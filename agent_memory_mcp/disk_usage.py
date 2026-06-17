"""Disk usage for agent data store and tracked project workspaces."""

from __future__ import annotations

import os
import subprocess
import time
from pathlib import Path
from typing import TYPE_CHECKING, Any, Dict, List, Optional

if TYPE_CHECKING:
    from .storage import Storage

SKIP_DIR_NAMES = frozenset({
    ".git",
    "node_modules",
    "__pycache__",
    ".venv",
    "venv",
    "dist",
    "build",
    ".next",
    "target",
    "vendor",
    ".turbo",
    ".cache",
    "coverage",
    ".pytest_cache",
    "graphify-out",
})

_CACHE: Dict[str, Any] = {}
_CACHE_TTL = 45.0


def format_bytes(num: int | float) -> str:
    n = float(num)
    if n < 0:
        n = 0.0
    units = ("B", "KB", "MB", "GB", "TB")
    for unit in units:
        if n < 1024 or unit == units[-1]:
            if unit == "B":
                return f"{int(n)} B"
            return f"{n:.1f} {unit}"
        n /= 1024
    return f"{n:.1f} TB"


def _file_size(path: Path) -> int:
    try:
        return path.stat().st_size if path.is_file() else 0
    except OSError:
        return 0


def _dir_tree_size(path: Path) -> int:
    total = 0
    if not path.exists():
        return 0
    if path.is_file():
        return _file_size(path)
    try:
        for entry in path.rglob("*"):
            if entry.is_file():
                total += _file_size(entry)
    except OSError:
        pass
    return total


def workspace_size_bytes(path: Path, timeout_sec: float = 25.0) -> Optional[int]:
    """Size of a project directory on disk (skips heavy folders via du where available)."""
    if not path.exists():
        return None
    try:
        proc = subprocess.run(
            ["du", "-sk", str(path)],
            capture_output=True,
            text=True,
            timeout=timeout_sec,
        )
        if proc.returncode == 0 and proc.stdout.strip():
            kb = int(proc.stdout.split()[0])
            return kb * 1024
    except (subprocess.TimeoutExpired, ValueError, OSError, FileNotFoundError):
        pass
    return _walk_size_capped(path)


def _walk_size_capped(root: Path, max_files: int = 40_000) -> int:
    total = 0
    count = 0
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in SKIP_DIR_NAMES]
        for name in filenames:
            count += 1
            if count > max_files:
                return total
            try:
                total += (Path(dirpath) / name).stat().st_size
            except OSError:
                continue
    return total


def _memory_bytes_by_repo(engine) -> Dict[str, Dict[str, int]]:
    """Bytes attributed to each repo in SQLite (content + metadata + embeddings)."""
    with engine._lock:
        text_rows = engine._conn.execute(
            """
            SELECT repo_id,
                   SUM(
                     LENGTH(content)
                     + LENGTH(COALESCE(metadata_json, ''))
                     + LENGTH(COALESCE(summary, ''))
                   ) AS text_bytes,
                   COUNT(*) AS memory_count
            FROM memories
            WHERE deleted = 0
            GROUP BY repo_id
            """
        ).fetchall()
        emb_rows = engine._conn.execute(
            """
            SELECT m.repo_id, SUM(LENGTH(COALESCE(e.embedding_json, ''))) AS emb_bytes
            FROM memory_embeddings e
            JOIN memories m ON m.id = e.memory_id
            WHERE m.deleted = 0
            GROUP BY m.repo_id
            """
        ).fetchall()

    out: Dict[str, Dict[str, int]] = {}
    for row in text_rows:
        rid = row["repo_id"]
        out[rid] = {
            "memory_bytes": int(row["text_bytes"] or 0),
            "embedding_bytes": 0,
            "memory_count": int(row["memory_count"] or 0),
        }
    for row in emb_rows:
        rid = row["repo_id"]
        if rid not in out:
            out[rid] = {"memory_bytes": 0, "embedding_bytes": 0, "memory_count": 0}
        out[rid]["embedding_bytes"] = int(row["emb_bytes"] or 0)
    for rid, vals in out.items():
        vals["memory_store_bytes"] = vals["memory_bytes"] + vals["embedding_bytes"]
    return out


