"""Tests for the FastAPI HTTP endpoints."""

import tempfile
from pathlib import Path

import pytest
from fastapi.testclient import TestClient

from agent_memory_mcp.api import create_app


@pytest.fixture
def client(tmp_path):
    app = create_app(tmp_path, serve_ui=False, auto_watcher=False)
    return TestClient(app)


def test_root(client):
    resp = client.get("/api/")
    assert resp.status_code == 200
    data = resp.json()
    assert data["name"] == "Agent Memory MCP"
    assert "version" in data


def test_list_repos(client):
    resp = client.get("/api/repos")
    assert resp.status_code == 200
    assert "repos" in resp.json()


def test_add_and_search_memory(client):
    # Add memory
    resp = client.post(
        "/api/memory/test-repo/failure",
        json={"text": "Connection refused on port 3306", "metadata": {"file": "db.py"}},
    )
    assert resp.status_code == 200
    assert resp.json()["ok"]

    # Search
    resp = client.post(
        "/api/search",
        json={"query": "connection refused", "repo_id": "test-repo"},
    )
    assert resp.status_code == 200
    results = resp.json()["results"]
    assert len(results) >= 1


def test_status(client):
    resp = client.get("/api/status")
    assert resp.status_code == 200
    data = resp.json()
    assert "running" in data
    assert data["running"] is False  # auto_watcher=False


def test_config(client):
    resp = client.get("/api/config")
    assert resp.status_code == 200
    data = resp.json()
    assert "watch_paths" in data


def test_list_projects(client):
    resp = client.get("/api/projects")
    assert resp.status_code == 200
    assert "projects" in resp.json()


def test_global_search(client):
    # Add some data first
    client.post("/api/memory/repo-x/failure", json={"text": "Null pointer exception"})
    resp = client.post("/api/search/global", json={"query": "null pointer"})
    assert resp.status_code == 200
    assert "results" in resp.json()


def test_pattern_report(client):
    client.post("/api/memory/repo-y/failure", json={"text": "Timeout error"})
    resp = client.get("/api/patterns/repo-y")
    assert resp.status_code == 200
    data = resp.json()
    assert "health_score" in data


def test_global_patterns(client):
    resp = client.get("/api/patterns")
    assert resp.status_code == 200
    assert "health_score" in resp.json()


def test_failure_hotspots(client):
    resp = client.get("/api/hotspots")
    assert resp.status_code == 200
    assert "hotspots" in resp.json()


def test_event_history(client):
    resp = client.get("/api/events/history")
    assert resp.status_code == 200
    assert "events" in resp.json()
