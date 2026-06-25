# Agent Memory MCP — Major Enhancement & SaaS Roadmap

**Version:** Draft v1 | **Date:** 2026-06-25 | **Status:** RFC

---

## Executive Summary

Agent Memory MCP (v0.4.0) has the infrastructure for storing agent memories — SQLite engine, FTS5 search, file watcher, MCP tools, REST API, dashboard UI. But in practice, it doesn't enhance the agent-LLM-human loop because of three compounding problems:

1. **Garbage in:** The watcher stores every file save as a low-value "attempt" (e.g., "Initial scan of dashboard.tsx"). Signal-to-noise ratio is near zero.
2. **Retrieval is weak:** The default 64-dim hash embedder produces near-random similarity scores (0.17–0.52 observed). Search returns irrelevant results.
3. **The agent never asks:** Memory is purely pull-based. LLMs don't know what they don't know — they won't call `search_memory` unprompted.

This document specifies three core enhancements to fix these problems, plus the architecture changes needed to ship as a SaaS product.

---

## Part 1: Core Enhancements

### Enhancement 1: Relevance Scoring System

**Problem:** All memories are treated equally. A critical production decision from last week ranks the same as "Added: {disk && (..." from a watcher scan. There's no mechanism for the system to learn which memories matter.

**Solution:** Track memory access patterns and blend relevance signals into search ranking.

#### 1.1 Schema Changes

```sql
-- New table: tracks every time a memory is surfaced
CREATE TABLE memory_access_log (
    id TEXT PRIMARY KEY,
    memory_id TEXT NOT NULL,
    access_type TEXT NOT NULL,  -- 'search_hit', 'context_inject', 'explicit_read'
    query_text TEXT,            -- the query that surfaced this memory
    session_id TEXT,
    host_ide TEXT,
    was_useful INTEGER,         -- NULL=unknown, 1=used, 0=ignored (feedback)
    created_at TEXT NOT NULL,
    FOREIGN KEY (memory_id) REFERENCES memories(id)
);

CREATE INDEX idx_access_log_memory ON memory_access_log(memory_id, created_at DESC);

-- New columns on memories table
ALTER TABLE memories ADD COLUMN access_count INTEGER DEFAULT 0;
ALTER TABLE memories ADD COLUMN last_accessed TEXT;
ALTER TABLE memories ADD COLUMN relevance_score REAL DEFAULT 0.0;
ALTER TABLE memories ADD COLUMN quality_tier TEXT DEFAULT 'unrated';
-- quality_tier: 'high', 'medium', 'low', 'noise', 'unrated'
```

#### 1.2 Access Tracking

Every time a memory appears in search results or context injection:

```python
def record_access(self, memory_id: str, access_type: str, query: str,
                  session_id: str = None, host_ide: str = None):
    """Called internally whenever a memory is surfaced to an agent."""
    now = _utc_now()
    self._conn.execute(
        """INSERT INTO memory_access_log
           (id, memory_id, access_type, query_text, session_id, host_ide, created_at)
           VALUES (?, ?, ?, ?, ?, ?, ?)""",
        (str(uuid.uuid4()), memory_id, access_type, query, session_id, host_ide, now)
    )
    self._conn.execute(
        """UPDATE memories
           SET access_count = access_count + 1, last_accessed = ?
           WHERE id = ?""",
        (now, memory_id)
    )
    self._conn.commit()
```

#### 1.3 Relevance Score Computation

Blend multiple signals with time decay:

```python
def compute_relevance(self, memory_id: str) -> float:
    """
    Score = weighted blend of:
      - access_frequency:  how often this memory gets surfaced (0-1)
      - recency:           time since creation, with half-life decay (0-1)
      - access_recency:    time since last accessed (0-1)
      - usefulness:        ratio of positive feedback to total accesses (0-1)
      - kind_weight:       decisions/facts > failures > attempts (static)
    """
    KIND_WEIGHTS = {
        'decision': 1.0, 'fact': 0.9, 'preference': 0.85,
        'failure': 0.7, 'attempt': 0.3
    }

    WEIGHTS = {
        'access_frequency': 0.25,
        'recency': 0.20,
        'access_recency': 0.20,
        'usefulness': 0.20,
        'kind_weight': 0.15,
    }

    # ... compute each signal, apply half-life decay (7 days for recency)
    # Return weighted sum clamped to [0, 1]
```

#### 1.4 Search Ranking Integration

