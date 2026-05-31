"""Graph state."""

from __future__ import annotations

from typing import Any, TypedDict


class BriefState(TypedDict, total=False):
    board_id: str
    sprint: dict[str, Any]
    issues: list[dict[str, Any]]
    brief_markdown: str
    recommendation_id: int
    approved: bool
    delivery_status: str
