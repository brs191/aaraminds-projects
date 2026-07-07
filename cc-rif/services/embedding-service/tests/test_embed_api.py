from __future__ import annotations

import json

import httpx
import numpy as np
import pytest
from fastapi.testclient import TestClient

import app as embedding_app
from app import EMBEDDING_DIM, Settings, create_app


class DummyEmbedder:
    model_name = "dummy"

    def __init__(self) -> None:
        self.last_texts: list[str] = []

    def encode(self, texts: list[str]) -> np.ndarray:
        self.last_texts = texts
        return np.ones((len(texts), EMBEDDING_DIM), dtype=np.float32)


def test_embed_returns_expected_shape() -> None:
    embedder = DummyEmbedder()
    app = create_app(
        settings=Settings.model_construct(
            embedding_provider="local",
            embedding_model="dummy",
            embedding_dim=EMBEDDING_DIM,
            model_path="/tmp/model",
            batch_size=32,
            max_text_len=512,
            port=8000,
        ),
        embedder=embedder,
    )
    client = TestClient(app)

    payload = [
        {"node_id": "n1", "text": "alpha"},
        {"node_id": "n2", "text": "beta"},
    ]
    resp = client.post("/embed", json=payload)

    assert resp.status_code == 200
    body = resp.json()
    assert len(body) == 2
    assert all(len(item["embedding"]) == EMBEDDING_DIM for item in body)


def test_embed_truncates_text_without_error() -> None:
    embedder = DummyEmbedder()
    app = create_app(
        settings=Settings.model_construct(
            embedding_provider="local",
            embedding_model="dummy",
            embedding_dim=EMBEDDING_DIM,
            model_path="/tmp/model",
            batch_size=32,
            max_text_len=8,
            port=8000,
        ),
        embedder=embedder,
    )
    client = TestClient(app)

    long_text = "x" * 100
    resp = client.post("/embed", json=[{"node_id": "n1", "text": long_text}])

    assert resp.status_code == 200
    assert embedder.last_texts == ["x" * 8]


def test_hash_provider_uses_configured_dimension() -> None:
    app = create_app(
        settings=Settings.model_construct(
            embedding_provider="hash",
            embedding_model="",
            embedding_dim=256,
            model_path=None,
            batch_size=32,
            max_text_len=128,
            port=8000,
        )
    )
    client = TestClient(app)

    health = client.get("/health")
    assert health.status_code == 200
    assert health.json()["dim"] == 256

    resp = client.post("/embed", json=[{"node_id": "n1", "text": "alpha"}])
    assert resp.status_code == 200
    assert len(resp.json()[0]["embedding"]) == 256


class WrongDimEmbedder:
    model_name = "wrong-dim"

    def encode(self, texts: list[str]) -> np.ndarray:
        return np.ones((len(texts), EMBEDDING_DIM - 1), dtype=np.float32)


def test_embed_dimension_mismatch_fails() -> None:
    app = create_app(
        settings=Settings.model_construct(
            embedding_provider="local",
            embedding_model="dummy",
            embedding_dim=EMBEDDING_DIM,
            model_path="/tmp/model",
            batch_size=32,
            max_text_len=128,
            port=8000,
        ),
        embedder=WrongDimEmbedder(),
    )
    client = TestClient(app)

    resp = client.post("/embed", json=[{"node_id": "n1", "text": "alpha"}])
    assert resp.status_code == 500
    assert "expected (*, 768)" in resp.json()["detail"]


def test_litellm_provider_path_uses_mocked_http_transport(monkeypatch) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        assert str(request.url) == "http://litellm.local/v1/embeddings"
        assert request.headers["Authorization"] == "Bearer test-key"
        payload = json.loads(request.read().decode("utf-8"))
        assert payload["model"] == "text-embedding-3-small"
        assert payload["dimensions"] == EMBEDDING_DIM
        return httpx.Response(
            status_code=200,
            json={
                "data": [
                    {"embedding": [0.1] * EMBEDDING_DIM},
                    {"embedding": [0.2] * EMBEDDING_DIM},
                ]
            },
        )

    transport = httpx.MockTransport(handler)
    real_httpx_client = httpx.Client

    def client_factory(*, timeout: float) -> httpx.Client:
        return real_httpx_client(transport=transport, timeout=timeout)

    monkeypatch.setattr(embedding_app.httpx, "Client", client_factory)

    app = create_app(
        settings=Settings.model_construct(
            embedding_provider="litellm",
            embedding_model="text-embedding-3-small",
            embedding_dim=EMBEDDING_DIM,
            model_path=None,
            batch_size=32,
            max_text_len=512,
            port=8000,
            litellm_base_url="http://litellm.local",
            litellm_api_key="test-key",
            litellm_timeout_seconds=30.0,
        )
    )
    client = TestClient(app)

    resp = client.post(
        "/embed",
        json=[
            {"node_id": "n1", "text": "alpha"},
            {"node_id": "n2", "text": "beta"},
        ],
    )
    assert resp.status_code == 200
    body = resp.json()
    assert len(body) == 2
    assert all(len(item["embedding"]) == EMBEDDING_DIM for item in body)


