from pathlib import Path
from typing import Dict, Any, List, Optional
import json
import hashlib
import os
import shutil
import time

from .engine.db import MemoryEngine, _normalize_kind
from .engine.migration import import_legacy_markdown, needs_migration
from .adapters.antigravity import AntigravityBrainAdapter
from .adapters.cursor_transcript import CursorTranscriptAdapter
from .extractor import MemoryExtractor


class Storage:
    def __init__(self, root: str | Path):
        self.root = Path(root).expanduser()
        self.root.mkdir(parents=True, exist_ok=True)
        self.memory_dir = self.root / "agent-memory"
        self.memory_dir.mkdir(parents=True, exist_ok=True)
        self.engine = MemoryEngine(self.root)
        self.brain_adapter = AntigravityBrainAdapter()
        self.cursor_adapter = CursorTranscriptAdapter()
        self.extractor = MemoryExtractor(self.engine)
        self._debounce: Dict[str, float] = {}
        self._debounce_seconds = 5.0
        self._run_migration()

    def _run_migration(self) -> None:
        if not needs_migration(self.root):
            return
        map_path = self.root / "repos.json"
        mapping = {}
        if map_path.exists():
            try:
                mapping = json.loads(map_path.read_text())
            except json.JSONDecodeError:
                pass
        import_legacy_markdown(self.engine, self.memory_dir, mapping)

    def resolve_repo(self, path: str | Path) -> str:
        p = Path(path).resolve()

        if str(self.memory_dir) in str(p):
            return p.name

        for _ in range(20):
            if (p / ".git").exists():
                repo_id = hashlib.sha1(str(p).encode()).hexdigest()[:10]
                self._update_repo_map(repo_id, str(p))
                self.engine.upsert_repo(repo_id, str(p))
                return repo_id
            if p.parent == p:
                break
            p = p.parent

        if "agent_memory_mcp" in str(path):
            rid = "agent-memory-v2"
            self._update_repo_map(rid, str(Path(path).parent))
            self.engine.upsert_repo(rid, str(Path(path).parent))
            return rid

        self.engine.upsert_repo("default", str(Path(path).resolve()))
        return "default"

    def _update_repo_map(self, repo_id: str, path: str) -> None:
        map_path = self.root / "repos.json"
        mapping = {}
        if map_path.exists():
            try:
                mapping = json.loads(map_path.read_text())
            except json.JSONDecodeError:
                pass
        if mapping.get(repo_id) != path:
            mapping[repo_id] = path
            map_path.write_text(json.dumps(mapping, indent=2))

    def capture_raw_log(self, repo_id: str, log_path: Path) -> Optional[Path]:
        """Archive a raw log file for a repo (ported from Agent-Observer)."""
        raw_dir = self.memory_dir / repo_id / "raw-logs"
        raw_dir.mkdir(parents=True, exist_ok=True)
        target = raw_dir / log_path.name
        try:
            shutil.copy2(log_path, target)
            return target
        except Exception:
            return None

    def _should_debounce(self, repo_id: str, key: str) -> bool:
        now = time.time()
        dk = f"{repo_id}:{key}"
        last = self._debounce.get(dk, 0)
        if now - last < self._debounce_seconds:
            return True
        self._debounce[dk] = now
        return False

    def append_memory(
        self,
        repo_id: str,
        kind: str,
        text: str,
        metadata: Optional[Dict] = None,
        source: str = "mcp",
    ) -> Optional[str]:
        meta = metadata or {}
        file_key = meta.get("file", "")
        if kind in ("attempts", "attempt") and file_key:
            if self._should_debounce(repo_id, file_key):
                return None

        mem_id, inserted = self.engine.insert_memory(
            repo_id, kind, text, source=source, metadata=meta
        )
        if inserted and mem_id:
            self.extractor.maybe_extract(
                mem_id, repo_id, _normalize_kind(kind), text
            )
        return mem_id

    def read_memory(self, repo_id: str) -> Dict[str, Any]:
        if repo_id == "agent-brain":
            return self.brain_adapter.read_virtual_memory()
        if repo_id == "cursor-transcripts":
            return self.cursor_adapter.read_virtual_memory()

        result = {
            "failures": self.engine.aggregate_markdown(repo_id, "failure"),
            "decisions": self.engine.aggregate_markdown(repo_id, "decision"),
            "attempts": self.engine.aggregate_markdown(repo_id, "attempt"),
            "facts": self.engine.aggregate_markdown(repo_id, "fact"),
            "state": {
                "failure_signatures": self.engine.get_failure_signatures(repo_id),
            },
        }
        return result

    def search(
        self,
        query: str,
        repo_id: Optional[str] = None,
        kinds: Optional[List[str]] = None,
        limit: int = 10,
    ) -> List[Dict[str, Any]]:
        return self.engine.search(query, repo_id=repo_id, kinds=kinds, limit=limit)

    def list_memories(
        self,
        repo_id: Optional[str] = None,
        kind: Optional[str] = None,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        return self.engine.list_memories(repo_id=repo_id, kind=kind, limit=limit)

    def get_repo_context(self, repo_id: str) -> Dict[str, str]:
        return self.engine.get_repo_context(repo_id)

    def mark_failure_resolved(self, repo_id: str, signature: str) -> bool:
        return self.engine.mark_failure_resolved(repo_id, signature)

    def forget(
        self,
        memory_id: Optional[str] = None,
        signature: Optional[str] = None,
        repo_id: Optional[str] = None,
    ) -> int:
        return self.engine.forget(memory_id=memory_id, signature=signature, repo_id=repo_id)

    def list_repos(self) -> List[Dict]:
        map_path = self.root / "repos.json"
        mapping = {}
        if map_path.exists():
            try:
                mapping = json.loads(map_path.read_text())
            except json.JSONDecodeError:
                pass

        results = []
        for rid, path in mapping.items():
            err_count = self.engine.failure_error_count(rid)
            base = self.memory_dir / rid
            results.append({
                "id": rid,
                "path": path,
                "error_count": err_count,
                "health": "CRITICAL" if err_count > 0 else "HEALTHY",
                "last_modified": (
                    time.ctime(os.path.getmtime(base))
                    if base.exists()
                    else "unknown"
                ),
            })

        results.append({
            "id": "agent-brain",
            "path": str(self.brain_adapter.brain_root),
            "health": "UNKNOWN",
            "error_count": 0,
        })
        results.append({
            "id": "cursor-transcripts",
            "path": str(self.cursor_adapter.projects_root),
            "health": "UNKNOWN",
            "error_count": 0,
        })
        return results
