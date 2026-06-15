#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
source .venv/bin/activate

export AGENT_MEMORY_LLM_PROVIDER="${AGENT_MEMORY_LLM_PROVIDER:-nvidia}"
export AGENT_MEMORY_USE_HERMES_AUTH="${AGENT_MEMORY_USE_HERMES_AUTH:-1}"

exec uvicorn agent_memory_mcp.demo_app:app --host 127.0.0.1 --port 9000
