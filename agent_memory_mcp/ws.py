"""WebSocket manager for real-time memory and watcher events."""

from __future__ import annotations

import asyncio
import json
import time
from dataclasses import dataclass, field
from enum import Enum
from typing import Any, Dict, List, Optional, Set


class EventType(str, Enum):
    MEMORY_CREATED = "memory.created"
    MEMORY_DELETED = "memory.deleted"
    FAILURE_DETECTED = "failure.detected"
    FAILURE_RESOLVED = "failure.resolved"
    FILE_CHANGED = "file.changed"
    REPO_DISCOVERED = "repo.discovered"
    WATCHER_STATUS = "watcher.status"


@dataclass
class Event:
    type: EventType
    data: Dict[str, Any]
    timestamp: float = field(default_factory=time.time)
    repo_id: Optional[str] = None

    def to_json(self) -> str:
        return json.dumps(
            {
                "type": self.type.value,
                "data": self.data,
                "timestamp": self.timestamp,
                "repo_id": self.repo_id,
            }
        )


class WebSocketManager:
    """Manages WebSocket connections and broadcasts events."""

    def __init__(self, max_history: int = 200):
        self._connections: Set[asyncio.Queue] = set()  # type: ignore[type-arg]
        self._history: List[Event] = []
        self._max_history = max_history
        self._lock = asyncio.Lock()

    async def connect(self) -> asyncio.Queue:  # type: ignore[type-arg]
        """Register a new WebSocket client. Returns a queue to read events from."""
        q: asyncio.Queue = asyncio.Queue(maxsize=100)  # type: ignore[type-arg]
        async with self._lock:
            self._connections.add(q)
        return q

    async def disconnect(self, q: asyncio.Queue) -> None:  # type: ignore[type-arg]
        """Unregister a WebSocket client."""
        async with self._lock:
            self._connections.discard(q)

    async def broadcast(self, event: Event) -> None:
        """Send an event to all connected clients."""
        self._history.append(event)
        if len(self._history) > self._max_history:
            self._history = self._history[-self._max_history :]

        dead: List[asyncio.Queue] = []  # type: ignore[type-arg]
        async with self._lock:
            for q in self._connections:
                try:
                    q.put_nowait(event)
                except asyncio.QueueFull:
                    dead.append(q)
            for q in dead:
                self._connections.discard(q)

    def emit_sync(self, event: Event) -> None:
        """Thread-safe emit from synchronous code (e.g., watcher callbacks)."""
        try:
            loop = asyncio.get_running_loop()
            loop.call_soon_threadsafe(asyncio.ensure_future, self.broadcast(event))
        except RuntimeError:
            self._history.append(event)
            if len(self._history) > self._max_history:
                self._history = self._history[-self._max_history :]

    def get_history(
        self,
        since: Optional[float] = None,
        event_types: Optional[List[str]] = None,
        repo_id: Optional[str] = None,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """Get recent event history with optional filters."""
        events = self._history
        if since:
            events = [e for e in events if e.timestamp >= since]
        if event_types:
            events = [e for e in events if e.type.value in event_types]
        if repo_id:
            events = [e for e in events if e.repo_id == repo_id]
        return [json.loads(e.to_json()) for e in events[-limit:]]

    @property
    def connection_count(self) -> int:
        return len(self._connections)


# Module-level singleton
ws_manager = WebSocketManager()
