"""Single-process demo: API + built UI on port 9000."""

from pathlib import Path

from .api import create_app

DATA_ROOT = Path("~/agent_companion_data").expanduser()
app = create_app(DATA_ROOT, serve_ui=True, auto_watcher=True)
