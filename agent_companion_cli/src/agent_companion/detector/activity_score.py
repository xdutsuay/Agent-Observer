import time
from collections import deque
class ActivityScorer:
    def __init__(self,h):
        self.w=h["churn_window_seconds"]; self.m=h["min_fs_events"]
        self.e=deque()
    def record_fs_event(self):
        t=time.time(); self.e.append(t)
        while self.e and self.e[0]<t-self.w: self.e.popleft()
    def score(self,p):
        c=1.0 if len(self.e)>=self.m else 0.0
        return 0.7*p+0.3*c
