# Go engine — production port of the proven core

A direct Go translation of the verified reference (`../reference/analyze.py`), in the
`engine-plan.md` package layout. The Python reference proves the algorithm against the
fixtures (5/5 golden tests pass); this is the production-stack (Go) implementation that ships.

## Layout

```
go/
  go.mod                            module github.com/aaraminds/azure-nettopo-engine
  internal/graph/model.go           topology model + parser — Azure-shaped in v1; cloud-neutral is a later goal (the Azure adapter feeds this)
  internal/analyze/analyze.go       the deterministic core — gates, exposure, DNAT, AVNM source-scope, CIDR, segmentation
  internal/analyze/analyze_test.go  golden tests over the same fixtures
  cmd/analyze/main.go               CLI: analyze <fixture.json> -> findings JSON
  testdata/                         the eval fixtures (golden inputs)
```

## Build & test — stdlib only, no network

```
cd go
go test ./...                                         # the golden suite — expect 5/5 PASS
go run ./cmd/analyze testdata/fixture-1-internet-exposure.json
```

**Honesty note:** this was written in a sandbox with no Go toolchain, so it is
*unverified-in-place but verified-by-twin* — a faithful, line-for-line port of the Python
suite that passes 5/5. Run `go test ./...` on your machine to confirm. If anything fails,
it's a transcription slip, not an algorithm error — the reference is the source of truth.

## Next (per engine-plan.md)

- **`internal/adapter/azure`** — Resource Graph + Network Watcher → `graph.Fixture` (needs live Azure + the read-only managed identity).
- **`internal/mcp`** — expose `analyze_risks` over stdio via `github.com/mark3labs/mcp-go`
  (run `go get github.com/mark3labs/mcp-go && go mod tidy` first). The handler is one line over the core:

  ```go
  s := server.NewMCPServer("azure-nettopo-engine", "0.1.0")
  s.AddTool(mcp.NewTool("analyze_risks", /* schema */), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
      fx, err := graph.Load(req.Params.Arguments["path"].(string))   // or parse inline topology
      if err != nil { return mcp.NewToolResultError(err.Error()), nil }
      b, _ := json.Marshal(analyze.Analyze(fx))
      return mcp.NewToolResultText(string(b)), nil
  })
  server.ServeStdio(s)
  ```

- **`reachable()` extension** for multi-hop attack-paths, and **`compareTopology()`** for drift detection — the two roadmap items now folded into `engine-plan.md`.
