"""Runtime configuration, read from environment variables.

Kept dependency-free (stdlib dataclass + os.getenv) so the skeleton has one fewer
moving part. Swap for pydantic-settings if validation needs grow.
"""

from __future__ import annotations

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Config:
    jira_mcp_url: str = os.getenv("JIRA_MCP_URL", "http://localhost:8080/mcp")
    teams_adapter_url: str = os.getenv("TEAMS_ADAPTER_URL", "http://localhost:8090")
    database_url: str = os.getenv(
        "DATABASE_URL", "postgresql://scrum:scrum@localhost:5432/scrum"
    )
    board_id: str = os.getenv("BOARD_ID", "1")
    # P0 default true: the demo resumes past the approval gate automatically so
    # `docker compose up` produces an end-to-end post. Set false to require an
    # explicit approval (see main.py) — that is the real human-in-the-loop path.
    auto_approve: bool = os.getenv("AUTO_APPROVE", "true").lower() == "true"
    stale_days: int = int(os.getenv("STALE_DAYS", "3"))


def load() -> Config:
    return Config()