Modify `_search_unlocked` in `engine/db.py`:

```python
# Current: final_score = semantic_similarity OR fts_rank
# New:
final_score = (
    0.50 * semantic_similarity +    # How well does this match the query?
    0.30 * relevance_score +         # How valuable is this memory historically?
    0.20 * recency_score             # How recent is this memory?
)
```

This replaces the current approach where FTS rank and cosine similarity are independent scores that don't blend.

#### 1.5 New MCP Tool: Feedback

```python
Tool(
    name="memory_feedback",
    description="Report whether a retrieved memory was useful in the current task.",
    inputSchema={
        "type": "object",
        "properties": {
            "memory_id": {"type": "string"},
            "useful": {"type": "boolean"},
            "context": {"type": "string", "description": "Brief note on why/why not"},
        },
        "required": ["memory_id", "useful"],
    },
)
```

When an LLM uses a memory and it leads to a successful outcome (no subsequent error, user accepts suggestion), the feedback can be recorded. This closes the loop.

#### 1.6 Automatic Noise Detection

Memories that have existed for 7+ days with zero accesses and kind="attempt" get auto-classified as `quality_tier='noise'`. They're excluded from search results by default but kept in the DB for audit.

```python
def auto_classify_noise(self, max_age_days: int = 7):
    """Periodic job: demote stale, never-accessed attempts to noise."""
    cutoff = (datetime.now(timezone.utc) - timedelta(days=max_age_days)).isoformat()
    self._conn.execute(
        """UPDATE memories SET quality_tier = 'noise'
           WHERE kind = 'attempt' AND access_count = 0
           AND created_at < ? AND quality_tier = 'unrated'""",
        (cutoff,)
    )
    self._conn.commit()
```

---

### Enhancement 2: Real Embeddings as Default

**Problem:** `_simple_embedding` (64-dim bag-of-words hash) is the active default. The `embeddings.py` module has `LocalEmbedder` and `APIEmbedder` but they're not wired in unless the user sets `AGENT_MEMORY_EMBED_PROVIDER`. The `MemoryEngine.__init__` doesn't call `create_embedder()`.

**Current code path:**
```
MemoryEngine.__init__(embedder=None)
  → self._embedder = None
  → _embed() falls through to _simple_embedding()
```

**Fix:**

#### 2.1 Wire `create_embedder()` into `MemoryEngine`

In `engine/db.py`:

```python
class MemoryEngine:
    def __init__(self, root: Path, embedder: Optional[Any] = None):
        ...
        if embedder is None:
            from .embeddings import create_embedder
            embedder = create_embedder()  # auto-detect: local → api → hash
        self._embedder = embedder
```

This single change activates `LocalEmbedder` (384-dim `all-MiniLM-L6-v2`) if `sentence-transformers` is installed, which it will be if the user installed with `pip install agent-memory[all]` or `pip install agent-memory[embeddings]`.

#### 2.2 Make Embeddings a Required Dependency

In `pyproject.toml`, move `sentence-transformers` from optional to required:

```toml
dependencies = [
    "fastapi>=0.95",
    "uvicorn[standard]>=0.22",
    "pydantic>=2.0",
    "watchdog>=3.0",
    "psutil>=5.9",
    "mcp>=1.0",
    "httpx>=0.27",
    "sentence-transformers>=2.2",  # MOVE FROM OPTIONAL
]
```

For SaaS, use `APIEmbedder` with OpenAI's `text-embedding-3-small` ($0.02/1M tokens) — cheaper than hosting a model. The local embedder stays as the self-hosted option.

#### 2.3 Auto-Reindex on Upgrade

When the DB has 64-dim embeddings and the new embedder produces 384-dim, detect the mismatch and reindex:

```python
def _check_embedding_dimension(self):
    """Reindex if stored embeddings don't match current provider dimension."""
    row = self._conn.execute(
        "SELECT embedding_json FROM memory_embeddings LIMIT 1"
    ).fetchone()
    if row:
        stored_dim = len(json.loads(row["embedding_json"]))
        if stored_dim != self._embedder.dim:
            print(f"[MEMORY] Embedding dimension changed {stored_dim}→{self._embedder.dim}, reindexing...")
            self.reindex_embeddings()
```

---

### Enhancement 3: Automatic Context Injection

**Problem:** The `inject_memory_context` MCP prompt exists but no MCP client auto-invokes prompts. The agent has to explicitly call `get_repo_context` or `search_memory`. In practice, it never does unless the user asks.

