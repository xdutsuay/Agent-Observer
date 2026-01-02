"""
Agent Memory MVP
Single-file, zero-packaging version.

Features:
- Persistent agent memory per repo
- HTTP API (FastAPI)
- Optional stdio JSON adapter (for MCP later)
- Clear startup logs
"""

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from pathlib import Path
import json
import sys
import threading
import uvicorn

# ============================================================
# CONFIG
# ============================================================

DATA_ROOT = Path.home() / "agent_companion_data"
DATA_ROOT.mkdir(parents=True, exist_ok=True)

# ============================================================
# STORAGE LAYER
# ============================================================

class Storage:
    def __init__(self, root: Path):
        self.root = root

    def _repo_memory_dir(self, repo_id: str) -> Path:
        path = self.root / "agent-memory" / repo_id / "memory"
        path.mkdir(parents=True, exist_ok=True)
        return path

    def append(self, repo_id: str, kind: str, text: str) -> None:
        path = self._repo_memory_dir(repo_id) / f"{kind}.md"
        with open(path, "a", encoding="utf-8") as f:
            f.write(text.rstrip() + "\n")

    def write_state(self, repo_id: str, state: dict) -> None:
        path = self._repo_memory_dir(repo_id) / "state.json"
        with open(path, "w", encoding="utf-8") as f:
            json.dump(state, f, indent=2)

    def read(self, repo_id: str) -> dict:
        base = self._repo_memory_dir(repo_id)
        result = {
            "attempts": [],
            "failures": [],
            "decisions": [],
            "state": {}
        }

        for key in ["attempts", "failures", "decisions"]:
            p = base / f"{key}.md"
            if p.exists():
                result[key] = p.read_text(encoding="utf-8").splitlines()

        state_path = base / "state.json"
        if state_path.exists():
            result["state"] = json.loads(state_path.read_text(encoding="utf-8"))

        return result

    def list_repos(self) -> list[str]:
        base = self.root / "agent-memory"
        if not base.exists():
            return []
        return [p.name for p in base.iterdir() if p.is_dir()]


store = Storage(DATA_ROOT)

# ============================================================
# HTTP API
# ============================================================

app = FastAPI(title="Agent Memory MVP")

class TextPayload(BaseModel):
    text: str

@app.get("/health")
def health():
    return {"status": "ok"}

@app.get("/repos")
def repos():
    return {"repos": store.list_repos()}

@app.get("/memory/{repo_id}")
def get_memory(repo_id: str):
    try:
        return store.read(repo_id)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/memory/{repo_id}/attempt")
def add_attempt(repo_id: str, payload: TextPayload):
    store.append(repo_id, "attempts", payload.text)
    return {"ok": True}

@app.post("/memory/{repo_id}/failure")
def add_failure(repo_id: str, payload: TextPayload):
    store.append(repo_id, "failures", payload.text)
    return {"ok": True}

@app.post("/memory/{repo_id}/decision")
def add_decision(repo_id: str, payload: TextPayload):
    store.append(repo_id, "decisions", payload.text)
    return {"ok": True}

@app.post("/memory/{repo_id}/state")
def set_state(repo_id: str, state: dict):
    store.write_state(repo_id, state)
    return {"ok": True}

# ============================================================
# STDIO ADAPTER (MCP-LIKE, OPTIONAL)
# ============================================================

def stdio_loop():
    """
    Simple JSON-over-stdio loop.
    Not used now, but kept for MCP integration later.
    """
    print("[STDIO] Adapter started", file=sys.stderr)
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            req = json.loads(line)
            req_id = req.get("id")
            method = req.get("method")
            params = req.get("params", {})

            if method == "list_repos":
                result = {"repos": store.list_repos()}
            elif method == "get_memory":
                result = store.read(params["repo_id"])
            elif method == "append_failure":
                store.append(params["repo_id"], "failures", params.get("text", ""))
                result = {"ok": True}
            else:
                result = {"error": "unknown method"}

            resp = {"id": req_id, "result": result}
        except Exception as e:
            resp = {"error": str(e)}

        sys.stdout.write(json.dumps(resp) + "\n")
        sys.stdout.flush()

# ============================================================
# ENTRY POINT
# ============================================================

if __name__ == "__main__":
    print("======================================")
    print(" Agent Memory MVP starting")
    print(f" Data dir: {DATA_ROOT}")
    print(" URL: http://127.0.0.1:9000")
    print("======================================")

    # Start HTTP server (blocking)
    uvicorn.run(app, host="127.0.0.1", port=9000, log_level="info")
