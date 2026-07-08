from __future__ import annotations

import asyncio
import hashlib
import json
import logging
from pathlib import Path
from typing import Any
from typing import Protocol
from typing import Sequence
from urllib.parse import urlparse

import httpx
import numpy as np
from fastapi import FastAPI
from fastapi import HTTPException
from pydantic import BaseModel, Field
from pydantic_settings import BaseSettings, SettingsConfigDict

DEFAULT_EMBEDDING_MODEL = "text-embedding-3-small"
LOCAL_FALLBACK_MODEL_NAME = "jinaai/jina-embeddings-v2-base-code"
HASH_MODEL_NAME = "hash-deterministic-v1"
EMBEDDING_DIM = 768

logger = logging.getLogger("embedding_service")


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

    embedding_provider: str = Field(default="local", alias="EMBEDDING_PROVIDER")
    embedding_model: str = Field(default=DEFAULT_EMBEDDING_MODEL, alias="EMBEDDING_MODEL")
    embedding_dim: int = Field(default=EMBEDDING_DIM, alias="EMBEDDING_DIM", ge=1)
    model_path: str | None = Field(default=None, alias="MODEL_PATH")
    batch_size: int = Field(default=32, alias="BATCH_SIZE", ge=1)
    max_batch_items: int = Field(default=256, alias="MAX_BATCH_ITEMS", ge=1)
    encode_concurrency: int = Field(default=2, alias="ENCODE_CONCURRENCY", ge=1)
    max_text_len: int = Field(default=512, alias="MAX_TEXT_LEN", ge=1)
    port: int = Field(default=8000, alias="PORT", ge=1, le=65535)
    litellm_base_url: str | None = Field(default=None, alias="LITELLM_BASE_URL")
    litellm_api_key: str | None = Field(default=None, alias="LITELLM_API_KEY")
    litellm_timeout_seconds: float = Field(default=120.0, alias="LITELLM_TIMEOUT_SECONDS", gt=0)
    litellm_encoding_format: str | None = Field(default=None, alias="LITELLM_ENCODING_FORMAT")
    litellm_user: str | None = Field(default=None, alias="LITELLM_USER")
    litellm_extra_body_json: str | None = Field(default=None, alias="LITELLM_EXTRA_BODY_JSON")


class EmbedInput(BaseModel):
    node_id: str
    text: str


class EmbedOutput(BaseModel):
    node_id: str
    embedding: list[float]


class HealthResponse(BaseModel):
    status: str
    model: str
    dim: int


class Embedder(Protocol):
    model_name: str

    def encode(self, texts: Sequence[str]) -> np.ndarray:
        ...


class LocalSentenceTransformerEmbedder:
    def __init__(self, settings: Settings) -> None:
        try:
            from sentence_transformers import SentenceTransformer
        except ImportError as exc:
            raise RuntimeError("sentence_transformers is required when EMBEDDING_PROVIDER=local") from exc
        if not settings.model_path:
            raise RuntimeError("MODEL_PATH is required when EMBEDDING_PROVIDER=local")
        model_dir = Path(settings.model_path)
        if not model_dir.exists():
            msg = (
                f"Model not found at {settings.model_path}. Pre-download with: "
                f"huggingface-cli download {LOCAL_FALLBACK_MODEL_NAME} --local-dir {settings.model_path}"
            )
            raise RuntimeError(msg)

        self._model = SentenceTransformer(
            str(model_dir),
            local_files_only=True,
            trust_remote_code=True,
        )
        requested_model = (settings.embedding_model or "").strip()
        if requested_model and requested_model != DEFAULT_EMBEDDING_MODEL:
            self.model_name = requested_model
        else:
            self.model_name = LOCAL_FALLBACK_MODEL_NAME

    def encode(self, texts: Sequence[str]) -> np.ndarray:
        vectors = self._model.encode(
            list(texts),
            convert_to_numpy=True,
            show_progress_bar=False,
        )
        return np.asarray(vectors, dtype=np.float32)


class HashEmbedder:
    def __init__(self, dim: int) -> None:
        self.dim = dim
        self.model_name = f"{HASH_MODEL_NAME}-{dim}"

    def encode(self, texts: Sequence[str]) -> np.ndarray:
        embeddings = [self._hash_embedding(text) for text in texts]
        return np.asarray(embeddings, dtype=np.float32)

    def _hash_embedding(self, text: str) -> list[float]:
        normalized = text.strip().lower()
        vec: list[float] = []
        seed = 0
        while len(vec) < self.dim:
            digest = hashlib.sha256(f"{seed}:{normalized}".encode("utf-8")).digest()
            for i in range(0, len(digest), 4):
                if len(vec) >= self.dim:
                    break
                value = int.from_bytes(digest[i : i + 4], byteorder="little", signed=False)
                scaled = (value / 0xFFFFFFFF) * 2.0 - 1.0
                vec.append(scaled)
            seed += 1
        norm = np.linalg.norm(vec)
        if norm > 0:
            vec = [v / norm for v in vec]
        return vec


