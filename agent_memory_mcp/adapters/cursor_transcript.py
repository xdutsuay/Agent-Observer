from pathlib import Path
from typing import Any, Dict, List

from .base import MemorySourceAdapter


class CursorTranscriptAdapter(MemorySourceAdapter):
    """Reads recent Cursor agent transcript snippets when available."""

    def __init__(self, projects_root: Path | None = None):
        self.projects_root = (projects_root or Path("~/.cursor/projects")).expanduser()

    def read_virtual_memory(self) -> Dict[str, Any]:
        result: Dict[str, Any] = {
            "failures": "",
            "decisions": "Recent Cursor agent sessions:\n",
            "attempts": "",
            "state": {"adapter": "cursor_transcript"},
        }
        snippets = self._latest_transcript_snippets(limit=3)
        for i, snip in enumerate(snippets, 1):
            result["decisions"] += f"\n--- session {i} ---\n{snip[:2000]}\n"
        return result

    def _latest_transcript_snippets(self, limit: int = 3) -> List[str]:
        if not self.projects_root.exists():
            return []
        transcripts: List[tuple[float, Path]] = []
        for proj in self.projects_root.iterdir():
            tdir = proj / "agent-transcripts"
            if not tdir.is_dir():
                continue
            for f in tdir.glob("*.jsonl"):
                try:
                    transcripts.append((f.stat().st_mtime, f))
                except OSError:
                    continue
        transcripts.sort(key=lambda x: x[0], reverse=True)
        out: List[str] = []
        for _, path in transcripts[:limit]:
            try:
                lines = path.read_text(encoding="utf-8", errors="ignore").splitlines()[-30:]
                out.append("\n".join(lines))
            except OSError:
                continue
        return out
