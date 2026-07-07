from __future__ import annotations

from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException

from agents import ArchitectureAgent, ImpactInvestigationAgent
from config import Settings
from mcp_client import MCPClient, MCPToolError
from models import (
    ExplainRequest,
    ExplainResponse,
    HealthResponse,
    InvestigateImpactRequest,
    InvestigateImpactResponse,
)


def create_app(settings: Settings | None = None, mcp_client: MCPClient | None = None) -> FastAPI:
    settings = settings or Settings()
    mcp_client = mcp_client or MCPClient(settings.mcp_server_url, timeout_seconds=settings.timeout_seconds)

    architecture_agent = ArchitectureAgent(
        mcp=mcp_client, max_hops=settings.max_hops, llm_model=settings.litellm_model, llm_api_key=settings.litellm_api_key
    )
    impact_agent = ImpactInvestigationAgent(
        mcp=mcp_client, max_hops=settings.max_hops, llm_model=settings.litellm_model, llm_api_key=settings.litellm_api_key
    )

    @asynccontextmanager
    async def lifespan(_: FastAPI):
        try:
            yield
        finally:
            await mcp_client.close()

    app = FastAPI(title="RIF Agent Service", version="0.1.0", lifespan=lifespan)

    @app.get("/health", response_model=HealthResponse)
    async def health() -> HealthResponse:
        return HealthResponse(status="ok", model=settings.litellm_model, max_hops=settings.max_hops)

    @app.post("/explain", response_model=ExplainResponse)
    async def explain(req: ExplainRequest) -> ExplainResponse:
        try:
            explanation, refs = await architecture_agent.run(req.repo_id, req.component)
        except MCPToolError as exc:
            raise HTTPException(status_code=502, detail=str(exc)) from exc
        except Exception as exc:
            raise HTTPException(status_code=500, detail=str(exc)) from exc
        return ExplainResponse(explanation=explanation, source_refs=refs)

    @app.post("/investigate_impact", response_model=InvestigateImpactResponse)
    async def investigate_impact(req: InvestigateImpactRequest) -> InvestigateImpactResponse:
        try:
            narrative, tiers, refs = await impact_agent.run(req.repo_id, req.changed_entity)
        except MCPToolError as exc:
            raise HTTPException(status_code=502, detail=str(exc)) from exc
        except Exception as exc:
            raise HTTPException(status_code=500, detail=str(exc)) from exc
        return InvestigateImpactResponse(narrative=narrative, tiers=tiers, source_refs=refs)

    return app


app = create_app()
