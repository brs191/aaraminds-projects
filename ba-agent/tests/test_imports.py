from __future__ import annotations

from ba_agent import __version__
from ba_agent.orchestrator import create_initial_graph_state


def test_package_imports() -> None:
    assert __version__ == "0.1.0"


def test_initial_graph_state_has_trace_and_version() -> None:
    state = create_initial_graph_state("trace-fixed")

    assert state.trace_id == "trace-fixed"
    assert state.graph_version == "phase2-synthetic-standup"
