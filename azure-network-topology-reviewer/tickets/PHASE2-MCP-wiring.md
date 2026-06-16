# Ticket PHASE2 — Wire `simulate_change` + `forecast_cost` MCP tools, then accept Phase 2

**Type:** feature / phase-close · **Effort:** M–L (~1 day incl. tests + memo) · **Needs:** Go 1.25
**Closes:** PA-01, PA-02 · **Status:** READY

## Context

Phase 2's engines are built and unit-tested but **never exposed** as MCP tools, and Phase 2 has no
acceptance memo (see `phase-2/PHASE_2_STATUS.md`). The engines:

- `simulator.ApplyDelta(fx *graph.Fixture, delta TopologyDelta) (*graph.Fixture, error)` — `simulator/apply.go:20`
- `simulator.DiffFindings(original, simulated []analyze.Finding) SecurityDelta` — `simulator/diff.go:51`
- `simulator.TopologyDelta` — JSON-shaped struct (`simulator/delta.go:19`): `addSubnet/removeSubnet/
  addNsgRule/removeNsgRule/addPeering/removePeering/addPublicIp/removePublicIp/modifyRoute`
- `forecast.ForecastCost(ctx, fx, delta, cache *PriceCache, flows FlowSummary, region string) (CostForecast, error)` — `forecast/forecast.go:86`
- `forecast.NewPriceCache()` — `forecast/prices.go:65`; `forecast.EstimateTrafficGB(...)` — `forecast/flowlogs.go:43`

Wire them following the existing tool pattern (`get_topology`/`analyze_risks`) — both are **read-only**
forecasts (mutating=false), no write path.

## Changes (exact)

### 1. `mcp/tools.go` — two handlers mirroring `analyzeRisksHandler` (`tools.go:113`)
```go
func simulateChangeHandler(fetcher TopologyFetcher) server.ToolHandlerFunc {
    return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
        subID, err := req.RequireString("subscription_id")
        if err != nil { return fmtErr("subscription_id is required"), nil }
        if verr := validateSubscriptionID(subID); verr != nil { return fmtErr("%s", verr), nil }
        var delta simulator.TopologyDelta
        if raw := req.GetString("delta", ""); raw != "" {
            if err := json.Unmarshal([]byte(raw), &delta); err != nil {
                return fmtErr("delta is not valid JSON: %v", err), nil
            }
        }
        fixture, err := fetcher.FetchFixture(ctx, subID)
        if err != nil { return fmtErr("fetch topology: %v", err), nil }
        before := analyze.Analyze(fixture)
        sim, err := simulator.ApplyDelta(fixture, delta)
        if err != nil { return fmtErr("apply delta: %v", err), nil }
        after := analyze.Analyze(sim)
        secDelta := simulator.DiffFindings(before, after)
        // marshal { subscription, security_delta: secDelta, before_count, after_count } and return
    }
}

func forecastCostHandler(fetcher TopologyFetcher) server.ToolHandlerFunc {
    return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
        // subscription_id + validate (as above)
        // delta := unmarshal req.GetString("delta","")   (empty delta = forecast current estate)
        // region := req.GetString("region", "eastus")
        // fixture := fetcher.FetchFixture(ctx, subID)
        // cache := forecast.NewPriceCache()
        // flows := forecast.FlowSummary{}   // v1: empty -> variable cost uses the estimated band
        // fc, err := forecast.ForecastCost(ctx, fixture, delta, cache, flows, region)
        // marshal fc (CostForecast) and return; surface err as fmtErr
    }
}
```
Reuse the package imports (`simulator`, `forecast`) — add them to the import block.

### 2. `mcp/server.go` — register both (mirror `analyze_risks` at server.go ~117)
```go
simulateTool := mcpgo.NewTool("simulate_change",
    mcpgo.WithDescription("Apply a proposed topology delta in-memory and return the security (reachability/severity) delta. Read-only."),
    mcpgo.WithString("subscription_id", mcpgo.Required(), mcpgo.Description("Azure subscription id")),
    mcpgo.WithString("delta", mcpgo.Description("TopologyDelta as JSON (addNsgRule/addPublicIp/modifyRoute/...)")),
)
s.AddTool(simulateTool, withMiddleware(logger, "simulate_change", false, simulateChangeHandler(fetcher), auditor))

forecastTool := mcpgo.NewTool("forecast_cost",
    mcpgo.WithDescription("Forecast fixed (SKU-exact) + variable (flow-log estimated) cost of the estate or a proposed delta. Read-only."),
    mcpgo.WithString("subscription_id", mcpgo.Required(), mcpgo.Description("Azure subscription id")),
    mcpgo.WithString("delta", mcpgo.Description("Optional TopologyDelta as JSON")),
    mcpgo.WithString("region", mcpgo.Description("Azure region for pricing (default eastus)")),
)
s.AddTool(forecastTool, withMiddleware(logger, "forecast_cost", false, forecastCostHandler(fetcher), auditor))
```

### 3. `mcp/audit.go` — add audit lines for both tools (mirror generate_topology), e.g. tool name,
`delta_hash` (SHA-256 of the delta JSON), finding counts before/after, forecast total. Keep the
`auditor != nil` guard.

### 4. `mcp/mcp_test.go` — add table tests asserting both tools register and return valid JSON on a golden
fixture (e.g. `addPublicIp` → simulate shows a new reachable finding; `forecast_cost` returns a non-nil
`CostForecast` with the mandatory caveats). The simulator/forecast unit tests already pass via `go test ./...`.

### 5. Acceptance + docs
- Write `phase-2/PHASE_2_ACCEPTANCE_MEMO.md` (use the Phase-3 memo as the template) with gates:
  G1 engine parity (`simulate` re-runs `analyze.Analyze` directly), G2 tool wiring (both registered,
  return on a fixture), G3 security (read-only, no `AZURE_CLIENT_SECRET`, no write/apply), G4 determinism,
  G5 cost caveats present (no unbaselined numbers).
- Update `phase-2/PHASE_2_STATUS.md`, the README status table, and `IMPLEMENTATION_PLAYBOOK.md` Phase-2 row
  from PARTIAL → ACCEPTED.

## Verify
```bash
cd engine/go && go build ./... && go vet ./... && go test ./...     # incl. new mcp tool tests
# optional live-stub: run the MCP server and call simulate_change/forecast_cost with a fixture-backed fetcher
```

## Acceptance criteria
- [ ] `simulate_change` and `forecast_cost` registered in `server.go` and callable; `go test ./mcp/...` green.
- [ ] `simulate_change` returns a `SecurityDelta`; `forecast_cost` returns a `CostForecast` with caveats.
- [ ] Both read-only — `grep -rn "AZURE_CLIENT_SECRET\|terraform apply" mcp/` clean; mutating=false.
- [ ] `phase-2/PHASE_2_ACCEPTANCE_MEMO.md` written; status tables updated to ACCEPTED.
- [ ] `engine-ci.yml` green (the workflow already runs `go test ./...`).
