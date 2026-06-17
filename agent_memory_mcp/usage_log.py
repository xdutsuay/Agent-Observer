"""Track MCP and HTTP interactions for usefulness analytics.

Persists to SQLite under the data root so the HTTP daemon and MCP stdio
process can share the same log.
"""

from __future__ import annotations

import json
import os
import sqlite3
import threading
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

import psutil

_INSTANCES: Dict[str, "UsageLog"] = {}
_LOCK = threading.Lock()

# Parent-process / client-name hints → friendly IDE label
IDE_HINTS: Tuple[Tuple[str, str], ...] = (
    ("claude code", "Claude Code"),
    ("claude-code", "Claude Code"),
    ("claude", "Claude Code"),
    ("cursor", "Cursor"),
    ("windsurf", "Windsurf"),
    ("antigravity", "Antigravity"),
    ("zed", "Zed"),
    ("visual studio code", "VS Code"),
    ("code helper", "VS Code"),
    ("vscode", "VS Code"),
    ("pycharm", "JetBrains"),
    ("intellij", "JetBrains"),
    ("continue", "Continue"),
    ("cline", "Cline"),
    ("aider", "Aider"),
    ("codex", "Codex CLI"),
    ("copilot", "GitHub Copilot"),
)


def _utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


def classify_host_ide(text: str) -> str:
    hay = (text or "").lower()
    for needle, label in IDE_HINTS:
        if needle in hay:
            return label
    return "Unknown"


def infer_host_ide() -> str:
    """Guess host IDE from MCP server's parent process."""
    try:
        parent = psutil.Process(os.getppid())
        name = parent.name() or ""
        cmd = " ".join(parent.cmdline() or [])
        return classify_host_ide(f"{name} {cmd}")
    except (psutil.Error, OSError):
        return "Unknown"


def detect_running_ides() -> List[Dict[str, Any]]:
    """Scan processes for known IDEs / agent CLIs (dashboard panel)."""
    found: Dict[str, Dict[str, Any]] = {}
    for proc in psutil.process_iter(["pid", "name", "cmdline"]):
        try:
            name = (proc.info.get("name") or "").lower()
            cmd = " ".join(proc.info.get("cmdline") or []).lower()
            hay = f"{name} {cmd}"
            label = classify_host_ide(hay)
            if label == "Unknown":
                continue
            if label not in found:
                found[label] = {
                    "label": label,
                    "process_count": 0,
                    "sample_pid": proc.info.get("pid"),
                }
            found[label]["process_count"] += 1
        except (psutil.Error, OSError):
            continue
    return sorted(found.values(), key=lambda x: x["label"])


def _summarize_query(method: str, query: Dict[str, Any]) -> str:
    if not query:
        return method
    parts = [method]
    for key in ("query", "path", "repo_id", "kind", "content", "text", "signature", "memory_id"):
        if key in query and query[key]:
            val = str(query[key])
            if len(val) > 120:
                val = val[:117] + "..."
            parts.append(f"{key}={val}")
    return " | ".join(parts)


def _preview(text: str, limit: int = 600) -> str:
    text = (text or "").strip()
    if len(text) <= limit:
        return text
    return text[: limit - 3] + "..."


