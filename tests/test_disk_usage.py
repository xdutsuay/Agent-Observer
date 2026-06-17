"""Tests for disk usage reporting."""

from pathlib import Path

from agent_memory_mcp.disk_usage import format_bytes, data_root_breakdown, build_disk_report
from agent_memory_mcp.storage import Storage


def test_format_bytes():
    assert format_bytes(500) == "500 B"
    assert format_bytes(2048) == "2.0 KB"
    assert format_bytes(5 * 1024 * 1024) == "5.0 MB"


def test_data_root_breakdown(tmp_path: Path):
    (tmp_path / "memory.db").write_bytes(b"x" * 100)
    (tmp_path / "repos.json").write_text("{}")
    bd = data_root_breakdown(tmp_path)
    assert bd["memory_db"] == 100
    assert bd["data_root_total"] >= 100


def test_build_disk_report(tmp_path: Path):
    store = Storage(tmp_path)
    (tmp_path / ".git").mkdir()
    rid = store.resolve_repo(tmp_path)
    store.append_memory(rid, "fact", "hello disk usage test", source="test")
    report = build_disk_report(store, include_workspace=False)
    assert report["overall"]["total_memory_attributed_bytes"] > 0
    assert any(p["repo_id"] == rid for p in report["projects"])
