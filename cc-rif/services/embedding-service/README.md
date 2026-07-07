# Phase 2 Embedding Service

FastAPI service for code embeddings with provider routing:

- `EMBEDDING_PROVIDER=local` (default): local `SentenceTransformer` model from filesystem
- `EMBEDDING_PROVIDER=litellm`: LiteLLM/OpenAI-compatible embeddings endpoint
- `EMBEDDING_PROVIDER=hash`: deterministic fallback vectors for degraded mode/testing

## Environment

Required:

- `EMBEDDING_PROVIDER`: `local`, `litellm`, or `hash` (default `local`)
- `EMBEDDING_DIM`: output vector dimension (default `768`)

Optional:

- `EMBEDDING_MODEL` (default `text-embedding-3-small`; local provider can override with a local model id)
- `MODEL_PATH`: local directory containing pre-downloaded model artifacts (required for `local`)
- `LITELLM_BASE_URL`: LiteLLM/OpenAI-compatible base URL (required for `litellm`)
- `LITELLM_API_KEY`: bearer token used for LiteLLM/OpenAI-compatible auth (required for `litellm`)
- `LITELLM_TIMEOUT_SECONDS` (default `120`)
- `LITELLM_ENCODING_FORMAT` (optional, e.g. `float` or `base64`)
- `LITELLM_USER` (optional user identifier pass-through)
- `LITELLM_EXTRA_BODY_JSON` (optional JSON object string merged into request body)
- `PORT` (default `8000`)
- `BATCH_SIZE` (default `32`)
- `MAX_TEXT_LEN` (default `512`)
- `DATABASE_URL` (required for `batch_embed.py`)
- `EMBED_SERVICE_URL` (default `http://127.0.0.1:8000`)

If `MODEL_PATH` does not exist, startup fails with:

`Model not found at {MODEL_PATH}. Pre-download with: huggingface-cli download jinaai/jina-embeddings-v2-base-code --local-dir {MODEL_PATH}`

## Run with uv (local provider)

```bash
cd services/embedding-service
uv sync
EMBEDDING_PROVIDER=local \
EMBEDDING_DIM=768 \
MODEL_PATH=/absolute/path/to/local-model \
uv run uvicorn app:app --host 0.0.0.0 --port ${PORT:-8000}
```

## Run with uv (hash fallback provider)

```bash
cd services/embedding-service
uv sync
EMBEDDING_PROVIDER=hash \
EMBEDDING_DIM=768 \
uv run uvicorn app:app --host 0.0.0.0 --port ${PORT:-8000}
```

## Run with uv (LiteLLM provider)

```bash
cd services/embedding-service
uv sync
EMBEDDING_PROVIDER=litellm \
EMBEDDING_MODEL=text-embedding-3-small \
EMBEDDING_DIM=768 \
LITELLM_BASE_URL=http://localhost:4000 \
LITELLM_API_KEY=your-token \
uv run uvicorn app:app --host 0.0.0.0 --port ${PORT:-8000}
```

LiteLLM mode sends `POST` to an OpenAI-compatible embeddings endpoint with normalization:

- `http://host` -> `http://host/v1/embeddings`
- `http://host/v1` -> `http://host/v1/embeddings`
- `http://host/embeddings` -> unchanged
- `http://host/v1/embeddings` -> unchanged

Payload always includes:

- `model` = `EMBEDDING_MODEL`
- `input` = list of request texts
- `dimensions` = `EMBEDDING_DIM`

Optional payload fields:

- `encoding_format` from `LITELLM_ENCODING_FORMAT`
- `user` from `LITELLM_USER`
- extra keys from `LITELLM_EXTRA_BODY_JSON` (required keys `model`/`input`/`dimensions` always win)

The service expects valid bearer credentials in `LITELLM_API_KEY`.

## API

- `GET /health` -> `{ "status": "ok", "model": "text-embedding-3-small", "dim": 768 }`
- `POST /embed` accepts:

```json
[{ "node_id": "method:123", "text": "public void doThing() { ... }" }]
```

Returns:

```json
[
  {"node_id": "method:123", "embedding": [0.01, -0.02, ...]}
]
```

## Batch backfill

```bash
cd services/embedding-service
DATABASE_URL=postgresql://... EMBEDDING_PROVIDER=litellm EMBEDDING_MODEL=text-embedding-3-small uv run uvicorn app:app --host 0.0.0.0 --port 8000
DATABASE_URL=postgresql://... EMBED_SERVICE_URL=http://127.0.0.1:8000 uv run python batch_cli.py --repo-id <repo_id> --concurrency 4
```

`batch_cli.py` reads pending `rif_meta.method_nodes` rows (`embedding IS NULL`, `origin = 'first_party'`), calls `/embed` in parallel, writes `embedding` and runtime `embedding_model` (from `/health`), and logs JSON metrics:

- `nodes_embedded`
- `batches_processed`
- `elapsed_time`
- `errors`

## Tests

```bash
cd services/embedding-service
uv run pytest
```

Integration test runs only when `MODEL_PATH` is set.

`/health` and `/embed` API contracts are unchanged across providers.
