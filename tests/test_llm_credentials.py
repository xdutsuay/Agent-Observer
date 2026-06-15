import json
import os
from pathlib import Path

from agent_memory_mcp.llm_credentials import (
    resolve_nvidia_credentials,
    nvidia_configured,
    _load_hermes_nvidia,
)


def test_resolve_from_env(monkeypatch):
    monkeypatch.setenv("AGENT_MEMORY_USE_HERMES_AUTH", "0")
    monkeypatch.setenv("AGENT_MEMORY_NVIDIA_API_KEY", "test-key-123")
    key, base, model = resolve_nvidia_credentials()
    assert key == "test-key-123"
    assert "nvidia.com" in base
    assert nvidia_configured()


def test_load_hermes_mock(tmp_path, monkeypatch):
    auth = {
        "credential_pool": {
            "nvidia": [
                {
                    "access_token": "hermes-token",
                    "base_url": "https://integrate.api.nvidia.com/v1",
                }
            ]
        }
    }
    path = tmp_path / "auth.json"
    path.write_text(json.dumps(auth))
    monkeypatch.setenv("HERMES_AUTH_PATH", str(path))
    monkeypatch.delenv("AGENT_MEMORY_NVIDIA_API_KEY", raising=False)
    monkeypatch.delenv("NVIDIA_API_KEY", raising=False)
    key, base = _load_hermes_nvidia()
    assert key == "hermes-token"
    assert "nvidia.com" in base
