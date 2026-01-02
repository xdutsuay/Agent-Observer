import psutil
class ProcessDetector:
    def __init__(self,h):
        self.names=[n.lower() for n in h["process_name_contains"]]
    def score(self):
        for p in psutil.process_iter(["name","cmdline"]):
            n=(p.info["name"] or "").lower()
            c=" ".join(p.info["cmdline"] or []).lower()
            if any(x in n or x in c for x in self.names): return 0.7
        return 0.0
