from __future__ import annotations

from collections.abc import Mapping
import json
from pathlib import Path

import pytest

from ba_agent.config import RuntimeSettings
from ba_agent.phase2.sandbox_mcp import (
    Phase2ConfluenceReadOnlyMcpAdapter,
    Phase2JiraReadOnlyMcpAdapter,
    Phase2SandboxBlockedError,
)
from ba_agent.validation import McpValidationRegister, load_validation_register


class FakeMcpClient:
    def __init__(self) -> None:
        self.calls: list[tuple[str, Mapping[str, object]]] = []

    def call_tool(self, tool_name: str, arguments: Mapping[str, object]) -> Mapping[str, object]:
        self.calls.append((tool_name, arguments))
        return {"tool_name": tool_name, "arguments": dict(arguments), "status": "ok"}


def test_phase2_jira_adapter_blocks_current_partial_register_before_any_call() -> None:
    client = FakeMcpClient()
    settings = RuntimeSettings(data_source_mode="sandbox_read")
    register = load_validation_register(Path("docs/development/mcp-validation-register.json"))

    with pytest.raises(Phase2SandboxBlockedError, match="fully validated"):
        Phase2JiraReadOnlyMcpAdapter(settings, register, client)

    assert client.calls == []


def test_phase2_jira_adapter_maps_requirement_metadata_to_allowed_fetch_tool(tmp_path: Path) -> None:
    client = FakeMcpClient()
    adapter = Phase2JiraReadOnlyMcpAdapter(
        RuntimeSettings(data_source_mode="sandbox_read"),
        _validated_jira_register(tmp_path),
        client,
    )

    response = adapter.fetch_requirement_issue_metadata({"jira_project_key": "SYN", "timeframe": "today"})

    assert response["tool_name"] == "FetchItrackJiraIssuesList"
    assert client.calls == [("FetchItrackJiraIssuesList", {"jira_project_key": "SYN", "timeframe": "today"})]


def test_phase2_jira_adapter_maps_job_status_and_validation_to_allowed_tools(tmp_path: Path) -> None:
    client = FakeMcpClient()
    adapter = Phase2JiraReadOnlyMcpAdapter(
        RuntimeSettings(data_source_mode="sandbox_read"),
        _validated_jira_register(tmp_path),
        client,
    )

    adapter.get_job_status("job-1")
    adapter.validate_issue_mapping("ITRACK-1", "JIRA-1")

    assert client.calls == [
        ("GetJiraItrackJobStatus", {"job_id": "job-1"}),
        ("JiraItrackValidate", {"itrack_issue_key": "ITRACK-1", "jira_issue_key": "JIRA-1"}),
    ]


@pytest.mark.parametrize(
    "tool_name",
    ["CreateJiraCloudIssue", "UpdateJiraCloudIssue", "UpdateJiraCloudStatus", "DeleteJiraCloudIssue", "RevertJiraItrackIssue"],
)
def test_phase2_jira_adapter_blocks_disallowed_advertised_write_tools(tmp_path: Path, tool_name: str) -> None:
    client = FakeMcpClient()
    adapter = Phase2JiraReadOnlyMcpAdapter(
        RuntimeSettings(data_source_mode="sandbox_read"),
        _validated_jira_register(tmp_path),
        client,
    )

    with pytest.raises(Phase2SandboxBlockedError, match="write-like or destructive"):
        adapter.call_upstream_tool_for_test(tool_name, {})

    assert client.calls == []


def test_phase2_jira_adapter_requires_sandbox_read_mode(tmp_path: Path) -> None:
    with pytest.raises(Phase2SandboxBlockedError, match="sandbox_read"):
        Phase2JiraReadOnlyMcpAdapter(
            RuntimeSettings(data_source_mode="synthetic"),
            _validated_jira_register(tmp_path),
            FakeMcpClient(),
        )


def test_phase2_confluence_adapter_blocks_current_partial_register_before_any_call() -> None:
    client = FakeMcpClient()
    settings = RuntimeSettings(data_source_mode="sandbox_read")
    register = load_validation_register(Path("docs/development/mcp-validation-register.json"))

    with pytest.raises(Phase2SandboxBlockedError, match="fully validated"):
        Phase2ConfluenceReadOnlyMcpAdapter(settings, register, client)

    assert client.calls == []


