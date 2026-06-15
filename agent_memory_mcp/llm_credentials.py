"""Resolve LLM API credentials from env or Hermes (NVIDIA NIM). Never log secrets."""

from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Optional, Tuple

NVIDIA_NIM_DEFAULT_BASE = "https://integrate.api.nvidia.com/v1"
NVIDIA_NIM_DEFAULT_MODEL = "nvidia/nemotron-3-super-120b-a12b"


def _hermes_auth_path() -> Path:
    return Path(os.environ.get("HERMES_AUTH_PATH", "~/.hermes/auth.json")).expanduser()


def _load_hermes_nvidia() -> Tuple[Optional[str], Optional[str]]:
    """Return (access_token, base_url) from Hermes auth if enabled."""
    if os.environ.get("AGENT_MEMORY_USE_HERMES_AUTH", "1").lower() in ("0", "false", "no"):
        return None, None
    path = _hermes_auth_path()
    if not path.exists():
        return None, None
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
        pool = data.get("credential_pool") or {}
        entries = pool.get("nvidia") or []
        if not entries:
            return None, None
        entry = entries[0] if isinstance(entries, list) else entries
        if not isinstance(entry, dict):
            return None, None
        token = entry.get("access_token") or entry.get("api_key")
        base = entry.get("base_url") or NVIDIA_NIM_DEFAULT_BASE
        return (str(token) if token else None, str(base) if base else None)
    except (OSError, json.JSONDecodeError, KeyError, TypeError):
        return None, None


def resolve_nvidia_credentials() -> Tuple[Optional[str], str, str]:
    """
    API key, base URL, model for NVIDIA NIM (OpenAI-compatible).
    Key order: AGENT_MEMORY_NVIDIA_API_KEY, NVIDIA_API_KEY, NVAPI_API_KEY, Hermes auth.
    """
    api_key = (
        os.environ.get("AGENT_MEMORY_NVIDIA_API_KEY")
        or os.environ.get("NVIDIA_API_KEY")
        or os.environ.get("NVAPI_API_KEY")
    )
    base_url = os.environ.get(
        "AGENT_MEMORY_NVIDIA_BASE_URL",
        os.environ.get("NVIDIA_BASE_URL", NVIDIA_NIM_DEFAULT_BASE),
    )
    model = os.environ.get("AGENT_MEMORY_NVIDIA_MODEL", NVIDIA_NIM_DEFAULT_MODEL)

    if not api_key:
        hermes_key, hermes_base = _load_hermes_nvidia()
        if hermes_key:
            api_key = hermes_key
            if hermes_base:
                base_url = hermes_base

    return api_key, base_url, model


def nvidia_configured() -> bool:
    return bool(resolve_nvidia_credentials()[0])
