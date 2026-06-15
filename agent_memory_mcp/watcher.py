from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler
from pathlib import Path
import queue
import time
import threading
from typing import Callable, Optional, Dict
from .log_parser import LogParser, LogEvent, Severity

class WatcherHandler(FileSystemEventHandler):
    def __init__(self, q: queue.Queue, extensions: set, ignore_patterns: set, parser: LogParser):
        self.q = q
        self.extensions = extensions
        self.ignore_patterns = ignore_patterns
        self.parser = parser
        self._content_cache = {}

    def on_modified(self, event):
        if event.is_directory:
            return
        
        path = Path(event.src_path)
        
        # Check ignores
        if any(p in str(path) for p in self.ignore_patterns):
            return

        if path.suffix in self.extensions:
            # Detect what changed if it's code
            change_summary = None
            if path.suffix in ['.py', '.ts', '.tsx', '.js']:
                change_summary = self._get_change_summary(path)

            # Categorize and analyze the file
            category = self.parser.categorize_file(path)
            
            event_data = {
                'path': path,
                'category': category,
                'timestamp': time.time(),
                'change_summary': change_summary
            }
            
            if category == 'log':
                log_events = self.parser.parse_file(path)
                event_data['log_events'] = log_events
                event_data['has_errors'] = any(e.severity == Severity.ERROR for e in log_events)
            elif category == 'code':
                analysis = self.parser.analyze_code_change(path)
                event_data['code_analysis'] = analysis
            
            self.q.put(event_data)

    def _get_change_summary(self, path: Path) -> Optional[str]:
        """Simple line-by-line diff to identify the last change."""
        try:
            current = path.read_text(encoding='utf-8', errors='ignore').splitlines()
            previous = self._content_cache.get(str(path), [])
            self._content_cache[str(path)] = current

            if not previous:
                return f"Initial scan of {path.name}"
            
            added = [line.strip() for line in current if line not in previous and line.strip()]
            if added:
                return f"Added: {added[0][:50]}..."
            return "Structural or whitespace change"
        except:
            return None

class FileWatcher:
    def __init__(self, paths: list, extensions: list, ignore_patterns: list = None):
        self.q = queue.Queue()
        self.extensions = set(extensions)
        self.ignore_patterns = set(ignore_patterns or [".git", "node_modules", "__pycache__", ".venv"])
        self.paths = [Path(p).expanduser() for p in paths]
        self.observer = Observer()
        self.parser = LogParser()
        self.handler = WatcherHandler(self.q, self.extensions, self.ignore_patterns, self.parser)
        
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
        """Generator that yields enhanced event data from the queue."""
        while self._running:
            try:
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
            print(f"Event: {ev['path']} [{ev['category']}]")
            if 'log_events' in ev:
                for log_ev in ev['log_events']:
                    print(f"  -> {log_ev.severity.value}: {log_ev.message}")
    except KeyboardInterrupt:
        w.stop()
