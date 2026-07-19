from __future__ import annotations

import pytest

from ba_agent.config import RuntimeSettings


def test_runtime_settings_default_to_local_synthetic() -> None:
    settings = RuntimeSettings.from_env({})

    assert settings.environment == "local"
    assert settings.live_integrations_enabled is False


def test_runtime_settings_reject_live_integrations() -> None:
    with pytest.raises(ValueError, match="LIVE_INTEGRATIONS_ENABLED=true"):
        RuntimeSettings.from_env({"LIVE_INTEGRATIONS_ENABLED": "true"})


def test_runtime_settings_reject_non_local_environment() -> None:
    with pytest.raises(ValueError, match="BA_AGENT_ENV=local"):
        RuntimeSettings.from_env({"BA_AGENT_ENV": "prod"})
