from pathlib import Path
import hashlib
class RepoResolver:
    def resolve(self,p):
        p=Path(p).parent
        for _ in range(20):
            if (p/".git").exists():
                return hashlib.sha1(str(p).encode()).hexdigest()[:10]
            if p.parent==p: break
            p=p.parent
        return None
