from __future__ import annotations

import subprocess


def test_make_cli_help_command_matches_docs() -> None:
    result = subprocess.run(
        ["make", "cli-help"],
        check=False,
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0
    assert "BA Agent local/synthetic command surface" in result.stdout


def test_make_no_live_command_matches_docs() -> None:
    result = subprocess.run(
        ["make", "no-live"],
        check=False,
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0
    assert '"live_integrations_enabled":false' in result.stdout


def test_make_validate_mcp_command_matches_docs() -> None:
    result = subprocess.run(
        ["make", "validate-mcp"],
        check=False,
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0
    assert '"version": "phase2-sandbox-auth-v0.5"' in result.stdout
    assert "get_sprint_status" in result.stdout
