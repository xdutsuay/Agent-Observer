from pathlib import Path
import shutil
class MemoryStore:
    def __init__(self,r):
        self.r=Path(r).expanduser()
    def capture_raw(self,rid,p):
        b=self.r/"agent-memory"/rid/"raw-logs"
        b.mkdir(parents=True,exist_ok=True)
        d=b/p.name; shutil.copy2(p,d); return d
    def update_memory(self,rid,p):
        m=self.r/"agent-memory"/rid/"memory"
        m.mkdir(parents=True,exist_ok=True)
        t=p.read_text(errors="ignore").lower()
        if "error" in t:
            with open(m/"failures.md","a") as f: f.write(f"- {p.name}\n")
        with open(m/"attempts.md","a") as f: f.write(f"- {p.name}\n")
