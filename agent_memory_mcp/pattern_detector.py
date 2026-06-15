"""Pattern detector: identifies recurring patterns, trends, and anomalies in memories."""

from __future__ import annotations

import re
from collections import Counter
from typing import TYPE_CHECKING, Any, Dict, List, Optional

if TYPE_CHECKING:
    from .engine.db import MemoryEngine


class PatternDetector:
    """Detects patterns across the memory database."""

    def __init__(self, engine: "MemoryEngine"):
        self.engine = engine

    def get_report(self, repo_id: Optional[str] = None) -> Dict[str, Any]:
        """Generate a full pattern report for a repo (or all repos)."""
        return {
            "recurring_failures": self._recurring_failures(repo_id),
            "activity_trends": self._activity_trends(repo_id),
            "common_error_categories": self._error_categories(repo_id),
            "decision_patterns": self._decision_patterns(repo_id),
            "health_score": self._health_score(repo_id),
        }

    def _recurring_failures(
        self, repo_id: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        if repo_id:
            sigs = self.engine.get_failure_signatures(repo_id)
        else:
            repos = self.engine.list_repos_db()
            sigs: List[Dict[str, Any]] = []
            for repo in repos:
                sigs.extend(self.engine.get_failure_signatures(repo["id"]))

        recurring = [s for s in sigs if s.get("count", 0) >= 2 and not s.get("resolved")]
        recurring.sort(key=lambda x: x.get("count", 0), reverse=True)
        return recurring[:20]

    def _activity_trends(self, repo_id: Optional[str] = None) -> Dict[str, Any]:
        memories = self.engine.list_memories(repo_id=repo_id, limit=500)
        if not memories:
            return {"total": 0, "by_day": {}, "by_kind": {}}

        by_day: Counter = Counter()
        by_kind: Counter = Counter()

        for mem in memories:
            created = mem.get("created_at", "")
            if created:
                day = created[:10]
                by_day[day] += 1
            by_kind[mem["kind"]] += 1

        return {
            "total": len(memories),
            "by_day": dict(sorted(by_day.items())[-30:]),
            "by_kind": dict(by_kind),
        }

    def _error_categories(
        self, repo_id: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        failures = self.engine.list_memories(repo_id=repo_id, kind="failure", limit=200)
        categories: Counter = Counter()
        for fail in failures:
            cat = self._categorize_error(fail["content"])
            categories[cat] += 1
        return [{"category": cat, "count": count} for cat, count in categories.most_common(15)]

    def _categorize_error(self, content: str) -> str:
        content_lower = content.lower()
        patterns = [
            (r"timeout|timed?\s*out", "Timeout"),
            (r"connection\s*(refused|reset|error)", "Connection Error"),
            (r"import\s*error|module\s*not\s*found|no\s*module", "Import Error"),
            (r"syntax\s*error", "Syntax Error"),
            (r"type\s*error|typeerror", "Type Error"),
            (r"permission|access\s*denied|forbidden", "Permission Error"),
            (r"not\s*found|404|missing", "Not Found"),
            (r"memory|oom|out\s*of\s*memory", "Memory Error"),
            (r"null|none|undefined|nil", "Null Reference"),
            (r"assertion|assert", "Assertion Error"),
            (r"test\s*fail", "Test Failure"),
            (r"build\s*fail|compile", "Build Error"),
        ]
        for pattern, category in patterns:
            if re.search(pattern, content_lower):
                return category
        return "Other"

    def _decision_patterns(self, repo_id: Optional[str] = None) -> Dict[str, Any]:
        decisions = self.engine.list_memories(repo_id=repo_id, kind="decision", limit=200)
        topic_counter: Counter = Counter()
        stopwords = {
            "this", "that", "with", "from", "have", "will", "been", "were",
            "than", "into", "also", "just", "more", "some", "when", "what",
            "each", "make", "like", "over", "such", "take", "only", "come",
            "could", "them", "made", "than", "after", "before", "should",
            "would", "about", "which", "their", "there", "other", "because",
            "these", "those", "being", "does", "done", "most", "very", "using",
        }
        for dec in decisions:
            words = re.findall(r"[a-z]{4,}", dec["content"].lower())
            words = [w for w in words if w not in stopwords]
            topic_counter.update(words)
        return {
            "total_decisions": len(decisions),
            "top_topics": [
                {"topic": t, "count": c} for t, c in topic_counter.most_common(15)
            ],
        }

    def _health_score(self, repo_id: Optional[str] = None) -> Dict[str, Any]:
        score = 100
        reasons: List[str] = []

        if repo_id:
            fail_count = self.engine.failure_error_count(repo_id)
        else:
            repos = self.engine.list_repos_db()
            fail_count = sum(self.engine.failure_error_count(r["id"]) for r in repos)

        if fail_count > 0:
            penalty = min(fail_count * 5, 40)
            score -= penalty
            reasons.append(f"{fail_count} unresolved failures (-{penalty})")

        recurring = self._recurring_failures(repo_id)
        high_count = [r for r in recurring if r.get("count", 0) >= 5]
        if high_count:
            penalty = min(len(high_count) * 10, 30)
            score -= penalty
            reasons.append(f"{len(high_count)} highly recurring failures (-{penalty})")

        decisions = self.engine.list_memories(repo_id=repo_id, kind="decision", limit=10)
        if len(decisions) >= 3:
            bonus = 10
            score = min(score + bonus, 100)
            reasons.append(f"Good decision documentation (+{bonus})")

        return {
            "score": max(score, 0),
            "grade": (
                "A" if score >= 90 else
                "B" if score >= 75 else
                "C" if score >= 60 else
                "D" if score >= 40 else "F"
            ),
            "reasons": reasons,
        }
