from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from mcp_client import MCPClient
from narrator import narrate_with_litellm
from models import Citation


def _dedupe(citations: list[Citation]) -> list[Citation]:
    seen: set[tuple[str, str, str]] = set()
    out: list[Citation] = []
    for citation in citations:
        key = (citation.tool_name, citation.result_excerpt, citation.confidence)
        if key not in seen:
            seen.add(key)
            out.append(citation)
    return out


def _refs_for_prompt(citations: list[Citation]) -> list[str]:
    return [f"{item.tool_name}: {item.result_excerpt} ({item.confidence})" for item in citations]


def _citations_from_search(tool_name: str, payload: dict[str, Any]) -> list[Citation]:
    rows = payload.get("results", [])
    refs: list[Citation] = []
    if isinstance(rows, list):
        for item in rows:
            if not isinstance(item, dict):
                continue
            excerpt = item.get("source_ref") or item.get("caller_ref") or item.get("call_site_ref")
            confidence = item.get("confidence") or item.get("tier") or "medium"
            if isinstance(excerpt, str) and excerpt.strip():
                refs.append(Citation(tool_name=tool_name, result_excerpt=excerpt.strip(), confidence=str(confidence)))
    return refs


def _citations_from_dependency(payload: dict[str, Any]) -> list[Citation]:
    refs: list[Citation] = []
    for key, confidence in (("direct_deps", "direct"), ("transitive_deps", "transitive")):
        values = payload.get(key, [])
        if isinstance(values, list):
            for value in values:
                if isinstance(value, str) and value.strip():
                    refs.append(Citation(tool_name="dependency_analysis", result_excerpt=value.strip(), confidence=confidence))
    return refs


def _citations_from_impact(payload: dict[str, Any]) -> list[Citation]:
    rows = payload.get("impacted", [])
    refs: list[Citation] = []
    if isinstance(rows, list):
        for item in rows:
            if not isinstance(item, dict):
                continue
            ref = item.get("source_ref")
            confidence = item.get("tier") or item.get("confidence") or "medium"
            if isinstance(ref, str) and ref.strip():
                refs.append(Citation(tool_name="impact_analysis", result_excerpt=ref.strip(), confidence=str(confidence)))
    return refs


@dataclass
class ArchitectureAgent:
    mcp: MCPClient
    max_hops: int = 3
    llm_model: str = "ollama/llama3.1:8b"
    llm_api_key: str | None = None

    async def run(self, repo_id: str, component: str) -> tuple[str, list[Citation]]:
        state: dict[str, Any] = {
            "repo_id": repo_id,
            "component": component,
            "plan": [component, f"dependencies of {component}", f"callers of {component}"],
            "search": {},
            "callers": {},
            "deps": {},
        }

        async def identify_component(s: dict[str, Any]) -> dict[str, Any]:
            s["search"] = await self.mcp.call_tool(
                "search_code", {"repo_id": repo_id, "query": component, "top_k": self.max_hops}
            )
            return s

        async def gather_callers(s: dict[str, Any]) -> dict[str, Any]:
            s["callers"] = await self.mcp.call_tool(
                "find_callers", {"repo_id": repo_id, "qualified_name": component, "depth": 2}
            )
            return s

        async def gather_dependencies(s: dict[str, Any]) -> dict[str, Any]:
            s["deps"] = await self.mcp.call_tool(
                "dependency_analysis", {"repo_id": repo_id, "entity": component, "depth": self.max_hops}
            )
            return s

        state = await _run_langgraph_pipeline(
            steps=[
                ("identify_component", identify_component),
                ("gather_callers", gather_callers),
                ("gather_dependencies", gather_dependencies),
            ],
            state=state,
        )

        deps = state["deps"]
        citations = _dedupe(
            _citations_from_search("search_code", state["search"])
            + _citations_from_search("find_callers", state["callers"])
            + _citations_from_dependency(deps)
        )
        if not citations:
            raise RuntimeError("ArchitectureAgent produced no citations")

        direct_count = len(deps.get("direct_deps", [])) if isinstance(deps.get("direct_deps"), list) else 0
        transitive_count = len(deps.get("transitive_deps", [])) if isinstance(deps.get("transitive_deps"), list) else 0
        explanation_prompt = (
            f"Explain the architecture of {component}. "
            f"The MCP plan searched for the component, its callers, and its dependencies. "
            f"It found {direct_count} direct dependencies and {transitive_count} transitive dependencies. "
            f"Write 3-4 sentences and ground the answer in the provided citations."
        )
        explanation = await narrate_with_litellm(
            model=self.llm_model,
            api_key=self.llm_api_key,
            prompt=explanation_prompt,
            citations=_refs_for_prompt(citations),
        )
        return explanation, citations