class LiteLLMEmbedder:
    def __init__(self, settings: Settings, http_client: httpx.Client | None = None) -> None:
        if not settings.litellm_base_url:
            raise RuntimeError("LITELLM_BASE_URL is required when EMBEDDING_PROVIDER=litellm")
        if not settings.litellm_api_key:
            raise RuntimeError("LITELLM_API_KEY is required when EMBEDDING_PROVIDER=litellm")

        self._model = settings.embedding_model
        self._dim = settings.embedding_dim
        self._api_key: str = settings.litellm_api_key
        self._endpoint = self._build_embeddings_endpoint(settings.litellm_base_url)
        self._client = http_client or httpx.Client(timeout=settings.litellm_timeout_seconds)
        self._encoding_format = settings.litellm_encoding_format
        self._user = settings.litellm_user
        self._extra_body = self._parse_extra_body_json(settings.litellm_extra_body_json)
        self.model_name = self._model

    @staticmethod
    def _build_embeddings_endpoint(base_url: str) -> str:
        normalized = base_url.strip().rstrip("/")
        parsed = urlparse(normalized)
        path = parsed.path.rstrip("/")

        if path.endswith("/embeddings") or path.endswith("/v1/embeddings"):
            endpoint_path = path
        elif path.endswith("/v1"):
            endpoint_path = f"{path}/embeddings"
        else:
            endpoint_path = f"{path}/v1/embeddings" if path else "/v1/embeddings"

        return parsed._replace(path=endpoint_path).geturl()

    @staticmethod
    def _parse_extra_body_json(value: str | None) -> dict[str, Any]:
        if value is None or value.strip() == "":
            return {}
        try:
            parsed = json.loads(value)
        except json.JSONDecodeError as exc:
            raise RuntimeError(f"LITELLM_EXTRA_BODY_JSON must be valid JSON: {exc.msg}") from exc
        if not isinstance(parsed, dict):
            raise RuntimeError("LITELLM_EXTRA_BODY_JSON must decode to a JSON object")
        return parsed

    def _build_payload(self, texts: Sequence[str]) -> dict[str, Any]:
        payload: dict[str, Any] = dict(self._extra_body)
        if self._encoding_format:
            payload["encoding_format"] = self._encoding_format
        if self._user:
            payload["user"] = self._user

        payload["model"] = self._model
        payload["input"] = list(texts)
        payload["dimensions"] = self._dim
        return payload

    def encode(self, texts: Sequence[str]) -> np.ndarray:
        headers = {
            "Authorization": f"Bearer {self._api_key}",
            "Content-Type": "application/json",
        }
        payload = self._build_payload(texts)
        try:
            response = self._client.post(self._endpoint, headers=headers, json=payload)
            response.raise_for_status()
        except httpx.HTTPError as exc:
            raise RuntimeError(f"LiteLLM embedding request failed: {exc}") from exc

        try:
            body = response.json()
        except ValueError as exc:
            raise RuntimeError("LiteLLM response is not valid JSON") from exc
        if not isinstance(body, dict):
            raise RuntimeError("LiteLLM response must be a JSON object")
        data = body.get("data")
        if not isinstance(data, list):
            raise RuntimeError("LiteLLM response field 'data' must be a list")

        embeddings: list[list[float]] = []
        for index, item in enumerate(data):
            if not isinstance(item, dict):
                raise RuntimeError(f"LiteLLM response data[{index}] must be an object")
            embedding = item.get("embedding")
            if not isinstance(embedding, list):
                raise RuntimeError(f"LiteLLM response data[{index}].embedding must be a list")
            converted: list[float] = []
            for value in embedding:
                if isinstance(value, bool) or not isinstance(value, (int, float)):
                    raise RuntimeError(
                        f"LiteLLM response data[{index}].embedding contains non-numeric values"
                    )
                converted.append(float(value))
            embeddings.append(converted)

        if len(embeddings) != len(texts):
            raise RuntimeError(
                f"LiteLLM response length mismatch: got {len(embeddings)} embeddings for {len(texts)} inputs"
            )

        return np.asarray(embeddings, dtype=np.float32)


