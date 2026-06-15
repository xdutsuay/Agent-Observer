"""Discover git repos under watch paths so the dashboard lists real workspaces."""

from __future__ import annotations

import os
from pathlib import Path
from typing import TYPE_CHECKING, List

if TYPE_CHECKING:
    from .storage import Storage

SKIP_DIR_NAMES = {
    ".git",
    "node_modules",
    "__pycache__",
    ".venv",
    "venv",
    "dist",
    ".next",
    "build",
    ".turbo",
    "ui/node_modules",
}


def default_watch_paths() -> List[str]:
    """Paths for filesystem watcher (avoid noisy Cursor terminal logs)."""
    home = Path.home()
    candidates = [
        home / "localcode",
        Path(__file__).resolve().parent.parent,
    ]
    return [str(p) for p in candidates if p.exists()]


def discover_scan_paths() -> List[str]:
    """Broader scan for git repo registration only (not watched)."""
    paths = list(default_watch_paths())
    cursor_projects = Path.home() / ".cursor" / "projects"
    if cursor_projects.exists():
        paths.append(str(cursor_projects))
    return paths


def discover_git_repos(store: "Storage", watch_paths: List[str]) -> int:
    """Register each git root under watch_paths. Returns count of repos touched."""
    seen_paths: set[str] = set()
    count = 0

    for watch in watch_paths:
        root = Path(watch).expanduser()
        if not root.is_dir():
            continue
        for dirpath, dirnames, _ in os.walk(root, topdown=True):
            dirnames[:] = [d for d in dirnames if d not in SKIP_DIR_NAMES]
            if ".git" not in dirnames:
                continue
            repo_path = Path(dirpath).resolve()
            key = str(repo_path)
            if key in seen_paths:
                dirnames.remove(".git")
                continue
            seen_paths.add(key)
            store.resolve_repo(repo_path)
            count += 1
            dirnames.remove(".git")

    return count
