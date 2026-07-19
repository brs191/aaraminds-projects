from __future__ import annotations

import os
import subprocess
import sys
import json


def _env() -> dict[str, str]:
    env = os.environ.copy()
    env["PYTHONPATH"] = "src"
    env.pop("LIVE_INTEGRATIONS_ENABLED", None)
    env["BA_AGENT_ENV"] = "local"
    return env


def test_module_help_runs() -> None:
    result = subprocess.run(
        [sys.executable, "-m", "ba_agent", "--help"],
        check=False,
        capture_output=True,
        text=True,
        env=_env(),
    )

    assert result.returncode == 0
    assert "BA Agent local/synthetic command surface" in result.stdout


def test_check_config_rejects_live_mode() -> None:
    env = _env()
    env["LIVE_INTEGRATIONS_ENABLED"] = "true"

    result = subprocess.run(
        [sys.executable, "-m", "ba_agent", "check-config"],
        check=False,
        capture_output=True,
        text=True,
        env=env,
    )

    assert result.returncode == 2
    assert "LIVE_INTEGRATIONS_ENABLED=true" in result.stderr


def test_placeholder_commands_run_without_network() -> None:
    for command in ("synthetic", "eval"):
        result = subprocess.run(
            [sys.executable, "-m", "ba_agent", command, "--help"],
            check=False,
            capture_output=True,
            text=True,
            env=_env(),
        )

        assert result.returncode == 0
        assert "usage:" in result.stdout


def test_synthetic_case_outputs_card_json() -> None:
    result = subprocess.run(
        [sys.executable, "-m", "ba_agent", "synthetic", "STD-001"],
        check=False,
        capture_output=True,
        text=True,
        env=_env(),
    )

    assert result.returncode == 0
    assert '"trace_id"' in result.stdout
    assert "Daily standup summary" in result.stdout


def test_eval_commands_pass() -> None:
    for eval_set in ("GTS-STANDUP", "GTS-ROUTER", "GTS-GATE", "GTS-PLANNING", "GTS-RETRO", "GTS-HEALTH", "GTS-MVP"):
        result = subprocess.run(
            [sys.executable, "-m", "ba_agent", "eval", eval_set],
            check=False,
            capture_output=True,
            text=True,
            env=_env(),
        )

        assert result.returncode == 0
        assert json.loads(result.stdout)["passed"] is True