class EmbeddingService:
    def __init__(self, settings: Settings, embedder: Embedder) -> None:
        self.settings = settings
        self.embedder = embedder
        # Bound concurrent encode calls. Without this, anyio's default thread
        # pool lets ~40 threads into one SentenceTransformer.encode and
        # concurrent PyTorch CPU inference thrashes (RIF review finding M11).
        self._encode_semaphore = asyncio.Semaphore(settings.encode_concurrency)

    def _truncate(self, texts: Sequence[str]) -> list[str]:
        max_len = self.settings.max_text_len
        return [text[:max_len] for text in texts]

    async def _encode_chunk(self, texts: list[str]) -> np.ndarray:
        async with self._encode_semaphore:
            return await asyncio.to_thread(self.embedder.encode, texts)

    async def embed(self, items: Sequence[EmbedInput]) -> list[EmbedOutput]:
        texts = self._truncate([item.text for item in items])
        # Chunk upstream encode calls by BATCH_SIZE instead of forwarding the
        # whole client payload in one call (RIF review finding H4/H9 — one
        # oversized upstream request blows provider limits and timeouts).
        chunk_size = self.settings.batch_size
        chunks = [texts[i : i + chunk_size] for i in range(0, len(texts), chunk_size)]
        vector_parts = [await self._encode_chunk(chunk) for chunk in chunks]
        vectors = np.concatenate(vector_parts, axis=0) if len(vector_parts) > 1 else vector_parts[0]
        if vectors.ndim != 2 or vectors.shape[1] != self.settings.embedding_dim:
            raise RuntimeError(
                f"Model returned embeddings with shape {vectors.shape}, expected (*, {self.settings.embedding_dim})"
            )
        if vectors.shape[0] != len(items):
            raise RuntimeError(
                f"Model returned {vectors.shape[0]} embeddings for {len(items)} inputs"
            )

        return [
            EmbedOutput(node_id=item.node_id, embedding=vectors[idx].tolist())
            for idx, item in enumerate(items)
        ]


def create_app(
    settings: Settings | None = None,
    embedder: Embedder | None = None,
) -> FastAPI:
    explicit_settings = settings
    explicit_embedder = embedder

    app = FastAPI(title="RIF Embedding Service", version="0.1.0")

    app.state.settings = explicit_settings
    app.state.embedder = explicit_embedder
    app.state.service = None

    def ensure_state_initialized() -> None:
        cfg = app.state.settings or Settings()
        emb = app.state.embedder
        if emb is None:
            provider = cfg.embedding_provider.lower().strip()
            if provider in {"local", "jina"}:
                emb = LocalSentenceTransformerEmbedder(cfg)
            elif provider == "litellm":
                emb = LiteLLMEmbedder(cfg)
            elif provider == "hash":
                emb = HashEmbedder(cfg.embedding_dim)
            else:
                raise RuntimeError(
                    f"Unsupported EMBEDDING_PROVIDER={cfg.embedding_provider}. Supported: local|jina|hash|litellm"
                )
        app.state.settings = cfg
        app.state.embedder = emb
        app.state.service = EmbeddingService(settings=cfg, embedder=emb)

    if explicit_settings is not None and explicit_embedder is not None:
        ensure_state_initialized()

    @app.on_event("startup")
    async def startup() -> None:
        ensure_state_initialized()

    @app.get("/health", response_model=HealthResponse)
    async def health() -> HealthResponse:
        if app.state.embedder is None:
            ensure_state_initialized()
        embedder_instance = app.state.embedder
        return HealthResponse(status="ok", model=embedder_instance.model_name, dim=app.state.settings.embedding_dim)

    @app.post("/embed", response_model=list[EmbedOutput])
    async def embed(batch: list[EmbedInput]) -> list[EmbedOutput]:
        if app.state.service is None:
            ensure_state_initialized()
        service = app.state.service
        if not batch:
            raise HTTPException(status_code=422, detail="batch must contain at least one item")
        max_items = app.state.settings.max_batch_items
        if len(batch) > max_items:
            raise HTTPException(
                status_code=413,
                detail=f"batch of {len(batch)} exceeds MAX_BATCH_ITEMS={max_items}; split the request",
            )
        try:
            return await service.embed(batch)
        except RuntimeError as exc:
            # Log the detail, return a generic message — upstream provider
            # URLs/response bodies must not leak to clients (finding M13).
            logger.error("embed_failed batch_size=%d", len(batch), exc_info=True)
            raise HTTPException(status_code=500, detail="embedding backend error") from exc

    return app


app = create_app()
