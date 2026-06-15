"""LangGraph orchestrator for the explanation pipeline.

Graph topology
--------------
START
  └─▶ enrich_findings   (fan-out: asyncio.gather per finding;
  │                       semaphore caps concurrent LLM calls at 5)
  └─▶ synthesise_summary (single LLM call over all explained findings)
  └─▶ assemble_report   (builds TopologyReport; no I/O)
  └─▶ END

State schema
------------
The TypedDict carries the minimal mutable state between nodes.
`report` is added beyond the spec's four fields so assemble_report can
store the finished TopologyReport in the graph's return value — the app
just reads `state["report"]`.

Dependency injection
--------------------
build_explainer_graph(llm, rag) captures the two clients in closures,
making the graph testable with mock clients.
"""

from __future__ import annotations

import asyncio
from datetime import datetime, timezone
from typing import Optional, TypedDict

import structlog

from explainer.llm import LLMClient
from explainer.models import (
    ExplainRequest,
    ExplainedFinding,
    FindingInput,
    TopologyReport,
)
from explainer.rag import RAGClient

log = structlog.get_logger(__name__)

_MAX_CONCURRENT_LLM = 5

# ---------------------------------------------------------------------------
# LangGraph state schema
# ---------------------------------------------------------------------------


class ExplainerState(TypedDict):
    request: ExplainRequest
    explained: list[ExplainedFinding]
    summary: Optional[str]
    error: Optional[str]
    report: Optional[TopologyReport]   # populated by assemble_report node


# ---------------------------------------------------------------------------
# Graph factory
# ---------------------------------------------------------------------------


def build_explainer_graph(llm: LLMClient, rag: RAGClient):
    """Return a compiled LangGraph that uses the provided clients.

    Clients are captured in closure so the graph is fully self-contained and
    trivially swappable with mocks in tests.
    """
    from langgraph.graph import END, START, StateGraph  # type: ignore[import]

    # ---------------------------------------------------------------
    # Node: enrich_findings
    # ---------------------------------------------------------------

    async def enrich_findings(state: ExplainerState) -> dict:
        """For each finding run RAG + LLM in parallel; honour semaphore."""
        semaphore = asyncio.Semaphore(_MAX_CONCURRENT_LLM)
        errors: list[str] = []

        async def _process_one(finding: FindingInput) -> ExplainedFinding:
            try:
                from opentelemetry import trace as otel_trace

                tracer = otel_trace.get_tracer(__name__)
                with tracer.start_as_current_span("explain.finding"):
                    return await _enrich(finding, semaphore)
            except Exception:
                # OTel might not be configured; run without span
                return await _enrich(finding, semaphore)

        async def _enrich(
            finding: FindingInput, sem: asyncio.Semaphore
        ) -> ExplainedFinding:
            """Call RAG and LLM concurrently; LLM waits for semaphore slot.

            Uses return_exceptions=True so a LLM failure does NOT discard a
            successful RAG result — the two I/O paths are independent.
            """

            async def _llm_guarded() -> str | None:
                async with sem:
                    return await llm.explain(finding)

            rec_text: str | None = None
            rag_source: str | None = None
            rag_grounded: bool = False
            explanation: str | None = None

            rag_result, llm_result = await asyncio.gather(
                rag.search(finding),
                _llm_guarded(),
                return_exceptions=True,
            )

            if isinstance(rag_result, BaseException):
                log.warning(
                    "enrich_finding.rag_error",
                    finding_type=finding.type,
                    error=str(rag_result),
                )
                errors.append(f"RAG: {rag_result}")
            else:
                rec_text, rag_source, rag_grounded = rag_result

            if isinstance(llm_result, BaseException):
                log.warning(
                    "enrich_finding.llm_error",
                    finding_type=finding.type,
                    error=str(llm_result),
                )
                errors.append(f"LLM: {llm_result}")
            else:
                explanation = llm_result

            return ExplainedFinding(
                **finding.model_dump(),
                explanation=explanation,
                recommendation=rec_text,
                rag_grounded=rag_grounded,
                rag_source=rag_source,
            )

        explained = await asyncio.gather(
            *[_process_one(f) for f in state["request"].findings]
        )

        return {
            "explained": list(explained),
            "error": "; ".join(errors) if errors else state.get("error"),
        }

    # ---------------------------------------------------------------
    # Node: synthesise_summary
    # ---------------------------------------------------------------

    async def synthesise_summary(state: ExplainerState) -> dict:
        """Single LLM call to produce a 2-sentence report-level summary."""
        if not state["explained"]:
            return {"summary": None}

        payload = [
            {
                "type": f.type,
                "severity": f.severity,
                "explanation": f.explanation,
            }
            for f in state["explained"]
        ]

        try:
            from opentelemetry import trace as otel_trace

            with otel_trace.get_tracer(__name__).start_as_current_span(
                "explain.summary"
            ):
                summary = await llm.summarise(payload)
        except Exception:
            summary = await llm.summarise(payload)

        return {"summary": summary}

    # ---------------------------------------------------------------
    # Node: assemble_report
    # ---------------------------------------------------------------

    def assemble_report(state: ExplainerState) -> dict:
        """Build the final TopologyReport from accumulated state.  No I/O."""
        findings = state["explained"]
        n = len(findings)
        high_critical = sum(
            1 for f in findings if f.severity in ("Critical", "High")
        )
        grounded = sum(1 for f in findings if f.rag_grounded)
        rag_pct = round(grounded / n, 4) if n > 0 else 0.0

        report = TopologyReport(
            subscription_id=state["request"].subscription_id,
            analyzed_at=datetime.now(tz=timezone.utc),
            findings=findings,
            summary=state.get("summary"),
            high_critical_count=high_critical,
            rag_grounded_pct=rag_pct,
            error=state.get("error"),
        )
        return {"report": report}

    # ---------------------------------------------------------------
    # Wire the graph
    # ---------------------------------------------------------------

    builder = StateGraph(ExplainerState)
    builder.add_node("enrich_findings", enrich_findings)
    builder.add_node("synthesise_summary", synthesise_summary)
    builder.add_node("assemble_report", assemble_report)

    builder.add_edge(START, "enrich_findings")
    builder.add_edge("enrich_findings", "synthesise_summary")
    builder.add_edge("synthesise_summary", "assemble_report")
    builder.add_edge("assemble_report", END)

    return builder.compile()
