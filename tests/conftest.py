"""Shared pytest fixtures for agent_memory_mcp tests."""

import tempfile
from pathlib import Path

import pytest

from agent_memory_mcp.engine.db import MemoryEngine
from agent_memory_mcp.storage import Storage


@pytest.fixture
def tmp_root(tmp_path):
    """Provides a temporary directory as a Path."""
    return tmp_path


@pytest.fixture
def engine(tmp_root):
    """MemoryEngine with a test repo pre-registered."""
    eng = MemoryEngine(tmp_root)
    eng.upsert_repo("test-repo", "/tmp/test-project")
    yield eng
    eng.close()


@pytest.fixture
def storage(tmp_root):
    """Storage instance backed by tmp_root."""
    return Storage(tmp_root)
