import sys
import json
from pathlib import Path
from .storage import Storage


class StdioAdapter:
    """
    Minimal JSON-over-stdio adapter.
    Intended to be wrapped by an MCP-compatible client.
    """

    def __init__(self, root: str | Path):
        self.store = Storage(root)

    def serve_forever(self) -> None:
        for line in sys.stdin:
            line = line.strip()
            if not line:
                continue

            try:
                request = json.loads(line)
            except json.JSONDecodeError:
                continue

            req_id = request.get("id")
            method = request.get("method")
            params = request.get("params", {})

            try:
                if method == "list_repos":
                    result = {"repos": self.store.list_repos()}
                elif method == "get_memory":
                    result = self.store.read_memory(params["repo_id"])
                elif method == "append_failure":
                    self.store.append_memory(
                        params["repo_id"], "failures", params.get("text", "")
                    )
                    result = {"ok": True}
                else:
                    result = {"error": "unknown method"}
            except Exception as e:
                result = {"error": str(e)}

            response = {"id": req_id, "result": result}
            sys.stdout.write(json.dumps(response) + "\n")
            sys.stdout.flush()
