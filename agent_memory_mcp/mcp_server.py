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
