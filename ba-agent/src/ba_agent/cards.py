from __future__ import annotations

from ba_agent.models import AdaptiveCardPayload, StandupSummary


def build_adaptive_card(summary: StandupSummary) -> AdaptiveCardPayload:
    body: list[dict[str, str]] = [
        {"type": "TextBlock", "text": f"Status snapshot: {summary.status_snapshot}"},
        {"type": "TextBlock", "text": _items_text("Completed", [_item_text(item.label, item.evidence_ref) for item in summary.completed_items])},
        {"type": "TextBlock", "text": _items_text("In progress", [_item_text(item.label, item.evidence_ref) for item in summary.in_progress_items])},
        {"type": "TextBlock", "text": _items_text("Blocked", [_item_text(item.label, item.evidence_ref) for item in summary.blocked_items])},
        {
            "type": "TextBlock",
            "text": _items_text("Risks", [_item_text(f"{risk.label} ({risk.rationale})", risk.evidence_ref) for risk in summary.risks]),
        },
        {"type": "TextBlock", "text": _items_text("Git activity", [_item_text(item.label, item.evidence_ref) for item in summary.git_activity])},
        {"type": "TextBlock", "text": _items_text("Data quality", summary.data_quality)},
        {"type": "TextBlock", "text": _items_text("Assumptions", summary.assumptions)},
        {"type": "TextBlock", "text": _items_text("Open questions", summary.open_questions)},
        {
            "type": "TextBlock",
            "text": (
                f"trace_id={summary.trace_id}; graph_version={summary.graph_version}; "
                f"fixture_version={summary.fixture_version}; case_id={summary.case_id}; "
                f"route={summary.route.value}; route_reason={summary.route_reason}"
            ),
        },
    ]
    return AdaptiveCardPayload(
        title=f"Daily standup summary — {summary.case_id}",
        body=body,
        evidence_refs=summary.evidence_refs,
        trace_id=summary.trace_id,
    )


def _items_text(label: str, items: list[str]) -> str:
    if not items:
        return f"{label}: none"
    return f"{label}: " + "; ".join(items)


def _item_text(label: str, evidence_ref: str) -> str:
    return f"{label} [evidence: {evidence_ref}]"


def send_adaptive_card_stub(_payload: AdaptiveCardPayload) -> None:
    raise RuntimeError("Live Teams posting is blocked in the synthetic thin slice")
