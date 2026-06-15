#!/usr/bin/env bash
# Start Agent Memory MCP — daemon + dashboard (and optional MCP-only mode).
#
# Usage:
#   ./scripts/start.sh              # API + built UI on :9000 (recommended)
#   ./scripts/start.sh --dev        # API :9000 + Vite dev UI :5000
#   ./scripts/start.sh --mcp        # MCP stdio only (for Cursor MCP config)
#   ./scripts/start.sh --api-only   # API only, no UI
#   ./scripts/start.sh --port 8080  # custom port
#
# Env (optional):
#   AGENT_MEMORY_LLM_PROVIDER=nvidia|openai|anthropic|none
#   AGENT_MEMORY_USE_HERMES_AUTH=1
#   AGENT_MEMORY_DATA_ROOT=~/agent_companion_data

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PORT="${PORT:-9000}"
DATA_ROOT="${AGENT_MEMORY_DATA_ROOT:-$HOME/agent_companion_data}"
MODE="serve"          # serve | dev | mcp | api-only
SKIP_UI_BUILD=0
NO_WATCHER=0

usage() {
  sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'
  exit "${1:-0}"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage 0 ;;
    --dev) MODE="dev" ;;
    --mcp) MODE="mcp" ;;
    --api-only) MODE="api-only" ;;
    --port) PORT="$2"; shift ;;
    --no-ui-build) SKIP_UI_BUILD=1 ;;
    --no-watcher) NO_WATCHER=1 ;;
    *) echo "Unknown option: $1" >&2; usage 1 ;;
  esac
  shift
done

log() { printf '\033[1;36m[start]\033[0m %s\n' "$*"; }

# --- Python venv ---
if [[ ! -d ".venv" ]]; then
  log "Creating virtualenv in .venv"
  python3 -m venv .venv
fi
# shellcheck disable=SC1091
source .venv/bin/activate

if ! python -c "import agent_memory_mcp" 2>/dev/null; then
  log "Installing Python package (editable + dev extras)"
  pip install -q -e ".[dev]"
fi

# --- LLM defaults (optional; watcher works without keys) ---
export AGENT_MEMORY_LLM_PROVIDER="${AGENT_MEMORY_LLM_PROVIDER:-nvidia}"
export AGENT_MEMORY_USE_HERMES_AUTH="${AGENT_MEMORY_USE_HERMES_AUTH:-1}"

# --- MCP stdio ---
if [[ "$MODE" == "mcp" ]]; then
  log "MCP stdio server (data: $DATA_ROOT)"
  exec agent-memory mcp --root "$DATA_ROOT"
fi

# --- UI build for production serve ---
need_ui_build() {
  [[ "$MODE" == "serve" ]] && [[ ! -f "ui/dist/index.html" ]]
}

if need_ui_build && [[ "$SKIP_UI_BUILD" -eq 0 ]]; then
  if ! command -v npm >/dev/null 2>&1; then
    echo "[start] ui/dist missing and npm not found. Install Node.js or run with --api-only" >&2
    exit 1
  fi
  log "Building UI (ui/dist) — first run may take a minute"
  (cd ui && npm install --silent && npm run build)
fi

# --- Dev: API + Vite ---
if [[ "$MODE" == "dev" ]]; then
  if ! command -v npm >/dev/null 2>&1; then
    echo "[start] --dev requires npm" >&2
    exit 1
  fi
  log "API on http://127.0.0.1:$PORT  |  UI dev server on http://127.0.0.1:5000"
  WATCHER_FLAG=()
  [[ "$NO_WATCHER" -eq 1 ]] && WATCHER_FLAG=(--no-watcher)
  trap 'kill 0 2>/dev/null || true' EXIT
  agent-memory serve --root "$DATA_ROOT" --port "$PORT" "${WATCHER_FLAG[@]}" &
  (cd ui && npm install --silent && npm run dev:client)
  exit 0
fi

# --- API only ---
if [[ "$MODE" == "api-only" ]]; then
  log "API only — http://127.0.0.1:$PORT/api/status"
  WATCHER_FLAG=()
  [[ "$NO_WATCHER" -eq 1 ]] && WATCHER_FLAG=(--no-watcher)
  exec agent-memory serve --root "$DATA_ROOT" --port "$PORT" "${WATCHER_FLAG[@]}"
fi

# --- Default: demo app (API + static UI) ---
log "Dashboard — http://127.0.0.1:$PORT"
log "Data root — $DATA_ROOT"
log "Watcher — $([[ "$NO_WATCHER" -eq 1 ]] && echo OFF || echo ON)"
if [[ "$NO_WATCHER" -eq 1 ]]; then
  echo "[start] Note: demo_app always enables watcher; use --api-only for --no-watcher" >&2
fi
exec uvicorn agent_memory_mcp.demo_app:app --host 127.0.0.1 --port "$PORT"
