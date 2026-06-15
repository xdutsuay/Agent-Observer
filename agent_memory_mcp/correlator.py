"""Memory correlator: finds relationships between memories across repos."""

from __future__ import annotations

import json
import re
from typing import TYPE_CHECKING, Any, Dict, List, Optional

if TYPE_CHECKING:
    from .engine.db import MemoryEngine


class MemoryCorrelator:
    """Finds related memories using content overlap, file references, and temporal proximity."""

    def __init__(self, engine: "MemoryEngine"):
        self.engine = engine

    def get_related(self, memory_id: str, limit: int = 10) -> List[Dict[str, Any]]:
        """Find memories related to the given memory."""
        rows = self.engine._conn.execute(
            "SELECT * FROM memories WHERE id = ? AND deleted = 0", (memory_id,)
        ).fetchall()
        if not rows:
            return []
        source = dict(rows[0])

        candidates: Dict[str, Dict[str, Any]] = {}

        # Strategy 1: Same file references
        file_refs = self._extract_file_refs(source["content"])
        if file_refs:
            for ref in file_refs[:3]:
                hits = self.engine.search(ref, limit=limit * 2)
                for hit in hits:
                    if hit["id"] != memory_id:
                        score = candidates.get(hit["id"], {}).get("relevance", 0)
                        candidates[hit["id"]] = {
                            **hit,
                            "relevance": score + 0.4,
                            "reason": "shared_file",
                        }

        # Strategy 2: Content similarity via search
        query_text = source["content"][:300]
        similar = self.engine.search(query_text, limit=limit * 2)
        for hit in similar:
            if hit["id"] != memory_id:
                score = candidates.get(hit["id"], {}).get("relevance", 0)
                sim_score = hit.get("score", 0.3)
                candidates[hit["id"]] = {
                    **hit,
                    "relevance": score + sim_score,
                    "reason": "content_similar",
                }

        # Strategy 3: Temporal proximity (within 5 minutes)
        created = source["created_at"]
        temporal = self.engine._conn.execute(
            """
            SELECT id, repo_id, kind, content, source, metadata_json, session_id,
                   summary, created_at
            FROM memories
            WHERE deleted = 0 AND id != ?
            AND abs(julianday(created_at) - julianday(?)) < (5.0 / 1440.0)
            ORDER BY created_at DESC LIMIT ?
            """,
            (memory_id, created, limit * 2),
        ).fetchall()
        for row in temporal:
            mid = row[0]
            if mid not in candidates:
                candidates[mid] = {
                    "id": mid,
                    "repo_id": row[1],
                    "kind": row[2],
                    "content": row[3],
                    "source": row[4],
                    "metadata": json.loads(row[5]) if row[5] else {},
                    "session_id": row[6],
                    "summary": row[7],
                    "created_at": row[8],
                    "relevance": 0.2,
                    "reason": "temporal_proximity",
                }
            else:
                candidates[mid]["relevance"] += 0.2

        ranked = sorted(
            candidates.values(), key=lambda x: x.get("relevance", 0), reverse=True
        )
        return ranked[:limit]

    def _extract_file_refs(self, text: str) -> List[str]:
        """Extract file path references from text."""
        patterns = [
            r"[\w/.-]+\.\w{1,5}",
            r"(?:src|lib|app|tests?)/[\w/.-]+",
        ]
        refs: set[str] = set()
        for pattern in patterns:
            for match in re.finditer(pattern, text):
                ref = match.group()
                if len(ref) > 5 and not ref.startswith("http"):
                    refs.add(ref)
        return list(refs)[:10]

    def build_dependency_graph(
        self, repo_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """Build a graph of memory relationships for visualization."""
        memories = self.engine.list_memories(repo_id=repo_id, limit=100)

        nodes = []
        edges = []

        for mem in memories:
            nodes.append(
                {
                    "id": mem["id"],
                    "kind": mem["kind"],
                    "repo_id": mem["repo_id"],
                    "label": (mem.get("summary") or mem["content"][:60]),
                    "created_at": mem["created_at"],
                }
            )

            related = self.get_related(mem["id"], limit=3)
            for rel in related:
                if rel.get("relevance", 0) > 0.3:
                    edges.append(
                        {
                            "source": mem["id"],
                            "target": rel["id"],
                            "weight": rel.get("relevance", 0),
                            "reason": rel.get("reason", "unknown"),
                        }
                    )

        return {"nodes": nodes, "edges": edges}
