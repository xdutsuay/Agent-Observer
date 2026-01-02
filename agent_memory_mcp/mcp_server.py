import sys
import json
from pathlib import Path
from typing import Dict, Any, List
from .storage import Storage

class MCPServer:
    """
    Experimental MCP Stdio Server for Agent Memory.
    Provides memory as resources and tools for adding logs.
    """
    def __init__(self, root: str | Path):
        self.store = Storage(root)

    def serve_forever(self):
        print("[MCP] Agent Memory Server starting...", file=sys.stderr)
        for line in sys.stdin:
            try:
                msg = json.loads(line)
                method = msg.get("method")
                params = msg.get("params", {})
                req_id = msg.get("id")

                if method == "initialize":
                    result = {
                        "protocolVersion": "2024-11-05",
                        "capabilities": {
                            "resources": {},
                            "tools": {}
                        },
                        "serverInfo": {"name": "agent-memory", "version": "0.1.0"}
                    }
                elif method == "resources/list":
                    # List repos as resources
                    repos = self.store.list_repos()
                    result = {
                        "resources": [
                            {
                                "uri": f"memory://{rid}/failures",
                                "name": f"Failure Memory ({path})",
                                "mimeType": "text/markdown"
                            } for rid, path in repos.items()
                        ]
                    }
                elif method == "resources/read":
                    uri = params.get("uri", "")
                    # Simple parse memory://{rid}/{kind}
                    if uri.startswith("memory://"):
                        parts = uri.replace("memory://", "").split("/")
                        if len(parts) == 2:
                            rid, kind = parts
                            mem = self.store.read_memory(rid)
                            content = mem.get(kind, "")
                            result = {
                                "contents": [{
                                    "uri": uri,
                                    "mimeType": "text/markdown",
                                    "text": content
                                }]
                            }
                        else:
                            result = {"error": "Invalid URI"}
                    else:
                        result = {"error": "Unknown scheme"}
                elif method == "tools/list":
                    result = {
                        "tools": [
                            {
                                "name": "add_memory",
                                "description": "Add an attempt, failure, or decision to the agent memory",
                                "inputSchema": {
                                    "type": "object",
                                    "properties": {
                                        "repo_path": {"type": "string"},
                                        "kind": {"type": "string", "enum": ["attempts", "failures", "decisions"]},
                                        "text": {"type": "string"}
                                    },
                                    "required": ["repo_path", "kind", "text"]
                                }
                            }
                        ]
                    }
                elif method == "tools/call":
                    tool = params.get("name")
                    args = params.get("arguments", {})
                    if tool == "add_memory":
                        rid = self.store.resolve_repo(args["repo_path"])
                        self.store.append_memory(rid, args["kind"], args["text"])
                        result = {"content": [{"type": "text", "text": "Memory added"}]}
                    else:
                        result = {"error": "Tool not found"}
                else:
                    result = {"error": "Method not implemented"}

                response = {"jsonrpc": "2.0", "id": req_id, "result": result}
                sys.stdout.write(json.dumps(response) + "\n")
                sys.stdout.flush()

            except Exception as e:
                resp = {"jsonrpc": "2.0", "id": msg.get("id"), "error": {"code": -32603, "message": str(e)}}
                sys.stdout.write(json.dumps(resp) + "\n")
                sys.stdout.flush()

