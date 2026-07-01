# Agent Memory MCP Go Restart Plan

Status: draft  
Branch: `feature/goport`  
Date: 2026-07-01

## Why restart this in Go

The current Python codebase proves the product is worth building: local-first repo memory, MCP tools, a dashboard, relevance scoring, cross-repo search, and operator visibility all already exist. But the implementation is now spread across watcher logic, SQLite access, MCP handlers, HTTP routes, roadmap experiments, and adapter code in a way that will get harder to scale cleanly.

A Go restart is justified because it gives us:

- one static binary for `mcp`, `serve`, background ingest, and maintenance jobs
- lower idle memory and better concurrency for watchers, indexing, and websocket fan-out
- easier long-running daemon behavior than the current Python thread mix
- a cleaner foundation for session-ingest, SaaS/API hardening, and team/multi-agent features
- faster startup and simpler installs for users who just want one executable

This should be a true restart, not a line-by-line translation. We should preserve behavior and product surface, while redesigning internals around Go package boundaries and explicit subsystems.

## What exists today and must survive

The port must keep every working feature that is already in this repo:

### Core memory system

- SQLite-backed repo registry and memory store
- memory kinds: `failure`, `decision`, `attempt`, `fact`, `preference`
- failure dedup via signatures
- full-text search plus embedding-backed semantic search
- relevance scoring with access tracking, feedback, recency, and quality tiers
- automatic noise classification for stale low-value attempts
- context bundle generation per repo
- `.agent-memory/context.md` generation

### MCP surface

- memory resources by repo/kind
- tools:
  - `remember`
  - `search_memory`
  - `get_repo_context`
  - `mark_failure_resolved`
  - `forget`
  - `list_projects`
  - `switch_project_context`
  - `global_search`
  - `find_similar_failures`
  - `get_related_memories`
  - `get_pattern_report`
  - `failure_hotspots`
  - `memory_feedback`
  - `smart_context`
  - `refresh_relevance`
- prompt:
  - `inject_memory_context`

### HTTP/dashboard surface

- REST API for memories, search, projects, patterns, usage, disk usage, smart context, feedback, relevance refresh, and context generation
- websocket event stream
- operator dashboard pages:
  - Dashboard
  - Projects
  - Memory
  - Search
  - Patterns
  - Timeline
  - Usage
  - Logs
  - Configuration

### Ingest and analysis

- filesystem watcher for code/log changes
- log parsing and error extraction
- project discovery across workspaces
- activity/process scoring
- memory correlation and pattern detection
- usage logging for MCP and HTTP interactions
- adapters for external/local memory-like sources

## Planned features that must be pulled forward

The restart also needs to absorb the planned features already documented in this repo instead of treating them as optional extras later:

- cross-tool session recall from Claude Code, Codex CLI, and Cursor
- adapter-driven session ingestion so more tools can be added by config
- smarter context injection that agents can call automatically
- multi-agent session awareness and conflict detection
- auth and SaaS-ready API boundaries
- export/import/sync
- stronger real-time UX
- better extraction and summarization pipelines

In other words: the Go port should target both feature parity and roadmap convergence.

## Key current pain points the Go design should fix

- watcher noise is still too high; the system captures many low-value file-save attempts
- product versioning is inconsistent across files (`README.md` says `0.4.0`, the app reports `0.3.0`)
- the current implementation mixes storage, ranking, ingest, and transport concerns
- session recall is only partially present today through ad hoc adapters and docs, not as a first-class subsystem
- process/IDE inspection is brittle in restricted environments; tests currently fail on `psutil` permission issues
- the dashboard and API are coupled to backend internals more tightly than they should be

## Reference repo idea we should steal

From `Kuberwastaken/reference`, the most valuable design to copy is not “BM25” by itself; it is the file-cache and incremental indexing model:

- transcript files are normalized into a common message model
- parsed results are cached per file using `mtime`
- the search index rebuilds only when the set of files or their mtimes changes
- adapters define transcript globs, memory globs, parser format, and behavior flags
- tool output is truncated and tagged, not dropped, so it remains searchable without exploding index size

We should adopt that exact operational idea, then improve retrieval quality beyond Reference by combining:

- transcript BM25 retrieval
- semantic reranking using embeddings
- recency weighting
- repository/project weighting
- existing relevance score signals from permanent memories

