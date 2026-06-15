"""Pluggable embedding providers for semantic search.

Replaces the toy _simple_embedding() with a proper abstraction:
- HashEmbedder: zero-dependency fallback (64-dim bag-of-words hash)
- LocalEmbedder: sentence-transformers (384-dim, good quality)
- APIEmbedder: OpenAI-compatible API (works with OpenAI, NVIDIA NIM, etc.)
"""

from __future__ import annotations

import math
import os
import re
from typing import List, Optional, Protocol, runtime_checkable


@runtime_checkable
class EmbeddingProvider(Protocol):
    """Any object that can embed text into a float vector."""

    @property
    def dim(self) -> int: ...

    def embed(self, text: str) -> List[float]: ...

    def embed_batch(self, texts: List[str]) -> List[List[float]]: ...


class HashEmbedder:
    """Original bag-of-words hash embedder (fallback, dim=64). Zero dependencies."""

    def __init__(self, dim: int = 64):
        self._dim = dim

    @property
    def dim(self) -> int:
        return self._dim

    def embed(self, text: str) -> List[float]:
        vec = [0.0] * self._dim
        tokens = re.findall(r"[a-z0-9]{3,}", text.lower())
        if not tokens:
            return vec
        for tok in tokens:
            h = hash(tok) % self._dim
            vec[h] += 1.0
        norm = math.sqrt(sum(x * x for x in vec)) or 1.0
        return [x / norm for x in vec]

    def embed_batch(self, texts: List[str]) -> List[List[float]]:
        return [self.embed(t) for t in texts]


class LocalEmbedder:
    """sentence-transformers based local embeddings.

    Falls back to HashEmbedder if sentence-transformers is not installed.
    """

    DEFAULT_MODEL = "all-MiniLM-L6-v2"  # 384-dim, fast, good quality

    def __init__(self, model_name: Optional[str] = None):
        self._model_name = model_name or os.environ.get(
            "AGENT_MEMORY_EMBED_MODEL", self.DEFAULT_MODEL
        )
        self._model = None
        self._dim_val: int = 64
        self._fallback: Optional[HashEmbedder] = None
        self._load_model()

    def _load_model(self) -> None:
        try:
            from sentence_transformers import SentenceTransformer

            self._model = SentenceTransformer(self._model_name)
            test = self._model.encode(["test"])
            self._dim_val = len(test[0])
        except (ImportError, Exception):
            self._fallback = HashEmbedder()
            self._dim_val = self._fallback.dim

    @property
    def dim(self) -> int:
        return self._dim_val

    def embed(self, text: str) -> List[float]:
        if self._fallback:
            return self._fallback.embed(text)
        arr = self._model.encode([text], show_progress_bar=False)  # type: ignore[union-attr]
        return arr[0].tolist()

    def embed_batch(self, texts: List[str]) -> List[List[float]]:
        if self._fallback:
            return self._fallback.embed_batch(texts)
        arr = self._model.encode(texts, show_progress_bar=False, batch_size=32)  # type: ignore[union-attr]
        return [v.tolist() for v in arr]


class APIEmbedder:
    """OpenAI-compatible API embedder (works with OpenAI, NVIDIA NIM, etc.)."""

    def __init__(
        self,
        api_key: Optional[str] = None,
        base_url: Optional[str] = None,
        model: str = "text-embedding-3-small",
        dim: int = 1536,
    ):
        self._api_key = api_key or os.environ.get("OPENAI_API_KEY", "")
        self._base_url = base_url or os.environ.get("OPENAI_BASE_URL")
        self._model = model
        self._dim_val = dim
        self._client = None

    def _get_client(self):  # type: ignore[no-untyped-def]
        if self._client is None:
            from openai import OpenAI

            kwargs: dict = {"api_key": self._api_key}
            if self._base_url:
                kwargs["base_url"] = self._base_url
            self._client = OpenAI(**kwargs)
        return self._client

    @property
    def dim(self) -> int:
        return self._dim_val

    def embed(self, text: str) -> List[float]:
        return self.embed_batch([text])[0]

    def embed_batch(self, texts: List[str]) -> List[List[float]]:
        client = self._get_client()
        resp = client.embeddings.create(input=texts, model=self._model)
        return [d.embedding for d in resp.data]


def create_embedder(provider: Optional[str] = None) -> EmbeddingProvider:
    """Factory: create the best available embedder.

    provider: "local", "api", "hash", or None (auto-detect).
    Auto-detect order: local (if sentence-transformers installed) -> hash fallback.
    """
    provider = provider or os.environ.get("AGENT_MEMORY_EMBED_PROVIDER", "auto")

    if provider == "api":
        return APIEmbedder()
    elif provider == "local":
        return LocalEmbedder()
    elif provider == "hash":
        return HashEmbedder()
    else:  # auto
        return LocalEmbedder()
