from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler
from pathlib import Path
import queue

class FileWatcher:
    def __init__(self,paths,ext):
        self.q=queue.Queue(); self.ext=set(ext)
        o=Observer(); h=self.H(self.q,self.ext)
        for p in paths: o.schedule(h,str(Path(p)),recursive=True)
        o.start(); self.o=o
    def events(self):
        while True: yield self.q.get()
    class H(FileSystemEventHandler):
        def __init__(s,q,e): s.q=q; s.e=e
        def on_modified(s,e):
            p=Path(e.src_path)
            if p.suffix in s.e: s.q.put(p)
