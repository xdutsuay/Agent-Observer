import sys
from pathlib import Path

import uvicorn

from .api import create_app
from .mcp_server import serve_forever


def run(
    mode: str = "http",
    port: int = 9000,
    root: str = "~/agent_companion_data",
    serve_ui: bool = False,
    auto_watcher: bool = True,
):
    root_path = Path(root).expanduser()

    if mode == "http":
        print(f"[MAIN] Starting HTTP Server on port {port}")
        print(f"[MAIN] Data root: {root_path}")
        app = create_app(root_path, serve_ui=serve_ui, auto_watcher=auto_watcher)
        uvicorn.run(app, host="127.0.0.1", port=port)
    else:
        print("[MAIN] Starting MCP Stdio Server", file=sys.stderr)
        serve_forever(root_path)
