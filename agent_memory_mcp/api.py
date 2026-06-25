from fastapi import FastAPI, HTTPException, Request, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
from fastapi.responses import FileResponse
from pydantic import BaseModel
from pathlib import Path
from typing import List, Dict, Optional
import asyncio
import time
import threading

from .storage import Storage
from .watcher import FileWatcher
from .activity_score import ActivityScorer
from .process_detector import ProcessDetector
from .metrics_collector import MetricsCollector
from .log_parser import Severity, LogParser
from .repo_discovery import default_watch_paths, discover_scan_paths, discover_git_repos
from .llm_credentials import nvidia_configured
from .project_registry import ProjectRegistry
from .cross_repo import CrossRepoSearch
from .correlator import MemoryCorrelator
from .pattern_detector import PatternDetector
from .ws import ws_manager, Event, EventType
from .usage_log import UsageLog, classify_host_ide
from .disk_usage import build_disk_report

class TextPayload(BaseModel):
    text: str
    metadata: Optional[Dict] = None

class SearchPayload(BaseModel):
    query: str
    repo_id: Optional[str] = None
    kinds: Optional[List[str]] = None
    limit: int = 10

class WatcherConfig(BaseModel):
    paths: Optional[List[str]] = None
    extensions: Optional[List[str]] = None
    ignore_patterns: Optional[List[str]] = None

class SmartContextPayload(BaseModel):
    task: str
    repo_id: str
    max_tokens: int = 3000

class FeedbackPayload(BaseModel):
    memory_id: str
    useful: bool
    context: Optional[str] = None

def build_default_config() -> Dict:
    return {
        "watch_paths": default_watch_paths(),
        "log_extensions": [".log", ".txt", ".md", ".py", ".ts", ".tsx", ".js"],
        "ignore_patterns": [
            ".git", "node_modules", "__pycache__", ".venv", "dist", ".next",
            "agent_companion_data", "/terminals/", "agent-transcripts", "mcps/",
            "memory.db", ".hermes",
        ],
        "churn_window_seconds": 60,
        "min_fs_events": 5,
        "process_name_contains": [
            "cursor", "vscode", "zed", "pycharm", "python", "node", "antigravity",
        ],
        "debounce_seconds": 5.0,
    }


DEFAULT_CONFIG = build_default_config()

class AgentObserver:
    def __init__(self, store: Storage, metrics: MetricsCollector, config: Dict):
        self.store = store
        self.metrics = metrics
        self.config = config
        self.watcher = None
        self.scorer = None
        self.thread = None
        self.live_events: List[Dict] = []
        self._recent_errors: Dict[str, float] = {}
        self._stop_event = threading.Event()
        self.store._debounce_seconds = config.get("debounce_seconds", 5.0)

    def start(self, config: Optional[Dict] = None) -> bool:
        if self.thread and self.thread.is_alive():
            return False

        cfg = {**self.config, **(config or {})}
        self._stop_event.clear()
        self.watcher = FileWatcher(
            paths=cfg.get("watch_paths", []),
            extensions=cfg.get("log_extensions", []),
            ignore_patterns=cfg.get("ignore_patterns", []),
        )
        self.scorer = ActivityScorer(cfg)
        self.thread = threading.Thread(target=self._run_loop, daemon=True)
        self.thread.start()
        return True

    def stop(self) -> bool:
        if self.watcher:
            self.watcher.stop()
            self._stop_event.set()
        return True

    def is_running(self) -> bool:
        return bool(self.thread and self.thread.is_alive())

    def _run_loop(self) -> None:
        self.watcher.start()
        print("[OBSERVER] Loop started")
        for event_data in self.watcher.events():
            if self._stop_event.is_set():
                break

            self.metrics.record_fs_event()
            if self.scorer:
                self.scorer.record_fs_event()

            path = event_data["path"]
            category = event_data.get("category", "unknown")
            repo_id = self.store.resolve_repo(path)

            live_event = {
                "id": f"{time.time()}_{path.name}",
                "timestamp": event_data["timestamp"],
                "message": event_data.get("change_summary") or f"Modified: {path.name}",
                "source": "WATCHER",
                "severity": "INFO",
                "category": category,
            }

            if event_data.get("log_events") and event_data.get("has_errors"):
                for log_event in event_data["log_events"]:
                    if log_event.severity != Severity.ERROR:
                        continue
                    if LogParser.is_noise_message(log_event.message):
                        continue
                    err_key = log_event.message[:80]
                    now = time.time()
                    if now - self._recent_errors.get(err_key, 0) < 30:
                        continue
                    self._recent_errors[err_key] = now
                    msg = (
                        f"{log_event.message}\n"
                        f"{log_event.stack_trace or ''}\n"
                        f"{log_event.category or ''}"
                    )
                    try:
                        self.store.append_memory(
                            repo_id,
                            "failures",
                            msg,
                            {
                                "file": str(log_event.file_path),
                                "line": log_event.line_number,
                            },
                            source="watcher",
                        )
                    except Exception as exc:
                        live_event["severity"] = "ERROR"
                        live_event["message"] = f"Storage error: {exc}"
                        continue
                    live_event["severity"] = "ERROR"
                    live_event["message"] = f"ERROR: {log_event.message[:80]}"

            elif "code_analysis" in event_data:
                analysis = event_data["code_analysis"]
                msg = event_data.get("change_summary") or f"Modified {path.name}"
                self.store.append_memory(
                    repo_id,
                    "attempts",
                    msg,
                    {
                        "file": str(path),
                        "complexity": analysis.get("complexity_hint"),
                        "lines": analysis.get("line_count"),
                    },
                    source="watcher",
                )

            self.live_events.append(live_event)
            if len(self.live_events) > 100:
                self.live_events.pop(0)


