import time
from collections import deque
from typing import Dict

class ActivityScorer:
    """
    Calculates an 'activity score' based on filesystem events and agent process presence.
    High score indicates high probability that an agent is actively working.
    """
    def __init__(self, config: Dict):
        self.window_seconds = config.get("churn_window_seconds", 60)
        self.min_events = config.get("min_fs_events", 5)
        self.events = deque()

    def record_fs_event(self):
        """Record a filesystem event timestamp."""
        now = time.time()
        self.events.append(now)
        self._cleanup(now)

    def _cleanup(self, now: float):
        """Remove events outside the window."""
        cutoff = now - self.window_seconds
        while self.events and self.events[0] < cutoff:
            self.events.popleft()

    def get_fs_activity_factor(self) -> float:
        """Returns 0.0 to 1.0 based on event density."""
        self._cleanup(time.time())
        count = len(self.events)
        if count == 0:
            return 0.0
        # Normalize: min_events = 1.0 activity
        return min(1.0, count / self.min_events)

    def calculate_score(self, process_score: float) -> float:
        """
        Final score: weighted average of process presence (70%) and FS activity (30%).
        """
        fs_factor = self.get_fs_activity_factor()
        # 0.7 weight for process detection, 0.3 for FS churn
        score = (0.7 * process_score) + (0.3 * fs_factor)
        return min(1.0, score)

