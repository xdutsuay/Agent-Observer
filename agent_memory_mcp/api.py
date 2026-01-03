from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel
from pathlib import Path
from typing import List, Dict, Optional
import time
import threading

from .storage import Storage
from .watcher import FileWatcher
from .activity_score import ActivityScorer
from .process_detector import ProcessDetector

app = FastAPI(title="Agent Memory MCP")

# Global state
DATA_ROOT = Path("~/agent_companion_data").expanduser()
store = Storage(DATA_ROOT)
scorer = None
watcher = None
watcher_thread = None

# Default config (can be moved to a file later)
DEFAULT_CONFIG = {
    "watch_paths": [str(Path.home() / "localcode")],
    "log_extensions": [".log", ".txt", ".md", ".py"],
    "ignore_patterns": [".git", "node_modules", "__pycache__", ".venv"],
    "churn_window_seconds": 60,
    "min_fs_events": 5,
    "process_name_contains": ["cursor", "vscode", "zed", "pycharm"]
}

class TextPayload(BaseModel):
    text: str
    metadata: Optional[Dict] = None

class WatcherConfig(BaseModel):
    paths: List[str]
    extensions: List[str]
    ignore_patterns: Optional[List[str]] = None

@app.get("/")
def root():
    return {
        "name": "Agent Memory MCP",
        "status": "active",
        "endpoints": ["/health", "/status", "/repos", "/memory/{repo_id}"]
    }

@app.get("/health")
def health():
    return {"status": "ok", "timestamp": time.time()}

@app.get("/repos")
def list_repos():
    return {"repos": store.list_repos()}

@app.get("/memory/{repo_id}")
def get_memory(repo_id: str):
    try:
        return store.read_memory(repo_id)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/memory/{repo_id}/attempt")
def add_attempt(repo_id: str, payload: TextPayload):
    store.append_memory(repo_id, "attempts", payload.text, payload.metadata)
    return {"ok": True}

@app.post("/memory/{repo_id}/failure")
def add_failure(repo_id: str, payload: TextPayload):
    store.append_memory(repo_id, "failures", payload.text, payload.metadata)
    return {"ok": True}

@app.post("/memory/{repo_id}/decision")
def add_decision(repo_id: str, payload: TextPayload):
    store.append_memory(repo_id, "decisions", payload.text, payload.metadata)
    return {"ok": True}

@app.post("/watcher/start")
def start_watcher(config: Optional[WatcherConfig] = None):
    global watcher, watcher_thread, scorer
    if watcher and watcher_thread and watcher_thread.is_alive():
        return {"status": "already running"}
    
    cfg = config.dict() if config else DEFAULT_CONFIG
    watcher = FileWatcher(
        paths=cfg.get("paths", DEFAULT_CONFIG["watch_paths"]),
        extensions=cfg.get("extensions", DEFAULT_CONFIG["log_extensions"]),
        ignore_patterns=cfg.get("ignore_patterns", DEFAULT_CONFIG["ignore_patterns"])
    )
    
    # Simple heuristics setup
    heuristics = {
        "churn_window_seconds": DEFAULT_CONFIG["churn_window_seconds"],
        "min_fs_events": DEFAULT_CONFIG["min_fs_events"],
        "process_name_contains": DEFAULT_CONFIG["process_name_contains"]
    }
    scorer = ActivityScorer(heuristics)
    det = ProcessDetector(heuristics)
    
    def watcher_loop():
        watcher.start()
        print("[API] Watcher loop started")
        for event in watcher.events():
            # Process event
            scorer.record_fs_event()
            # In a real scenario, we'd also check process detector here
            # and decide if we want to capture the log
            repo_id = store.resolve_repo(event)
            if repo_id:
                raw_path = store.capture_raw_log(repo_id, event)
                # Auto-detect failure in log (very basic)
                try:
                    content = event.read_text(errors="ignore").lower()
                    if "error" in content or "failed" in content:
                        store.append_memory(repo_id, "failures", f"Detected error in {event.name}", {"timestamp": time.ctime()})
                    else:
                        store.append_memory(repo_id, "attempts", f"Detected activity in {event.name}", {"timestamp": time.ctime()})
                except Exception as e:
                    print(f"Error reading {event}: {e}")

    watcher_thread = threading.Thread(target=watcher_loop, daemon=True)
    watcher_thread.start()
    
    return {"status": "started"}

@app.post("/watcher/stop")
def stop_watcher():
    global watcher
    if watcher:
        watcher.stop()
        return {"status": "stopped"}
    return {"status": "not running"}

@app.get("/status")
def get_status():
    running = watcher_thread and watcher_thread.is_alive()
    return {
        "running": running,
        "data_root": str(DATA_ROOT),
        "repos_count": len(store.list_repos())
    }