def test_phase2_confluence_adapter_maps_reads_to_allowed_tools(tmp_path: Path) -> None:
    client = FakeMcpClient()
    adapter = Phase2ConfluenceReadOnlyMcpAdapter(
        RuntimeSettings(data_source_mode="sandbox_read"),
        _validated_confluence_register(tmp_path),
        client,
    )

    adapter.search({"cql": "space = SYN"})
    adapter.get_page("page-1")
    adapter.list_spaces()
    adapter.list_space_pages("SYN")
    adapter.list_page_children("page-1")
    adapter.list_page_attachments("page-1")
    adapter.list_page_comments("page-1")

    assert client.calls == [
        ("confluence_search", {"cql": "space = SYN"}),
        ("confluence_get_page", {"page_id": "page-1"}),
        ("confluence_list_spaces", {}),
        ("confluence_space_pages", {"space_key": "SYN"}),
        ("confluence_page_children", {"page_id": "page-1"}),
        ("confluence_page_attachments", {"page_id": "page-1"}),
        ("confluence_page_comments", {"page_id": "page-1"}),
    ]


@pytest.mark.parametrize(
    "tool_name",
    ["confluence_create_page", "confluence_update_page", "confluence_delete_page", "confluence_add_comment"],
)
def test_phase2_confluence_adapter_blocks_disallowed_write_like_tools(tmp_path: Path, tool_name: str) -> None:
    client = FakeMcpClient()
    adapter = Phase2ConfluenceReadOnlyMcpAdapter(
        RuntimeSettings(data_source_mode="sandbox_read"),
        _validated_confluence_register(tmp_path),
        client,
    )

    with pytest.raises(Phase2SandboxBlockedError, match="write-like or destructive"):
        adapter.call_upstream_tool_for_test(tool_name, {})

    assert client.calls == []


def test_phase2_confluence_adapter_blocks_unlisted_root_tools(tmp_path: Path) -> None:
    client = FakeMcpClient()
    adapter = Phase2ConfluenceReadOnlyMcpAdapter(
        RuntimeSettings(data_source_mode="sandbox_read"),
        _validated_confluence_register(tmp_path),
        client,
    )

    with pytest.raises(Phase2SandboxBlockedError, match="not on the Phase 2 read-only allowlist"):
        adapter.call_upstream_tool_for_test("grafana_search_dashboards", {})

    assert client.calls == []


def test_phase2_confluence_adapter_requires_sandbox_read_mode(tmp_path: Path) -> None:
    with pytest.raises(Phase2SandboxBlockedError, match="sandbox_read"):
        Phase2ConfluenceReadOnlyMcpAdapter(
            RuntimeSettings(data_source_mode="synthetic"),
            _validated_confluence_register(tmp_path),
            FakeMcpClient(),
        )


def _validated_jira_register(tmp_path: Path) -> McpValidationRegister:
    path = tmp_path / "register.json"
    path.write_text(
        json.dumps(
            {
                "version": "test",
                "rows": [
                    {
                        "tool_name": "get_sprint_status",
                        "mcp_server_name": "apm0045942-cc-mcp-server:/jira-cloud/mcp",
                        "environment": "sandbox",
                        "permission": "read",
                        "approved_scopes": ["SYNTHETIC-PROJECT"],
                        "implementation_status": "ready",
                        "validation_status": "validated",
                        "owner": "RAJA acting owner",
                        "actual_request_schema_ref": "captured-tools-list",
                        "actual_response_schema_ref": "captured-output-schema",
                        "schema_diff_ref": "phase-2-sandbox-authorization-package.md",
                        "auth_model_ref": "approved-auth-model",
                        "rate_limit_ref": "approved-rate-limit",
                        "approval_evidence_ref": "approval:RAJA",
                        "validated_at": "2026-07-10T10:30:00Z",
                        "open_blockers": [],
                    }
                ],
            }
        ),
        encoding="utf-8",
    )
    return load_validation_register(path)


def _validated_confluence_register(tmp_path: Path) -> McpValidationRegister:
    path = tmp_path / "register.json"
    path.write_text(
        json.dumps(
            {
                "version": "test",
                "rows": [
                    {
                        "tool_name": "get_confluence_source_pages",
                        "mcp_server_name": "apm0045942-cc-mcp-server:/",
                        "environment": "sandbox",
                        "permission": "read",
                        "approved_scopes": ["SYNTHETIC-SPACE"],
                        "implementation_status": "ready",
                        "validation_status": "validated",
                        "owner": "RAJA acting owner",
                        "actual_request_schema_ref": "captured-confluence-tools-list",
                        "actual_response_schema_ref": "captured-confluence-output-schema",
                        "schema_diff_ref": "phase-2-confluence-sandbox-evidence-package.md",
                        "auth_model_ref": "approved-auth-model",
                        "rate_limit_ref": "approved-rate-limit",
                        "approval_evidence_ref": "approval:RAJA",
                        "validated_at": "2026-07-13T07:45:00Z",
                        "open_blockers": [],
                    }
                ],
            }
        ),
        encoding="utf-8",
    )
    return load_validation_register(path)