**Solution:** Multiple injection strategies, from simplest to most sophisticated.

#### 3.1 Strategy A: MCP Sampling (MCP spec feature)

The MCP spec supports server-initiated sampling requests. When a client connects and lists tools, the server can proactively send a sampling message with context:

```python
@server.on_client_connected()
async def on_connect(session):
    # Detect which repo the client is working in
    # (from client params, working directory, or recent activity)
    repo_id = infer_active_repo(session)
    if repo_id:
        context = store.get_repo_context(repo_id)
        top_memories = get_top_relevant_memories(repo_id, limit=5)
        # Push context to client via sampling
        await session.send_sampling_request(
            messages=[{
                "role": "system",
                "content": format_memory_context(context, top_memories)
            }]
        )
```

**Limitation:** Most MCP clients don't support server-initiated sampling yet.

#### 3.2 Strategy B: CLAUDE.md / .cursorrules Generation

The most practical approach today. Auto-generate a `MEMORY_CONTEXT.md` file in the project root that IDEs load automatically:

```python
def generate_context_file(self, repo_id: str, project_path: Path):
    """Generate a context file that IDEs auto-load."""
    ctx = self.engine.get_repo_context(repo_id)
    top = self.engine.search("", repo_id=repo_id, limit=10)  # top by relevance
    top = sorted(top, key=lambda m: m.get('relevance_score', 0), reverse=True)[:5]

    content = f"""# Agent Memory Context (auto-generated)
# Last updated: {_utc_now()}
# Do not edit — regenerated by agent-memory-mcp

## Key Decisions
{ctx['decisions']}

## Known Issues
{ctx['failures']}

## Project Facts
{ctx['facts']}

## High-Relevance Memories
{chr(10).join(f'- {m["content"][:200]}' for m in top if m.get('quality_tier') != 'noise')}
"""
    # Write to .agent-memory/context.md
    out = project_path / ".agent-memory" / "context.md"
    out.parent.mkdir(exist_ok=True)
    out.write_text(content)
    return out
```

Then instruct users to add to their `.cursorrules` or `CLAUDE.md`:
```
Read .agent-memory/context.md at the start of every session for project context.
```

#### 3.3 Strategy C: Tool-Hint in Tool Descriptions (Immediate Win)

Modify the `remember` and `search_memory` tool descriptions to hint that the agent should use them proactively:

```python
Tool(
    name="get_repo_context",
    description=(
        "Get failures, decisions, facts, and recent attempts for a project. "
        "IMPORTANT: Call this at the START of every coding session to load "
        "relevant project context before making changes."
    ),
    ...
)
```

This is the cheapest change and works with every MCP client today.

#### 3.4 Strategy D: Smart Context API (SaaS)

For the SaaS version, expose a `/api/context/smart` endpoint that returns only the most relevant memories for a given task description:

```python
@app.post("/api/context/smart")
def smart_context(task: str, repo_id: str, max_tokens: int = 2000):
    """Return the optimal memory context for a specific task."""
    # 1. Search for task-relevant memories
    relevant = store.search(task, repo_id=repo_id, limit=20)

    # 2. Filter by quality tier (exclude noise)
    relevant = [m for m in relevant if m.get('quality_tier') != 'noise']

    # 3. Rank by blended relevance score
    relevant.sort(key=lambda m: m.get('relevance_score', 0), reverse=True)

    # 4. Pack into token budget
    context_parts = []
    token_count = 0
    for mem in relevant:
        est_tokens = len(mem['content'].split()) * 1.3
        if token_count + est_tokens > max_tokens:
            break
        context_parts.append(mem)
        token_count += est_tokens

    return {"context": context_parts, "token_estimate": token_count}
```

---

## Part 2: SaaS Architecture Roadmap

### Phase 1: Foundation (Weeks 1–4)

**Goal:** Multi-tenant cloud deployment with auth.

#### Database: SQLite → Turso (libSQL)

Why Turso: SQLite-compatible (minimal code changes), edge-replicated, per-tenant databases, generous free tier.

```
Current: Storage → MemoryEngine → sqlite3.connect("memory.db")
Target:  Storage → MemoryEngine → libsql_client.connect(tenant_db_url)
```

The SQL schema stays identical. The FTS5 and embedding tables work unchanged. The main change is connection management and per-tenant DB URLs.

