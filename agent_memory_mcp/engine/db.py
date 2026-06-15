"""SQLite memory engine with FTS5 search and optional embeddings."""

from __future__ import annotations

import json
import math
import re
import sqlite3
import threading
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

KIND_ALIASES = {
    "failures": "failure",
    "failure": "failure",
    "decisions": "decision",
    "decision": "decision",
    "attempts": "attempt",
    "attempt": "attempt",
    "facts": "fact",
    "fact": "fact",
    "preferences": "preference",
    "preference": "preference",
}

LEGACY_KINDS = ("failure", "decision", "attempt")


def _normalize_kind(kind: str) -> str:
    k = kind.lower().strip()
    return KIND_ALIASES.get(k, k)


def _utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


def _simple_embedding(text: str, dim: int = 64) -> List[float]:
    """Deterministic bag-of-words style vector (legacy fallback).

    Prefer using EmbeddingProvider from engine.embeddings instead.
    """
    vec = [0.0] * dim
    tokens = re.findall(r"[a-z0-9]{3,}", text.lower())
    if not tokens:
        return vec
    for tok in tokens:
        h = hash(tok) % dim
        vec[h] += 1.0
    norm = math.sqrt(sum(x * x for x in vec)) or 1.0
    return [x / norm for x in vec]


def _cosine(a: List[float], b: List[float]) -> float:
    if len(a) != len(b):
        return 0.0
    dot = sum(x * y for x, y in zip(a, b))
    na = math.sqrt(sum(x * x for x in a)) or 1.0
    nb = math.sqrt(sum(x * x for x in b)) or 1.0
    return dot / (na * nb)


