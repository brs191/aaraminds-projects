from __future__ import annotations

from pathlib import Path
import json

import pytest

from ba_agent.adapters import SandboxReadBlockedError, build_standup_read_adapter
from ba_agent.config import RuntimeSettings
from ba_agent.validation import load_validation_register


REGISTER = Path("docs/development/mcp-validation-register.json")


def test_validation_register_loads_and_blocks_unvalidated_tools() -> None:
    register = load_validation_register(REGISTER)

    assert register.version == "phase2-sandbox-auth-v0.5"
    assert register.can_enable_all(["get_sprint_status", "get_recent_activity"]) is False


def test_synthetic_adapter_is_default() -> None:
    adapter = build_standup_read_adapter(RuntimeSettings.from_env({}))

    assert adapter.get_standup_case("STD-001").case_id == "STD-001"


def test_sandbox_adapter_fails_closed_without_validated_register() -> None:
    settings = RuntimeSettings(environment="local", data_source_mode="sandbox_read")

    with pytest.raises(SandboxReadBlockedError, match="validated Jira/Git"):
        build_standup_read_adapter(settings, REGISTER)


def test_incomplete_validated_row_does_not_enable_sandbox_read(tmp_path: Path) -> None:
    register_data = {
        "version": "test",
        "rows": [
            {
                "tool_name": "get_sprint_status",
                "mcp_server_name": "jira-mcp",
                "environment": "sandbox",
                "permission": "read",
                "approved_scopes": ["SYN"],
                "implementation_status": "ready",
                "validation_status": "validated",
                "owner": "jira-owner",
                "open_blockers": []
            },
            {
                "tool_name": "get_recent_activity",
                "mcp_server_name": "git-mcp",
                "environment": "sandbox",
                "permission": "read",
                "approved_scopes": ["synthetic-repo"],
                "implementation_status": "ready",
                "validation_status": "validated",
                "owner": "git-owner",
                "open_blockers": []
            }
        ]
    }
    path = tmp_path / "register.json"
    path.write_text(json.dumps(register_data), encoding="utf-8")
    register = load_validation_register(path)

    assert register.can_enable_all(["get_sprint_status", "get_recent_activity"]) is False


def test_complete_validated_rows_can_enable_sandbox_read(tmp_path: Path) -> None:
    row_common = {
        "environment": "sandbox",
        "permission": "read",
        "implementation_status": "ready",
        "validation_status": "validated",
        "actual_request_schema_ref": "schemas/request.json",
        "actual_response_schema_ref": "schemas/response.json",
        "schema_diff_ref": "schemas/diff.md",
        "auth_model_ref": "docs/auth.md",
        "rate_limit_ref": "docs/rate-limit.md",
        "approval_evidence_ref": "approval:external:RAJA-G4",
        "validated_at": "2026-07-03T00:00:00Z",
        "open_blockers": [],
    }
    register_data = {
        "version": "test",
        "rows": [
            {
                **row_common,
                "tool_name": "get_sprint_status",
                "mcp_server_name": "jira-mcp",
                "approved_scopes": ["SYN"],
                "owner": "jira-owner",
            },
            {
                **row_common,
                "tool_name": "get_recent_activity",
                "mcp_server_name": "git-mcp",
                "approved_scopes": ["synthetic-repo"],
                "owner": "git-owner",
            },
        ],
    }
    path = tmp_path / "register.json"
    path.write_text(json.dumps(register_data), encoding="utf-8")
    register = load_validation_register(path)

    assert register.can_enable_all(["get_sprint_status", "get_recent_activity"]) is True


def test_blank_validated_fields_do_not_enable_sandbox_read(tmp_path: Path) -> None:
    register_data = {
        "version": "test",
        "rows": [
            {
                "tool_name": "get_sprint_status",
                "mcp_server_name": "",
                "environment": "sandbox",
                "permission": "read",
                "approved_scopes": [""],
                "implementation_status": "ready",
                "validation_status": "validated",
                "owner": "",
                "actual_request_schema_ref": "",
                "actual_response_schema_ref": "",
                "schema_diff_ref": "",
                "auth_model_ref": "",
                "rate_limit_ref": "",
                "approval_evidence_ref": "",
                "validated_at": "",
                "open_blockers": [],
            }
        ],
    }
    path = tmp_path / "register.json"
    path.write_text(json.dumps(register_data), encoding="utf-8")
    register = load_validation_register(path)

    assert register.row_for("get_sprint_status").can_enable_sandbox_read() is False
