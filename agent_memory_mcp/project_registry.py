"""Project registry for multi-project workspace intelligence."""

from __future__ import annotations

import json
import time
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import TYPE_CHECKING, Any, Dict, List, Optional

if TYPE_CHECKING:
    from .storage import Storage


@dataclass
class ProjectInfo:
    repo_id: str
    path: str
    name: str
    language: Optional[str] = None
    framework: Optional[str] = None
    last_activity: Optional[float] = None
    memory_count: int = 0
    failure_count: int = 0
    tags: List[str] = field(default_factory=list)


class ProjectRegistry:
    """Rich project metadata and discovery beyond simple repo registration."""

    def __init__(self, storage: "Storage"):
        self.storage = storage
        self._cache: Dict[str, ProjectInfo] = {}
        self._cache_time: float = 0
        self._cache_ttl: float = 60.0

    def list_projects(self, refresh: bool = False) -> List[ProjectInfo]:
        if not refresh and self._cache and (time.time() - self._cache_time < self._cache_ttl):
            return list(self._cache.values())

        repos = self.storage.list_repos()
        projects = []
        for repo in repos:
            info = self._enrich_repo(repo)
            self._cache[info.repo_id] = info
            projects.append(info)
        self._cache_time = time.time()
        return projects

    def get_project(self, repo_id: str) -> Optional[ProjectInfo]:
        self.list_projects()
        return self._cache.get(repo_id)

    def _enrich_repo(self, repo: Dict) -> ProjectInfo:
        path = Path(repo.get("path", ""))
        name = path.name if path.name else repo["id"]
        language = self._detect_language(path)
        framework = self._detect_framework(path)

        memory_count = len(self.storage.engine.list_memories(repo_id=repo["id"], limit=500))
        failure_count = self.storage.engine.failure_error_count(repo["id"])
        memories = self.storage.engine.list_memories(repo_id=repo["id"], limit=1)

        return ProjectInfo(
            repo_id=repo["id"],
            path=str(path),
            name=name,
            language=language,
            framework=framework,
            last_activity=time.time() if memories else None,
            memory_count=memory_count,
            failure_count=failure_count,
            tags=self._auto_tags(path, language, framework),
        )

    def _detect_language(self, path: Path) -> Optional[str]:
        if not path.exists():
            return None
        markers = {
            "pyproject.toml": "python",
            "setup.py": "python",
            "requirements.txt": "python",
            "package.json": "typescript/javascript",
            "tsconfig.json": "typescript",
            "Cargo.toml": "rust",
            "go.mod": "go",
            "pom.xml": "java",
            "build.gradle": "java",
            "Gemfile": "ruby",
        }
        for marker, lang in markers.items():
            if (path / marker).exists():
                return lang
        return None

    def _detect_framework(self, path: Path) -> Optional[str]:
        if not path.exists():
            return None
        checks = [
            (path / "next.config.js", "Next.js"),
            (path / "next.config.ts", "Next.js"),
            (path / "vite.config.ts", "Vite"),
            (path / "angular.json", "Angular"),
            (path / "manage.py", "Django"),
        ]
        for check_path, fw in checks:
            if check_path.exists():
                return fw
        pkg = path / "package.json"
        if pkg.exists():
            try:
                data = json.loads(pkg.read_text())
                deps = {**data.get("dependencies", {}), **data.get("devDependencies", {})}
                if "react" in deps:
                    return "React"
                if "vue" in deps:
                    return "Vue"
                if "svelte" in deps:
                    return "Svelte"
                if "express" in deps:
                    return "Express"
            except (json.JSONDecodeError, OSError):
                pass
        return None

    def _auto_tags(
        self, path: Path, language: Optional[str], framework: Optional[str]
    ) -> List[str]:
        tags: List[str] = []
        if language:
            tags.append(language)
        if framework:
            tags.append(framework)
        name = path.name.lower()
        if any(kw in name for kw in ("api", "server", "backend")):
            tags.append("backend")
        if any(kw in name for kw in ("ui", "frontend", "web", "dashboard")):
            tags.append("frontend")
        if any(kw in name for kw in ("lib", "sdk", "package")):
            tags.append("library")
        return tags

    def to_json(self) -> List[Dict[str, Any]]:
        return [asdict(p) for p in self.list_projects()]
