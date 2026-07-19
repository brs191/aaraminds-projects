"""Phase 2 read-only sandbox MCP wrappers.

These wrappers enforce the BA Agent allowlist before any upstream MCP tool can
be called. They do not authorize sandbox execution by themselves; the validation
register must be fully validated before an adapter can be constructed.
"""
from __future__ import annotations

from collections.abc import Mapping
from typing import Protocol

from ba_agent.config import RuntimeSettings
from ba_agent.gateway import evaluate_phase2_sandbox_upstream_tool
from ba_agent.models import ToolStatus
from ba_agent.validation import McpValidationRegister


class Phase2UpstreamMcpClient(Protocol):
    def call_tool(self, tool_name: str, arguments: Mapping[str, object]) -> Mapping[str, object]:
        """Call an upstream MCP tool and return its structured response."""


class Phase2SandboxBlockedError(RuntimeError):
    """Raised when Phase 2 sandbox execution remains blocked."""


class Phase2JiraReadOnlyMcpAdapter:
    """Read-only Jira adapter over the approved Phase 2 MCP allowlist."""

    REGISTER_TOOL_NAME = "get_sprint_status"
    CANDIDATE_ID = "P2-SBX-JIRA-READ"

    def __init__(
        self,
        settings: RuntimeSettings,
        register: McpValidationRegister,
        upstream_client: Phase2UpstreamMcpClient,
    ) -> None:
        if settings.data_source_mode != "sandbox_read":
            raise Phase2SandboxBlockedError("Phase 2 Jira MCP adapter requires BA_AGENT_DATA_SOURCE_MODE=sandbox_read")
        if settings.live_integrations_enabled:
            raise Phase2SandboxBlockedError("Phase 2 Jira MCP adapter rejects LIVE_INTEGRATIONS_ENABLED=true")
        register_row = register.row_for(self.REGISTER_TOOL_NAME)
        if not register_row.can_enable_sandbox_read():
            raise Phase2SandboxBlockedError(
                "Phase 2 Jira MCP adapter requires a fully validated get_sprint_status register row"
            )
        self._upstream_client = upstream_client

    def fetch_requirement_issue_metadata(self, arguments: Mapping[str, object]) -> Mapping[str, object]:
        return self._call_allowed("FetchItrackJiraIssuesList", arguments)

    def get_job_status(self, job_id: str) -> Mapping[str, object]:
        return self._call_allowed("GetJiraItrackJobStatus", {"job_id": job_id})

    def validate_issue_mapping(self, itrack_issue_key: str, jira_issue_key: str | None = None) -> Mapping[str, object]:
        arguments: dict[str, object] = {"itrack_issue_key": itrack_issue_key}
        if jira_issue_key is not None:
            arguments["jira_issue_key"] = jira_issue_key
        return self._call_allowed("JiraItrackValidate", arguments)

    def call_upstream_tool_for_test(self, tool_name: str, arguments: Mapping[str, object]) -> Mapping[str, object]:
        """Test seam proving advertised but disallowed upstream tools cannot pass."""
        return self._call_allowed(tool_name, arguments)

    def _call_allowed(self, tool_name: str, arguments: Mapping[str, object]) -> Mapping[str, object]:
        decision = evaluate_phase2_sandbox_upstream_tool(self.CANDIDATE_ID, tool_name)
        if decision.status != ToolStatus.OK:
            raise Phase2SandboxBlockedError(decision.message)
        return self._upstream_client.call_tool(tool_name, arguments)


class Phase2ConfluenceReadOnlyMcpAdapter:
    """Read-only Confluence adapter over the approved Phase 2 MCP allowlist."""

    REGISTER_TOOL_NAME = "get_confluence_source_pages"
    CANDIDATE_ID = "P2-SBX-CONF-READ"

    def __init__(
        self,
        settings: RuntimeSettings,
        register: McpValidationRegister,
        upstream_client: Phase2UpstreamMcpClient,
    ) -> None:
        if settings.data_source_mode != "sandbox_read":
            raise Phase2SandboxBlockedError(
                "Phase 2 Confluence MCP adapter requires BA_AGENT_DATA_SOURCE_MODE=sandbox_read"
            )
        if settings.live_integrations_enabled:
            raise Phase2SandboxBlockedError("Phase 2 Confluence MCP adapter rejects LIVE_INTEGRATIONS_ENABLED=true")
        register_row = register.row_for(self.REGISTER_TOOL_NAME)
        if not register_row.can_enable_sandbox_read():
            raise Phase2SandboxBlockedError(
                "Phase 2 Confluence MCP adapter requires a fully validated get_confluence_source_pages register row"
            )
        self._upstream_client = upstream_client

    def search(self, arguments: Mapping[str, object]) -> Mapping[str, object]:
        return self._call_allowed("confluence_search", arguments)

    def get_page(self, page_id: str) -> Mapping[str, object]:
        return self._call_allowed("confluence_get_page", {"page_id": page_id})

    def list_spaces(self, arguments: Mapping[str, object] | None = None) -> Mapping[str, object]:
        return self._call_allowed("confluence_list_spaces", arguments or {})

    def list_space_pages(self, space_key: str) -> Mapping[str, object]:
        return self._call_allowed("confluence_space_pages", {"space_key": space_key})

    def list_page_children(self, page_id: str) -> Mapping[str, object]:
        return self._call_allowed("confluence_page_children", {"page_id": page_id})

    def list_page_attachments(self, page_id: str) -> Mapping[str, object]:
        return self._call_allowed("confluence_page_attachments", {"page_id": page_id})

    def list_page_comments(self, page_id: str) -> Mapping[str, object]:
        return self._call_allowed("confluence_page_comments", {"page_id": page_id})

    def call_upstream_tool_for_test(self, tool_name: str, arguments: Mapping[str, object]) -> Mapping[str, object]:
        """Test seam proving advertised but disallowed upstream tools cannot pass."""
        return self._call_allowed(tool_name, arguments)

    def _call_allowed(self, tool_name: str, arguments: Mapping[str, object]) -> Mapping[str, object]:
        decision = evaluate_phase2_sandbox_upstream_tool(self.CANDIDATE_ID, tool_name)
        if decision.status != ToolStatus.OK:
            raise Phase2SandboxBlockedError(decision.message)
        return self._upstream_client.call_tool(tool_name, arguments)
