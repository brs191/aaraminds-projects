from __future__ import annotations

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

    mcp_server_url: str = Field(default="http://127.0.0.1:8081/mcp", alias="MCP_SERVER_URL")
    litellm_model: str = Field(default="ollama/llama3.1:8b", alias="LITELLM_MODEL")
    litellm_api_key: str | None = Field(default=None, alias="LITELLM_API_KEY")
    max_hops: int = Field(default=3, alias="MAX_HOPS", ge=1, le=8)
    timeout_seconds: float = Field(default=30.0, alias="TIMEOUT_SECONDS", gt=0)
