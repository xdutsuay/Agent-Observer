# Agent Memory MCP

**Persistent, intelligent memory for AI coding agents** — works with Cursor, Claude Code, Claude Desktop, Windsurf, Zed, or any MCP client.

Your agent forgets everything between sessions. Agent Memory **remembers failures, decisions, and context per git repo**, learns which memories matter through relevance scoring, and surfaces the right context at the right time — automatically.

> **No cloud required.** Data stays on your machine in SQLite. SaaS version coming soon.

[![Python 3.10+](https://img.shields.io/badge/python-3.10%2B-blue)]()
[![MCP](https://img.shields.io/badge/MCP-compatible-green)]()
[![License: MIT](https://img.shields.io/badge/license-MIT-lightgrey)]()

---

## Dashboard

![Agent Memory Dashboard](docs/screenshots/dashboard.png)

*Real-time operator console — health scores, memory distribution, agent usage, failure hotspots, and MCP tool status.*

![Memory Browser](docs/screenshots/memory.png)

*Browse memories by workspace, view relevance scores, quality tiers, and give feedback to improve ranking.*

---

## Why use this?

| Problem | How Agent Memory solves it |
|--------|---------------------------|
| Agent repeats the same mistake | Failures are captured and surfaced automatically via relevance-ranked search |
| Context lost between sessions | Smart Context API packs the most relevant memories into a token budget |
| Search returns irrelevant results | Real semantic embeddings (384-dim) + blended scoring (semantic × relevance × recency) |
| No way to tell the system what's useful | Thumbs up/down feedback loop — memories that help get ranked higher |
| Noisy low-value memories pile up | Automatic noise classification demotes stale, never-accessed attempts |
| No visibility into agent behavior | Operator dashboard with usage logs, health scores, and failure tracking |
| Vendor lock-in | Standard MCP stdio — works with any IDE that supports MCP |

---

## Quick start

```bash
git clone https://github.com/xdutsuay/Agent-Observer.git
cd Agent-Observer

python3 -m venv .venv
source .venv/bin/activate
pip install -e ".[dev]"

./scripts/start.sh
```

Open **http://127.0.0.1:9000** — dashboard, memory browser, cross-repo search, agent usage.

### Connect to your IDE

Add to your MCP settings (Cursor, Claude Code, Claude Desktop, Windsurf):

```json
{
  "mcpServers": {
    "agent-memory": {
      "command": "agent-memory",
      "args": ["mcp", "--root", "~/agent_companion_data"]
    }
  }
}
```

Restart MCP in your IDE. The agent now has access to 15 memory tools.

---

## Features

### Relevance scoring

Not all memories are equal. The system tracks how often each memory is accessed, whether it was useful (via feedback), and how recent it is. Search results blend three signals:

```
final_score = 0.45 × semantic_similarity + 0.30 × relevance_score + 0.25 × recency
```

Memories that are frequently accessed and marked useful rise to the top. Stale, never-accessed attempts are automatically classified as noise and excluded from results.

### Smart Context API

Instead of dumping all memories, the Smart Context endpoint returns only what's relevant to your current task, packed into a token budget:

```bash
curl -X POST http://localhost:9000/api/v1/context/smart \
  -H "Content-Type: application/json" \
  -d '{"task": "Fix the auth timeout bug", "repo_id": "my-app", "max_tokens": 3000}'
```

Returns ranked memories + a ready-to-use system prompt fragment.

### Feedback loop

Agents (or humans) can report whether a memory was useful. This closes the loop — the system gets smarter over time:

```bash
curl -X POST http://localhost:9000/api/v1/feedback \
  -d '{"memory_id": "abc123", "useful": true}'
```

### Real semantic search

384-dimensional embeddings via `sentence-transformers` (all-MiniLM-L6-v2) with automatic fallback. The system auto-detects embedding dimension changes and reindexes on startup.

### Noise classification

Memories that have existed for 7+ days with zero accesses and kind `attempt` are automatically demoted to `noise` tier and excluded from search. They stay in the DB for audit but stop polluting results.

---

## MCP Tools (15 active)

### Core tools

| Tool | Purpose |
|------|---------|
| `remember` | Store a failure, decision, attempt, fact, or preference |
| `search_memory` | Semantic + keyword search with relevance-blended ranking |
| `get_repo_context` | Load failures + decisions + facts for a project at session start |
| `global_search` | Search across all tracked projects |
| `list_projects` | All tracked projects with metadata |
| `switch_project_context` | Full project context bundle |
| `get_pattern_report` | Health scores, trends, error categories |
| `get_related_memories` | Find related by content or time |
| `find_similar_failures` | Cross-project failure matching |
| `failure_hotspots` | Projects with the most unresolved failures |
| `mark_failure_resolved` | Resolve failure signatures |
| `forget` | Soft-delete memories |

### Intelligence tools (new in v0.5)

| Tool | Purpose |
|------|---------|
| `smart_context` | Task-specific memory retrieval with token budgeting |
| `memory_feedback` | Report if a memory was useful (closes the feedback loop) |
| `refresh_relevance` | Recompute all relevance scores + classify noise |

---

## Operator Dashboard

| Page | What it shows |
|------|---------------|
| **Dashboard** | Health score, memory stats, activity stream, failure hotspots, MCP tool status |
| **Projects** | Auto-discovered git repos with language/framework detection |
| **Memory** | Browse memories with relevance bars, quality tiers, feedback buttons |
| **Search** | Cross-repo search + Smart Context tab with copyable prompt fragments |
| **Patterns** | Recurring failures, error categories, health trends |
| **Timeline** | Real-time event stream |
| **Agent Usage** | Which IDE connected, what was queried, what was returned |
| **Config** | Live daemon config + disk breakdown |

---

## HTTP API

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/status` | GET | Server status, repo count |
| `/api/memories` | GET | List memory records |
| `/api/search` | POST | Hybrid search (single repo) |
| `/api/search/global` | POST | Cross-repo search |
| `/api/v1/context/smart` | POST | **Smart context with token budget** |
| `/api/v1/feedback` | POST | **Memory feedback (useful/not useful)** |
| `/api/v1/relevance/refresh` | POST | **Recompute relevance scores** |
| `/api/v1/context/generate` | POST | **Generate IDE context file** |
| `/api/patterns` | GET | Health score + error trends |
| `/api/usage/summary` | GET | Agent adoption metrics |
| `/api/disk-usage` | GET | Storage per project |
| `/ws/events` | WS | Real-time timeline events |

Full OpenAPI docs at **http://localhost:9000/docs** when the server is running.

---

## Architecture

```
IDE (Cursor / Claude Code / Windsurf / …)
        │ MCP stdio
        ▼
┌─────────────────────────────────────────────────┐
│  agent-memory                                   │
│  ┌─────────────┐  ┌──────────────────────────┐ │
│  │ MCP server  │  │ FastAPI + React dashboard │ │
│  │ (15 tools)  │  │ (9 pages)                │ │
│  └──────┬──────┘  └────────────┬─────────────┘ │
│         │                      │                │
│         └──────────┬───────────┘                │
│                    ▼                            │
│  Storage → SQLite (FTS5 + 384-dim vectors)      │
│              ├── memory_access_log (tracking)   │
│              ├── relevance scoring              │
│              └── noise classification           │
│                    ▲                            │
│  Filesystem watcher (watchdog) ─────────────── │
└─────────────────────────────────────────────────┘
        │
        ▼
~/agent_companion_data/
  ├── memory.db    (memories, embeddings, access logs)
  ├── usage.db     (agent interaction log)
  └── repos.json   (repo id → path map)
```

---

## Install options

```bash
# Core + tests
pip install -e ".[dev]"

# With real embeddings (recommended)
pip install -e ".[embeddings]"

# With LLM extraction
pip install -e ".[llm]"
```

## Run modes

```bash
./scripts/start.sh              # API + UI on :9000 (recommended)
./scripts/start.sh --dev        # API :9000 + Vite live UI :5000
./scripts/start.sh --api-only   # HTTP API only
./scripts/start.sh --mcp        # MCP stdio only (for IDE config)
./scripts/start.sh --port 8080  # Custom port
```

---

## Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `AGENT_MEMORY_LLM_PROVIDER` | `none` | `openai`, `anthropic`, `nvidia`, or `none` |
| `AGENT_MEMORY_DATA_ROOT` | `~/agent_companion_data` | SQLite + config location |
| `AGENT_MEMORY_EMBED_PROVIDER` | auto-detect | `local`, `openai`, or `hash` |
| `OPENAI_API_KEY` / `ANTHROPIC_API_KEY` | — | For optional extraction / API embeddings |

**LLM is optional.** All core features (memory, search, relevance, dashboard) work without any API key.

---

## Requirements

- **Python 3.10+**
- **Node.js 18+** (only to build the UI; `start.sh` handles this)
- **macOS / Linux** recommended

---

## Roadmap

### Shipped (v0.5.0)

- [x] Relevance scoring with access tracking + time decay
- [x] Memory feedback loop (thumbs up/down)
- [x] Smart Context API with token budgeting
- [x] Automatic noise classification
- [x] Real 384-dim semantic embeddings wired by default
- [x] Auto-reindex on embedding dimension change
- [x] Context file generation for IDE auto-injection
- [x] Dashboard UI with relevance bars, quality tiers, feedback buttons
- [x] Cross-repo search with blended scoring
- [x] 15 MCP tools (up from 12)

### Coming next — SaaS (v1.0)

- [ ] **Multi-tenancy** — per-org databases via Turso (libSQL), connection provider abstraction
- [ ] **Auth** — signup/login, JWT, API key generation for MCP/CLI
- [ ] **Cloud MCP proxy** — HTTP/SSE transport so any MCP client can connect remotely
- [ ] **Ingestion API** — structured event push from CI/CD, replacing the noisy file watcher
- [ ] **Memory distillation** — LLM pass to merge duplicates, extract facts from attempts, auto-tag
- [ ] **Team features** — shared project memories, attribution, access control, review queue
- [ ] **Billing** — Stripe integration, free/pro/team tiers

See [`docs/ENHANCEMENT_AND_SAAS_ROADMAP.md`](docs/ENHANCEMENT_AND_SAAS_ROADMAP.md) for the full spec.

---

## Tests

```bash
source .venv/bin/activate
pytest -q
```

---

## Contributing

Issues and PRs welcome at [github.com/xdutsuay/Agent-Observer](https://github.com/xdutsuay/Agent-Observer).

---

## License

MIT
