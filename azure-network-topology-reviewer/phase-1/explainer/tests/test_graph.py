"""Graph integration tests — all I/O mocked.

Test matrix
-----------
- TestHappyPath       : 2 findings → full TopologyReport with explanations + summary.
- TestLLMFailure      : LLM raises → explanation=None, report.error set, no exception.
- TestRAGNoMatch      : RAG returns empty → recommendation=None, rag_grounded=False.
- TestConcurrencyBound: 10 findings → ≤5 LLM calls concurrent (semaphore respected).

Dependencies: pytest-asyncio (asyncio_mode="auto" in pyproject.toml).
"""

from __future__ import annotations

import asyncio
from unittest.mock import AsyncMock, MagicMock

import pytest

from explainer.graph import build_explainer_graph
from explainer.models import ExplainRequest, FindingInput, TopologyReport

# ---------------------------------------------------------------------------
# Fixtures / helpers
# ---------------------------------------------------------------------------


def _make_finding(
    idx: int = 0,
    severity: str = "High",
    reachable: bool = True,
) -> FindingInput:
    return FindingInput(
        type=f"INTERNET_EXPOSURE_{idx}",
        severity=severity,
        resource=f"vnet-{idx}",
        evidence=f"evidence-{idx}",
        reachable=reachable,
    )


def _make_request(n: int, severity: str = "High") -> ExplainRequest:
    return ExplainRequest(
        subscription_id="sub-test-001",
        findings=[_make_finding(i, severity=severity) for i in range(n)],
    )


def _initial_state(request: ExplainRequest) -> dict:
    return {
        "request": request,
        "explained": [],
        "summary": None,
        "error": None,
        "report": None,
    }


def _stub_llm(explanation: str = "Stub explanation.") -> MagicMock:
    """Mock LLMClient with explain() and summarise() returning fixed strings."""
    mock = MagicMock()
    mock.explain = AsyncMock(return_value=explanation)
    mock.summarise = AsyncMock(return_value="Stub two-sentence summary.")
    return mock


def _stub_rag(
    recommendation: str | None = "Per AT&T §1: Fix it. Source: doc.",
    title: str | None = "AT&T Standard",
    grounded: bool = True,
) -> MagicMock:
    """Mock RAGClient with search() returning a fixed tuple."""
    mock = MagicMock()
    mock.search = AsyncMock(return_value=(recommendation, title, grounded))
    return mock


# ---------------------------------------------------------------------------
# TestHappyPath
# ---------------------------------------------------------------------------


class TestHappyPath:
    async def test_two_findings_produce_report(self):
        llm = _stub_llm()
        rag = _stub_rag()
        graph = build_explainer_graph(llm, rag)

        state = await graph.ainvoke(_initial_state(_make_request(2)))

        report: TopologyReport = state["report"]
        assert isinstance(report, TopologyReport)
        assert len(report.findings) == 2

    async def test_explanations_populated(self):
        llm = _stub_llm("Some explanation.")
        rag = _stub_rag()
        graph = build_explainer_graph(llm, rag)

        state = await graph.ainvoke(_initial_state(_make_request(2)))
        for f in state["report"].findings:
            assert f.explanation == "Some explanation."

    async def test_summary_populated(self):
        llm = _stub_llm()
        rag = _stub_rag()
        graph = build_explainer_graph(llm, rag)

        state = await graph.ainvoke(_initial_state(_make_request(2)))
        assert state["report"].summary == "Stub two-sentence summary."

    async def test_rag_fields_populated(self):
        llm = _stub_llm()
        rag = _stub_rag(
            recommendation="Per AT&T §2: Restrict. Source: NSG Guide.",
            title="NSG Guide",
            grounded=True,
        )
        graph = build_explainer_graph(llm, rag)

        state = await graph.ainvoke(_initial_state(_make_request(2)))
        for f in state["report"].findings:
            assert f.rag_grounded is True
            assert f.rag_source == "NSG Guide"
            assert "NSG Guide" in f.recommendation

    async def test_high_critical_count_correct(self):
        llm = _stub_llm()
        rag = _stub_rag()
        graph = build_explainer_graph(llm, rag)

        req = ExplainRequest(
            subscription_id="sub-test",
            findings=[
                _make_finding(0, severity="Critical"),
                _make_finding(1, severity="High"),
                _make_finding(2, severity="Medium"),
            ],
        )
        state = await graph.ainvoke(_initial_state(req))
        assert state["report"].high_critical_count == 2

    async def test_no_error_on_happy_path(self):
        llm = _stub_llm()
        rag = _stub_rag()
        graph = build_explainer_graph(llm, rag)

        state = await graph.ainvoke(_initial_state(_make_request(2)))
        assert state["report"].error is None


# ---------------------------------------------------------------------------
# TestLLMFailure
# ---------------------------------------------------------------------------


