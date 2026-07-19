from __future__ import annotations

from typing import Protocol
from uuid import uuid4

from ba_agent.cards import build_adaptive_card
from ba_agent.constants import GRAPH_VERSION
from ba_agent.fixtures import load_fixture_case
from ba_agent.models import GraphState
from ba_agent.standup import build_standup_summary


class ModelClient(Protocol):
    def summarize(self, prompt: str) -> str:
        """Return a local summary."""


class OfflineModelClient:
    """Offline fake used before any approved model integration."""

    def summarize(self, prompt: str) -> str:
        return f"offline-placeholder:{len(prompt)}"


def create_initial_graph_state(trace_id: str | None = None) -> GraphState:
    return GraphState(
        trace_id=trace_id or f"trace-{uuid4()}",
        graph_version=GRAPH_VERSION,
    )


def run_synthetic_standup(case_id: str, trace_id: str | None = None) -> str:
    fixture_set, case = load_fixture_case(case_id)
    state = create_initial_graph_state(trace_id)
    summary = build_standup_summary(case, fixture_set.manifest.fixture_version, state.trace_id)
    card = build_adaptive_card(summary)
    return card.model_dump_json(indent=2)
