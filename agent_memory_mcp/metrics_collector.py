import psutil
import time
from typing import Dict, List
from dataclasses import dataclass, asdict
from collections import deque

@dataclass
class SystemMetrics:
    cpu_percent: float
    memory_used_gb: float
    memory_total_gb: float
    network_mbps: float
    fs_events_per_sec: float
    active_processes: int
    timestamp: float

class MetricsCollector:
    """Collects real-time system metrics for the dashboard."""
    
    def __init__(self, window_seconds: int = 60):
        self.window_seconds = window_seconds
        self.fs_events = deque(maxlen=1000)
        self.last_network_io = psutil.net_io_counters()
        self.last_network_time = time.time()
        
    def record_fs_event(self):
        """Record a filesystem event occurrence."""
        self.fs_events.append(time.time())
    
    def get_current_metrics(self) -> SystemMetrics:
        """Get current system metrics snapshot."""
        
        # CPU (avg over 1 second)
        cpu_percent = psutil.cpu_percent(interval=0.1)
        
        # Memory
        memory = psutil.virtual_memory()
        memory_used_gb = memory.used / (1024 ** 3)
        memory_total_gb = memory.total / (1024 ** 3)
        
        # Network (calculate MB/s since last call)
        current_network_io = psutil.net_io_counters()
        current_time = time.time()
        time_diff = current_time - self.last_network_time
        
        if time_diff > 0:
            bytes_diff = (current_network_io.bytes_sent + current_network_io.bytes_recv) - \
                        (self.last_network_io.bytes_sent + self.last_network_io.bytes_recv)
            network_mbps = (bytes_diff / time_diff) / (1024 * 1024)
        else:
            network_mbps = 0.0
        
        self.last_network_io = current_network_io
        self.last_network_time = current_time
        
        # FS Events rate (events in last window / window size)
        now = time.time()
        cutoff = now - self.window_seconds
        recent_events = [t for t in self.fs_events if t > cutoff]
        fs_events_per_sec = len(recent_events) / self.window_seconds if self.window_seconds > 0 else 0
        
        # Active processes
        active_processes = len(psutil.pids())
        
        return SystemMetrics(
            cpu_percent=round(cpu_percent, 1),
            memory_used_gb=round(memory_used_gb, 1),
            memory_total_gb=round(memory_total_gb, 1),
            network_mbps=round(network_mbps, 2),
            fs_events_per_sec=round(fs_events_per_sec, 1),
            active_processes=active_processes,
            timestamp=now
        )
    
    def get_metrics_dict(self) -> Dict:
        """Get metrics as a dictionary for JSON serialization."""
        metrics = self.get_current_metrics()
        return asdict(metrics)
    
    def detect_agent_processes(self, keywords: List[str]) -> List[Dict]:
        """Detect running processes that match agent keywords."""
        matching = []
        
        for proc in psutil.process_iter(['pid', 'name', 'cmdline']):
            try:
                name = proc.info['name'].lower() if proc.info['name'] else ''
                cmdline = ' '.join(proc.info['cmdline']).lower() if proc.info['cmdline'] else ''
                
                for keyword in keywords:
                    if keyword.lower() in name or keyword.lower() in cmdline:
                        matching.append({
                            'pid': proc.info['pid'],
                            'name': proc.info['name'],
                            'keyword_matched': keyword
                        })
                        break
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                continue
        
        return matching
