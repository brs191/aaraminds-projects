"""Pydantic v2 models and typed settings for the explainer service.

Design rule: every boundary is typed.  Pydantic validates at the edge; the rest
of the service trusts the typed objects.
"""

from __future__ import annotations

from datetime import datetime
from typing import Literal, Optional

from pydantic import BaseModel, Field, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict

# ---------------------------------------------------------------------------
# Finding / request / report models
# ---------------------------------------------------------------------------


class FindingInput(BaseModel):
    """A single finding produced by the Go deterministic analysis engine."""

    type: str
    severity: Literal["Critical", "High", "Medium", "Informational"]
    resource: str
    evidence: str
    reachable: bool


class ExplainedFinding(FindingInput):
    """FindingInput enriched by the LLM + RAG layer."""

    explanation: str | None = None        # LLM-generated; None on LLM failure
    recommendation: str | None = None     # RAG-grounded; None if no match
    rag_grounded: bool = False
    rag_source: str | None = None         # index document title


class ExplainRequest(BaseModel):
    """Inbound request to POST /explain."""

    subscription_id: str
    findings: list[FindingInput]


class TopologyReport(BaseModel):
    """Final report returned to the caller.  Never 500 — errors go in .error."""

    subscription_id: str
    analyzed_at: datetime
    findings: list[ExplainedFinding]
    summary: str | None = None            # 2-sentence LLM synthesis
    high_critical_count: int
    rag_grounded_pct: float               # 0.0–1.0
    error: str | None = None             # set on LLM failure

    @model_validator(mode="after")
    def _compute_derived(self) -> "TopologyReport":
        """Validate that derived fields are consistent with findings list.

        Both high_critical_count and rag_grounded_pct must be set by callers
        (assemble_report), but this validator guards against accidental drift.
        """
        expected_hc = sum(
            1 for f in self.findings if f.severity in ("Critical", "High")
        )
        if self.high_critical_count != expected_hc:
            raise ValueError(
                f"high_critical_count={self.high_critical_count} "
                f"but computed {expected_hc} from findings"
            )
        n = len(self.findings)
        if n > 0:
            grounded = sum(1 for f in self.findings if f.rag_grounded)
            expected_pct = round(grounded / n, 4)
            if abs(self.rag_grounded_pct - expected_pct) > 0.0001:
                raise ValueError(
                    f"rag_grounded_pct={self.rag_grounded_pct} "
                    f"but computed {expected_pct}"
                )
        else:
            if self.rag_grounded_pct != 0.0:
                raise ValueError("rag_grounded_pct must be 0.0 for empty findings")
        return self


# ---------------------------------------------------------------------------
# Typed settings (one object per subsystem; validated at startup)
# ---------------------------------------------------------------------------


class LLMSettings(BaseSettings):
    """AskAT&T GenAI client configuration.

    All values come from environment variables (or Key Vault references injected
    as env-vars by Container Apps).  ASKAT_CLIENT_SECRET is never logged.
    """

    model_config = SettingsConfigDict(env_prefix="ASKAT_", case_sensitive=False)

    endpoint: str = ""
    model: str = "gpt-4o"
    client_id: str = ""
    client_secret: str = ""    # from KV reference; never logged
    token_url: str = ""
    scope: str = "https://cognitiveservices.azure.com/.default"


class RAGSettings(BaseSettings):
    """Azure AI Search (RAG) client configuration."""

    model_config = SettingsConfigDict(case_sensitive=False)

    azure_search_endpoint: str = ""
    azure_search_key: str = ""           # use AzureKeyCredential when set
    azure_search_index: str = ""
