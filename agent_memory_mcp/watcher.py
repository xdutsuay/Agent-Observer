from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler
from pathlib import Path
import queue
import time
import threading

class WatcherHandler(FileSystemEventHandler):
    def __init__(self, q: queue.Queue, extensions: set, ignore_patterns: set):
        self.q = q
        self.extensions = extensions
        self.ignore_patterns = ignore_patterns

    def on_modified(self, event):
        if event.is_directory:
            return
        
        path = Path(event.src_path)
        
        # Check ignores
        if any(p in str(path) for p in self.ignore_patterns):
            return

        if path.suffix in self.extensions:
            self.q.put(path)

class FileWatcher:
    def __init__(self, paths: list, extensions: list, ignore_patterns: list = None):
        self.q = queue.Queue()
        self.extensions = set(extensions)
        self.ignore_patterns = set(ignore_patterns or [".git", "node_modules", "__pycache__", ".venv"])
        self.paths = [Path(p).expanduser() for p in paths]
        self.observer = Observer()
        self.handler = WatcherHandler(self.q, self.extensions, self.ignore_patterns)
        
        for p in self.paths:
            if p.exists():
                self.observer.schedule(self.handler, str(p), recursive=True)

        self._running = False

    def start(self):
        if not self._running:
            self.observer.start()
            self._running = True

    def stop(self):
        if self._running:
            self.observer.stop()
            self.observer.join()
            self._running = False

    def events(self):
        """Generator that yields events from the queue."""
        while self._running:
            try:
                # Use timeout to allow checking self._running
                yield self.q.get(timeout=1.0)
            except queue.Empty:
                continue

if __name__ == "__main__":
    # Test
    w = FileWatcher(["."], [".py", ".log", ".txt"])
    w.start()
    try:
        print("Watching... Press Ctrl+C to stop.")
        for ev in w.events():
            print(f"File modified: {ev}")
    except KeyboardInterrupt:
        w.stop()
