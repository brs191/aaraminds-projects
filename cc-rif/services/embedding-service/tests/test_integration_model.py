from __future__ import annotations

import os

import numpy as np
import pytest
from fastapi.testclient import TestClient

from app import create_app


@pytest.mark.skipif(not os.getenv("MODEL_PATH"), reason="MODEL_PATH is not set")
def test_similar_strings_have_high_cosine_similarity() -> None:
    app = create_app()
    client = TestClient(app)

    payload = [
        {"node_id": "a", "text": "calculate account balance for customer"},
        {"node_id": "b", "text": "compute customer account balance"},
    ]
    resp = client.post("/embed", json=payload)

    assert resp.status_code == 200
    data = resp.json()
    v1 = np.array(data[0]["embedding"], dtype=np.float32)
    v2 = np.array(data[1]["embedding"], dtype=np.float32)
    cosine = float(np.dot(v1, v2) / (np.linalg.norm(v1) * np.linalg.norm(v2)))
    assert cosine > 0.8
