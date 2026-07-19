from __future__ import annotations

import pytest

from ba_agent.cards import build_adaptive_card, send_adaptive_card_stub
from ba_agent.fixtures import load_fixture_set
from ba_agent.standup import build_standup_summary


def test_adaptive_card_contains_required_sections() -> None:
    fixture_set = load_fixture_set()
    case = fixture_set.get_case("STD-001")
    summary = build_standup_summary(case, fixture_set.manifest.fixture_version, "trace-card")

    card = build_adaptive_card(summary)

    assert card.type == "AdaptiveCard"
    assert card.trace_id == "trace-card"
    text = "\n".join(block["text"] for block in card.body)
    assert "Status snapshot" in text
    assert "Completed" in text
    assert "Assumptions" in text
    assert "Data quality" in text
    assert "trace_id=trace-card" in text
    assert "route=standup" in text
    assert "[evidence: jira:synthetic:SYN/SYN-1]" in text
    assert summary.evidence_refs == card.evidence_refs


def test_adaptive_card_send_stub_fails_closed() -> None:
    fixture_set = load_fixture_set()
    case = fixture_set.get_case("STD-001")
    summary = build_standup_summary(case, fixture_set.manifest.fixture_version, "trace-card")
    card = build_adaptive_card(summary)

    with pytest.raises(RuntimeError, match="blocked"):
        send_adaptive_card_stub(card)
