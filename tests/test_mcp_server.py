"""Tests for MCP server tools (via Storage layer — thin wrappers)."""

import tempfile
from pathlib import Path

import pytest

from agent_memory_mcp.storage import Storage
from agent_memory_mcp.mcp_server import create_mcp_server
from agent_memory_mcp.project_registry import ProjectRegistry
from agent_memory_mcp.cross_repo import CrossRepoSearch
from agent_memory_mcp.correlator import MemoryCorrelator
from agent_memory_mcp.pattern_detector import PatternDetector


@pytest.fixture
def store(tmp_path):
    s = Storage(tmp_path)
    s.engine.upsert_repo("proj-a", str(tmp_path / "project-a"))
    s.engine.upsert_repo("proj-b", str(tmp_path / "project-b"))
    # Also register in repos.json so list_repos() finds them
    s._update_repo_map("proj-a", str(tmp_path / "project-a"))
    s._update_repo_map("proj-b", str(tmp_path / "project-b"))
    return s


def test_remember_and_search(store):
    """remember → search_memory round-trip."""
    store.append_memory("proj-a", "failure", "Connection timeout on port 5432", source="mcp")
    store.append_memory("proj-a", "decision", "Use connection pooling", source="mcp")
    hits = store.search("timeout connection", repo_id="proj-a", limit=5)
    assert len(hits) >= 1
    assert any("timeout" in h["content"].lower() for h in hits)


def test_get_repo_context(store):
    store.append_memory("proj-a", "failure", "ImportError: no module foo")
    store.append_memory("proj-a", "decision", "Pin foo==1.2.3 in requirements")
    ctx = store.get_repo_context("proj-a")
    assert "ImportError" in ctx["failures"]
    assert "Pin foo" in ctx["decisions"]


def test_forget(store):
    mid = store.append_memory("proj-a", "fact", "The sky is blue")
    assert mid
    count = store.forget(memory_id=mid)
    assert count == 1
    # Should no longer appear in search
    hits = store.search("sky blue", repo_id="proj-a")
    assert all(h["id"] != mid for h in hits)


def test_mark_failure_resolved(store):
    store.append_memory("proj-a", "failure", "Disk full on /var/log")
    ok = store.mark_failure_resolved("proj-a", "Disk full on /var/log")
    assert ok


def test_list_projects(store):
    registry = ProjectRegistry(store)
    projects = registry.list_projects()
    ids = [p.repo_id for p in projects]
    assert "proj-a" in ids
    assert "proj-b" in ids


def test_cross_repo_global_search(store):
    store.append_memory("proj-a", "failure", "Redis connection refused")
    store.append_memory("proj-b", "failure", "Redis timeout after 30s")
    cross = CrossRepoSearch(store)
    hits = cross.global_search("Redis", limit=10)
    assert len(hits) >= 2
    repos_found = {h["repo_id"] for h in hits}
    assert "proj-a" in repos_found
    assert "proj-b" in repos_found


def test_find_similar_failures(store):
    store.append_memory("proj-a", "failure", "ModuleNotFoundError: No module named 'requests'")
    store.append_memory("proj-b", "failure", "ModuleNotFoundError: No module named 'flask'")
    cross = CrossRepoSearch(store)
    similar = cross.find_similar_failures("proj-a", limit=5)
    # Should find proj-b's similar failure
    assert any(h["repo_id"] == "proj-b" for h in similar)


def test_correlator_get_related(store):
    mid1 = store.append_memory("proj-a", "failure", "Error in src/api/auth.py line 42: token expired")
    mid2 = store.append_memory("proj-a", "attempt", "Fix token refresh in src/api/auth.py")
    assert mid1 and mid2
    corr = MemoryCorrelator(store.engine)
    related = corr.get_related(mid1, limit=5)
    assert len(related) >= 1


def test_pattern_detector_report(store):
    store.append_memory("proj-a", "failure", "Connection timeout on DB")
    store.append_memory("proj-a", "failure", "Connection timeout on DB")  # deduped
    store.append_memory("proj-a", "decision", "Add retry logic for DB connections")
    detector = PatternDetector(store.engine)
    report = detector.get_report("proj-a")
    assert "health_score" in report
    assert "activity_trends" in report
    assert report["health_score"]["score"] <= 100


def test_mcp_server_creates(tmp_path):
    """Verify create_mcp_server doesn't crash."""
    srv = create_mcp_server(tmp_path)
    assert srv is not None