def create_app(
    data_root: Path,
    serve_ui: bool = False,
    auto_watcher: bool = True,
) -> FastAPI:
    store = Storage(data_root)
    metrics_collector = MetricsCollector(window_seconds=60)
    observer = AgentObserver(store, metrics_collector, DEFAULT_CONFIG.copy())
    usage_log = UsageLog.for_root(data_root)

    def _http_client(request: Request) -> tuple[str, str, str]:
        ua = request.headers.get("user-agent", "Dashboard UI")
        host = classify_host_ide(ua)
        if host == "Unknown":
            host = "Web UI"
        return ua[:120], "", host

    def _log_http(
        request: Request,
        method: str,
        query: Dict,
        response_preview: str,
        duration_ms: float,
        ok: bool = True,
    ) -> None:
        client_name, client_version, host_ide = _http_client(request)
        usage_log.record(
            transport="http",
            method=method,
            query=query,
            response_preview=response_preview,
            client_name=client_name,
            client_version=client_version,
            host_ide=host_ide,
            duration_ms=duration_ms,
            ok=ok,
        )

    app = FastAPI(title="Agent Memory MCP", version="0.3.0")
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_methods=["*"],
        allow_headers=["*"],
    )

    @app.on_event("startup")
    def _startup() -> None:
        discovered = discover_git_repos(store, discover_scan_paths())
        print(f"[OBSERVER] Discovered {discovered} git repo(s)")
        if auto_watcher:
            started = observer.start()
            print(f"[OBSERVER] Watcher {'started' if started else 'already running'}")

    @app.get("/api/")
    def root():
        return {"name": "Agent Memory MCP", "version": "0.3.0"}

    @app.get("/api/repos")
    def list_repos():
        return {"repos": store.list_repos()}

    @app.get("/api/memory/{repo_id}")
    def get_memory(repo_id: str, request: Request):
        t0 = time.perf_counter()
        data = store.read_memory(repo_id)
        preview = "\n".join(f"{k}: {str(v)[:80]}" for k, v in list(data.items())[:4])
        _log_http(
            request,
            "http_get_memory",
            {"repo_id": repo_id},
            preview,
            (time.perf_counter() - t0) * 1000,
        )
        return data

    @app.get("/api/memories")
    def list_memory_records(
        repo_id: Optional[str] = None,
        kind: Optional[str] = None,
        limit: int = 50,
    ):
        return {"memories": store.list_memories(repo_id=repo_id, kind=kind, limit=limit)}

    @app.post("/api/search")
    def search_memory(payload: SearchPayload, request: Request):
        t0 = time.perf_counter()
        results = store.search(
            payload.query,
            repo_id=payload.repo_id,
            kinds=payload.kinds,
            limit=payload.limit,
        )
        preview = f"{len(results)} hit(s)"
        if results:
            preview += f" — top: {results[0].get('content', '')[:120]}"
        _log_http(
            request,
            "http_search",
            payload.model_dump(),
            preview,
            (time.perf_counter() - t0) * 1000,
        )
        return {"results": results}

    @app.get("/api/failures/{repo_id}")
    def failure_signatures(repo_id: str):
        return {"signatures": store.engine.get_failure_signatures(repo_id)}

    @app.post("/api/memory/{repo_id}/{kind}")
    def add_memory(repo_id: str, kind: str, payload: TextPayload, request: Request):
        allowed = ["attempts", "failures", "decisions", "facts", "preferences",
                   "attempt", "failure", "decision", "fact", "preference"]
        if kind not in allowed:
            raise HTTPException(status_code=400, detail="Invalid kind")
        t0 = time.perf_counter()
        store.append_memory(repo_id, kind, payload.text, payload.metadata, source="http")
        _log_http(
            request,
            "http_add_memory",
            {"repo_id": repo_id, "kind": kind, "text": payload.text[:200]},
            f"Stored {kind} in {repo_id}",
            (time.perf_counter() - t0) * 1000,
        )
        return {"ok": True}

    @app.get("/api/metrics")
    def get_metrics():
        m = metrics_collector.get_metrics_dict()
        procs = metrics_collector.detect_agent_processes(
            DEFAULT_CONFIG["process_name_contains"]
        )
        detector = ProcessDetector({"process_name_contains": DEFAULT_CONFIG["process_name_contains"]})
        process_score = detector.score()
        m["agent_processes_detected"] = len(procs)
        m["activity_score"] = (
            observer.scorer.calculate_score(process_score) if observer.scorer else 0.0
        )
        return m

    @app.get("/api/events")
    def get_events():
        return {"events": observer.live_events}

    @app.post("/api/watcher/start")
    def start_watcher(config: Optional[WatcherConfig] = None):
        cfg = DEFAULT_CONFIG.copy()
        if config:
            if config.paths:
                cfg["watch_paths"] = config.paths
            if config.extensions:
                cfg["log_extensions"] = config.extensions
            if config.ignore_patterns:
                cfg["ignore_patterns"] = config.ignore_patterns
        if observer.start(cfg):
            return {"status": "started"}
        return {"status": "already running"}

    @app.post("/api/watcher/stop")
    def stop_watcher():
        observer.stop()
        return {"status": "stopped"}

    @app.get("/api/status")
    def get_status():
        return {
            "running": observer.is_running(),
            "data_root": str(data_root),
            "repos_count": len(store.list_repos()),
            "llm_provider": store.extractor.provider,
            "nvidia_configured": nvidia_configured(),
            "watch_paths": observer.config.get("watch_paths", []),
        }

    @app.get("/api/config")
    def get_config():
        return DEFAULT_CONFIG

    @app.get("/api/usage/summary")
    def usage_summary():
        return usage_log.summary()

    @app.get("/api/usage/sessions")
    def usage_sessions(limit: int = 20):
        return {"sessions": usage_log.list_sessions(limit=limit)}

    @app.get("/api/usage/interactions")
    def usage_interactions(limit: int = 50, host_ide: Optional[str] = None):
        return {"interactions": usage_log.list_interactions(limit=limit, host_ide=host_ide)}

    @app.get("/api/disk-usage")
    def disk_usage(include_workspace: bool = True):
        return build_disk_report(store, include_workspace=include_workspace)

    # --- Phase 1.5: Multi-project intelligence ---
    project_registry = ProjectRegistry(store)
    cross_repo_search = CrossRepoSearch(store)

    @app.get("/api/projects")
    def list_projects():
        return {"projects": project_registry.to_json()}

    @app.get("/api/projects/{repo_id}")
    def get_project(repo_id: str):
        info = project_registry.get_project(repo_id)
        if not info:
            raise HTTPException(404, "Project not found")
        from dataclasses import asdict
        return asdict(info)

    @app.post("/api/search/global")
    def global_search(payload: SearchPayload, request: Request):
        t0 = time.perf_counter()
        results = cross_repo_search.global_search(
            payload.query, kinds=payload.kinds, limit=payload.limit
        )
        preview = f"{len(results)} cross-repo hit(s)"
        if results:
            preview += f" — {results[0].get('repo_id')}: {results[0].get('content', '')[:80]}"
        _log_http(
            request,
            "http_global_search",
            payload.model_dump(),
            preview,
            (time.perf_counter() - t0) * 1000,
        )
        return {"results": results}

    @app.get("/api/hotspots")
    def failure_hotspots():
        return {"hotspots": cross_repo_search.failure_hotspots()}

    # --- Phase 3: Pattern intelligence ---
    mem_correlator = MemoryCorrelator(store.engine)
    pattern_detect = PatternDetector(store.engine)

    @app.get("/api/patterns/{repo_id}")
    def get_pattern_report(repo_id: str):
        return pattern_detect.get_report(repo_id)

    @app.get("/api/patterns")
    def get_global_patterns():
        return pattern_detect.get_report()

    @app.get("/api/related/{memory_id}")
    def get_related_memories(memory_id: str, limit: int = 10):
        return {"related": mem_correlator.get_related(memory_id, limit=limit)}

    # --- Phase 2: WebSocket events ---
    @app.websocket("/ws/events")
    async def websocket_events(websocket: WebSocket):
        await websocket.accept()
        q = await ws_manager.connect()
        try:
            while True:
                event: Event = await q.get()
                await websocket.send_text(event.to_json())
        except WebSocketDisconnect:
            pass
        finally:
            await ws_manager.disconnect(q)

    @app.get("/api/events/history")
    def event_history(
        since: Optional[float] = None,
        event_types: Optional[str] = None,
        repo_id: Optional[str] = None,
        limit: int = 50,
    ):
        types_list = event_types.split(",") if event_types else None
        return {"events": ws_manager.get_history(
            since=since, event_types=types_list, repo_id=repo_id, limit=limit
        )}

    # --- Relevance scoring & smart context ---

    @app.post("/api/v1/context/smart")
    def smart_context(payload: SmartContextPayload, request: Request):
        t0 = time.perf_counter()
        rid = payload.repo_id
        hits = store.search(payload.task, repo_id=rid, limit=30)
        hits = [h for h in hits if h.get("quality_tier") != "noise"]
        hits.sort(key=lambda m: (m.get("score", 0), m.get("relevance_score", 0)), reverse=True)

        context_parts = []
        token_count = 0
        for mem in hits:
            est_tokens = int(len(mem["content"].split()) * 1.3)
            if token_count + est_tokens > payload.max_tokens:
                break
            context_parts.append(mem)
            token_count += est_tokens
            try:
                store.engine.record_access(mem["id"], "context_inject", payload.task)
            except Exception:
                pass

        _log_http(
            request, "smart_context",
            {"repo_id": rid, "task": payload.task[:200]},
            f"{len(context_parts)} memories, ~{token_count} tokens",
            (time.perf_counter() - t0) * 1000,
        )
        return {
            "memories": context_parts,
            "token_estimate": token_count,
            "system_prompt_fragment": _build_prompt_fragment(rid, context_parts),
        }

    @app.post("/api/v1/feedback")
    def memory_feedback(payload: FeedbackPayload):
        store.engine.record_feedback(payload.memory_id, payload.useful, payload.context or "")
        new_score = store.engine.compute_relevance_score(payload.memory_id)
        store.engine._conn.execute(
            "UPDATE memories SET relevance_score = ? WHERE id = ?",
            (new_score, payload.memory_id),
        )
        store.engine._conn.commit()
        return {"ok": True, "new_relevance_score": new_score}

    @app.post("/api/v1/relevance/refresh")
    def refresh_relevance(repo_id: Optional[str] = None):
        noise_count = store.engine.classify_noise()
        refresh_count = store.engine.refresh_relevance_scores(repo_id)
        return {
            "refreshed": refresh_count,
            "noise_classified": noise_count,
        }

    @app.post("/api/v1/context/generate")
    def generate_context_file(repo_id: str):
        """Generate .agent-memory/context.md for a project."""
        repos_map = {}
        map_path = store.root / "repos.json"
        if map_path.exists():
            import json as _json
            try:
                repos_map = _json.loads(map_path.read_text())
            except Exception:
                pass
        project_path = repos_map.get(repo_id)
        if not project_path:
            raise HTTPException(404, f"No path found for repo {repo_id}")
        out = store.generate_context_file(repo_id, project_path)
        return {"path": str(out), "repo_id": repo_id}

    def _build_prompt_fragment(repo_id: str, memories: List[Dict]) -> str:
        if not memories:
            return ""
        lines = [f"You are working on project `{repo_id}`. Relevant context from memory:\n"]
        for m in memories:
            lines.append(f"- [{m['kind']}] {m['content'][:300]}")
        return "\n".join(lines)

    ui_dist = Path(__file__).resolve().parent.parent / "ui" / "dist"
    if serve_ui and ui_dist.exists():
        # Serve static assets
        app.mount("/assets", StaticFiles(directory=str(ui_dist / "assets")), name="assets")

        # SPA catch-all: serve index.html for any non-API route
        @app.get("/{full_path:path}")
        async def spa_fallback(request: Request, full_path: str):
            # If it's a real file in dist, serve it
            file_path = ui_dist / full_path
            if full_path and file_path.exists() and file_path.is_file():
                return FileResponse(str(file_path))
            # Otherwise serve index.html for SPA routing
            return FileResponse(str(ui_dist / "index.html"))

    return app


# Backward-compatible module-level app for imports
DATA_ROOT = Path("~/agent_companion_data").expanduser()
app = create_app(DATA_ROOT, serve_ui=False, auto_watcher=True)
