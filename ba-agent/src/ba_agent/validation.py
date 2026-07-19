from __future__ import annotations

import json
from pathlib import Path
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field


ValidationStatus = Literal["not_validated", "schema_observed_not_validated_for_execution", "validated", "blocked"]


class McpValidationRow(BaseModel):
    model_config = ConfigDict(frozen=True)

    tool_name: str
    mcp_server_name: str
    environment: str
    permission: str
    approved_scopes: list[str] = Field(default_factory=list)
    implementation_status: str
    validation_status: ValidationStatus
    owner: str
    actual_request_schema_ref: str | None = None
    actual_response_schema_ref: str | None = None
    schema_diff_ref: str | None = None
    auth_model_ref: str | None = None
    rate_limit_ref: str | None = None
    approval_evidence_ref: str | None = None
    validated_at: str | None = None
    open_blockers: list[str] = Field(default_factory=list)

    def can_enable_sandbox_read(self) -> bool:
        return (
            self.validation_status == "validated"
            and self.environment == "sandbox"
            and self.permission == "read"
            and _is_resolved(self.mcp_server_name)
            and _is_resolved(self.owner)
            and bool(self.approved_scopes)
            and all(_is_resolved(scope) for scope in self.approved_scopes)
            and _is_resolved(self.actual_request_schema_ref)
            and _is_resolved(self.actual_response_schema_ref)
            and _is_resolved(self.schema_diff_ref)
            and _is_resolved(self.auth_model_ref)
            and _is_resolved(self.rate_limit_ref)
            and _is_resolved(self.approval_evidence_ref)
            and _is_resolved(self.validated_at)
            and not self.open_blockers
        )


class McpValidationRegister(BaseModel):
    model_config = ConfigDict(frozen=True)

    version: str
    rows: list[McpValidationRow]

    def row_for(self, tool_name: str) -> McpValidationRow:
        for row in self.rows:
            if row.tool_name == tool_name:
                return row
        raise KeyError(f"missing validation row for {tool_name}")

    def can_enable_all(self, tool_names: list[str]) -> bool:
        return all(self.row_for(tool_name).can_enable_sandbox_read() for tool_name in tool_names)

    def summary(self) -> dict[str, object]:
        return {
            "version": self.version,
            "validated": [row.tool_name for row in self.rows if row.validation_status == "validated"],
            "blocked": [row.tool_name for row in self.rows if not row.can_enable_sandbox_read()],
        }


def load_validation_register(path: Path) -> McpValidationRegister:
    return McpValidationRegister.model_validate_json(path.read_text(encoding="utf-8"))


def validation_summary_json(path: Path) -> str:
    register = load_validation_register(path)
    return json.dumps(register.summary(), indent=2)


def _is_resolved(value: str | None) -> bool:
    if value is None:
        return False
    stripped = value.strip()
    return bool(stripped) and stripped != "[RAJA]"
