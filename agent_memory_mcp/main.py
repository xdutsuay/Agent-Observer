import uvicorn
from pathlib import Path
from .api import app
from .mcp_server import MCPServer

def run(mode: str = "http", port: int = 9000, root: str = "~/agent_companion_data"):
    root_path = Path(root).expanduser()
    
    if mode == "http":
        print(f"[MAIN] Starting HTTP Server on port {port}")
        print(f"[MAIN] Data root: {root_path}")
        uvicorn.run(app, host="127.0.0.1", port=port)
    else:
        print(f"[MAIN] Starting MCP Stdio Server", file=sys.stderr)
        server = MCPServer(root_path)
        server.serve_forever()