class MemoryEngine:
    def __init__(self, root: Path, embedder: Optional[Any] = None):
        self.root = root.expanduser()
        self.root.mkdir(parents=True, exist_ok=True)
        self.db_path = self.root / "memory.db"
        self._conn = sqlite3.connect(str(self.db_path), check_same_thread=False)
        self._conn.row_factory = sqlite3.Row
        self._lock = threading.RLock()
        self._embedder = embedder  # EmbeddingProvider or None (uses _simple_embedding)
        self._init_schema()

    def _init_schema(self) -> None:
        c = self._conn
        c.executescript(
            """
            CREATE TABLE IF NOT EXISTS repos (
                id TEXT PRIMARY KEY,
                path TEXT NOT NULL,
                created_at TEXT NOT NULL
            );
            CREATE TABLE IF NOT EXISTS memories (
                id TEXT PRIMARY KEY,
                repo_id TEXT NOT NULL,
                kind TEXT NOT NULL,
                content TEXT NOT NULL,
                source TEXT NOT NULL DEFAULT 'mcp',
                metadata_json TEXT,
                session_id TEXT,
                summary TEXT,
                deleted INTEGER NOT NULL DEFAULT 0,
                created_at TEXT NOT NULL,
                FOREIGN KEY (repo_id) REFERENCES repos(id)
            );
            CREATE INDEX IF NOT EXISTS idx_memories_repo_kind
                ON memories(repo_id, kind, deleted, created_at DESC);
            CREATE TABLE IF NOT EXISTS memory_embeddings (
                memory_id TEXT PRIMARY KEY,
                embedding_json TEXT NOT NULL,
                FOREIGN KEY (memory_id) REFERENCES memories(id)
            );
            CREATE TABLE IF NOT EXISTS failure_signatures (
                repo_id TEXT NOT NULL,
                signature TEXT NOT NULL,
                count INTEGER NOT NULL DEFAULT 1,
                first_seen TEXT NOT NULL,
                last_seen TEXT NOT NULL,
                resolved INTEGER NOT NULL DEFAULT 0,
                memory_id TEXT,
                PRIMARY KEY (repo_id, signature)
            );
            """
        )
        self._ensure_standalone_fts(c)
        c.commit()

    def _ensure_standalone_fts(self, c: sqlite3.Connection) -> None:
        """Standalone FTS5 (external content= caused 'API misuse' on insert)."""
        row = c.execute(
            "SELECT sql FROM sqlite_master WHERE type='table' AND name='memories_fts'"
        ).fetchone()
        rebuilt = False
        if row and row[0] and "content=" in str(row[0]):
            c.execute("DROP TABLE IF EXISTS memories_fts")
            rebuilt = True
        c.execute(
            """
            CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
                content, kind, repo_id
            )
            """
        )
        if rebuilt:
            rows = c.execute(
                "SELECT rowid, content, kind, repo_id FROM memories WHERE deleted = 0"
            ).fetchall()
            for r in rows:
                c.execute(
                    "INSERT INTO memories_fts(rowid, content, kind, repo_id) VALUES (?, ?, ?, ?)",
                    (r["rowid"], r["content"], r["kind"], r["repo_id"]),
                )

    def upsert_repo(self, repo_id: str, path: str) -> None:
        with self._lock:
            self._upsert_repo_unlocked(repo_id, path)

    def _upsert_repo_unlocked(self, repo_id: str, path: str) -> None:
        self._conn.execute(
            """
            INSERT INTO repos (id, path, created_at) VALUES (?, ?, ?)
            ON CONFLICT(id) DO UPDATE SET path = excluded.path
            """,
            (repo_id, path, _utc_now()),
        )
        self._conn.commit()

    def insert_memory(
        self,
        repo_id: str,
        kind: str,
        content: str,
        source: str = "mcp",
        metadata: Optional[Dict] = None,
        session_id: Optional[str] = None,
        skip_failure_dedup: bool = False,
    ) -> Tuple[Optional[str], bool]:
        """
        Insert a memory row. Returns (memory_id, inserted).
        For duplicate failures, returns (None, False) when dedup applies.
        """
        with self._lock:
            return self._insert_memory_unlocked(
                repo_id, kind, content, source, metadata, session_id, skip_failure_dedup
            )

    def _insert_memory_unlocked(
        self,
        repo_id: str,
        kind: str,
        content: str,
        source: str = "mcp",
        metadata: Optional[Dict] = None,
        session_id: Optional[str] = None,
        skip_failure_dedup: bool = False,
    ) -> Tuple[Optional[str], bool]:
        kind_n = _normalize_kind(kind)
        meta = metadata or {}
        now = _utc_now()

        if kind_n == "failure" and not skip_failure_dedup:
            sig = content.splitlines()[0][:100].strip() if content else content[:100]
            existing = self._conn.execute(
                """
                SELECT signature, count, memory_id FROM failure_signatures
                WHERE repo_id = ? AND signature = ?
                """,
                (repo_id, sig),
            ).fetchone()
            if existing:
                self._conn.execute(
                    """
                    UPDATE failure_signatures
                    SET count = count + 1, last_seen = ?
                    WHERE repo_id = ? AND signature = ?
                    """,
                    (now, repo_id, sig),
                )
                self._conn.commit()
                return (existing["memory_id"], False)

        mem_id = str(uuid.uuid4())
        self._conn.execute(
            """
            INSERT INTO memories
            (id, repo_id, kind, content, source, metadata_json, session_id, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (
                mem_id,
                repo_id,
                kind_n,
                content,
                source,
                json.dumps(meta),
                session_id,
                now,
            ),
        )
        row = self._conn.execute(
            "SELECT rowid FROM memories WHERE id = ?", (mem_id,)
        ).fetchone()
        if row:
            self._conn.execute(
                "INSERT INTO memories_fts(rowid, content, kind, repo_id) VALUES (?, ?, ?, ?)",
                (row["rowid"], content, kind_n, repo_id),
            )

        emb = self._embed(content)
        self._conn.execute(
            "INSERT OR REPLACE INTO memory_embeddings (memory_id, embedding_json) VALUES (?, ?)",
            (mem_id, json.dumps(emb)),
        )

        if kind_n == "failure":
            sig = content.splitlines()[0][:100].strip() if content else content[:100]
            self._conn.execute(
                """
                INSERT INTO failure_signatures
                (repo_id, signature, count, first_seen, last_seen, resolved, memory_id)
                VALUES (?, ?, 1, ?, ?, 0, ?)
                ON CONFLICT(repo_id, signature) DO UPDATE SET
                    count = count + 1, last_seen = excluded.last_seen, memory_id = excluded.memory_id
                """,
                (repo_id, sig, now, now, mem_id),
            )

        self._conn.commit()
        return (mem_id, True)

    def list_memories(
        self,
        repo_id: Optional[str] = None,
        kind: Optional[str] = None,
        limit: int = 50,
        include_deleted: bool = False,
    ) -> List[Dict[str, Any]]:
        with self._lock:
            return self._list_memories_unlocked(repo_id, kind, limit, include_deleted)

    def _list_memories_unlocked(
        self,
        repo_id: Optional[str] = None,
        kind: Optional[str] = None,
        limit: int = 50,
        include_deleted: bool = False,
    ) -> List[Dict[str, Any]]:
        clauses = ["deleted = 0"] if not include_deleted else ["1=1"]
        params: List[Any] = []
        if repo_id:
            clauses.append("repo_id = ?")
            params.append(repo_id)
        if kind:
            clauses.append("kind = ?")
            params.append(_normalize_kind(kind))
        where = " AND ".join(clauses)
        params.append(limit)
        rows = self._conn.execute(
            f"""
            SELECT id, repo_id, kind, content, source, metadata_json, session_id,
                   summary, created_at
            FROM memories WHERE {where}
            ORDER BY created_at DESC LIMIT ?
            """,
            params,
        ).fetchall()
        return [self._row_to_dict(r) for r in rows]

    def _row_to_dict(self, row: sqlite3.Row) -> Dict[str, Any]:
        meta = {}
        if row["metadata_json"]:
            try:
                meta = json.loads(row["metadata_json"])
            except json.JSONDecodeError:
                pass
        return {
            "id": row["id"],
            "repo_id": row["repo_id"],
            "kind": row["kind"],
            "content": row["content"],
            "source": row["source"],
            "metadata": meta,
            "session_id": row["session_id"],
            "summary": row["summary"],
            "created_at": row["created_at"],
        }

    def search(
        self,
        query: str,
        repo_id: Optional[str] = None,
        kinds: Optional[List[str]] = None,
        limit: int = 10,
    ) -> List[Dict[str, Any]]:
        with self._lock:
            return self._search_unlocked(query, repo_id, kinds, limit)

    def _search_unlocked(
        self,
        query: str,
        repo_id: Optional[str] = None,
        kinds: Optional[List[str]] = None,
        limit: int = 10,
    ) -> List[Dict[str, Any]]:
        query = query.strip()
        if not query:
            return self.list_memories(repo_id=repo_id, limit=limit)

        results: Dict[str, Dict[str, Any]] = {}
        q_emb = self._embed(query)

        # FTS5 keyword search
        fts_query = " ".join(f'"{w}"' for w in query.split() if w)
        try:
            fts_sql = """
                SELECT m.id, m.repo_id, m.kind, m.content, m.source, m.metadata_json,
                       m.session_id, m.summary, m.created_at,
                       bm25(memories_fts) AS rank
                FROM memories_fts f
                JOIN memories m ON m.rowid = f.rowid
                WHERE memories_fts MATCH ? AND m.deleted = 0
            """
            fts_params: List[Any] = [fts_query]
            if repo_id:
                fts_sql += " AND m.repo_id = ?"
                fts_params.append(repo_id)
            if kinds:
                kn = [_normalize_kind(k) for k in kinds]
                fts_sql += f" AND m.kind IN ({','.join('?' * len(kn))})"
                fts_params.extend(kn)
            fts_sql += " ORDER BY rank LIMIT ?"
            fts_params.append(limit * 2)
            for row in self._conn.execute(fts_sql, fts_params).fetchall():
                d = self._row_to_dict(row)
                d["score"] = float(-row["rank"]) if row["rank"] else 0.5
                d["match"] = "keyword"
                results[d["id"]] = d
        except sqlite3.OperationalError:
            pass

        # Vector similarity over recent memories
        clauses = ["m.deleted = 0"]
        params: List[Any] = []
        if repo_id:
            clauses.append("m.repo_id = ?")
            params.append(repo_id)
        if kinds:
            kn = [_normalize_kind(k) for k in kinds]
            clauses.append(f"m.kind IN ({','.join('?' * len(kn))})")
            params.extend(kn)
        where = " AND ".join(clauses)
        params.append(limit * 5)
        vec_rows = self._conn.execute(
            f"""
            SELECT m.id, m.repo_id, m.kind, m.content, m.source, m.metadata_json,
                   m.session_id, m.summary, m.created_at, e.embedding_json
            FROM memories m
            JOIN memory_embeddings e ON e.memory_id = m.id
            WHERE {where}
            ORDER BY m.created_at DESC LIMIT ?
            """,
            params,
        ).fetchall()
        for row in vec_rows:
            emb = json.loads(row["embedding_json"])
            sim = _cosine(q_emb, emb)
            if sim < 0.05:
                continue
            d = self._row_to_dict(row)
            d["score"] = sim
            d["match"] = "vector"
            prev = results.get(d["id"])
            if not prev or d["score"] > prev.get("score", 0):
                results[d["id"]] = d

        ranked = sorted(results.values(), key=lambda x: x.get("score", 0), reverse=True)
        return ranked[:limit]

    def aggregate_markdown(self, repo_id: str, kind: str) -> str:
        """Build legacy markdown buffer for MCP resources."""
        kind_n = _normalize_kind(kind)
        rows = self.list_memories(repo_id=repo_id, kind=kind_n, limit=200)
        parts = []
        for r in reversed(rows):
            ts = r.get("created_at", "")
            parts.append(f"### {ts}\n{r['content'].rstrip()}\n\n")
        return "".join(parts)

    def get_repo_context(self, repo_id: str, attempt_limit: int = 15) -> Dict[str, str]:
        failures = self.list_memories(repo_id=repo_id, kind="failure", limit=10)
        decisions = self.list_memories(repo_id=repo_id, kind="decision", limit=10)
        attempts = self.list_memories(repo_id=repo_id, kind="attempt", limit=attempt_limit)
        facts = self.list_memories(repo_id=repo_id, kind="fact", limit=10)

        def fmt(rows: List[Dict]) -> str:
            if not rows:
                return "(none)"
            return "\n".join(
                f"- [{r['created_at']}] {r['content'][:500]}"
                for r in rows
            )

        sigs = self.get_failure_signatures(repo_id)
        unresolved = [s for s in sigs if not s.get("resolved")]

        return {
            "failures": fmt(failures),
            "decisions": fmt(decisions),
            "attempts": fmt(attempts),
            "facts": fmt(facts),
            "failure_signatures": json.dumps(unresolved[:20], indent=2),
        }

    def get_failure_signatures(self, repo_id: str) -> List[Dict[str, Any]]:
        rows = self._conn.execute(
            """
            SELECT signature, count, first_seen, last_seen, resolved, memory_id
            FROM failure_signatures WHERE repo_id = ?
            ORDER BY count DESC
            """,
            (repo_id,),
        ).fetchall()
        return [dict(r) for r in rows]

    def mark_failure_resolved(self, repo_id: str, signature: str) -> bool:
        cur = self._conn.execute(
            """
            UPDATE failure_signatures SET resolved = 1
            WHERE repo_id = ? AND signature = ?
            """,
            (repo_id, signature),
        )
        self._conn.commit()
        return cur.rowcount > 0

    def forget(self, memory_id: Optional[str] = None, signature: Optional[str] = None, repo_id: Optional[str] = None) -> int:
        count = 0
        if memory_id:
            cur = self._conn.execute(
                "UPDATE memories SET deleted = 1 WHERE id = ?", (memory_id,)
            )
            count += cur.rowcount
        if signature and repo_id:
            cur = self._conn.execute(
                """
                UPDATE memories SET deleted = 1
                WHERE id IN (
                    SELECT memory_id FROM failure_signatures
                    WHERE repo_id = ? AND signature = ?
                )
                """,
                (repo_id, signature),
            )
            count += cur.rowcount
        self._conn.commit()
        return count

    def update_summary(self, memory_id: str, summary: str) -> None:
        self._conn.execute(
            "UPDATE memories SET summary = ? WHERE id = ?", (summary, memory_id)
        )
        self._conn.commit()

    def list_repos_db(self) -> List[Dict[str, str]]:
        rows = self._conn.execute(
            "SELECT id, path, created_at FROM repos ORDER BY created_at DESC"
        ).fetchall()
        return [{"id": r["id"], "path": r["path"], "created_at": r["created_at"]} for r in rows]

    def failure_error_count(self, repo_id: str) -> int:
        row = self._conn.execute(
            """
            SELECT COALESCE(SUM(count), 0) AS total FROM failure_signatures
            WHERE repo_id = ? AND resolved = 0
            """,
            (repo_id,),
        ).fetchone()
        return int(row["total"]) if row else 0

    def _embed(self, text: str) -> List[float]:
        """Embed text using the configured provider, or fall back to _simple_embedding."""
        if self._embedder is not None:
            try:
                return self._embedder.embed(text)
            except Exception:
                pass
        return _simple_embedding(text)

    def set_embedder(self, embedder: Any) -> None:
        """Hot-swap the embedding provider."""
        self._embedder = embedder

    def reindex_embeddings(self, batch_size: int = 50) -> int:
        """Re-embed all memories with the current provider. Returns count reindexed."""
        if self._embedder is None:
            return 0
        with self._lock:
            rows = self._conn.execute(
                "SELECT id, content FROM memories WHERE deleted = 0"
            ).fetchall()
            count = 0
            batch_ids: List[str] = []
            batch_texts: List[str] = []
            for row in rows:
                batch_ids.append(row["id"])
                batch_texts.append(row["content"])
                if len(batch_texts) >= batch_size:
                    self._reindex_batch(batch_ids, batch_texts)
                    count += len(batch_ids)
                    batch_ids, batch_texts = [], []
            if batch_texts:
                self._reindex_batch(batch_ids, batch_texts)
                count += len(batch_ids)
            return count

    def _reindex_batch(self, ids: List[str], texts: List[str]) -> None:
        try:
            embeddings = self._embedder.embed_batch(texts)  # type: ignore[union-attr]
        except Exception:
            return
        for mid, emb in zip(ids, embeddings):
            self._conn.execute(
                "INSERT OR REPLACE INTO memory_embeddings (memory_id, embedding_json) VALUES (?, ?)",
                (mid, json.dumps(emb)),
            )
        self._conn.commit()

    def close(self) -> None:
        self._conn.close()