Reference gives us the right ingest/index invalidation pattern. Agent Memory MCP should provide the stronger ranking model and memory promotion workflow.

## Proposed Go architecture

### Product shape

Keep one repo, one Go module, one frontend workspace.

- Go backend:
  - MCP stdio server
  - HTTP API
  - websocket server
  - watcher/index workers
  - maintenance jobs
- React frontend:
  - keep the existing UI stack initially
  - point it to the new Go API
  - optionally embed built assets into the Go binary for local distribution

### Suggested repo layout

```text
cmd/
  agent-memory/
    main.go
internal/
  app/
  config/
  mcp/
  httpapi/
  ws/
  store/
    sqlite/
    migrations/
  memory/
  search/
  embeddings/
  sessions/
    adapters/
    parsers/
    index/
  ingest/
    watcher/
    logs/
    extractor/
  projects/
  patterns/
  usage/
  disk/
  context/
  auth/
  sync/
  export/
ui/
  client/
docs/
```

### Runtime modes

The new binary should support:

- `agent-memory serve`
- `agent-memory mcp`
- `agent-memory reindex`
- `agent-memory refresh-relevance`
- `agent-memory sessions index`
- `agent-memory export`
- `agent-memory import`
- `agent-memory doctor`

## Core subsystem plan

### 1. Storage

Use SQLite as the primary local store again. Keep it as the canonical source of truth.

Requirements:

- preserve memory records, repo registry, failure signatures, access logs, usage logs
- keep FTS5 for keyword retrieval
- add explicit schema versioning and deterministic migrations from day 1
- separate transcript/session index tables from permanent memory tables
- store provenance aggressively: source tool, file path, session id, repo path, event type

Recommended tables:

- `repos`
- `memories`
- `memory_embeddings`
- `failure_signatures`
- `memory_access_log`
- `usage_sessions`
- `usage_interactions`
- `session_sources`
- `session_files`
- `session_turns`
- `session_turn_embeddings`
- `index_state`
- later: `users`, `orgs`, `api_keys`, `sync_events`

### 2. Search and ranking

Define search as a first-class subsystem instead of burying it in the DB layer.

Search modes:

- repo memory search
- global/cross-repo memory search
- session transcript search
- blended recall across permanent memory and transcripts
- smart-context packing under a token budget

Ranking pipeline:

1. FTS/BM25 retrieval
2. semantic similarity scoring
3. relevance score blending
4. recency boost
5. repo/project affinity boost
6. quality-tier filtering

### 3. Embeddings

Expose embeddings behind a provider interface from the start.

Providers to support:

- OpenAI-compatible API embeddings
- local HTTP embedding provider such as Ollama/NIM
- deterministic hash fallback
- optional native ONNX local embedder as a later parity milestone

Important point: do not couple the first Go port to Python `sentence-transformers`. The port should stay Go-native, with provider abstraction preserving the feature instead of preserving the exact implementation.

### 4. Session recall

This should be a major pillar of the restart, not a side feature.

Capabilities:

- parse Claude Code, Codex CLI, and Cursor transcripts into one `SessionTurn` model
- index `AGENTS.md`, `CLAUDE.md`, repo memory markdown, and transcript turns
- cache parsed files by `mtime`
- rebuild session indexes only when file signature changes
- support `search_sessions`, `list_sessions`, `get_session`, and `recall`
- allow promoting a session turn into permanent memory

Suggested Go package split:

- `sessions/adapters`
- `sessions/parsers`
- `sessions/cache`
- `sessions/index`
- `sessions/search`

### 5. Ingestion and watcher

The watcher should survive, but in a less noisy form.

Rules:

- do not store every save as a memory
- classify events first, persist only meaningful ones
- support debouncing, file filters, and per-repo rate controls
- separate:
  - raw event capture
  - classification
  - memory creation
  - websocket/UI broadcast

Better default policy:

- failures/log errors become memories
- code changes become candidate events first
- candidate attempts are promoted only if they cross a significance threshold or are summarized

### 6. MCP and API

Implement transports as thin layers over domain services.

That means:

- MCP handlers call service interfaces, not raw SQL
- HTTP handlers call the same service interfaces
- websocket events are emitted from domain events, not route code
- prompt generation and smart-context packing live in `context/`, not in the MCP server itself

### 7. Frontend

