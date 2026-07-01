# Go Port Implementation Plan — MCP Parity First, Full Data Migration

This file tracks the agreed implementation direction for the Go restart.

## Summary

- Build a clean Go core on `feature/goport`.
- Prioritize storage/read-path parity first, then MCP parity, then HTTP/dashboard reconnect.
- Preserve the current local data root: `memory.db`, `usage.db`, `repos.json`, and legacy markdown import.
- Use Go-native interfaces and packages so the Python runtime is not in the critical path.

## Current implementation focus

1. Scaffold the Go module and package layout.
2. Port SQLite schema, migration bootstrap, and `repos.json` compatibility behavior.
3. Port the memory/read-path service, search, relevance refresh, and usage logging.
4. Add a minimal runnable CLI and HTTP entrypoint so the port is executable while MCP/API parity is built out.

## Constraints

- Data compatibility is required.
- Search ranking should preserve current blended memory behavior.
- Process/IDE inspection must be best-effort and permission-safe.
- Session recall, MCP parity, and the full dashboard reconnect come after the storage foundation is stable.
