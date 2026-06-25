"""Tests for the relevance scoring, access tracking, and noise classification."""

import tempfile
import time
from pathlib import Path

from agent_memory_mcp.engine.db import MemoryEngine
from agent_memory_mcp.storage import Storage


def _make_engine():
    tmp = tempfile.mkdtemp()
    engine = MemoryEngine(Path(tmp))
    engine.upsert_repo("test-repo", "/tmp/test-project")
    return engine, tmp


def test_access_tracking():
    engine, _ = _make_engine()
    mid, ok = engine.insert_memory("test-repo", "decision", "Use Postgres over DynamoDB")
    assert ok and mid

    engine.record_access(mid, "search_hit", "database choice")
    engine.record_access(mid, "search_hit", "postgres vs dynamo")

    # Check access count updated
    row = engine._conn.execute(
        "SELECT access_count, last_accessed FROM memories WHERE id = ?", (mid,)
    ).fetchone()
    assert row["access_count"] == 2
    assert row["last_accessed"] is not None

    # Check access log entries
    logs = engine._conn.execute(
        "SELECT * FROM memory_access_log WHERE memory_id = ?", (mid,)
    ).fetchall()
    assert len(logs) == 2
    assert logs[0]["access_type"] == "search_hit"


def test_feedback_recording():
    engine, _ = _make_engine()
    mid, _ = engine.insert_memory("test-repo", "failure", "Timeout on auth service")
    engine.record_access(mid, "search_hit", "auth timeout")
    engine.record_feedback(mid, useful=True, context="helped fix the bug")

    log = engine._conn.execute(
        "SELECT was_useful FROM memory_access_log WHERE memory_id = ? ORDER BY created_at DESC LIMIT 1",
        (mid,),
    ).fetchone()
    assert log["was_useful"] == 1


def test_relevance_score_computation():
    engine, _ = _make_engine()

    # High-value: decision with accesses
    mid_dec, _ = engine.insert_memory("test-repo", "decision", "Use event sourcing for audit trail")
    for _ in range(5):
        engine.record_access(mid_dec, "search_hit", "event sourcing")
    engine.record_feedback(mid_dec, useful=True)

    # Low-value: attempt with no accesses
    mid_att, _ = engine.insert_memory("test-repo", "attempt", "Initial scan of foo.py")

    score_dec = engine.compute_relevance_score(mid_dec)
    score_att = engine.compute_relevance_score(mid_att)

    assert score_dec > score_att, f"Decision ({score_dec}) should score higher than attempt ({score_att})"
    assert score_dec > 0.3
    assert score_att < 0.5


def test_noise_classification():
    engine, _ = _make_engine()

    # Insert old attempts (fake old created_at)
    from datetime import datetime, timedelta, timezone
    old_date = (datetime.now(timezone.utc) - timedelta(days=10)).isoformat()

    for i in range(5):
        mid = f"old-attempt-{i}"
        engine._conn.execute(
            """INSERT INTO memories (id, repo_id, kind, content, source, created_at, access_count, quality_tier)
               VALUES (?, 'test-repo', 'attempt', ?, 'watcher', ?, 0, 'unrated')""",
            (mid, f"Initial scan of file{i}.py", old_date),
        )
    engine._conn.commit()

    # Insert a recent attempt (should NOT be classified as noise)
    engine.insert_memory("test-repo", "attempt", "Recent code change")

    noise_count = engine.classify_noise(max_age_days=7)
    assert noise_count == 5

    # Verify they got tagged
    rows = engine._conn.execute(
        "SELECT quality_tier FROM memories WHERE id LIKE 'old-attempt-%'"
    ).fetchall()
    assert all(r["quality_tier"] == "noise" for r in rows)


def test_refresh_relevance_scores():
    engine, _ = _make_engine()
    engine.insert_memory("test-repo", "decision", "Use Redis for caching")
    engine.insert_memory("test-repo", "fact", "API rate limit is 100 req/s")
    engine.insert_memory("test-repo", "attempt", "Modified cache.py")

    count = engine.refresh_relevance_scores("test-repo")
    assert count == 3

    rows = engine._conn.execute(
        "SELECT relevance_score FROM memories WHERE repo_id = 'test-repo' AND deleted = 0"
    ).fetchall()
    assert all(r["relevance_score"] > 0 for r in rows)


def test_search_excludes_noise():
    engine, _ = _make_engine()
    from datetime import datetime, timedelta, timezone

    old_date = (datetime.now(timezone.utc) - timedelta(days=10)).isoformat()

    # Insert noise
    engine._conn.execute(
        """INSERT INTO memories (id, repo_id, kind, content, source, created_at, access_count, quality_tier)
           VALUES ('noise-1', 'test-repo', 'attempt', 'timeout connection error', 'watcher', ?, 0, 'noise')""",
        (old_date,),
    )
    # Insert embedding for it
    import json
    engine._conn.execute(
        "INSERT INTO memory_embeddings (memory_id, embedding_json) VALUES ('noise-1', ?)",
        (json.dumps(engine._embed("timeout connection error")),),
    )
    engine._conn.commit()

    # Insert a real decision
    engine.insert_memory("test-repo", "decision", "Handle timeout with retry and exponential backoff")

    hits = engine.search("timeout", repo_id="test-repo", limit=10)
    ids = [h["id"] for h in hits]
    assert "noise-1" not in ids


def test_search_records_access():
    engine, _ = _make_engine()
    mid, _ = engine.insert_memory("test-repo", "failure", "Connection timeout on database port 5432")

    hits = engine.search("timeout database connection", repo_id="test-repo", limit=5)
    assert len(hits) >= 1

    # Verify access was recorded
    row = engine._conn.execute(
        "SELECT access_count FROM memories WHERE id = ?", (mid,)
    ).fetchone()
    assert row["access_count"] >= 1


def test_context_file_generation():
    with tempfile.TemporaryDirectory() as tmp:
        store = Storage(tmp)
        store.engine.upsert_repo("ctx-test", tmp)
        store.append_memory(
            store.resolve_repo(tmp), "decision", "Use gRPC for internal services", source="test"
        )
        store.append_memory(
            store.resolve_repo(tmp), "fact", "Deployment target is Kubernetes 1.28", source="test"
        )

        rid = store.resolve_repo(tmp)
        out = store.generate_context_file(rid, tmp)
        assert out.exists()
        content = out.read_text()
        assert "auto-generated" in content
        assert "gRPC" in content or "Kubernetes" in content