def data_root_breakdown(root: Path) -> Dict[str, int]:
    root = root.expanduser()
    files = {
        "memory_db": _file_size(root / "memory.db"),
        "usage_db": _file_size(root / "usage.db"),
        "repos_json": _file_size(root / "repos.json"),
        "legacy_markdown": _dir_tree_size(root / "agent-memory"),
    }
    files["other"] = max(
        0,
        _dir_tree_size(root)
        - sum(files.values()),
    )
    files["data_root_total"] = sum(files.values())
    return files


def build_disk_report(storage: "Storage", include_workspace: bool = True) -> Dict[str, Any]:
    cache_key = str(storage.root.expanduser())
    cached = _CACHE.get(cache_key)
    if cached and (time.time() - cached["ts"]) < _CACHE_TTL:
        return cached["data"]

    memory_by_repo = _memory_bytes_by_repo(storage.engine)
    repos = storage.list_repos()
    projects: List[Dict[str, Any]] = []

    for repo in repos:
        rid = repo["id"]
        path = Path(repo.get("path", ""))
        mem = memory_by_repo.get(
            rid,
            {"memory_bytes": 0, "embedding_bytes": 0, "memory_store_bytes": 0, "memory_count": 0},
        )
        workspace_bytes: Optional[int] = None
        if include_workspace and path.exists():
            workspace_bytes = workspace_size_bytes(path)
        projects.append({
            "repo_id": rid,
            "name": path.name or rid,
            "path": str(path),
            "memory_bytes": mem["memory_bytes"],
            "embedding_bytes": mem["embedding_bytes"],
            "memory_store_bytes": mem["memory_store_bytes"],
            "memory_count": mem["memory_count"],
            "workspace_bytes": workspace_bytes,
            "workspace_bytes_human": format_bytes(workspace_bytes) if workspace_bytes is not None else None,
            "memory_store_bytes_human": format_bytes(mem["memory_store_bytes"]),
        })

    projects.sort(key=lambda p: p["memory_store_bytes"] + (p["workspace_bytes"] or 0), reverse=True)

    breakdown = data_root_breakdown(storage.root)
    total_memory_store = sum(v["memory_store_bytes"] for v in memory_by_repo.values())
    total_workspace = sum(p["workspace_bytes"] or 0 for p in projects if p["workspace_bytes"] is not None)

    # Repos with memories but not in registry
    known_ids = {p["repo_id"] for p in projects}
    for rid, mem in memory_by_repo.items():
        if rid in known_ids or mem["memory_store_bytes"] == 0:
            continue
        projects.append({
            "repo_id": rid,
            "name": rid,
            "path": "",
            "memory_bytes": mem["memory_bytes"],
            "embedding_bytes": mem["embedding_bytes"],
            "memory_store_bytes": mem["memory_store_bytes"],
            "memory_count": mem["memory_count"],
            "workspace_bytes": None,
            "workspace_bytes_human": None,
            "memory_store_bytes_human": format_bytes(mem["memory_store_bytes"]),
        })
    projects.sort(key=lambda p: p["memory_store_bytes"] + (p["workspace_bytes"] or 0), reverse=True)

    report = {
        "data_root": str(storage.root.expanduser()),
        "overall": {
            "data_root_bytes": breakdown["data_root_total"],
            "data_root_bytes_human": format_bytes(breakdown["data_root_total"]),
            "memory_db_bytes": breakdown["memory_db"],
            "usage_db_bytes": breakdown["usage_db"],
            "legacy_markdown_bytes": breakdown["legacy_markdown"],
            "total_memory_attributed_bytes": total_memory_store,
            "total_memory_attributed_human": format_bytes(total_memory_store),
            "total_workspace_bytes": total_workspace,
            "total_workspace_human": format_bytes(total_workspace),
            "project_count": len(projects),
        },
        "breakdown": {k: v for k, v in breakdown.items()},
        "breakdown_human": {k: format_bytes(v) for k, v in breakdown.items()},
        "projects": projects,
    }

    _CACHE[cache_key] = {"ts": time.time(), "data": report}
    return report