def test_litellm_provider_dimension_mismatch_returns_500(monkeypatch) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            status_code=200,
            json={
                "data": [
                    {"embedding": [0.1] * (EMBEDDING_DIM - 1)},
                ]
            },
        )

    transport = httpx.MockTransport(handler)
    real_httpx_client = httpx.Client

    def client_factory(*, timeout: float) -> httpx.Client:
        return real_httpx_client(transport=transport, timeout=timeout)

    monkeypatch.setattr(embedding_app.httpx, "Client", client_factory)

    app = create_app(
        settings=Settings.model_construct(
            embedding_provider="litellm",
            embedding_model="text-embedding-3-small",
            embedding_dim=EMBEDDING_DIM,
            model_path=None,
            batch_size=32,
            max_text_len=512,
            port=8000,
            litellm_base_url="http://litellm.local",
            litellm_api_key="test-key",
            litellm_timeout_seconds=30.0,
        )
    )
    client = TestClient(app)

    resp = client.post("/embed", json=[{"node_id": "n1", "text": "alpha"}])
    assert resp.status_code == 500
    assert "expected (*, 768)" in resp.json()["detail"]


@pytest.mark.parametrize(
    ("base_url", "expected"),
    [
        ("http://host", "http://host/v1/embeddings"),
        ("http://host/v1", "http://host/v1/embeddings"),
        ("http://host/embeddings", "http://host/embeddings"),
        ("http://host/v1/embeddings", "http://host/v1/embeddings"),
    ],
)
def test_litellm_endpoint_normalization_matrix(base_url: str, expected: str) -> None:
    assert embedding_app.LiteLLMEmbedder._build_embeddings_endpoint(base_url) == expected


def test_litellm_payload_optional_fields_and_extra_body_merge(monkeypatch) -> None:
    captured_payload: dict[str, object] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal captured_payload
        captured_payload = json.loads(request.read().decode("utf-8"))
        return httpx.Response(
            status_code=200,
            json={"data": [{"embedding": [0.1] * EMBEDDING_DIM}]},
        )

    transport = httpx.MockTransport(handler)
    real_httpx_client = httpx.Client

    def client_factory(*, timeout: float) -> httpx.Client:
        return real_httpx_client(transport=transport, timeout=timeout)

    monkeypatch.setattr(embedding_app.httpx, "Client", client_factory)

    app = create_app(
        settings=Settings.model_construct(
            embedding_provider="litellm",
            embedding_model="text-embedding-3-small",
            embedding_dim=EMBEDDING_DIM,
            model_path=None,
            batch_size=32,
            max_text_len=512,
            port=8000,
            litellm_base_url="http://litellm.local",
            litellm_api_key="test-key",
            litellm_timeout_seconds=30.0,
            litellm_encoding_format="base64",
            litellm_user="user-123",
            litellm_extra_body_json=json.dumps(
                {
                    "metadata": {"source": "test"},
                    "model": "must-not-override",
                    "input": "must-not-override",
                    "dimensions": 1,
                    "custom_flag": True,
                }
            ),
        )
    )
    client = TestClient(app)

    resp = client.post("/embed", json=[{"node_id": "n1", "text": "alpha"}])
    assert resp.status_code == 200

    assert captured_payload["model"] == "text-embedding-3-small"
    assert captured_payload["input"] == ["alpha"]
    assert captured_payload["dimensions"] == EMBEDDING_DIM
    assert captured_payload["encoding_format"] == "base64"
    assert captured_payload["user"] == "user-123"
    assert captured_payload["metadata"] == {"source": "test"}
    assert captured_payload["custom_flag"] is True
