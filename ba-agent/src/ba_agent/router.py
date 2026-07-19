from __future__ import annotations

from ba_agent.models import Route, RouteDecision

PHASE2_TERMS = (
    "brd",
    "frd",
    "prd",
    "requirement discovery",
    "user story",
    "acceptance criteria",
    "process map",
    "gap analysis",
    "impact analysis",
    "traceability",
    "test scenario",
)

WRITE_INTENT_TERMS = (
    "approve",
    "approval",
    "publish",
    "update sprint",
    "commit sprint",
    "post to all channels",
)


def route_prompt(prompt: str) -> RouteDecision:
    normalized = prompt.lower()
    has_standup_intent = "standup" in normalized or "yesterday" in normalized or "team finish" in normalized
    if has_standup_intent and any(term in normalized for term in WRITE_INTENT_TERMS):
        return RouteDecision(
            route=Route.STANDUP,
            reason="Standup request contains write/approval intent; execute only advisory standup path and block the write intent.",
            blocked=True,
        )
    if any(term in normalized for term in PHASE2_TERMS):
        return RouteDecision(
            route=Route.PHASE2_BLOCKED,
            reason="Phase 2 Enterprise BA capability requested before G7.",
            blocked=True,
        )
    if has_standup_intent:
        return RouteDecision(route=Route.STANDUP, reason="Prompt asks for standup-style sprint status.")
    if "planning" in normalized or "plan next sprint" in normalized:
        return RouteDecision(
            route=Route.PLANNING_PLACEHOLDER,
            reason="Planning is an MVP capability placeholder outside the Phase 2 standup thin slice.",
            blocked=True,
        )
    if "retro" in normalized or "retrospective" in normalized:
        return RouteDecision(
            route=Route.RETRO_PLACEHOLDER,
            reason="Retrospective is an MVP capability placeholder outside the Phase 2 standup thin slice.",
            blocked=True,
        )
    if "health" in normalized or "risk" in normalized:
        return RouteDecision(
            route=Route.HEALTH_PLACEHOLDER,
            reason="Health monitoring is an MVP capability placeholder outside the Phase 2 standup thin slice.",
            blocked=True,
        )
    return RouteDecision(route=Route.UNSUPPORTED, reason="Prompt is outside current synthetic standup scope.", blocked=True)