class TestLLMFailure:
    async def test_explanation_is_none_on_llm_error(self):
        llm = MagicMock()
        llm.explain = AsyncMock(side_effect=Exception("LLM timeout"))
        llm.summarise = AsyncMock(return_value=None)
        rag = _stub_rag()
        graph = build_explainer_graph(llm, rag)

        state = await graph.ainvoke(_initial_state(_make_request(2)))
        for f in state["report"].findings:
            assert f.explanation is None

    async def test_no_exception_propagated_on_llm_error(self):
        """The graph must never raise; failures go into report.error."""
        llm = MagicMock()
        llm.explain = AsyncMock(side_effect=RuntimeError("broken"))
        llm.summarise = AsyncMock(return_value=None)
        rag = _stub_rag()
        graph = build_explainer_graph(llm, rag)

        # Should not raise
        state = await graph.ainvoke(_initial_state(_make_request(1)))
        assert state["report"] is not None

    async def test_error_field_set_on_llm_failure(self):
        llm = MagicMock()
        llm.explain = AsyncMock(side_effect=Exception("429 rate limited"))
        llm.summarise = AsyncMock(return_value=None)
        rag = _stub_rag()
        graph = build_explainer_graph(llm, rag)

        state = await graph.ainvoke(_initial_state(_make_request(1)))
        assert state["report"].error is not None
        assert "429" in state["report"].error

    async def test_rag_still_works_when_llm_fails(self):
        """RAG runs independently; its results should survive LLM failure."""
        llm = MagicMock()
        llm.explain = AsyncMock(side_effect=Exception("LLM down"))
        llm.summarise = AsyncMock(return_value=None)
        rag = _stub_rag(grounded=True)
        graph = build_explainer_graph(llm, rag)

        state = await graph.ainvoke(_initial_state(_make_request(1)))
        f = state["report"].findings[0]
        assert f.rag_grounded is True


# ---------------------------------------------------------------------------
# TestRAGNoMatch
# ---------------------------------------------------------------------------


class TestRAGNoMatch:
    async def test_recommendation_none_on_empty_rag(self):
        llm = _stub_llm()
        rag = _stub_rag(recommendation=None, title=None, grounded=False)
        graph = build_explainer_graph(llm, rag)

        state = await graph.ainvoke(_initial_state(_make_request(2)))
        for f in state["report"].findings:
            assert f.recommendation is None
            assert f.rag_grounded is False
            assert f.rag_source is None

    async def test_rag_grounded_false_on_empty(self):
        llm = _stub_llm()
        rag = _stub_rag(recommendation=None, title=None, grounded=False)
        graph = build_explainer_graph(llm, rag)

        state = await graph.ainvoke(_initial_state(_make_request(1)))
        assert state["report"].rag_grounded_pct == 0.0

    async def test_rag_failure_does_not_block_explanation(self):
        """LLM explanation should still be populated even when RAG raises."""
        llm = _stub_llm("Good explanation despite RAG failure.")
        rag = MagicMock()
        rag.search = AsyncMock(side_effect=Exception("search timeout"))
        graph = build_explainer_graph(llm, rag)

        state = await graph.ainvoke(_initial_state(_make_request(1)))
        # return_exceptions=True: LLM explanation survives RAG failure
        assert state["report"] is not None
        f = state["report"].findings[0]
        assert f.explanation == "Good explanation despite RAG failure."
        assert f.rag_grounded is False


# ---------------------------------------------------------------------------
# TestConcurrencyBound
# ---------------------------------------------------------------------------


class TestConcurrencyBound:
    async def test_max_5_concurrent_llm_calls(self):
        """With 10 findings and semaphore=5, peak concurrency ≤ 5."""
        max_concurrent: list[int] = [0]
        current_concurrent: list[int] = [0]
        counter_lock = asyncio.Lock()

        async def _tracked_explain(finding: FindingInput) -> str:
            async with counter_lock:
                current_concurrent[0] += 1
                if current_concurrent[0] > max_concurrent[0]:
                    max_concurrent[0] = current_concurrent[0]
            # Simulate non-trivial I/O so tasks truly overlap
            await asyncio.sleep(0.05)
            async with counter_lock:
                current_concurrent[0] -= 1
            return "explanation"

        llm = MagicMock()
        llm.explain = _tracked_explain
        llm.summarise = AsyncMock(return_value="Summary.")
        rag = _stub_rag()
        graph = build_explainer_graph(llm, rag)

        await graph.ainvoke(_initial_state(_make_request(10)))

        assert max_concurrent[0] <= 5, (
            f"Expected ≤5 concurrent LLM calls, observed {max_concurrent[0]}"
        )

    async def test_all_findings_processed_with_semaphore(self):
        """Semaphore must not starve findings — all 10 must be processed."""
        llm = MagicMock()
        llm.explain = AsyncMock(return_value="ok")
        llm.summarise = AsyncMock(return_value="Summary.")
        rag = _stub_rag()
        graph = build_explainer_graph(llm, rag)

        state = await graph.ainvoke(_initial_state(_make_request(10)))
        assert len(state["report"].findings) == 10
