from __future__ import annotations

from pydantic import BaseModel, Field


class Citation(BaseModel):
    tool_name: str = Field(min_length=1)
    result_excerpt: str = Field(min_length=1)
    confidence: str = Field(min_length=1)


class ExplainRequest(BaseModel):
    repo_id: str = Field(min_length=1)
    component: str = Field(min_length=1)


class ExplainResponse(BaseModel):
    explanation: str = Field(min_length=1)
    source_refs: list[Citation] = Field(min_length=1)


class InvestigateImpactRequest(BaseModel):
    repo_id: str = Field(min_length=1)
    changed_entity: str = Field(min_length=1)


class InvestigateImpactResponse(BaseModel):
    narrative: str = Field(min_length=1)
    tiers: dict[str, list[str]]
    source_refs: list[Citation] = Field(min_length=1)


class HealthResponse(BaseModel):
    status: str
    model: str
    max_hops: int
