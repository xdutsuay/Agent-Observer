import argparse
import sys
from .main import run

def entry():
    parser = argparse.ArgumentParser(prog="agent-memory")
    subparsers = parser.add_subparsers(dest="command", help="Commands")

    # Serve command
    serve_parser = subparsers.add_parser("serve", help="Start the FastAPI server & Watcher")
    serve_parser.add_argument("--port", type=int, default=9000, help="Port to run on")
    serve_parser.add_argument("--root", default="~/agent_companion_data", help="Data root directory")
    serve_parser.add_argument("--ui", action="store_true", help="Also serve UI (if built)")

    # MCP command
    mcp_parser = subparsers.add_parser("mcp", help="Run as MCP stdio server")
    mcp_parser.add_argument("--root", default="~/agent_companion_data", help="Data root directory")

    args = parser.parse_args()

    if args.command == "serve":
        run(mode="http", port=args.port, root=args.root)
    elif args.command == "mcp":
        run(mode="stdio", root=args.root)
    else:
        parser.print_help()

if __name__ == "__main__":
    entry()
