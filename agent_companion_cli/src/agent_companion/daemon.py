import time
from agent_companion.config_loader import load_config, load_heuristics
from agent_companion.detector.activity_score import ActivityScorer
from agent_companion.detector.process_detector import ProcessDetector
from agent_companion.watcher.fs_watcher import FileWatcher
from agent_companion.repo.resolver import RepoResolver
from agent_companion.memory.store import MemoryStore
from agent_companion.runtime_state import RuntimeState

class AgentCompanionDaemon:
    def __init__(self):
        self.config = load_config()
        self.heur = load_heuristics()
        self.state = RuntimeState(self.config["output_root"])
        self.det = ProcessDetector(self.heur)
        self.score = ActivityScorer(self.heur)
        self.repo = RepoResolver()
        self.mem = MemoryStore(self.config["output_root"])
        self.watch = FileWatcher(self.config["watch_paths"], self.config["log_extensions"])

    def run(self):
        for ev in self.watch.events():
            self.score.record_fs_event()
            s = self.score.score(self.det.score())
            self.state.update(s)
            if s < self.heur["activation_threshold"]: continue
            rid = self.repo.resolve(ev)
            if not rid: continue
            p = self.mem.capture_raw(rid, ev)
            self.mem.update_memory(rid, p)
            time.sleep(self.config["poll_interval_seconds"])

    def print_status(self):
        print(self.state.read())