@dataclass
class ImpactInvestigationAgent:
    mcp: MCPClient
    max_hops: int = 3
    llm_model: str = "ollama/llama3.1:8b"
    llm_api_key: str | None = None

    async def run(self, repo_id: str, changed_entity: str) -> tuple[str, dict[str, list[str]], list[Citation]]:
        state: dict[str, Any] = {"plan": [], "search": {}, "impact": {}, "tiers": {}}

        async def identify_changed_entity(s: dict[str, Any]) -> dict[str, Any]:
            s["plan"] = [changed_entity, f"blast radius of {changed_entity}", f"tiering for {changed_entity}"]
            s["search"] = await self.mcp.call_tool(
                "search_code", {"repo_id": repo_id, "query": changed_entity, "top_k": self.max_hops}
            )
            return s

        async def run_impact_analysis(s: dict[str, Any]) -> dict[str, Any]:
            s["impact"] = await self.mcp.call_tool(
                "impact_analysis", {"repo_id": repo_id, "changed_entity": changed_entity, "depth": self.max_hops}
            )
            return s

        async def classify_by_tier(s: dict[str, Any]) -> dict[str, Any]:
            tiers: dict[str, list[str]] = {}
            impacted = s.get("impact", {}).get("impacted", [])
            if isinstance(impacted, list):
                for item in impacted:
                    if not isinstance(item, dict):
                        continue
                    tier = item.get("tier")
                    ref = item.get("source_ref")
                    if isinstance(tier, str) and isinstance(ref, str) and ref.strip():
                        tiers.setdefault(tier, []).append(ref.strip())
            s["tiers"] = tiers
            return s

        state = await _run_langgraph_pipeline(
            steps=[
                ("identify_changed_entity", identify_changed_entity),
                ("run_impact_analysis", run_impact_analysis),
                ("classify_by_tier", classify_by_tier),
            ],
            state=state,
        )
        impact = state["impact"]
        tiers = state["tiers"] if isinstance(state["tiers"], dict) else {}
        citations = _dedupe(_citations_from_search("search_code", state["search"]) + _citations_from_impact(impact))
        if not citations:
            raise RuntimeError("ImpactInvestigationAgent produced no citations")

        caveat = impact.get("completeness_caveat")
        caveat_text = caveat if isinstance(caveat, str) and caveat.strip() else "Graph reachability is bounded."
        tier_counts = ", ".join(f"{tier}:{len(refs)}" for tier, refs in sorted(tiers.items())) or "no impacted tiers"
        narrative_prompt = (
            f"Investigate the impact of changing {changed_entity}. "
            f"Summarise the affected tiers in priority order ({tier_counts}) and close with this caveat: {caveat_text}."
        )
        narrative = await narrate_with_litellm(
            model=self.llm_model,
            api_key=self.llm_api_key,
            prompt=narrative_prompt,
            citations=_refs_for_prompt(citations),
        )
        return narrative, tiers, citations


async def _run_langgraph_pipeline(*, steps: list[tuple[str, Any]], state: dict[str, Any]) -> dict[str, Any]:
    try:
        from langgraph.graph import END, StateGraph
    except Exception:
        current = state
        for _, step in steps:
            current = await step(current)
        return current

    graph = StateGraph(dict)
    for name, step in steps:
        graph.add_node(name, step)
    for i in range(len(steps) - 1):
        graph.add_edge(steps[i][0], steps[i + 1][0])
    graph.set_entry_point(steps[0][0])
    graph.add_edge(steps[-1][0], END)
    compiled = graph.compile()
    result = await compiled.ainvoke(state)
    return result if isinstance(result, dict) else state
