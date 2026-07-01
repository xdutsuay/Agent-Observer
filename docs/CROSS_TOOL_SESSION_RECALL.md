# Cross-Tool Session Recall — Implementation Plan

**Feature:** Read and search past session transcripts from Claude Code, Codex CLI, and Cursor.
**Inspired by:** [Reference MCP](https://github.com/Kuberwastaken/reference) — but with our semantic search + relevance scoring advantage.
**Version target:** v0.6.0

---

## What this adds

Agents using Agent Memory MCP will be able to search past conversations from Claude Code, Codex, and Cursor — not just stored memories, but the actual session transcripts. This means an agent can recall "what did I try last time I worked on the auth module?" and get real answers from prior sessions, even if nobody explicitly saved a memory.

Our advantage over Reference: they do BM25 keyword search only. We re-rank with 384-dim semantic embeddings + our relevance scoring model, so results are actually good.

---

## Phase 1: Session transcript parsers (~3 hours)

### New module: `agent_memory_mcp/session_ingest/`

```
session_ingest/
├── __init__.py
├── models.py        # SessionTurn dataclass
├── parsers.py       # Claude Code + Codex JSONL parsers
├── adapters.py      # Tool adapter configs (globs, format, memory globs)
└── indexer.py       # File discovery + mtime cache
```

### `models.py` — Uniform data model

```python
@dataclass
class SessionTurn:
    source: str          # "claude_code" | "codex" | "cursor"
    role: str            # "user" | "assistant" | "system"
    text: str            # Normalized plain text (blocks collapsed)
    timestamp: float     # Unix epoch (from JSONL or file mtime)
    session_id: str      # Derived from file path
    project: str         # Parent directory name
    file_path: str       # Source JSONL path (for dedup)
    uuid: str | None     # Message UUID if available
```

### `parsers.py` — JSONL format handlers

**Claude Code format** (`~/.claude/projects/**/*.jsonl`):
Each line is JSON with `type` field: `"user"`, `"assistant"`, `"summary"`.
The `message.content` field contains content blocks:

```python
def parse_claude_code(line: dict) -> SessionTurn | None:
    # type field → role mapping
    # message.content → list of blocks
    # blocks_to_text() handles: text, thinking, tool_use, tool_result
    # Truncate tool_result to 600 chars, tool_input to 160 chars
    # Extract timestamp from message.created_at or file mtime
```

**Codex format** (`~/.codex/sessions/**/*.jsonl`):
Similar structure, parse `role` + content blocks.

**Cursor format** (`~/.cursor/projects/*/agent-transcripts/*.jsonl`):
Already partially handled by our existing `CursorTranscriptAdapter` — upgrade it to use proper JSONL parsing instead of raw line reading.

### `adapters.py` — Tool adapter registry

```python
@dataclass
class SessionAdapter:
    name: str
    session_globs: list[str]    # Where to find session files
    session_format: str          # "claude" | "codex" | "cursor"
    memory_globs: list[str]     # Config/memory files to also index

BUILTIN_ADAPTERS = [
    SessionAdapter(
        name="claude_code",
        session_globs=["~/.claude/projects/**/*.jsonl"],
        session_format="claude",
        memory_globs=["~/.claude/CLAUDE.md", "~/.claude/projects/**/CLAUDE.md"],
    ),
    SessionAdapter(
        name="codex",
        session_globs=["~/.codex/sessions/**/*.jsonl"],
        session_format="codex",
        memory_globs=["~/.codex/AGENTS.md"],
    ),
    SessionAdapter(
        name="cursor",
        session_globs=["~/.cursor/projects/*/agent-transcripts/*.jsonl"],
        session_format="cursor",
        memory_globs=[],
    ),
]
```

**Extensibility:** Users can add custom adapters via `~/.agent-memory/session_sources.toml`:

```toml
[[sources]]
name = "windsurf"
session_globs = ["~/.windsurf/sessions/**/*.jsonl"]
session_format = "claude"  # reuse parser if format matches
memory_globs = []
```

### `indexer.py` — File discovery with mtime cache

```python
class SessionIndexer:
    def __init__(self, adapters: list[SessionAdapter]):
        self._cache: dict[str, tuple[float, list[SessionTurn]]] = {}
        # key: file path, value: (mtime, parsed turns)

    def discover_sessions(self) -> list[SessionMeta]:
        """List all session files across all adapters."""
        # Expand globs, collect file paths + mtimes
        # Return metadata (path, source, project, turn_count, last_modified)

    def get_turns(self, file_path: str) -> list[SessionTurn]:
        """Parse a session file, using cache if mtime unchanged."""
        # Check mtime against cache → skip re-parse if unchanged
        # Parse with appropriate parser based on adapter format
        # Cache result
```

---

## Phase 2: Search integration (~2 hours)

### `session_ingest/search.py` — Hybrid session search

```python
class SessionSearch:
    def __init__(self, indexer: SessionIndexer, embedder):
        self.indexer = indexer
        self.embedder = embedder
        self._bm25_cache: dict[str, tuple[float, Any]] = {}

    def search(
        self,
        query: str,
        source: str | None = None,    # Filter by tool
        project: str | None = None,   # Filter by project
        since_days: int | None = None, # Recency filter
        limit: int = 20,
    ) -> list[SessionHit]:
        # 1. BM25 pass over all cached turns (fast, keyword-based)
        # 2. Take top-50 BM25 hits
        # 3. Embed query + hit texts with our 384-dim embedder
        # 4. Cosine re-rank: final = 0.4 * bm25_norm + 0.4 * cosine + 0.2 * recency
        # 5. Return top-N with scores

@dataclass
class SessionHit:
    turn: SessionTurn
    score: float
    bm25_score: float
    semantic_score: float
```

### Wire into `cross_repo.py`

Add optional `include_sessions: bool = False` param to `global_search()`. When true, also search session transcripts and merge results:

```python
def global_search(self, query, kinds=None, limit=20, include_sessions=False):
    memory_hits = self.storage.search(query, repo_id=None, kinds=kinds, limit=limit)
    if include_sessions:
        session_hits = self._session_search.search(query, limit=limit)
        # Merge and re-rank combined results
        # Tag each result with source="memory" or source="session"
    return combined[:limit]
```

---

## Phase 3: MCP tools + API (~2 hours)

### New MCP tools (add to `mcp_server.py`)

**`recall_sessions`** — Search past session transcripts
```
Params: query (str), source? (str), project? (str), since_days? (int), limit? (int)
Returns: Ranked session turns matching query, with scores and source info
```

**`list_sessions`** — List recent sessions across all tools
```
Params: source? (str), limit? (int)
Returns: Session metadata (tool, project, time range, turn count)
```

**`ingest_session_turn`** — Promote a session turn to permanent memory
```
Params: session_file (str), turn_index (int), kind (str), tags? (list)
Returns: Created memory ID
```
This is our unique value-add: agents find useful turns and save them permanently with full relevance scoring.

### New REST endpoints (add to `api.py`)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/sessions` | GET | List discovered sessions with metadata |
| `/api/sessions/search` | POST | Search session transcripts |
| `/api/sessions/ingest` | POST | Promote turn to permanent memory |

---

## Phase 4: Smart context enhancement (~1 hour)

### Update `smart_context` in `api.py`

When building context for a task, also search session transcripts and include the top 3-5 relevant past turns in the prompt fragment:

```python
# In smart_context endpoint:
session_hits = session_search.search(task, limit=5)
if session_hits:
    fragment += "\n\n## Relevant past sessions\n"
    for hit in session_hits:
        fragment += f"[{hit.turn.source} · {hit.turn.project}]: {hit.turn.text[:300]}\n"
```

### Index memory files

Also read and index `CLAUDE.md`, `AGENTS.md`, and `memory/*.md` files discovered via adapter `memory_globs`. These contain high-value context that agents wrote for themselves.

---

## Phase 5: Dashboard UI + tests (~2 hours)

### New page: `ui/client/src/pages/sessions.tsx`

- Source filter tabs (All / Claude Code / Codex / Cursor)
- Search bar with query + project filter
- Results with source badges, project paths, timestamps
- "Ingest to Memory" button on each turn → calls `/api/sessions/ingest`
- Session detail view: expand to see full conversation

### Update existing pages

- `dashboard.tsx`: Add "Sessions" stat card (count of discovered sessions), update tool count to 18
- `App.tsx`: Add `/sessions` route
- `layout.tsx`: Add "Sessions" nav link with MessageSquare icon

### Tests: `tests/test_session_ingest.py`

- Parser tests: valid Claude Code JSONL, valid Codex JSONL, malformed lines, empty files
- Indexer tests: glob expansion, mtime caching, re-parse on change
- Search tests: BM25 scoring, semantic re-rank, source/project filters
- Integration: ingest turn → verify it appears in memory search

---

## Integration points with existing code

| Existing file | Change |
|--------------|--------|
| `storage.py` | Import `SessionIndexer` + `SessionSearch`, init in `__init__` |
| `cross_repo.py` | Add `include_sessions` param to `global_search()` |
| `mcp_server.py` | Register 3 new tools (`recall_sessions`, `list_sessions`, `ingest_session_turn`) |
| `api.py` | Add 3 new endpoints + update `smart_context` |
| `adapters/cursor_transcript.py` | Deprecate in favor of `session_ingest` (keep for backward compat) |

## Dependencies

No new dependencies. We use:
- `pathlib.glob` for file discovery
- `json` for JSONL parsing
- Our existing `create_embedder()` for semantic search
- Standard `math` for BM25 IDF calculations

---

## Why this beats Reference

| Aspect | Reference | Agent Memory MCP |
|--------|-----------|-----------------|
| Search | BM25 keyword only | BM25 → semantic re-rank (384-dim) |
| Scoring | IDF + recency boost | Semantic × relevance × recency |
| Storage | In-memory only, rebuilt each run | SQLite + FTS5, persistent |
| Feedback | None | Thumbs up/down loop improves ranking |
| Ingest | Read-only recall | Promote turns to permanent memories |
| Dashboard | None | Full UI with search, browse, ingest |
| Smart context | None | Token-budgeted prompt fragments |
| Extensibility | TOML config | TOML config (same) |

---

## Estimated effort

| Phase | Hours | Priority |
|-------|-------|----------|
| 1. Parsers | 3 | P0 — foundation |
| 2. Search | 2 | P0 — core value |
| 3. MCP tools + API | 2 | P0 — user-facing |
| 4. Smart context | 1 | P1 — enhancement |
| 5. UI + tests | 2 | P1 — polish |
| **Total** | **~10 hours** | |
