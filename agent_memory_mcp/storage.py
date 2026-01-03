from pathlib import Path
from typing import Dict, Any, List, Optional
import json
import hashlib
import shutil
import os

class Storage:
    def __init__(self, root: str | Path):
        self.root = Path(root).expanduser()
        self.root.mkdir(parents=True, exist_ok=True)
        self.memory_dir = self.root / "agent-memory"
        self.memory_dir.mkdir(parents=True, exist_ok=True)
        self.brain_root = Path("~/.gemini/antigravity/brain").expanduser()

    def resolve_repo(self, path: str | Path) -> str:
        """Finds the root of the repo (containing .git) and returns a short hash of the path."""
        p = Path(path).resolve()
        for _ in range(20):
            if (p / ".git").exists():
                repo_id = hashlib.sha1(str(p).encode()).hexdigest()[:10]
                self._update_repo_map(repo_id, str(p))
                return repo_id
            if p.parent == p:
                break
            p = p.parent
        
        # Fallback: if it's a known project folder, use name as ID
        if "agent_memory_mcp" in str(path):
            rid = "agent-memory-v2"
            self._update_repo_map(rid, str(Path(path).parent))
            return rid
            
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
        raw_dir = self._repo_base(repo_id) / "raw-logs"
        raw_dir.mkdir(parents=True, exist_ok=True)
        target = raw_dir / log_path.name
        shutil.copy2(log_path, target)
        return target

    def append_memory(self, repo_id: str, kind: str, text: str, metadata: Optional[Dict] = None) -> None:
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
        sig_path = self._repo_base(repo_id) / "failure_signatures.json"
        sigs = []
        if sig_path.exists():
            try:
                sigs = json.loads(sig_path.read_text())
            except: pass
        
        lines = text.splitlines()
        if lines:
            sig = lines[0][:100]
            if sig not in sigs:
                sigs.append(sig)
                sig_path.write_text(json.dumps(sigs, indent=2))

    def read_memory(self, repo_id: str) -> Dict[str, Any]:
        if repo_id == "agent-brain":
            return self._read_brain_memory()

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

    def _read_brain_memory(self) -> Dict[str, Any]:
        """Virtual repo that reads from the agent's internal brain logs."""
        result = {
            "failures": "Analysis of past failures in the current session:\n",
            "decisions": "Recent strategic decisions logged by the agent.\n",
            "attempts": "Live trace of task attempts.\n",
            "state": {"session_active": True},
            "signatures": []
        }
        
        # Try to find the most recent conversation grain
        try:
            conv_dirs = [d for d in self.brain_root.iterdir() if d.is_dir()]
            conv_dirs.sort(key=lambda d: d.stat().st_mtime, reverse=True)
            if conv_dirs:
                latest = conv_dirs[0]
                task_file = latest / "task.md"
                if task_file.exists():
                    result["decisions"] += f"\n[INTERNAL TASK LOG]\n{task_file.read_text()}"
                
                # Fetch snippets from walkthrough if it exists
                walk_file = latest / "walkthrough.md"
                if walk_file.exists():
                    result["attempts"] += f"\n[INTERNAL PROGRESS]\n{walk_file.read_text()}"
        except Exception as e:
            result["failures"] += f"Error reading brain: {e}"
            
        return result

    def list_repos(self) -> Dict[str, str]:
        mapping = {}
        map_path = self.root / "repos.json"
        if map_path.exists():
            try:
                mapping = json.loads(map_path.read_text())
            except: pass
            
        if self.memory_dir.exists():
            for p in self.memory_dir.iterdir():
                if p.is_dir() and p.name not in mapping:
                    mapping[p.name] = str(p)
                    
        # Add virtual brain repo
        mapping["agent-brain"] = str(self.brain_root)
                    
        return mapping
