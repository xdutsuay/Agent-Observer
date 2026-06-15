import tempfile
from pathlib import Path

from agent_memory_mcp.engine.db import MemoryEngine
from agent_memory_mcp.engine.migration import import_legacy_markdown, needs_migration
from agent_memory_mcp.storage import Storage


def test_insert_and_search():
    with tempfile.TemporaryDirectory() as tmp:
        engine = MemoryEngine(Path(tmp))
        engine.upsert_repo("repo1", "/tmp/project")
        mid, ok = engine.insert_memory("repo1", "failure", "Connection timeout on port 5432")
        assert ok and mid
        engine.insert_memory("repo1", "decision", "Use SQLite for local storage")
        hits = engine.search("timeout connection", repo_id="repo1", limit=5)
        assert len(hits) >= 1
        assert any("timeout" in h["content"].lower() for h in hits)


def test_failure_dedup():
    with tempfile.TemporaryDirectory() as tmp:
        engine = MemoryEngine(Path(tmp))
        engine.upsert_repo("r1", "/proj")
        _, ok1 = engine.insert_memory("r1", "failure", "Same error\nline 2")
        _, ok2 = engine.insert_memory("r1", "failure", "Same error\nother")
        assert ok1
        assert not ok2
        sigs = engine.get_failure_signatures("r1")
        assert sigs[0]["count"] == 2


def test_legacy_migration():
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        mem_dir = root / "agent-memory" / "abc123" / "memory"
        mem_dir.mkdir(parents=True)
        (mem_dir / "failures.md").write_text("### Mon Jan 1\nImport test failure\n\n")
        (root / "repos.json").write_text('{"abc123": "/fake/path"}')
        engine = MemoryEngine(root)
        n = import_legacy_markdown(engine, root / "agent-memory", {"abc123": "/fake/path"})
        assert n >= 1
        assert not needs_migration(root)


def test_storage_resolve_repo():
    with tempfile.TemporaryDirectory() as tmp:
        store = Storage(tmp)
        rid = store.resolve_repo(__file__)
        assert rid
        store.append_memory(rid, "decision", "test decision", source="test")
        ctx = store.get_repo_context(rid)
        assert "test decision" in ctx["decisions"]
