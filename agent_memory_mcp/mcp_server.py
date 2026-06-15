"""Agent Memory MCP server using the official MCP Python SDK."""

from __future__ import annotations

import asyncio
from pathlib import Path
from typing import Any

from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp.types import (
    Resource,
    TextContent,
    Tool,
    Prompt,
    PromptMessage,
    GetPromptResult,
)

from .storage import Storage
from .project_registry import ProjectRegistry
from .cross_repo import CrossRepoSearch
from .correlator import MemoryCorrelator
from .pattern_detector import PatternDetector

server = Server("agent-memory")
_store: Storage | None = None

KINDS_RESOURCE = ("failures", "decisions", "attempts", "facts")


def _get_store(root: Path) -> Storage:
    global _store
    if _store is None:
        _store = Storage(root)
    return _store


def create_mcp_server(root: str | Path) -> Server:
    root_path = Path(root).expanduser()
    store = _get_store(root_path)
    registry = ProjectRegistry(store)
    cross_repo = CrossRepoSearch(store)
    correlator = MemoryCorrelator(store.engine)
    pattern_detector = PatternDetector(store.engine)

    @server.list_resources()
    async def list_resources() -> list[Resource]:
        resources: list[Resource] = []
        for repo in store.list_repos():
            rid = repo["id"]
            for kind in KINDS_RESOURCE:
                resources.append(
                    Resource(
                        uri=f"memory://{rid}/{kind}",
                        name=f"{kind.capitalize()} ({rid})",
                        mimeType="text/markdown",
                    )
                )
        return resources

    @server.read_resource()
    async def read_resource(uri: str) -> str:
        if not uri.startswith("memory://"):
            raise ValueError(f"Unsupported URI: {uri}")
        parts = uri.replace("memory://", "").split("/", 1)
        if len(parts) != 2:
            raise ValueError(f"Invalid URI: {uri}")
        rid, kind = parts
        mem = store.read_memory(rid)
        key = kind if kind in mem else kind.rstrip("s") + "s" if kind + "s" in mem else kind
        if key.endswith("s") and key not in mem:
            for k in (kind, kind + "s", kind.rstrip("s")):
                if k in mem:
                    key = k
                    break
        return mem.get(key, mem.get(kind, ""))

    @server.list_tools()
    async def list_tools() -> list[Tool]:
        return [
            Tool(
                name="remember",
                description="Store a memory entry for a project (failures, decisions, attempts, facts, preferences).",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "path": {"type": "string", "description": "Project path or file path"},
                        "kind": {
                            "type": "string",
                            "enum": [
                                "failure", "failures",
                                "decision", "decisions",
                                "attempt", "attempts",
                                "fact", "preference",
                            ],
                        },
                        "content": {"type": "string"},
                    },
                    "required": ["path", "kind", "content"],
                },
            ),
            Tool(
                name="search_memory",
                description="Search repo-scoped memories by keyword and semantic similarity.",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "query": {"type": "string"},
                        "path": {"type": "string", "description": "Optional project path to scope search"},
                        "kinds": {"type": "array", "items": {"type": "string"}},
                        "limit": {"type": "integer", "default": 10},
                    },
                    "required": ["query"],
                },
            ),
            Tool(
                name="get_repo_context",
                description="Get a pre-packaged bundle of failures, decisions, attempts, and facts for a repo.",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "path": {"type": "string"},
                    },
                    "required": ["path"],
                },
            ),
            Tool(
                name="mark_failure_resolved",
                description="Mark a recurring failure signature as resolved.",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "path": {"type": "string"},
                        "signature": {"type": "string"},
                    },
                    "required": ["path", "signature"],
                },
            ),
            Tool(
                name="forget",
                description="Soft-delete a memory by id or failure signature.",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "memory_id": {"type": "string"},
                        "path": {"type": "string"},
                        "signature": {"type": "string"},
                    },
                },
            ),
            Tool(
                name="add_memory",
                description="Legacy alias for remember.",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "path": {"type": "string"},
                        "kind": {"type": "string", "enum": ["attempts", "failures", "decisions"]},
                        "text": {"type": "string"},
                    },
                    "required": ["path", "kind", "text"],
                },
            ),
            Tool(
                name="list_projects",
                description="List all tracked projects with metadata (language, framework, health).",
                inputSchema={"type": "object", "properties": {}},
            ),
            Tool(
                name="switch_project_context",
                description="Get full context for a project by repo_id — failures, decisions, attempts, facts, and health score.",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "repo_id": {"type": "string", "description": "Project repo ID from list_projects"},
                    },
                    "required": ["repo_id"],
                },
            ),
            Tool(
                name="global_search",
                description="Search across ALL projects (cross-repo semantic + keyword search).",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "query": {"type": "string"},
                        "kinds": {"type": "array", "items": {"type": "string"}},
                        "limit": {"type": "integer", "default": 20},
                    },
                    "required": ["query"],
                },
            ),
            Tool(
                name="find_similar_failures",
                description="Find failures in other projects similar to a given project's failures.",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "path": {"type": "string", "description": "Project path"},
                        "limit": {"type": "integer", "default": 10},
                    },
                    "required": ["path"],
                },
            ),
            Tool(
                name="get_related_memories",
                description="Find memories related to a specific memory (by content, files, temporal proximity).",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "memory_id": {"type": "string"},
                        "limit": {"type": "integer", "default": 10},
                    },
                    "required": ["memory_id"],
                },
            ),
            Tool(
                name="get_pattern_report",
                description="Get a pattern analysis report: recurring failures, trends, health score, error categories.",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "path": {"type": "string", "description": "Optional project path (omit for global report)"},
                    },
                },
            ),
            Tool(
                name="failure_hotspots",
                description="Find projects with most unresolved failures.",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "limit": {"type": "integer", "default": 10},
                    },
                },
            ),
        ]

    @server.call_tool()
    async def call_tool(name: str, arguments: dict[str, Any]) -> list[TextContent]:
        if name == "remember" or name == "add_memory":
            path = arguments["path"]
            kind = arguments["kind"]
            content = arguments.get("content") or arguments.get("text", "")
            rid = store.resolve_repo(path)
            mid = store.append_memory(rid, kind, content, source="mcp")
            return [TextContent(type="text", text=f"Memory stored in {rid}" + (f" (id={mid})" if mid else " (deduped)"))]

        if name == "search_memory":
            rid = store.resolve_repo(arguments["path"]) if arguments.get("path") else None
            hits = store.search(
                arguments["query"],
                repo_id=rid,
                kinds=arguments.get("kinds"),
                limit=arguments.get("limit", 10),
            )
            if not hits:
                return [TextContent(type="text", text="No memories found.")]
            lines = []
            for h in hits:
                lines.append(
                    f"[{h.get('score', 0):.2f}] ({h['kind']}) {h['created_at']}\n{h['content'][:600]}\n"
                )
            return [TextContent(type="text", text="\n---\n".join(lines))]

        if name == "get_repo_context":
            rid = store.resolve_repo(arguments["path"])
            ctx = store.get_repo_context(rid)
            text = "\n\n".join(f"## {k.upper()}\n{v}" for k, v in ctx.items())
            return [TextContent(type="text", text=text)]

        if name == "mark_failure_resolved":
            rid = store.resolve_repo(arguments["path"])
            ok = store.mark_failure_resolved(rid, arguments["signature"])
            return [TextContent(type="text", text="Resolved." if ok else "Signature not found.")]

        if name == "forget":
            rid = store.resolve_repo(arguments["path"]) if arguments.get("path") else None
            n = store.forget(
                memory_id=arguments.get("memory_id"),
                signature=arguments.get("signature"),
                repo_id=rid,
            )
            return [TextContent(type="text", text=f"Forgot {n} record(s).")]

        if name == "list_projects":
            projects = registry.to_json()
            if not projects:
                return [TextContent(type="text", text="No projects tracked yet.")]
            lines = []
            for p in projects:
                health = "CRITICAL" if p["failure_count"] > 0 else "HEALTHY"
                lang = p.get("language") or "unknown"
                fw = f" ({p['framework']})" if p.get("framework") else ""
                lines.append(
                    f"- **{p['name']}** [{p['repo_id']}] — {lang}{fw} | "
                    f"{p['memory_count']} memories | {health}"
                )
            return [TextContent(type="text", text="\n".join(lines))]

        if name == "switch_project_context":
            rid = arguments["repo_id"]
            ctx = store.get_repo_context(rid)
            report = pattern_detector.get_report(rid)
            health = report["health_score"]
            text = (
                f"## Project: {rid}\n"
                f"Health: {health['grade']} ({health['score']}/100)\n\n"
                + "\n\n".join(f"## {k.upper()}\n{v}" for k, v in ctx.items())
            )
            if health["reasons"]:
                text += "\n\n## HEALTH NOTES\n" + "\n".join(f"- {r}" for r in health["reasons"])
            return [TextContent(type="text", text=text)]

        if name == "global_search":
            hits = cross_repo.global_search(
                arguments["query"],
                kinds=arguments.get("kinds"),
                limit=arguments.get("limit", 20),
            )
            if not hits:
                return [TextContent(type="text", text="No memories found across any project.")]
            lines = []
            for h in hits:
                lines.append(
                    f"[{h.get('score', 0):.2f}] ({h['repo_id']}/{h['kind']}) {h['created_at']}\n{h['content'][:600]}\n"
                )
            return [TextContent(type="text", text="\n---\n".join(lines))]

        if name == "find_similar_failures":
            rid = store.resolve_repo(arguments["path"])
            hits = cross_repo.find_similar_failures(rid, limit=arguments.get("limit", 10))
            if not hits:
                return [TextContent(type="text", text="No similar failures found in other projects.")]
            lines = []
            for h in hits:
                lines.append(
                    f"[{h['repo_id']}] {h['content'][:200]}\n  Similar to: {h.get('similar_to', '')[:100]}"
                )
            return [TextContent(type="text", text="\n\n".join(lines))]

        if name == "get_related_memories":
            related = correlator.get_related(
                arguments["memory_id"], limit=arguments.get("limit", 10)
            )
            if not related:
                return [TextContent(type="text", text="No related memories found.")]
            lines = []
            for r in related:
                lines.append(
                    f"[{r.get('relevance', 0):.2f} | {r.get('reason', '')}] "
                    f"({r['repo_id']}/{r['kind']}) {r['content'][:300]}"
                )
            return [TextContent(type="text", text="\n\n".join(lines))]

        if name == "get_pattern_report":
            rid = store.resolve_repo(arguments["path"]) if arguments.get("path") else None
            report = pattern_detector.get_report(rid)
            import json as _json
            return [TextContent(type="text", text=_json.dumps(report, indent=2, default=str))]

        if name == "failure_hotspots":
            hotspots = cross_repo.failure_hotspots(limit=arguments.get("limit", 10))
            if not hotspots:
                return [TextContent(type="text", text="No failure hotspots — all projects healthy!")]
            lines = [
                f"- {h['repo_id']}: {h['unresolved_failures']} unresolved failures ({h['path']})"
                for h in hotspots
            ]
            return [TextContent(type="text", text="\n".join(lines))]

        raise ValueError(f"Unknown tool: {name}")

    @server.list_prompts()
    async def list_prompts() -> list[Prompt]:
        return [
            Prompt(
                name="inject_memory_context",
                description="Inject repo failure/decision context before starting a task.",
                arguments=[
                    {"name": "path", "description": "Project path", "required": True},
                ],
            )
        ]

    @server.get_prompt()
    async def get_prompt(name: str, arguments: dict[str, str] | None) -> GetPromptResult:
        if name != "inject_memory_context":
            raise ValueError(f"Unknown prompt: {name}")
        args = arguments or {}
        path = args.get("path", ".")
        rid = store.resolve_repo(path)
        ctx = store.get_repo_context(rid)
        body = (
            f"You are working on repo `{rid}`. Relevant memory:\n\n"
            f"### Failures\n{ctx['failures']}\n\n"
            f"### Decisions\n{ctx['decisions']}\n\n"
            f"### Recent attempts\n{ctx['attempts']}\n\n"
            f"### Facts\n{ctx.get('facts', '')}\n"
        )
        return GetPromptResult(
            description=f"Memory context for {rid}",
            messages=[PromptMessage(role="user", content=TextContent(type="text", text=body))],
        )

    return server


async def run_mcp_stdio(root: str | Path) -> None:
    create_mcp_server(root)
    async with stdio_server() as (read_stream, write_stream):
        await server.run(read_stream, write_stream, server.create_initialization_options())


def serve_forever(root: str | Path) -> None:
    """Blocking entry for CLI."""
    asyncio.run(run_mcp_stdio(root))
