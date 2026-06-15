# Low-Level Plan (LLP): Agent Memory MCP — Future Enhancements

> **Project:** agent_memory_mcp v0.3.0  
> **GitHub:** [xdutsuay/Agent-Observer](https://github.com/xdutsuay/Agent-Observer) (outdated — MVP only, 9 commits)  
> **Status:** Draft — do not implement without explicit phase approval.  
> **Created:** 2026-06-15

---

## Current State Summary

Agent Memory MCP is an IDE-agnostic, repo-scoped memory system with: triple-buffer model (failures/decisions/attempts + facts/preferences), filesystem watcher for passive observation, SQLite + FTS5 + local bag-of-words vectors for hybrid search, MCP tools (remember, search_memory, get_repo_context, mark_failure_resolved, forget), HTTP API + React dashboard UI, and optional LLM extraction (OpenAI/Anthropic/NVIDIA).

### GitHub Sync Status

The GitHub repo (`xdutsuay/Agent-Observer`) is significantly behind the local project. GitHub has the MVP (basic HTTP API + stdio adapter). Local has grown to v0.3.0 with: engine/db.py (full SQLite+FTS5+embeddings), watcher.py (filesystem observer), extractor.py (LLM extraction), adapters/ (Antigravity, Cursor transcript), log_parser.py, process_detector.py, metrics_collector.py, activity_score.py, repo_discovery.py, and a React UI. **No `.git` directory exists locally** — the repo was not cloned from GitHub; it evolved independently. A full re-push is needed.

### Multi-Project Context

The developer actively works on **30+ projects** in `~/localcode/`, including: LLMLAB, Mainframe_cobol, MoneyManager, career-ops, cerebral, doom_air, fast-job-apply, kaustubhtripathi.com, larql, linkedin-growth-engine, shatrunZ, stockswing, and more. Agent Memory MCP's primary value is giving any agent (Cursor, Claude, Zed) instant context about whichever project the developer is touching — failures seen before, decisions already made, patterns across repos. The enhancement plan below prioritizes **multi-repo awareness** and **cross-project intelligence** accordingly.

**Key gaps identified:** no `.git` initialized locally, GitHub outdated, no CI/CD, toy embeddings (64-dim bag-of-words hash), no auth, no multi-user support, no real-time UI updates, no cross-repo search, limited test coverage (3 test files), no project-switching intelligence.

---

## Phase 0 — Foundation & DevOps (Week 1)

**Goal:** Git init, CI, packaging — make the project shippable.

| Action | Symbol / File | Why |
|--------|---------------|-----|
| **Add** | `git init` + force-push to `xdutsuay/Agent-Observer` | Local has no `.git`; GitHub repo is outdated MVP. Force-push v0.3.0 as new baseline |
| **Add** | `.github/workflows/ci.yml` | Automated test + lint on PR; zero CI today |
| **Add** | `Dockerfile`, `docker-compose.yml` | Reproducible deployment; currently manual `pip install -e .` only |
| **Change** | `pyproject.toml` → add `[project.urls]`, classifiers | PyPI-readiness metadata missing |
| **Add** | `tests/conftest.py` + fixtures | No shared test fixtures; each test bootstraps its own engine |
| **Add** | `tests/test_mcp_server.py`, `tests/test_api.py` | MCP server and HTTP API have zero test coverage |
| **Change** | `.gitignore` → add `*.egg-info/`, `graphify-out/`, `demo_test.log` | Currently tracks build artifacts |

**Risks:** None significant. Pure infrastructure.  
**Tests:** CI pipeline itself is the test — green on first push.

---

## Phase 1 — Real Embeddings & Smarter Search (Week 2–3)

**Goal:** Replace toy bag-of-words vectors with actual sentence embeddings; add cross-repo search.

| Action | Symbol / File | Why |
|--------|---------------|-----|
| **Add** | `agent_memory_mcp/engine/embeddings.py` — `EmbeddingProvider` protocol, `LocalEmbedder` (sentence-transformers), `APIEmbedder` (OpenAI/NVIDIA) | `_simple_embedding()` in db.py is a 64-dim hash — no real semantics |
| **Change** | `MemoryEngine.__init__`, `insert_memory`, `_search_unlocked` → use `EmbeddingProvider` | Decouple embedding logic from DB engine |
| **Add** | `memory_embeddings` table: add `model` column | Track which model generated each embedding for re-indexing |
| **Add** | `MemoryEngine.reindex_embeddings(model)` | Batch re-embed when upgrading model |
| **Add** | `engine/migration.py` → `migrate_v2_embeddings()` | Schema migration for new embedding column |
| **Change** | `mcp_server.py` → `search_memory` → add `cross_repo: bool` param | Currently all searches are repo-scoped; users want global search |
| **Add** | `api.py` → `GET /api/search/global` | HTTP equivalent of cross-repo search |

**Risks:** sentence-transformers adds ~500MB dependency. Mitigation: make it optional (`pip install -e ".[embeddings]"`), fall back to current hash vectors.  
**Tests:** `tests/test_embeddings.py` — cosine similarity sanity, provider fallback, reindex idempotency.

---

## Phase 1.5 — Multi-Project Intelligence (Week 3)

**Goal:** Make the system aware of all 30+ projects and enable cross-project context switching, so any agent can instantly know "what was I doing in project X last time?"

| Action | Symbol / File | Why |
|--------|---------------|-----|
| **Add** | `agent_memory_mcp/project_registry.py` — `ProjectRegistry` | No unified registry of all active projects; `repos.json` only tracks repos the watcher has seen |
| **Add** | `ProjectRegistry.scan_workspace(path)` → auto-discover all git repos under `~/localcode` | Currently relies on watcher to discover repos one-by-one; should bulk-scan on startup |
| **Add** | `ProjectRegistry.get_project_summary(repo_id)` → last activity, open failures, recent decisions | Agents need a quick "what's the state of this project?" answer |
| **Add** | MCP tool: `list_projects()` → all known projects with health/activity summary | No way for an agent to ask "what projects exist and what shape are they in?" |
| **Add** | MCP tool: `switch_project_context(path)` → returns full context bundle for the new project | When developer opens a different project, the agent needs instant context |
| **Add** | `agent_memory_mcp/cross_repo.py` — `CrossRepoSearch` | Search across all 30+ projects for shared patterns, common failures |
| **Add** | `CrossRepoSearch.find_similar_failures(content)` → failures from ANY repo matching this pattern | If project A has the same error that project B already solved, surface it |
| **Change** | `repo_discovery.py` → scan `~/localcode` recursively on startup, register all git repos | Currently only discovers repos from hardcoded watch paths |
| **Change** | `api.py` → `GET /api/projects` with activity scores, last-touched timestamps | Dashboard needs a project overview page |
| **Add** | `ui/client/src/pages/ProjectOverview.tsx` | Visual overview of all projects, health status, activity heatmap |

**Risks:** Scanning 30+ repos on startup could be slow. Mitigation: async scan, cache results in `repos.json`, only re-scan on explicit request or daily.  
**Tests:** `tests/test_project_registry.py` — scan discovers repos, summary aggregation, cross-repo search returns results from multiple repos.

---

## Phase 2 — Real-Time UI & WebSocket Events (Week 4–5)

**Goal:** Dashboard updates live instead of polling; add timeline view.

| Action | Symbol / File | Why |
|--------|---------------|-----|
| **Add** | `agent_memory_mcp/ws.py` — WebSocket manager | UI currently has no live updates; events list is polled |
| **Change** | `api.py` → `AgentObserver._run_loop` → broadcast events via WS | Observer has events but no push mechanism |
| **Add** | `app.websocket("/ws/events")` in `api.py` | WebSocket endpoint for UI |
| **Add** | `ui/client/src/hooks/useEventStream.ts` | React hook consuming WS events |
| **Change** | `ui/client/src/components/EventFeed` → real-time rendering | Replace polling with WS subscription |
| **Add** | `ui/client/src/pages/Timeline.tsx` | Chronological view of all memory across repos |
| **Add** | `ui/client/src/pages/SearchPage.tsx` | Dedicated search UI with filters, facets |

**Risks:** WebSocket adds connection management complexity. Mitigation: fall back to SSE if WS unavailable.  
**Tests:** `tests/test_ws.py` — connect, receive event, disconnect.

---

## Phase 3 — Memory Intelligence (Week 5–6)

**Goal:** Auto-correlate failures with decisions; detect recurring patterns; suggest resolutions.

| Action | Symbol / File | Why |
|--------|---------------|-----|
| **Add** | `agent_memory_mcp/correlator.py` — `MemoryCorrelator` | No cross-kind analysis; failures/decisions/attempts live in silos |
| **Add** | `MemoryCorrelator.find_related(memory_id)` → returns related memories across kinds | Enable "this failure was caused by this decision" linking |
| **Add** | `agent_memory_mcp/pattern_detector.py` — `PatternDetector` | Recurring failure patterns detected only by exact signature match today |
| **Add** | `PatternDetector.detect_cycles(repo_id)` → failure→fix→regression chains | Catch "we fixed this before and it came back" |
| **Change** | `extractor.py` → extract `tags`, `related_files`, `root_cause` in addition to summary/fact | LLM extraction is underutilized — only summary + one fact |
| **Add** | `memories` table: add `tags TEXT`, `related_ids TEXT` columns | Enable tagging and explicit relationships |
| **Add** | MCP tool: `get_related_memories(memory_id)` | Expose correlation to agents |
| **Add** | MCP tool: `get_pattern_report(repo_id)` | Surface detected patterns to agents |

**Risks:** Correlation quality depends on embedding quality (Phase 1 prerequisite). False positives in pattern detection could be noisy.  
**Tests:** `tests/test_correlator.py`, `tests/test_pattern_detector.py` — known failure-decision pairs, cycle detection with synthetic data.

---

## Phase 4 — Multi-Agent & Auth (Week 7–8)

**Goal:** Support multiple concurrent agents with session isolation and optional auth.

| Action | Symbol / File | Why |
|--------|---------------|-----|
| **Add** | `agent_memory_mcp/auth.py` — API key auth middleware | Zero auth today; anyone on localhost can read/write |
| **Add** | `memories` table: use `session_id` column (already exists but unused) | Track which agent session wrote each memory |
| **Change** | `mcp_server.py` → pass `session_id` from MCP init to all writes | Enable per-session filtering |
| **Add** | MCP tool: `list_sessions(repo_id)` | Let agents see what other sessions have done |
| **Add** | `agent_memory_mcp/conflict.py` — `ConflictResolver` | Multiple agents may store contradictory decisions |
| **Add** | `ConflictResolver.detect_conflicts(repo_id)` | Flag contradictory decisions from different sessions |
| **Change** | `api.py` → add auth middleware, `X-API-Key` header | Protect HTTP API |

**Risks:** Auth adds friction for single-user local use. Mitigation: auth disabled by default, enabled via `AGENT_MEMORY_AUTH_KEY` env var.  
**Tests:** `tests/test_auth.py` — key validation, unauthorized rejection. `tests/test_conflict.py` — conflicting decisions detected.

---

## Phase 5 — Export, Sync & Ecosystem (Week 9–10)

**Goal:** Memory portability — export, import, sync across machines.

| Action | Symbol / File | Why |
|--------|---------------|-----|
| **Add** | `agent_memory_mcp/export.py` — `MemoryExporter` | No way to export/backup memories today |
| **Add** | CLI: `agent-memory export --repo <id> --format json|md` | Dump memories to portable format |
| **Add** | CLI: `agent-memory import <file>` | Restore from export |
| **Add** | `agent_memory_mcp/sync.py` — `MemorySync` | No cross-machine sync; memory is local-only |
| **Add** | `MemorySync.push(remote_url)`, `MemorySync.pull(remote_url)` | P2P or server-mediated sync |
| **Add** | MCP resource: `memory://global/search` | Cross-client search resource |
| **Change** | `cli.py` → add `export`, `import`, `sync` subcommands | CLI currently only has `serve` and `mcp` |
| **Add** | `agent_memory_mcp/adapters/claude_desktop.py` | Adapter for Claude Desktop memory format |

**Risks:** Sync introduces merge conflicts. Mitigation: append-only with tombstones (current soft-delete model already supports this). Start with export/import, add live sync later.  
**Tests:** `tests/test_export.py` — round-trip export→import, format validation.

---

## Quick Reference — All New Symbols by Phase

### Phase 0 — DevOps
- `.github/workflows/ci.yml`, `Dockerfile`, `docker-compose.yml`
- `tests/conftest.py`, `tests/test_mcp_server.py`, `tests/test_api.py`

### Phase 1 — Embeddings
- `EmbeddingProvider`, `LocalEmbedder`, `APIEmbedder` → `engine/embeddings.py`
- `MemoryEngine.reindex_embeddings()` → `engine/db.py`
- `migrate_v2_embeddings()` → `engine/migration.py`

### Phase 1.5 — Multi-Project Intelligence
- `ProjectRegistry` → `project_registry.py`
- `CrossRepoSearch` → `cross_repo.py`
- MCP tools: `list_projects`, `switch_project_context`
- `ProjectOverview` → `ui/client/src/pages/`

### Phase 2 — Real-Time UI
- `WebSocketManager` → `ws.py`
- `useEventStream` → `ui/client/src/hooks/`
- `Timeline`, `SearchPage` → `ui/client/src/pages/`

### Phase 3 — Intelligence
- `MemoryCorrelator` → `correlator.py`
- `PatternDetector` → `pattern_detector.py`
- MCP tools: `get_related_memories`, `get_pattern_report`

### Phase 4 — Multi-Agent
- `AuthMiddleware` → `auth.py`
- `ConflictResolver` → `conflict.py`
- MCP tool: `list_sessions`

### Phase 5 — Export & Sync
- `MemoryExporter` → `export.py`
- `MemorySync` → `sync.py`
- CLI: `export`, `import`, `sync`
- `ClaudeDesktopAdapter` → `adapters/claude_desktop.py`

---

## Out of Scope

- Replacing SQLite with Postgres/other RDBMS (SQLite is the right choice for local-first)
- Building a SaaS/hosted version
- Supporting non-MCP agent protocols
- Mobile UI

---

**Approve Phase 0** to begin. Each subsequent phase requires explicit approval.
