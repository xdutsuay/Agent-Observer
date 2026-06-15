from pathlib import Path
from typing import Any, Dict

from .base import MemorySourceAdapter


class AntigravityBrainAdapter(MemorySourceAdapter):
    def __init__(self, brain_root: Path | None = None):
        self.brain_root = (brain_root or Path("~/.gemini/antigravity/brain")).expanduser()

    def read_virtual_memory(self) -> Dict[str, Any]:
        result: Dict[str, Any] = {
            "failures": "Analysis of past failures in the current session:\n",
            "decisions": "Recent strategic decisions logged by the agent.\n",
            "attempts": "Live trace of task attempts.\n",
            "state": {"session_active": True},
        }
        try:
            if not self.brain_root.exists():
                return result
            conv_dirs = [d for d in self.brain_root.iterdir() if d.is_dir()]
            conv_dirs.sort(key=lambda d: d.stat().st_mtime, reverse=True)
            if not conv_dirs:
                return result
            latest = conv_dirs[0]
            task_file = latest / "task.md"
            if task_file.exists():
                result["decisions"] += f"\n[INTERNAL TASK LOG]\n{task_file.read_text()}"
            walk_file = latest / "walkthrough.md"
            if walk_file.exists():
                result["attempts"] += f"\n[INTERNAL PROGRESS]\n{walk_file.read_text()}"
        except OSError as e:
            result["failures"] += f"Error reading brain: {e}"
        return result
