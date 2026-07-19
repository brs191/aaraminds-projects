from __future__ import annotations

from pathlib import Path
from typing import Protocol

from ba_agent.config import RuntimeSettings
from ba_agent.fixtures import load_fixture_case
from ba_agent.models import StandupFixtureCase
from ba_agent.validation import McpValidationRegister, load_validation_register


class StandupReadAdapter(Protocol):
    def get_standup_case(self, case_id: str) -> StandupFixtureCase:
        """Return standup input data."""


class SyntheticStandupReadAdapter:
    def get_standup_case(self, case_id: str) -> StandupFixtureCase:
        _fixture_set, case = load_fixture_case(case_id)
        return case


class SandboxReadBlockedError(RuntimeError):
    pass


class SandboxReadAdapter:
    REQUIRED_TOOLS = ["get_sprint_status", "get_recent_activity"]

    def __init__(self, register: McpValidationRegister) -> None:
        self.register = register
        if not register.can_enable_all(self.REQUIRED_TOOLS):
            raise SandboxReadBlockedError("sandbox read mode requires validated Jira/Git read tools and approved scopes")

    def get_standup_case(self, _case_id: str) -> StandupFixtureCase:
        raise SandboxReadBlockedError("actual sandbox reads are not implemented in Phase 4")


def build_standup_read_adapter(
    settings: RuntimeSettings,
    register_path: Path | None = None,
) -> StandupReadAdapter:
    if settings.data_source_mode == "synthetic":
        return SyntheticStandupReadAdapter()
    if register_path is None:
        raise SandboxReadBlockedError("sandbox read mode requires a validation register")
    return SandboxReadAdapter(load_validation_register(register_path))
