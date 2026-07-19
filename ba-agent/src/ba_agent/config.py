from __future__ import annotations

import os
from collections.abc import Mapping
from typing import Self

from pydantic import BaseModel, ConfigDict, Field


def _parse_bool(value: str | None) -> bool:
    if value is None:
        return False
    normalized = value.strip().lower()
    if normalized in {"1", "true", "yes", "on"}:
        return True
    if normalized in {"0", "false", "no", "off", ""}:
        return False
    raise ValueError(f"invalid boolean value for LIVE_INTEGRATIONS_ENABLED: {value!r}")


class RuntimeSettings(BaseModel):
    """Runtime settings for local/synthetic Phase 1 execution."""

    model_config = ConfigDict(frozen=True)

    environment: str = Field(default="local")
    live_integrations_enabled: bool = Field(default=False)
    data_source_mode: str = Field(default="synthetic")

    @classmethod
    def from_env(cls, env: Mapping[str, str] | None = None) -> Self:
        source = os.environ if env is None else env
        settings = cls(
            environment=source.get("BA_AGENT_ENV", "local"),
            live_integrations_enabled=_parse_bool(source.get("LIVE_INTEGRATIONS_ENABLED")),
            data_source_mode=source.get("BA_AGENT_DATA_SOURCE_MODE", "synthetic"),
        )
        settings.require_local_only()
        return settings

    def require_local_only(self) -> None:
        if self.environment != "local":
            raise ValueError("Phase 1 only supports BA_AGENT_ENV=local")
        if self.live_integrations_enabled:
            raise ValueError("Phase 1 rejects LIVE_INTEGRATIONS_ENABLED=true")
        if self.data_source_mode not in {"synthetic", "sandbox_read"}:
            raise ValueError("BA_AGENT_DATA_SOURCE_MODE must be synthetic or sandbox_read")