**Migration path:**
1. Abstract the connection in `MemoryEngine` behind a `ConnectionProvider` interface
2. `LocalConnectionProvider` wraps current sqlite3 logic (self-hosted mode)
3. `TursoConnectionProvider` uses libsql-client (SaaS mode)
4. Feature flag: `AGENT_MEMORY_MODE=local|cloud`

#### Authentication

```
POST /api/auth/signup     → create org + first user
POST /api/auth/login      → JWT
POST /api/auth/api-key    → generate API key for MCP/CLI usage

Headers: Authorization: Bearer <jwt_or_api_key>
```

Every MCP connection and HTTP request is scoped to an org. API keys are what users put in their MCP config.

#### Multi-Tenancy Model

```
org
  ├── members (users with roles: owner, admin, member)
  ├── api_keys
  └── projects (repos)
       └── memories
```

Each org gets its own Turso database. No cross-org queries possible by design.

### Phase 2: Ingestion Overhaul (Weeks 3–6)

**Goal:** Replace the local file watcher with a structured event ingestion API.

#### Kill the Watcher, Build an Ingestion API

The file watcher is the root cause of noise. It watches FS events indiscriminately and creates low-value "attempt" records. For SaaS, agents and CI pipelines should push structured events.

```
POST /api/v1/ingest
{
    "repo_id": "my-project",
    "events": [
        {
            "kind": "decision",
            "content": "Chose Postgres over DynamoDB for the user table because we need complex joins for the reporting feature",
            "metadata": {"file": "src/db/schema.sql", "author": "claude"}
        },
        {
            "kind": "failure",
            "content": "TypeError: Cannot read property 'id' of undefined at UserService.getProfile (user-service.ts:42)",
            "metadata": {"stack": "...", "resolved": false}
        }
    ]
}
```

#### IDE Plugin SDKs

Lightweight plugins for each IDE that capture high-value events:

- **Error events:** When the agent encounters an error (stack trace, resolution attempt)
- **Decision events:** When the agent makes an architectural choice (explicit `remember` call)
- **Fact events:** Project-specific knowledge (endpoints, conventions, team preferences)
- **Session summaries:** At session end, summarize what was done and learned

```python
# SDK usage in a Claude Code hook:
from agent_memory import MemoryClient

memory = MemoryClient(api_key="am_...", project="my-app")
memory.remember("decision", "Switched from REST to gRPC for internal services due to latency requirements")
```

NOT captured automatically: file saves, line-by-line diffs, "Initial scan of X" noise.

### Phase 3: Intelligence Layer (Weeks 5–8)

**Goal:** The system gets smarter over time.

#### Embedding Service

Centralized embedding via OpenAI `text-embedding-3-small`:
- $0.02 per 1M tokens — negligible cost
- Consistent quality across all tenants
- No model hosting burden
- Cache embeddings aggressively (content-hash → embedding)

#### Relevance Feedback Loop

Deploy Enhancement 1 (relevance scoring). The access tracking table lives alongside memories in the tenant DB. Relevance scores are recomputed on a schedule (hourly for active projects).

#### Memory Distillation

Periodic LLM pass over raw memories to:
1. Merge duplicates ("same error reported 5 times" → single entry with count)
2. Extract facts from attempts ("from 15 file changes, extract: migrated auth from JWT to session cookies")
3. Auto-tag with categories
4. Generate 1-line summaries

This is where the LLM extractor (currently off by default) becomes a core product feature. Run it server-side on the SaaS, charged per-memory or included in tier.

### Phase 4: Smart Context API (Weeks 7–10)

**Goal:** The primary product surface — automatic context injection.

#### Context Endpoint

```
POST /api/v1/context
{
    "project": "my-app",
    "task": "Fix the authentication timeout bug in the login flow",
    "max_tokens": 3000,
    "exclude_noise": true
}

Response:
{
    "memories": [...],
    "system_prompt_fragment": "You are working on my-app. Known issues: ...",
    "token_count": 2847
}
```

This is the key SaaS value prop. The agent (or IDE plugin) calls this at session start and injects the response into the system prompt. The user never has to manually ask for context.

#### MCP Proxy Mode

For users who want to keep using MCP, offer a cloud-hosted MCP server:

```json
{
    "mcpServers": {
        "agent-memory": {
            "url": "https://api.agentmemory.dev/mcp",
            "headers": {"Authorization": "Bearer am_..."}
        }
    }
}
```

