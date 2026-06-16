# Azure Network Topology Reviewer — Phase 1 Status

Phase 1 has not started. See `../IMPLEMENTATION_PLAYBOOK.md` for the full step-by-step guide.

## Ready to begin

All Phase 0 prerequisites are met:

- `engine/go/internal/analyze/analyze.go` — deterministic `Analyze()` function (5/5 golden tests pass)
- `engine/go/internal/graph/model.go` — `graph.Fixture` type (the contract the adapter must produce)
- `engine/go/testdata/` — 5 golden fixtures
- `phase-0/FINDINGS_MEMO.md` — Phase 0 acceptance document

## Step 1.1 is the entry point

Start with:

```
Agent: aara-project-architect
Task:  phase-1/design/TOPOLOGY_MODEL.md
Prompt: copy from IMPLEMENTATION_PLAYBOOK.md § Step 1.1
```

## Gate table (not yet run)

| Gate | Criterion | Verdict |
|---|---|---|
| G1 | Adapter produces correct graph.Fixture | ⬜ |
| G2 | analyze_risks returns same verdicts as internal engine | ⬜ |
| G3 | Precision ≥ 0.95; recall ≥ 0.90 for High+Critical | ⬜ |
| G4 | LLM never in severity/reachability path | ⬜ |
| G5 | Managed Identity read-only; JFrog for registry; OIDC for Azure | ⬜ |
