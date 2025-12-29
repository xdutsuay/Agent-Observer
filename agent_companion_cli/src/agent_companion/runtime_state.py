import json, time
from pathlib import Path

class RuntimeState:
    def __init__(self, root):
        self.p = Path(root).expanduser()/ "runtime"
        self.p.mkdir(parents=True, exist_ok=True)

    def update(self, score):
        with open(self.p/"status.json","w") as f:
            json.dump({"score":score,"ts":time.time()},f)

    def read(self):
        p=self.p/"status.json"
        return p.read_text() if p.exists() else "{}"