class UsageLog:
    def __init__(self, root: Path):
        self.root = root.expanduser()
        self.root.mkdir(parents=True, exist_ok=True)
        self.db_path = self.root / "usage.db"
        self._conn = sqlite3.connect(str(self.db_path), check_same_thread=False)
        self._conn.row_factory = sqlite3.Row
        self._lock = threading.RLock()
        self._init_schema()

    @classmethod
    def for_root(cls, root: Path | str) -> "UsageLog":
        key = str(Path(root).expanduser())
        with _LOCK:
            if key not in _INSTANCES:
                _INSTANCES[key] = cls(Path(key))
            return _INSTANCES[key]

    def _init_schema(self) -> None:
        with self._lock:
            self._conn.executescript(
                """
                CREATE TABLE IF NOT EXISTS mcp_sessions (
                    id TEXT PRIMARY KEY,
                    client_name TEXT,
                    client_version TEXT,
                    host_ide TEXT,
                    transport TEXT,
                    connected_at TEXT,
                    last_seen_at TEXT
                );
                CREATE TABLE IF NOT EXISTS interactions (
                    id TEXT PRIMARY KEY,
                    session_id TEXT,
                    transport TEXT NOT NULL,
                    method TEXT NOT NULL,
                    client_name TEXT,
                    client_version TEXT,
                    host_ide TEXT,
                    query_summary TEXT,
                    query_json TEXT,
                    response_preview TEXT,
                    duration_ms REAL,
                    ok INTEGER DEFAULT 1,
                    created_at TEXT NOT NULL
                );
                CREATE INDEX IF NOT EXISTS idx_interactions_created
                    ON interactions(created_at DESC);
                CREATE INDEX IF NOT EXISTS idx_interactions_host
                    ON interactions(host_ide);
                """
            )
            self._conn.commit()

    def _session_key(
        self,
        transport: str,
        client_name: str,
        host_ide: str,
    ) -> str:
        return f"{transport}:{client_name}:{host_ide}"

    def _touch_session(
        self,
        transport: str,
        client_name: str,
        client_version: str,
        host_ide: str,
    ) -> str:
        sid = self._session_key(transport, client_name, host_ide)
        now = _utc_now()
        with self._lock:
            row = self._conn.execute(
                "SELECT id FROM mcp_sessions WHERE id = ?", (sid,)
            ).fetchone()
            if row:
                self._conn.execute(
                    "UPDATE mcp_sessions SET last_seen_at = ?, client_version = ? WHERE id = ?",
                    (now, client_version, sid),
                )
            else:
                self._conn.execute(
                    """INSERT INTO mcp_sessions
                       (id, client_name, client_version, host_ide, transport, connected_at, last_seen_at)
                       VALUES (?, ?, ?, ?, ?, ?, ?)""",
                    (sid, client_name, client_version, host_ide, transport, now, now),
                )
            self._conn.commit()
        return sid

    def record(
        self,
        *,
        transport: str,
        method: str,
        query: Optional[Dict[str, Any]] = None,
        response_preview: str = "",
        client_name: str = "unknown",
        client_version: str = "",
        host_ide: Optional[str] = None,
        duration_ms: float = 0.0,
        ok: bool = True,
    ) -> str:
        host = host_ide or classify_host_ide(client_name) or infer_host_ide()
        if host == "Unknown" and client_name != "unknown":
            host = classify_host_ide(client_name)
        sid = self._touch_session(transport, client_name, client_version, host)
        iid = str(uuid.uuid4())
        q = query or {}
        with self._lock:
            self._conn.execute(
                """INSERT INTO interactions
                   (id, session_id, transport, method, client_name, client_version, host_ide,
                    query_summary, query_json, response_preview, duration_ms, ok, created_at)
                   VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)""",
                (
                    iid,
                    sid,
                    transport,
                    method,
                    client_name,
                    client_version,
                    host,
                    _summarize_query(method, q),
                    json.dumps(q, default=str)[:4000],
                    _preview(response_preview, 2000),
                    duration_ms,
                    1 if ok else 0,
                    _utc_now(),
                ),
            )
            self._conn.commit()
        return iid

    def list_interactions(self, limit: int = 50, host_ide: Optional[str] = None) -> List[Dict[str, Any]]:
        limit = max(1, min(limit, 200))
        with self._lock:
            if host_ide:
                rows = self._conn.execute(
                    """SELECT * FROM interactions WHERE host_ide = ?
                       ORDER BY created_at DESC LIMIT ?""",
                    (host_ide, limit),
                ).fetchall()
            else:
                rows = self._conn.execute(
                    "SELECT * FROM interactions ORDER BY created_at DESC LIMIT ?",
                    (limit,),
                ).fetchall()
        return [dict(r) for r in rows]

    def list_sessions(self, limit: int = 20) -> List[Dict[str, Any]]:
        limit = max(1, min(limit, 100))
        with self._lock:
            rows = self._conn.execute(
                """SELECT s.*,
                          (SELECT COUNT(*) FROM interactions i WHERE i.session_id = s.id) AS call_count,
                          (SELECT MAX(created_at) FROM interactions i WHERE i.session_id = s.id) AS last_call
                   FROM mcp_sessions s
                   ORDER BY last_seen_at DESC LIMIT ?""",
                (limit,),
            ).fetchall()
        return [dict(r) for r in rows]

    def summary(self) -> Dict[str, Any]:
        with self._lock:
            total = self._conn.execute("SELECT COUNT(*) FROM interactions").fetchone()[0]
            last_24h = self._conn.execute(
                """SELECT COUNT(*) FROM interactions
                   WHERE datetime(created_at) >= datetime('now', '-1 day')"""
            ).fetchone()[0]
            by_method = self._conn.execute(
                """SELECT method, COUNT(*) AS n FROM interactions
                   GROUP BY method ORDER BY n DESC LIMIT 10"""
            ).fetchall()
            by_host = self._conn.execute(
                """SELECT host_ide, COUNT(*) AS n FROM interactions
                   GROUP BY host_ide ORDER BY n DESC"""
            ).fetchall()
            by_transport = self._conn.execute(
                """SELECT transport, COUNT(*) AS n FROM interactions
                   GROUP BY transport ORDER BY n DESC"""
            ).fetchall()
            reads = self._conn.execute(
                """SELECT COUNT(*) FROM interactions
                   WHERE method IN ('search_memory','global_search','get_repo_context',
                                    'switch_project_context','get_related_memories',
                                    'find_similar_failures','read_resource','inject_memory_context',
                                    'http_search','http_global_search','http_get_memory')"""
            ).fetchone()[0]
            writes = self._conn.execute(
                """SELECT COUNT(*) FROM interactions
                   WHERE method IN ('remember','add_memory','http_add_memory')"""
            ).fetchone()[0]
        return {
            "total_interactions": total,
            "last_24h": last_24h,
            "reads": reads,
            "writes": writes,
            "by_method": [{"method": m, "count": n} for m, n in by_method],
            "by_host_ide": [{"host_ide": h, "count": n} for h, n in by_host],
            "by_transport": [{"transport": t, "count": n} for t, n in by_transport],
            "running_ides": detect_running_ides(),
        }
