from abc import ABC, abstractmethod
from typing import Any, Dict


class MemorySourceAdapter(ABC):
    @abstractmethod
    def read_virtual_memory(self) -> Dict[str, Any]:
        """Return dict with failures, decisions, attempts, state keys."""
