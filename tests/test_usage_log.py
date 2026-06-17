"""Tests for usage interaction logging."""

from pathlib import Path

from agent_memory_mcp.usage_log import UsageLog, classify_host_ide, _summarize_query


def test_classify_host_ide():
    assert classify_host_ide("Cursor/1.0") == "Cursor"
    assert classify_host_ide("claude-code CLI") == "Claude Code"
    assert classify_host_ide("random-app") == "Unknown"


def test_summarize_query():
    s = _summarize_query("search_memory", {"query": "auth bug", "path": "/tmp/proj"})
    assert "search_memory" in s
    assert "auth bug" in s


def test_usage_log_record_and_list(tmp_path: Path):
    log = UsageLog.for_root(tmp_path)
    log.record(
        transport="mcp",
        method="search_memory",
        query={"query": "test"},
        response_preview="2 hits",
        client_name="cursor-vscode",
        host_ide="Cursor",
        duration_ms=12.5,
    )
    rows = log.list_interactions(limit=10)
    assert len(rows) == 1
    assert rows[0]["method"] == "search_memory"
    assert rows[0]["host_ide"] == "Cursor"

    summary = log.summary()
    assert summary["total_interactions"] == 1
    assert summary["reads"] >= 1

    sessions = log.list_sessions()
    assert len(sessions) == 1
    assert sessions[0]["call_count"] == 1
