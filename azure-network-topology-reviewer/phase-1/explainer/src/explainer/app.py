"""FastAPI application — composition root for the explainer service.

Startup sequence
----------------
1. Validate required env-vars; abort if ASKAT_ENDPOINT is missing.
2. Initialise structlog (JSON, contextvars, redaction).
3. Initialise OpenTelemetry (OTLP if endpoint set, else no-op).
4. Instantiate LLMClient + RAGClient; build the LangGraph.
5. FastAPI auto-instruments (httpx, fastapi) via OTel instrumentors.

Runtime invariants
------------------
- POST /explain always returns 200.  LLM failures surface in TopologyReport.error.
- Authorization header is never emitted to any log.
- structlog context always carries subscription_id (never finding evidence).
- trace_id is injected into every log entry from the current OTel span.
"""

from __future__ import annotations

import logging
import os
from contextlib import asynccontextmanager
from typing import Optional

import structlog
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from explainer.graph import build_explainer_graph
from explainer.llm import LLMClient
from explainer.models import (
    ExplainRequest,
    LLMSettings,
    RAGSettings,
    TopologyReport,
)
from explainer.rag import RAGClient
from explainer.stub import StubLLMClient, StubRAGClient

# ---------------------------------------------------------------------------
# structlog configuration
# ---------------------------------------------------------------------------


def _redact_sensitive(
    logger: object, method: str, event_dict: dict
) -> dict:
    """Strip any field that must never appear in logs."""
    _REDACTED = frozenset(
        {
            "authorization",
            "Authorization",
            "client_secret",
            "askat_client_secret",
            "ASKAT_CLIENT_SECRET",
        }
    )
    for key in list(event_dict.keys()):
        if key.lower() in {k.lower() for k in _REDACTED}:
            del event_dict[key]
    return event_dict


def _inject_otel_trace_id(
    logger: object, method: str, event_dict: dict
) -> dict:
    """Add trace_id from the active OTel span (no-op if OTel not active)."""
    try:
        from opentelemetry import trace as otel_trace

        span = otel_trace.get_current_span()
        ctx = span.get_span_context()
        if ctx.is_valid:
            event_dict["trace_id"] = format(ctx.trace_id, "032x")
    except Exception:
        pass
    return event_dict


def configure_structlog() -> None:
    structlog.configure(
        processors=[
            structlog.contextvars.merge_contextvars,
            structlog.stdlib.add_log_level,
            structlog.stdlib.add_logger_name,
            structlog.processors.TimeStamper(fmt="iso"),
            _redact_sensitive,
            _inject_otel_trace_id,
            structlog.processors.StackInfoRenderer(),
            structlog.processors.JSONRenderer(),
        ],
        wrapper_class=structlog.stdlib.BoundLogger,
        context_class=dict,
        logger_factory=structlog.stdlib.LoggerFactory(),
        cache_logger_on_first_use=True,
    )
    # Bridge stdlib logging → structlog so third-party libraries are captured
    handler = logging.StreamHandler()
    handler.setFormatter(
        structlog.stdlib.ProcessorFormatter(
            processor=structlog.processors.JSONRenderer(),
        )
    )
    root = logging.getLogger()
    root.handlers = [handler]
    root.setLevel(logging.INFO)


# ---------------------------------------------------------------------------
# OpenTelemetry configuration
# ---------------------------------------------------------------------------


def configure_otel(service_name: str = "azure-nettopo-explainer") -> None:
    from opentelemetry import trace
    from opentelemetry.sdk.resources import SERVICE_NAME, Resource
    from opentelemetry.sdk.trace import TracerProvider
    from opentelemetry.sdk.trace.export import BatchSpanProcessor

    resource = Resource.create({SERVICE_NAME: service_name})
    provider = TracerProvider(resource=resource)

    otlp_endpoint = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
    if otlp_endpoint:
        from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import (
            OTLPSpanExporter,
        )

        exporter = OTLPSpanExporter(endpoint=otlp_endpoint)
        provider.add_span_processor(BatchSpanProcessor(exporter))

    trace.set_tracer_provider(provider)

    # Auto-instrument FastAPI and httpx
    try:
        from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor  # type: ignore[import]
        from opentelemetry.instrumentation.httpx import HTTPXClientInstrumentor  # type: ignore[import]

        FastAPIInstrumentor().instrument()
        HTTPXClientInstrumentor().instrument()
    except Exception:
        pass


