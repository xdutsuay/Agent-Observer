#!/bin/bash
# ==========================================================
# Merge Agent-Observer + agent_memory_mcp → single repo
# Then push to https://github.com/xdutsuay/Agent-Observer
# ==========================================================
set -e

PROJECT_DIR="$HOME/localcode/agent_memory_mcp"
OLD_DIR="$HOME/localcode/Agent-Observer"
GITHUB_URL="https://github.com/xdutsuay/Agent-Observer.git"

cd "$PROJECT_DIR"

# 1. Clean up stale .git from sandbox attempt (if any)
if [ -d ".git" ]; then
    echo "Removing stale .git from previous attempt..."
    rm -rf .git
fi

# 2. Initialize fresh git repo
echo "Initializing git..."
git init
git branch -M main

git config user.email "kaustubh.best01@gmail.com"
git config user.name "kaustubh"

# 3. Stage and commit
git add -A
git commit -m "v0.3.0: Full agent memory system (merged from Agent-Observer + agent_memory_mcp)

Merged two parallel projects into one canonical codebase.

Core:
- SQLite + FTS5 + local vector search engine (engine/db.py)
- MCP server with remember/search/context/forget tools (mcp_server.py)
- FastAPI HTTP API with full CRUD (api.py)
- Filesystem watcher for passive observation (watcher.py)
- LLM extraction: OpenAI/Anthropic/NVIDIA (extractor.py)
- Adapters: Antigravity brain, Cursor transcripts
- Log parser, process detector, metrics collector
- Activity scoring and repo discovery
- Raw log capture (ported from Agent-Observer)

UI:
- React dashboard (Vite + shadcn/ui)
- Dashboard, Memory, Logs, Configuration pages

Tests:
- engine, log_parser, llm_credentials

Docs:
- Future enhancement LLP plan"

# 4. Add remote and push
git remote add origin "$GITHUB_URL" 2>/dev/null || \
    git remote set-url origin "$GITHUB_URL"

echo ""
echo "=========================================="
echo " Ready to push!"
echo "=========================================="
echo ""
echo "This will OVERWRITE the old Agent-Observer repo on GitHub."
echo "Run:  git push -f origin main"
echo ""
read -p "Force push now? (y/N) " answer
if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
    git push -f origin main
    echo ""
    echo "Done! GitHub is synced."
    echo ""
    echo "You can now optionally remove the old project:"
    echo "  rm -rf $OLD_DIR"
else
    echo ""
    echo "Skipped. When ready, run:"
    echo "  cd $PROJECT_DIR && git push -f origin main"
fi
