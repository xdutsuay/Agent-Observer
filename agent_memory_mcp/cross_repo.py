"""Cross-repository search and pattern finding."""

from __future__ import annotations

import re
from typing import TYPE_CHECKING, Any, Dict, List, Optional

if TYPE_CHECKING:
    from .storage import Storage


class CrossRepoSearch:
    """Search and find patterns across all repositories."""

    def __init__(self, storage: "Storage"):
        self.storage = storage

    def global_search(
        self,
        query: str,
        kinds: Optional[List[str]] = None,
        limit: int = 20,
    ) -> List[Dict[str, Any]]:
        """Search across ALL repos."""
        return self.storage.search(query, repo_id=None, kinds=kinds, limit=limit)

    def find_similar_failures(
        self, repo_id: str, limit: int = 10
    ) -> List[Dict[str, Any]]:
        """Find failures in OTHER repos similar to this repo's failures."""
        my_failures = self.storage.engine.list_memories(
            repo_id=repo_id, kind="failure", limit=20
        )
        if not my_failures:
            return []

        results: List[Dict[str, Any]] = []
        seen_ids: set[str] = set()
        for fail in my_failures[:5]:
            query = fail["content"][:200]
            hits = self.storage.search(query, repo_id=None, kinds=["failure"], limit=limit)
            for hit in hits:
                if hit["repo_id"] != repo_id and hit["id"] not in seen_ids:
                    hit["similar_to"] = fail["content"][:100]
                    results.append(hit)
                    seen_ids.add(hit["id"])

        return results[:limit]

    def failure_hotspots(self, limit: int = 10) -> List[Dict[str, Any]]:
        """Find repos with most unresolved failures."""
        repos = self.storage.list_repos()
        hotspots = []
        for repo in repos:
            count = self.storage.engine.failure_error_count(repo["id"])
            if count > 0:
                hotspots.append(
                    {
                        "repo_id": repo["id"],
                        "path": repo.get("path", ""),
                        "unresolved_failures": count,
                    }
                )
        hotspots.sort(key=lambda x: x["unresolved_failures"], reverse=True)
        return hotspots[:limit]

    def common_patterns(
        self, kind: str = "failure", limit: int = 10
    ) -> List[Dict[str, Any]]:
        """Find patterns that appear across multiple repos."""
        memories = self.storage.engine.list_memories(kind=kind, limit=500)

        sig_repos: Dict[str, set] = {}
        sig_content: Dict[str, str] = {}
        for mem in memories:
            sig = (
                mem["content"].splitlines()[0][:80].strip() if mem["content"] else ""
            )
            if not sig:
                continue
            normalized = re.sub(r"\d+", "N", sig)
            normalized = re.sub(r"/[\w/.-]+", "PATH", normalized)
            if normalized not in sig_repos:
                sig_repos[normalized] = set()
                sig_content[normalized] = sig
            sig_repos[normalized].add(mem["repo_id"])

        patterns = []
        for norm_sig, repos in sig_repos.items():
            if len(repos) >= 2:
                patterns.append(
                    {
                        "pattern": sig_content[norm_sig],
                        "normalized": norm_sig,
                        "repo_count": len(repos),
                        "repos": list(repos),
                    }
                )
        patterns.sort(key=lambda x: x["repo_count"], reverse=True)
        return patterns[:limit]