This works with Claude Code, Cursor, Windsurf — any MCP client. The tools are the same, but backed by the cloud DB with real embeddings and relevance scoring.

### Phase 5: Team Features (Weeks 9–12)

**Goal:** Multi-user collaboration on shared memory.

- **Shared project memories:** Team members' agents contribute to the same memory pool
- **Memory attribution:** Who/what created each memory (agent, human, CI)
- **Access control:** Project-level permissions (read/write/admin)
- **Memory review queue:** Human approval for auto-generated facts before they enter the shared pool
- **Audit log:** Who accessed what, when, for compliance

---

## Part 3: Pricing Model

### Free Tier
- 1 project
- 500 memories
- Hash embeddings only (no semantic search)
- Local MCP mode
- Community support

### Pro ($19/month per user)
- Unlimited projects
- 50,000 memories
- Real embeddings (text-embedding-3-small)
- Smart context API
- Cloud MCP proxy
- Relevance scoring
- Memory distillation (100 distillations/month)

### Team ($39/month per user)
- Everything in Pro
- Shared team memories
- Memory review queue
- Audit log
- SSO / SAML
- Priority support
- 500 distillations/month

### Enterprise (Custom)
- Self-hosted option
- Custom embedding models
- SLA
- Dedicated support
- Unlimited distillation
- On-prem deployment

---

## Part 4: Implementation Priority

### Must-do first (the order matters)

| Priority | Enhancement | Impact | Effort | Why This Order |
|----------|------------|--------|--------|----------------|
| **P0** | Wire `create_embedder()` into MemoryEngine | Fixes search quality | 1 hour | Everything downstream depends on good retrieval |
| **P1** | Tool description hints (Strategy C) | Agents start calling tools | 30 min | Zero-cost way to increase usage today |
| **P2** | Relevance scoring tables + access tracking | Memories improve over time | 1 week | Needs real embeddings first to be meaningful |
| **P3** | Noise auto-classification | Clean up existing garbage | 2 days | Needs access tracking data to classify |
| **P4** | Context file generation (Strategy B) | Auto-injection for local users | 3 days | Needs clean, ranked memories |
| **P5** | Smart context API | SaaS product surface | 1 week | Needs all of the above |

### The 1-hour fix that changes everything

In `storage.py`, line 22:

```python
# Current:
self.engine = MemoryEngine(self.root)

# Change to:
from .engine.embeddings import create_embedder
self.engine = MemoryEngine(self.root, embedder=create_embedder())
```

This single line activates real embeddings for every user who has `sentence-transformers` installed. Search quality jumps from "random noise" to "actually useful" immediately.

---

## Part 5: Success Metrics

How to know these enhancements are working:

| Metric | Current | Target (30 days) | Target (90 days) |
|--------|---------|-------------------|-------------------|
| Search relevance (top-3 precision) | ~10% | 60% | 80% |
| Memory access rate (% memories ever retrieved) | <5% | 30% | 50% |
| Agent-initiated context loads per session | 0 | 0.5 | 1.0+ |
| Noise ratio (attempt/total memories) | ~90% | 40% | 20% |
| Positive feedback rate | N/A | 40% | 60% |
| Paid conversions (SaaS) | 0 | 50 | 500 |

---

## Appendix: Current Architecture Reference

```
agent_memory_mcp/
├── engine/
│   ├── db.py              # MemoryEngine: SQLite + FTS5 + embeddings
│   ├── embeddings.py       # HashEmbedder, LocalEmbedder, APIEmbedder
│   └── migration.py        # Legacy markdown → SQLite migration
├── mcp_server.py           # MCP stdio server (12 tools, 1 prompt)
├── api.py                  # FastAPI HTTP server + file watcher
├── storage.py              # Storage facade over MemoryEngine
├── watcher.py              # watchdog-based file system observer
├── extractor.py            # Optional LLM extraction (off by default)
├── pattern_detector.py     # Recurring failure/pattern analysis
├── cross_repo.py           # Cross-repository search
├── correlator.py           # Memory relationship finding
├── project_registry.py     # Project metadata tracking
├── usage_log.py            # MCP/HTTP interaction logging
└── ui/                     # React dashboard
```

**Key data files:**
- `memory.db` — Main SQLite database (memories, embeddings, failure_signatures)
- `usage.db` — Interaction/usage tracking
- `repos.json` — Repo ID → path mapping
