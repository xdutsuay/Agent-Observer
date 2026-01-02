from pathlib import Path
from typing import Dict, Any, List, Optional
import json
import hashlib
import shutil

class Storage:
    def __init__(self, root: str | Path):
        self.root = Path(root).expanduser()
        self.root.mkdir(parents=True, exist_ok=True)
        self.memory_dir = self.root / "agent-memory"
        self.memory_dir.mkdir(parents=True, exist_ok=True)

    def resolve_repo(self, path: str | Path) -> str:
        """Finds the root of the repo (containing .git) and returns a short hash of the path."""
        p = Path(path).resolve()
        for _ in range(20):
            if (p / ".git").exists():
                repo_id = hashlib.sha1(str(p).encode()).hexdigest()[:10]
                # Store a mapping of hash to path for UI display
                self._update_repo_map(repo_id, str(p))
                return repo_id
            if p.parent == p:
                break
            p = p.parent
        return "default"

    def _update_repo_map(self, repo_id: str, path: str):
        map_path = self.root / "repos.json"
        mapping = {}
        if map_path.exists():
            try:
                mapping = json.loads(map_path.read_text())
            except: pass
        mapping[repo_id] = path
        map_path.write_text(json.dumps(mapping, indent=2))

    def _repo_base(self, repo_id: str) -> Path:
        base = self.memory_dir / repo_id
        base.mkdir(parents=True, exist_ok=True)
        return base

    def capture_raw_log(self, repo_id: str, log_path: Path):
        """Copies a log file to a 'raw' directory for later analysis."""
        raw_dir = self._repo_base(repo_id) / "raw-logs"
        raw_dir.mkdir(parents=True, exist_ok=True)
        target = raw_dir / log_path.name
        shutil.copy2(log_path, target)
        return target

    def append_memory(self, repo_id: str, kind: str, text: str, metadata: Optional[Dict] = None) -> None:
        """Appends structured memory. Kind can be 'attempts', 'failures', 'decisions'."""
        mem_dir = self._repo_base(repo_id) / "memory"
        mem_dir.mkdir(parents=True, exist_ok=True)
        path = mem_dir / f"{kind}.md"
        
        timestamp = metadata.get('timestamp', '---') if metadata else '---'
        entry = f"### {timestamp}\n{text.rstrip()}\n\n"
        with open(path, "a", encoding="utf-8") as f:
            f.write(entry)

        if kind == "failures":
            self._update_failure_signatures(repo_id, text)

    def _update_failure_signatures(self, repo_id: str, text: str):
        """Simple signature tracking to prevent repeated failures."""
        sig_path = self._repo_base(repo_id) / "failure_signatures.json"
        sigs = []
        if sig_path.exists():
            try:
                sigs = json.loads(sig_path.read_text())
            except: pass
        
        # Simple heuristic: first line if it looks like an error
        lines = text.splitlines()
        if lines:
            sig = lines[0][:100]
            if sig not in sigs:
                sigs.append(sig)
                sig_path.write_text(json.dumps(sigs, indent=2))

    def write_state(self, repo_id: str, state: Dict[str, Any]) -> None:
        mem_dir = self._repo_base(repo_id) / "memory"
        mem_dir.mkdir(parents=True, exist_ok=True)
        path = mem_dir / "state.json"
        with open(path, "w", encoding="utf-8") as f:
            json.dump(state, f, indent=2)

    def read_memory(self, repo_id: str) -> Dict[str, Any]:
        mem_dir = self._repo_base(repo_id) / "memory"
        result = {
            "failures": "",
            "decisions": "",
            "attempts": "",
            "state": {},
            "signatures": []
        }

        for key in ("failures", "decisions", "attempts"):
            path = mem_dir / f"{key}.md"
            if path.exists():
                result[key] = path.read_text(encoding="utf-8")

        state_path = mem_dir / "state.json"
        if state_path.exists():
            result["state"] = json.loads(state_path.read_text(encoding="utf-8"))

        sig_path = self._repo_base(repo_id) / "failure_signatures.json"
        if sig_path.exists():
            try:
                result["signatures"] = json.loads(sig_path.read_text())
            except: pass

        return result

    def list_repos(self) -> Dict[str, str]:
        map_path = self.root / "repos.json"
        if map_path.exists():
            try:
                return json.loads(map_path.read_text())
            except: pass
        return {}
