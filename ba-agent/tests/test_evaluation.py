from __future__ import annotations

from ba_agent.evaluation import run_eval


def test_gts_standup_seed_eval_passes() -> None:
    result = run_eval("GTS-STANDUP")

    assert result.passed is True
    assert result.total_cases >= 1
    assert result.failed_cases == []


def test_gts_router_seed_eval_passes() -> None:
    result = run_eval("GTS-ROUTER")

    assert result.passed is True
    assert result.metrics["phase_separation_violations"] == 0


def test_gts_gate_seed_eval_passes() -> None:
    result = run_eval("GTS-GATE")

    assert result.passed is True
    assert result.metrics["approval_gate_bypass_count"] == 0


def test_mvp_capability_seed_evals_pass() -> None:
    for eval_set in ("GTS-PLANNING", "GTS-RETRO", "GTS-HEALTH", "GTS-MVP"):
        result = run_eval(eval_set)

        assert result.passed is True


def test_gts_p2_req_seed_eval_passes() -> None:
    result = run_eval("GTS-P2-REQ")

    assert result.passed is True
    assert result.total_cases == 8
    assert result.metrics["phase2_executable_fixture_count"] == 8
    assert result.metrics["approval_gate_bypass_count"] == 0
    assert result.metrics["phase_separation_violations"] == 0
