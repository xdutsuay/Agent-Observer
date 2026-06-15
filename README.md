# Agent Memory MCP

IDE-agnostic **repo-scoped agent memory**: passive workspace observation, structured SQLite storage, semantic search, and MCP tools for any compatible client (Cursor, Claude Desktop, Zed, etc.).

## Features

- **Triple-buffer model**: failures, decisions, attempts (+ facts/preferences from LLM extraction)
- **Filesystem watcher**: captures log errors and code churn automatically
- **SQLite + FTS5 + local vectors**: hybrid search without a cloud vector DB
- **MCP tools**: `remember`, `search_memory`, `get_repo_context`, `mark_failure_resolved`, `forget`
- **Operator UI**: dashboard and memory browser wired to the same backend

## Install

```bash
cd agent_memory_mcp
pip install -e ".[dev]"
# Optional LLM extraction:
pip install -e ".[llm]"
```

## MCP (primary)

Copy [`mcp.json.example`](mcp.json.example) into your MCP client config:

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

### Tools

| Tool | Description |
|------|-------------|
| `remember` | Store memory for a project path |
| `search_memory` | Keyword + semantic search |
| `get_repo_context` | Failures, decisions, attempts bundle |
| `mark_failure_resolved` | Clear recurring failure signature |
| `forget` | Soft-delete by id or signature |

### Prompt

- `inject_memory_context` — pass `path` to preload repo memory into a session

## Daemon + UI

```bash
# API on :9000 — watcher ON by default, scans ~/localcode + Cursor projects for git repos
AGENT_MEMORY_LLM_PROVIDER=nvidia agent-memory serve --root ~/agent_companion_data

# With built UI (npm run build in ui/ first)
agent-memory serve --ui

# Dev UI (proxies /api → :9000)
cd ui && npm run dev:client
```

## Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `AGENT_MEMORY_LLM_PROVIDER` | `none` | `openai`, `anthropic`, `nvidia`, or `none` |
| `AGENT_MEMORY_USE_HERMES_AUTH` | `1` | When `nvidia`, read API key from `~/.hermes/auth.json` if env unset |
| `AGENT_MEMORY_NVIDIA_API_KEY` | — | NVIDIA NIM key (or `NVIDIA_API_KEY` / `NVAPI_API_KEY`) |
| `AGENT_MEMORY_NVIDIA_BASE_URL` | NVIDIA NIM v1 | OpenAI-compatible base URL |
| `AGENT_MEMORY_NVIDIA_MODEL` | `nvidia/nemotron-3-super-120b-a12b` | Model id for extraction |
| `OPENAI_API_KEY` | — | For extraction when provider=openai |
| `ANTHROPIC_API_KEY` | — | For extraction when provider=anthropic |

**NVIDIA via Hermes:** set `AGENT_MEMORY_LLM_PROVIDER=nvidia` and keep Hermes logged in; the daemon reuses the NVIDIA credential from `~/.hermes/auth.json` (no need to duplicate the key in `.env`).

## Data layout

```
~/agent_companion_data/
  memory.db          # SQLite source of truth
  repos.json         # repo id → path map
  agent-memory/      # legacy markdown (migrated once)
```

## Cursor hook (optional)

POST to the daemon after edits:

```bash
curl -s -X POST "http://127.0.0.1:9000/api/memory/$(agent-memory resolve 2>/dev/null || echo default)/decisions" \
  -H "Content-Type: application/json" \
  -d '{"text":"Completed refactor of auth module"}'
```

Or use MCP `remember` from the agent directly (recommended).

## Architecture

```
Watcher / MCP / HTTP → Storage → MemoryEngine (SQLite)
                              ↓
                    Optional LLM extractor → facts
```

## Tests

```bash
pytest -q
```

## License

MIT