# ---------------------------------------------------------------------------
# Startup validation
# ---------------------------------------------------------------------------


def _explainer_mode() -> str:
    """Return EXPLAINER_MODE env-var, defaulting to 'live'."""
    return os.getenv("EXPLAINER_MODE", "live").lower()


def _validate_env() -> tuple[LLMSettings, RAGSettings]:
    """Parse and validate settings; abort startup on missing required vars.

    In stub mode (EXPLAINER_MODE=stub) the ASKAT_ENDPOINT check is skipped —
    no Azure credentials are required.
    """
    log = structlog.get_logger(__name__)

    llm_settings = LLMSettings()
    rag_settings = RAGSettings()

    mode = _explainer_mode()
    if mode == "stub":
        log.info(
            "startup.stub_mode",
            hint="EXPLAINER_MODE=stub — using canned responses, no Azure calls",
        )
        return llm_settings, rag_settings

    if not llm_settings.endpoint:
        log.critical(
            "startup.missing_required_env",
            var="ASKAT_ENDPOINT",
            hint="Set ASKAT_ENDPOINT to the AskAT&T GenAI endpoint URL",
        )
        raise RuntimeError("ASKAT_ENDPOINT is required")

    log.info(
        "startup.settings_validated",
        model=llm_settings.model,
        search_endpoint=rag_settings.azure_search_endpoint or "<not set>",
        search_index=rag_settings.azure_search_index or "<not set>",
    )
    return llm_settings, rag_settings


# ---------------------------------------------------------------------------
# Lifespan — create / tear down I/O clients once per process
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(app: FastAPI):
    configure_structlog()
    configure_otel()

    log = structlog.get_logger(__name__)

    try:
        llm_settings, rag_settings = _validate_env()
    except RuntimeError as exc:
        log.critical("startup.failed", error=str(exc))
        raise

    mode = _explainer_mode()
    if mode == "stub":
        llm = StubLLMClient()
        rag = StubRAGClient()
        log.info("startup.complete", mode="stub")
    else:
        llm = LLMClient(llm_settings)
        rag = RAGClient(rag_settings)
        log.info("startup.complete", auth_hint="MI-first then client-credentials")

    graph = build_explainer_graph(llm, rag)

    app.state.llm = llm
    app.state.rag = rag
    app.state.graph = graph

    yield

    log.info("shutdown.started")
    await llm.aclose()
    await rag.aclose()
    log.info("shutdown.complete")


# ---------------------------------------------------------------------------
# Application
# ---------------------------------------------------------------------------

app = FastAPI(
    title="Azure NetTopo Explainer",
    version="1.0.0",
    lifespan=lifespan,
)

_log = structlog.get_logger(__name__)


@app.post("/explain", response_model=TopologyReport)
async def explain(request: Request, body: ExplainRequest) -> TopologyReport:
    """Enrich Go engine findings with LLM explanations and RAG recommendations.

    Always returns 200.  LLM failures are surfaced in TopologyReport.error —
    the caller is never given a 500.
    """
    structlog.contextvars.clear_contextvars()
    structlog.contextvars.bind_contextvars(
        subscription_id=body.subscription_id,
        finding_count=len(body.findings),
    )

    _log.info("explain.request.received")

    try:
        state = await request.app.state.graph.ainvoke(
            {
                "request": body,
                "explained": [],
                "summary": None,
                "error": None,
                "report": None,
            }
        )
        report: TopologyReport = state["report"]
    except Exception as exc:
        # Absolute last-resort catch — the graph itself guarantees no-raise,
        # but defensive belt-and-suspenders here.
        _log.error("explain.graph.unexpected_error", error=str(exc))
        from datetime import datetime, timezone
        report = TopologyReport(
            subscription_id=body.subscription_id,
            analyzed_at=datetime.now(tz=timezone.utc),
            findings=[],
            summary=None,
            high_critical_count=0,
            rag_grounded_pct=0.0,
            error=f"Internal error: {exc}",
        )

    _log.info(
        "explain.request.complete",
        high_critical=report.high_critical_count,
        rag_grounded_pct=report.rag_grounded_pct,
        has_error=report.error is not None,
    )

    return report


@app.get("/health")
async def health() -> dict:
    return {"status": "ok", "version": "1.0.0"}
