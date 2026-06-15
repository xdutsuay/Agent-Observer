"""Optional LLM extraction for facts and summaries."""

from __future__ import annotations

import os
import time
import threading
from collections import deque
from typing import TYPE_CHECKING, Optional

if TYPE_CHECKING:
    from .engine.db import MemoryEngine

from .llm_credentials import resolve_nvidia_credentials

RATE_WINDOW_SEC = 3600
MAX_CALLS_PER_REPO_PER_HOUR = 20


class MemoryExtractor:
    def __init__(self, engine: "MemoryEngine"):
        self.engine = engine
        self.provider = os.environ.get("AGENT_MEMORY_LLM_PROVIDER", "none").lower()
        self._calls: deque = deque()
        self._lock = threading.Lock()

    def _rate_ok(self, repo_id: str) -> bool:
        now = time.time()
        with self._lock:
            self._calls = deque((t, r) for t, r in self._calls if now - t < RATE_WINDOW_SEC)
            count = sum(1 for _, r in self._calls if r == repo_id)
            if count >= MAX_CALLS_PER_REPO_PER_HOUR:
                return False
            self._calls.append((now, repo_id))
        return True

    def maybe_extract(self, memory_id: str, repo_id: str, kind: str, content: str) -> None:
        if self.provider == "none":
            return
        if kind not in ("failure", "decision", "attempt"):
            return
        if not self._rate_ok(repo_id):
            return
        if len(content) < 40:
            return

        threading.Thread(
            target=self._extract_worker,
            args=(memory_id, repo_id, kind, content),
            daemon=True,
        ).start()

    def _extract_worker(self, memory_id: str, repo_id: str, kind: str, content: str) -> None:
        try:
            summary, fact = self._call_llm(kind, content)
            if summary:
                self.engine.update_summary(memory_id, summary)
            if fact:
                self.engine.insert_memory(
                    repo_id, "fact", fact, source="llm_extractor", metadata={"from": memory_id}
                )
        except Exception:
            pass

    def _call_llm(self, kind: str, content: str) -> tuple[Optional[str], Optional[str]]:
        prompt = (
            f"Summarize this agent {kind} in one sentence. "
            f"If there is a reusable lesson, add a second line starting with FACT:.\n\n{content[:3000]}"
        )
        if self.provider in ("openai", "nvidia"):
            return self._openai_compatible(prompt, provider=self.provider)
        if self.provider == "anthropic":
            return self._anthropic(prompt)
        return None, None

    def _openai_compatible(
        self, prompt: str, provider: str = "openai"
    ) -> tuple[Optional[str], Optional[str]]:
        try:
            from openai import OpenAI
        except ImportError:
            return None, None

        if provider == "nvidia":
            api_key, base_url, model = resolve_nvidia_credentials()
            if not api_key:
                return None, None
            client = OpenAI(api_key=api_key, base_url=base_url)
        else:
            api_key = os.environ.get("OPENAI_API_KEY")
            if not api_key:
                return None, None
            base_url = os.environ.get("OPENAI_BASE_URL")
            model = os.environ.get("AGENT_MEMORY_OPENAI_MODEL", "gpt-4o-mini")
            client = (
                OpenAI(api_key=api_key, base_url=base_url)
                if base_url
                else OpenAI(api_key=api_key)
            )
        resp = client.chat.completions.create(
            model=model,
            messages=[{"role": "user", "content": prompt}],
            max_tokens=200,
        )
        text = resp.choices[0].message.content or ""
        return self._parse_response(text)

    def _openai(self, prompt: str) -> tuple[Optional[str], Optional[str]]:
        return self._openai_compatible(prompt, provider="openai")

    def _anthropic(self, prompt: str) -> tuple[Optional[str], Optional[str]]:
        try:
            import anthropic
        except ImportError:
            return None, None
        api_key = os.environ.get("ANTHROPIC_API_KEY")
        if not api_key:
            return None, None
        client = anthropic.Anthropic(api_key=api_key)
        model = os.environ.get("AGENT_MEMORY_ANTHROPIC_MODEL", "claude-3-5-haiku-20241022")
        resp = client.messages.create(
            model=model,
            max_tokens=200,
            messages=[{"role": "user", "content": prompt}],
        )
        text = ""
        for block in resp.content:
            if hasattr(block, "text"):
                text += block.text
        return self._parse_response(text)

    @staticmethod
    def _parse_response(text: str) -> tuple[Optional[str], Optional[str]]:
        lines = [ln.strip() for ln in text.strip().splitlines() if ln.strip()]
        summary = lines[0] if lines else None
        fact = None
        for ln in lines[1:]:
            if ln.upper().startswith("FACT:"):
                fact = ln[5:].strip()
                break
        return summary, fact
