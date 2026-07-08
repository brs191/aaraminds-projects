from __future__ import annotations

import argparse
import asyncio
import json
import time
from dataclasses import dataclass

import httpx
import psycopg
from pgvector.psycopg import register_vector
from pydantic import BaseModel, Field
from pydantic_settings import BaseSettings, SettingsConfigDict

SELECT_PENDING_SQL = """
SELECT
    mn.node_id,
    mn.repo_id,
    mn.qualified_name,
    mn.simple_name,
    mn.source_ref
FROM rif_meta.method_nodes mn
WHERE mn.embedding IS NULL
  AND mn.origin = 'first_party'
  AND (%(repo_id)s::text IS NULL OR mn.repo_id = %(repo_id)s::text)
ORDER BY mn.repo_id, mn.qualified_name;
"""

UPSERT_EMBEDDING_SQL = """
UPDATE rif_meta.method_nodes
SET embedding = %(embedding)s,
    embedding_model = %(embedding_model)s,
    upserted_at = NOW()
WHERE node_id = %(node_id)s;
"""


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

    database_url: str = Field(alias="DATABASE_URL")
    service_url: str = Field(default="http://127.0.0.1:8000", alias="EMBED_SERVICE_URL")
    batch_size: int = Field(default=32, alias="BATCH_SIZE", ge=1)
    concurrency: int = Field(default=4, alias="EMBED_CONCURRENCY", ge=1)


class EmbedRequest(BaseModel):
    node_id: str
    text: str


class EmbedResponse(BaseModel):
    node_id: str
    embedding: list[float]


class HealthResponse(BaseModel):
    status: str
    model: str
    dim: int


@dataclass
class Metrics:
    nodes_embedded: int = 0
    batches_processed: int = 0
    errors: int = 0


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Parallel embedding backfill into rif_meta.method_nodes")
    parser.add_argument("--repo-id", required=True, help="Repository id to backfill")
    parser.add_argument("--limit", type=int, default=None, help="Maximum number of nodes to embed")
    parser.add_argument("--batch-size", type=int, default=None, help="Override batch size per /embed request")
    parser.add_argument("--concurrency", type=int, default=None, help="Override concurrent in-flight /embed requests")
    return parser.parse_args()


def fetch_pending_nodes(conn: psycopg.Connection, repo_id: str, limit: int | None) -> list[EmbedRequest]:
    with conn.cursor() as cur:
        cur.execute(SELECT_PENDING_SQL, {"repo_id": repo_id})
        rows = cur.fetchall()
    if limit is not None:
        rows = rows[:limit]

    items: list[EmbedRequest] = []
    for node_id, _repo_id, qualified_name, simple_name, source_ref in rows:
        parts = [qualified_name or "", simple_name or "", source_ref or ""]
        items.append(EmbedRequest(node_id=node_id, text="\n".join(part for part in parts if part).strip()))
    return items


def chunked(items: list[EmbedRequest], size: int) -> list[list[EmbedRequest]]:
    return [items[index : index + size] for index in range(0, len(items), size)]


async def fetch_health(client: httpx.AsyncClient, service_url: str) -> HealthResponse:
    resp = await client.get(f"{service_url}/health", timeout=10.0)
    resp.raise_for_status()
    return HealthResponse.model_validate(resp.json())


async def embed_batch(client: httpx.AsyncClient, service_url: str, batch: list[EmbedRequest]) -> list[EmbedResponse]:
    resp = await client.post(f"{service_url}/embed", json=[item.model_dump() for item in batch], timeout=120.0)
    resp.raise_for_status()
    return [EmbedResponse.model_validate(item) for item in resp.json()]


async def embed_batch_with_retry(
    client: httpx.AsyncClient,
    service_url: str,
    batch: list[EmbedRequest],
    attempts: int = 3,
    backoff_seconds: float = 2.0,
) -> list[EmbedResponse]:
    last_exc: Exception | None = None
    for attempt in range(1, attempts + 1):
        try:
            return await embed_batch(client, service_url, batch)
        except (httpx.HTTPError, ValueError) as exc:
            last_exc = exc
            if attempt < attempts:
                await asyncio.sleep(backoff_seconds * attempt)
    raise RuntimeError(f"batch of {len(batch)} failed after {attempts} attempts") from last_exc


async def run_and_persist_batches(
    conn: psycopg.Connection,
    service_url: str,
    batches: list[list[EmbedRequest]],
    concurrency: int,
    embedding_model: str,
    metrics: Metrics,
) -> None:
    """Embed batches concurrently, persisting each batch as it completes.

    Previous behavior gathered ALL batches before the first DB write, so one
    failed batch discarded every completed embedding and held all vectors in
    memory simultaneously (RIF review finding H7). Now each completed batch is
    committed immediately; a failed batch is counted and skipped, never fatal
    to its siblings.
    """
    semaphore = asyncio.Semaphore(concurrency)

    async with httpx.AsyncClient() as client:

        async def run_one(batch: list[EmbedRequest]) -> list[EmbedResponse]:
            async with semaphore:
                return await embed_batch_with_retry(client, service_url, batch)

        tasks = [asyncio.create_task(run_one(batch)) for batch in batches]
        for completed in asyncio.as_completed(tasks):
            try:
                rows = await completed
            except Exception as exc:
                metrics.errors += 1
                print(json.dumps({"event": "batch_failed", "error": str(exc)}))
                continue
            metrics.nodes_embedded += write_embeddings(conn, rows, embedding_model)
            metrics.batches_processed += 1


def write_embeddings(conn: psycopg.Connection, rows: list[EmbedResponse], embedding_model: str) -> int:
    with conn.cursor() as cur:
        cur.executemany(
            UPSERT_EMBEDDING_SQL,
            [
                {"node_id": row.node_id, "embedding": row.embedding, "embedding_model": embedding_model}
                for row in rows
            ],
        )
    conn.commit()
    return len(rows)


async def async_main() -> None:
    args = parse_args()
    settings = Settings()
    batch_size = args.batch_size or settings.batch_size
    concurrency = args.concurrency or settings.concurrency
    metrics = Metrics()
    started = time.monotonic()

    with psycopg.connect(settings.database_url) as conn:
        register_vector(conn)
        pending = fetch_pending_nodes(conn, args.repo_id, args.limit)
        async with httpx.AsyncClient() as client:
            health = await fetch_health(client, settings.service_url)
        batches = chunked(pending, batch_size)
        await run_and_persist_batches(conn, settings.service_url, batches, concurrency, health.model, metrics)

    elapsed = time.monotonic() - started
    print(
        json.dumps(
            {
                "repo_id": args.repo_id,
                "embedding_model": health.model,
                "embedding_dim": health.dim,
                "nodes_embedded": metrics.nodes_embedded,
                "batches_processed": metrics.batches_processed,
                "concurrency": concurrency,
                "elapsed_time": round(elapsed, 3),
                "errors": metrics.errors,
            }
        )
    )


def main() -> None:
    asyncio.run(async_main())


if __name__ == "__main__":
    main()
