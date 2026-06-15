"""One-time import of legacy markdown memory files into SQLite."""

from __future__ import annotations

import json
import re
from pathlib import Path
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from .db import MemoryEngine

ENTRY_RE = re.compile(r"^### (.+)$", re.MULTILINE)


def import_legacy_markdown(engine: "MemoryEngine", memory_dir: Path, repos_map: dict) -> int:
    """Import */memory/*.md and state.json failure_analytics. Returns count imported."""
    if not memory_dir.exists():
        return 0

    imported = 0
    for repo_dir in memory_dir.iterdir():
        if not repo_dir.is_dir():
            continue
        repo_id = repo_dir.name
        mem_path = repo_dir / "memory"
        if not mem_path.exists():
            continue

        path = repos_map.get(repo_id, str(repo_dir))
        engine.upsert_repo(repo_id, path)

        for kind_file, kind in [
            ("failures.md", "failure"),
            ("decisions.md", "decision"),
            ("attempts.md", "attempt"),
        ]:
            fpath = mem_path / kind_file
            if not fpath.exists():
                continue
            text = fpath.read_text(encoding="utf-8")
            blocks = _split_markdown_entries(text)
            for ts, body in blocks:
                if not body.strip():
                    continue
                _, inserted = engine.insert_memory(
                    repo_id,
                    kind,
                    body.strip(),
                    source="import",
                    metadata={"legacy_timestamp": ts},
                    skip_failure_dedup=True,
                )
                if inserted:
                    imported += 1

        state_path = mem_path / "state.json"
        if state_path.exists():
            try:
                state = json.loads(state_path.read_text())
                for sig, data in state.get("failure_analytics", {}).items():
                    engine._conn.execute(
                        """
                        INSERT OR IGNORE INTO failure_signatures
                        (repo_id, signature, count, first_seen, last_seen, resolved, memory_id)
                        VALUES (?, ?, ?, ?, ?, ?, NULL)
                        """,
                        (
                            repo_id,
                            sig,
                            data.get("count", 1),
                            data.get("first_seen", ""),
                            data.get("last_seen", ""),
                            1 if data.get("resolved") else 0,
                        ),
                    )
                engine._conn.commit()
            except (json.JSONDecodeError, OSError):
                pass

    marker = memory_dir.parent / ".migrated_to_sqlite"
    if imported > 0 or memory_dir.exists():
        marker.write_text("ok")
    return imported


def _split_markdown_entries(text: str) -> list[tuple[str, str]]:
    parts: list[tuple[str, str]] = []
    matches = list(ENTRY_RE.finditer(text))
    if not matches:
        if text.strip():
            parts.append(("", text))
        return parts
    for i, m in enumerate(matches):
        ts = m.group(1)
        start = m.end()
        end = matches[i + 1].start() if i + 1 < len(matches) else len(text)
        parts.append((ts, text[start:end]))
    return parts


def needs_migration(root: Path) -> bool:
    marker = root / ".migrated_to_sqlite"
    memory_dir = root / "agent-memory"
    return memory_dir.exists() and not marker.exists()
