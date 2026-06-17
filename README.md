# Agent Memory MCP

**Local, repo-scoped memory for AI coding agents** — works with Cursor, Claude Code, Claude Desktop, Zed, or any MCP client.

Your agent forgets context between sessions. This project **remembers failures, decisions, and attempts per git repo**, watches your workspace passively, and exposes memory through **MCP tools** plus an **operator dashboard**.

> **No cloud required.** No API key required for core features. Data stays on your machine in SQLite.

[![Python 3.10+](https://img.shields.io/badge/python-3.10%2B-blue)]()
[![MCP](https://img.shields.io/badge/MCP-compatible-green)]()
[![License: MIT](https://img.shields.io/badge/license-MIT-lightgrey)]()

---

## Why use this?

| Problem | How Agent Memory helps |
|--------|-------------------------|
| Agent repeats the same mistake | Failures are captured from logs + `remember` and surfaced via `get_repo_context` |
| Context lost between chats | MCP `search_memory` + `inject_memory_context` preload prior work |
| No visibility into what agents actually use | **Agent Usage** page logs MCP/HTTP queries and responses |
| Scattered notes across tools | One SQLite store per machine, scoped by git repo |
| Vendor lock-in | Standard MCP stdio — not tied to a single IDE |

---

## Quick start (5 minutes)

```bash
git clone https://github.com/xdutsuay/Agent-Observer.git
cd Agent-Observer

python3 -m venv .venv
source .venv/bin/activate
pip install -e ".[dev]"

./scripts/start.sh
```

Open **http://127.0.0.1:9000** — dashboard, memory browser, agent usage, disk usage.

`start.sh` creates the venv if needed, installs deps, builds the UI on first run, and starts the daemon with the filesystem watcher **on** by default.

### Connect to Cursor (recommended)

1. Ensure `agent-memory` is on your PATH (venv activated), or use the full path to `.venv/bin/agent-memory`.
2. Add to Cursor MCP settings (or copy [`mcp.json.example`](mcp.json.example)):

```json
{
  "mcpServers": {
    "agent-memory": {
      "command": "agent-memory",
      "args": ["mcp", "--root", "~/agent_companion_data"],
      "env": {
        "AGENT_MEMORY_LLM_PROVIDER": "none"
      }
    }
  }
}
```

3. Restart MCP in Cursor. Ask the agent to call `get_repo_context` or `search_memory` for your project path.

### Connect to Claude Code / Claude Desktop

Use the same MCP block above in your client's MCP config. The server speaks **stdio MCP** — no HTTP port needed for the agent integration.

---

## What you get

### MCP tools (agent-facing)

| Tool | Use when |
|------|----------|
| `remember` | Store a failure, decision, attempt, fact, or preference |
| `search_memory` | Find relevant past context for the current task |
| `get_repo_context` | Bundle failures + decisions + attempts for a repo |
| `inject_memory_context` | Prompt: preload memory before starting work |
| `global_search` | Search across all tracked projects |
| `list_projects` / `switch_project_context` | Multi-repo workspaces |
| `find_similar_failures` / `failure_hotspots` | Cross-repo failure intelligence |
| `mark_failure_resolved` / `forget` | Curate memory over time |

### Operator dashboard (human-facing)

| Page | Purpose |
|------|---------|
| **Dashboard** | Health, activity, projects, disk + usage summaries |
| **Projects** | Auto-discovered git repos under `~/localcode` |
| **Memory** | Browse and search stored memories |
| **Agent Usage** | Which IDE connected, what was queried, what was returned |
| **Configuration** | Live daemon config + disk breakdown |
| **Timeline / Patterns** | Events and failure trends |

### Passive watcher (zero agent effort)

With the daemon running, the watcher observes configured paths, ingests log errors into **failures** and code churn into **attempts** — no manual note-taking.

---

## Requirements

- **Python 3.10+**
- **Node.js 18+** (only to build the UI; `start.sh` handles this)
- **macOS / Linux** recommended (`du` used for workspace disk sizing)
- **Optional:** OpenAI, Anthropic, or NVIDIA NIM API key for background fact extraction

---

## Install options

```bash
# Core + tests
pip install -e ".[dev]"

# Optional LLM extraction
pip install -e ".[llm]"

# Optional better local embeddings
pip install -e ".[embeddings]"
```

---

## Run modes

```bash
./scripts/start.sh              # API + UI on :9000 (recommended)
./scripts/start.sh --dev        # API :9000 + Vite live UI :5000
./scripts/start.sh --api-only   # HTTP API only
./scripts/start.sh --mcp        # MCP stdio only (for IDE config)
./scripts/start.sh --port 8080  # Custom port
```

Manual equivalent:

```bash
agent-memory serve --root ~/agent_companion_data --ui --port 9000
agent-memory mcp --root ~/agent_companion_data
```

---

## Environment variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `AGENT_MEMORY_LLM_PROVIDER` | `none` | `openai`, `anthropic`, `nvidia`, or `none` |
| `AGENT_MEMORY_USE_HERMES_AUTH` | `1` | When `nvidia`, read key from `~/.hermes/auth.json` if unset |
| `AGENT_MEMORY_NVIDIA_API_KEY` | — | NVIDIA NIM key |
| `AGENT_MEMORY_DATA_ROOT` | `~/agent_companion_data` | SQLite + config location |
| `OPENAI_API_KEY` / `ANTHROPIC_API_KEY` | — | For optional extraction |

**LLM is optional.** Watcher, MCP, search, and dashboard work without any API key.

---

## Data on disk

```
~/agent_companion_data/
  memory.db       # Memories, FTS5 search, embeddings
  usage.db        # Agent usage / MCP interaction log
  repos.json      # repo id → path map
  agent-memory/   # Legacy markdown (migrated once)
```

**Disk usage API:** `GET /api/disk-usage` — per-project memory size and workspace size.

---

## HTTP API (integrations)

| Endpoint | Purpose |
|----------|---------|
| `GET /api/status` | Watcher, repo count, LLM provider |
| `GET /api/memories` | List memory records |
| `POST /api/search` | Hybrid search |
| `GET /api/usage/summary` | Agent adoption metrics |
| `GET /api/usage/interactions` | Query/response audit log |
| `GET /api/disk-usage` | Storage per project + overall |
| `WS /ws/events` | Real-time timeline events |

Full OpenAPI docs when the daemon is running: **http://127.0.0.1:9000/docs**

---

## Architecture

```
IDE (Cursor / Claude Code / …)
        │ MCP stdio
        ▼
┌───────────────────────────────────────────┐
│  agent-memory                             │
│  ┌─────────────┐  ┌──────────────────┐  │
│  │ MCP server  │  │ FastAPI + UI     │  │
│  └──────┬──────┘  └────────┬─────────┘  │
│         │                  │              │
│         └────────┬─────────┘              │
│                  ▼                        │
│         Storage → SQLite (FTS5 + vectors) │
│                  ▲                        │
│         Filesystem watcher (watchdog)     │
└───────────────────────────────────────────┘
        │
        ▼
~/agent_companion_data/
```

---

## Adoption checklist

- [ ] Run `./scripts/start.sh` and confirm **http://127.0.0.1:9000/api/status** shows `running: true`
- [ ] Add MCP config to your IDE and verify tools appear
- [ ] Open a repo under `~/localcode` (or your watch paths) — it should appear in **Projects**
- [ ] Call `remember` or `search_memory` from an agent — check **Agent Usage** for the log entry
- [ ] (Optional) Set `AGENT_MEMORY_LLM_PROVIDER` for automatic fact extraction

---

## Cursor hook (optional)

See [`docs/cursor-hook.example.json`](docs/cursor-hook.example.json) to POST memory after file edits. MCP `remember` is simpler when the agent is already connected.

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