Keep the React UI for now. Rewrite backend contracts first, not the frontend.

Plan:

- define stable JSON response contracts in Go
- keep the existing routes/pages
- upgrade only the API client and any shape mismatches
- embed static UI assets into the Go binary after the backend stabilizes

## Feature mapping from Python to Go

| Current area | Go target |
|---|---|
| `engine/db.py` | `internal/store/sqlite`, `internal/search`, `internal/memory` |
| `storage.py` | service layer in `internal/app` / `internal/memory` |
| `mcp_server.py` | `internal/mcp` |
| `api.py` | `internal/httpapi` |
| `watcher.py` + `log_parser.py` | `internal/ingest/watcher` + `internal/ingest/logs` |
| `project_registry.py` + `repo_discovery.py` | `internal/projects` |
| `cross_repo.py` | `internal/search` |
| `correlator.py` + `pattern_detector.py` | `internal/patterns` |
| `usage_log.py` | `internal/usage` |
| `disk_usage.py` | `internal/disk` |
| transcript recall plan docs + adapters | `internal/sessions/*` |

## Delivery plan

### Phase 0: Design freeze and parity inventory

- freeze current Python feature inventory
- finalize MCP tool contracts to preserve
- define JSON API contracts to preserve or deliberately change
- write SQLite schema v1 for Go
- choose main libraries:
  - HTTP router
  - MCP SDK strategy
  - SQLite driver
  - migration tool
  - websocket stack

Deliverable:

- architecture doc
- parity checklist
- Go module scaffold

### Phase 1: Storage and read-path core

- implement config loading
- implement SQLite migrations
- implement repo registry
- implement memory CRUD
- implement failure signatures
- implement keyword search
- implement relevance columns and refresh jobs

Deliverable:

- CLI smoke tests
- storage tests
- search tests

### Phase 2: MCP parity

- port all current MCP tools
- port memory resources
- port `inject_memory_context`
- port usage logging around tool calls

Deliverable:

- MCP contract tests against the parity checklist

### Phase 3: HTTP/API parity

- port current HTTP endpoints
- port websocket events
- port disk usage and usage analytics
- connect existing frontend to the Go backend

Deliverable:

- dashboard works against Go backend

### Phase 4: Watcher and ingest cleanup

- port watcher
- port log parsing
- port repo discovery
- reduce noisy attempt creation
- emit domain events cleanly

Deliverable:

- long-running daemon works reliably
- lower noise than current Python implementation

### Phase 5: Session recall subsystem

- implement adapter config
- implement transcript parsers
- implement `mtime` parse cache
- implement index signature invalidation
- implement transcript search and recall tools
- implement session-to-memory promotion

Deliverable:

- first-class cross-tool session recall
- better than Reference because it can merge transcript hits with permanent memory

### Phase 6: Intelligence parity

- port correlator
- port pattern detector
- port smart context packing
- port feedback loop and relevance refresh

Deliverable:

- search quality and smart context at least as good as current repo

### Phase 7: Roadmap carry-forward

- auth
- multi-agent session isolation
- export/import
- sync
- SaaS boundary prep

Deliverable:

- roadmap features are now being built on top of the Go architecture instead of fighting the Python one

## Acceptance criteria

We should not call the port “ready” until all of this is true:

- every current MCP tool has a Go equivalent
- the dashboard works end to end against the Go backend
- SQLite migration/versioning is explicit and tested
- session recall works across Claude, Codex, and Cursor
- search can blend keyword, semantic, relevance, and recency signals
- watcher noise is lower than current Python behavior
- usage analytics work in restricted environments without crashing
- local install is one binary plus optional UI assets

## Recommended implementation sequence right now

1. Create the Go module and architecture scaffold on `feature/goport`.
2. Port the storage schema and tests before any transport layer.
3. Port MCP next, because it is the core product contract.
4. Reconnect the existing React UI to Go once the API is stable.
5. Build the Reference-style session cache/index subsystem before auth/sync work.

## Strong recommendation

Do not do a mixed-language “Python core with Go wrapper” port. That will keep the current complexity and add IPC complexity on top.

Do a clean Go core with:

- SQLite
- pluggable embeddings
- first-class session indexing
- thin MCP/HTTP transports
- reused React frontend

That gives this project the best chance of becoming the fast, portable, cross-tool memory system it wants to be.
