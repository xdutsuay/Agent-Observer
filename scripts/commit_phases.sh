#!/bin/bash
set -e
cd "$HOME/localcode/agent_memory_mcp"

git add -A
git commit -m "v0.4.0: Phases 1-3 — embeddings, multi-project, WebSocket, memory intelligence

Phase 1 — Real Embeddings + Cross-Repo Search:
- engine/embeddings.py: EmbeddingProvider protocol, HashEmbedder (fallback),
  LocalEmbedder (sentence-transformers), APIEmbedder (OpenAI-compatible)
- MemoryEngine now accepts pluggable embedder, with reindex_embeddings()
- cross_repo.py: global search, similar failures, failure hotspots, common patterns

Phase 1.5 — Multi-Project Intelligence:
- project_registry.py: ProjectRegistry with language/framework detection,
  auto-tagging, health stats
- MCP tools: list_projects, switch_project_context, global_search,
  find_similar_failures, failure_hotspots

Phase 2 — WebSocket Real-Time Events:
- ws.py: WebSocketManager with broadcast, history, sync emit from threads
- /ws/events WebSocket endpoint + /api/events/history REST endpoint

Phase 3 — Memory Intelligence:
- correlator.py: MemoryCorrelator — finds related memories by file refs,
  content similarity, temporal proximity; builds dependency graph
- pattern_detector.py: PatternDetector — recurring failures, activity trends,
  error categorization, decision patterns, health scoring (A-F grades)
- MCP tools: get_related_memories, get_pattern_report

Tests: 30/30 passing (test_engine, test_mcp_server, test_api, test_log_parser, test_llm_credentials)"

git push origin main
echo "Done! Pushed v0.4.0 to GitHub."
