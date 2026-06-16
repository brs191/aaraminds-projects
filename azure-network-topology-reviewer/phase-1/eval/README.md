# Azure Network Topology Reviewer — Eval Harness

Phase 1 evaluation harness for the deterministic analysis engine. Runs all fixtures against the Go CLI, scores findings against answer keys, and gates CI on per-severity recall and overall precision.

## Quick Start

```bash
cd /path/to/azure-network-topology-reviewer
python phase-1/eval/run_eval.py
```

Open `phase-1/eval/report.html` in a browser after a run to view the interactive results table.

## Requirements

- **Python 3.11+** (stdlib only — no `pip install` required)
- **Go 1.21+** in `PATH`
- Engine source at `engine/go/` (default; override with `--engine-dir`)

## Gate Thresholds

| Gate | Metric | Threshold |
|---|---|---|
| Overall precision | TP / (TP + FP) across all fixtures | ≥ 0.95 |
| High+Critical recall | TP / (TP + FN) for High and Critical findings | ≥ 0.90 |
| Medium recall | TP / (TP + FN) for Medium findings | ≥ 0.80 |

CI exits with code `1` if any gate is missed. A zero exit means all gates passed.

## CLI Options

```
python run_eval.py [--fixtures-dir DIR] [--answer-keys-dir DIR] [--engine-dir DIR] [--output PATH]

--fixtures-dir     Directory of fixture JSON files    (default: ./fixtures)
--answer-keys-dir  Directory of answer key JSON files (default: ./answer-keys)
--engine-dir       Directory with engine go.mod        (default: ../../engine/go)
--output           Path for last_run.json report       (default: ./last_run.json)
```

## Fixture Corpus

### Engine fixtures (13) — copied from `engine/go/testdata/`

| Fixture | Key findings |
|---|---|
| `fixture-1-internet-exposure.json` | SSH internet reachable, latent NSG, orphaned PIP |
| `fixture-2-segmentation-peering.json` | Missing tier segmentation on sensitive DB NIC |
| `fixture-3-cidr-avnm.json` | AVNM AlwaysAllow reachable, AVNM Deny latent, CIDR overlap |
| `fixture-h1-dnat-multihop.json` | Firewall DNAT exposes NIC without public IP |
| `fixture-h2-blackhole-tags.json` | AzureCloud tag reachable, black-hole route latent |
| `fixture-f6-pe-dns-misconfiguration.json` | Private DNS zone not linked to VNet |
| `fixture-f7-appgw-waf-disabled.json` | App GW WAF disabled, WAF in detection mode |
| `fixture-f8-aks-and-crosssub-peering.json` | AKS non-private, cross-sub peering without firewall |
| `fixture-f10-elb-nat.json` | ELB NAT exposes NICs without direct public IPs |
| `fixture-f11-apim-exposure.json` | APIM None isolation, APIM External without WAF |
| `fixture-f12-bastion-bypass.json` | Bastion bypass on port 22/3389 |
| `fixture-f13-vwan-unsecured.json` | vWAN hub unsecured, hub firewall bypasses private traffic |
| `fixture-f14-frontdoor-waf.json` | Front Door WAF disabled, WAF in detection mode |

### Eval fixtures (10) — adversarial and edge cases in `fixtures/`

| Fixture | Scenario | Expected |
|---|---|---|
| `eval-fixture-6.json` | 3 orphaned PIPs + 1 clean NIC | 3× orphaned-endpoint Low; zero reachable |
| `eval-fixture-7.json` | AVNM Deny overrides open NSG | latent Informational; NOT High reachable |
| `eval-fixture-8.json` | Route → VirtualAppliance (NVA) | latent Informational; NOT High reachable |
| `eval-fixture-9.json` | Mixed severity: Critical + High + Medium + Informational | 1 Critical, 1 High, 1 Medium, 1 Informational |
| `eval-fixture-10.json` | Firewall DNAT to two NICs | 2× DNAT High; NOT Critical (DNAT code path ignores tags) |
| `eval-fixture-11.json` | AllowVnetInBound + DenyVnetInBound (deny wins) | Zero findings (VirtualNetwork source, not internet) |
| `eval-fixture-12.json` | All-clean hub-spoke (no false positives) | Zero findings |
| `eval-fixture-13.json` | PE DNS zone correctly linked | Zero findings |
| `eval-fixture-14.json` | NIC with empty effective rules + public IP | Zero findings (no allow rule to match) |
| `eval-fixture-15.json` | Front Door Prevention + APIM Internal | Zero findings |

## Answer Key Format

```json
{
  "fixture": "fixture-name.json",
  "expected_findings": [
    { "type": "over-permissive NSG (reachable)", "severity": "High", "resource": "nic-vm-web-a" }
  ],
  "trap_assertions": [
    { "resource": "nic-vm-web-b", "must_not_have": "reachable High — description" }
  ]
}
```

- `expected_findings`: scored by `run_eval.py` — each entry must be matched by an engine finding
- `trap_assertions`: documentation only — describes what the engine must NOT produce; not scored programmatically
- **Type matching uses substring**: `expected["type"] in engine_finding["type"]` (consistent with engine test assertions)
- **Severity and resource**: exact match

## Adding a New Fixture

1. Create `fixtures/my-scenario.json` following the engine's JSON schema (see `engine/go/internal/graph/model.go`)
2. Run the engine manually to observe actual output:
   ```bash
   cd engine/go
   go run ./cmd/analyze/... ../../phase-1/eval/fixtures/my-scenario.json
   ```
3. Create `answer-keys/my-scenario.json` with `expected_findings` matching the verified output
4. Re-run the eval: `python phase-1/eval/run_eval.py`

## CI Integration

```yaml
# .github/workflows/eval.yml (example)
- name: Run evaluation harness
  run: python phase-1/eval/run_eval.py
  # Exits 1 if any gate is missed — blocks merge
```

## Output Files

- `last_run.json` — machine-readable report consumed by `report.html`
- `report.html` — static interactive report; open locally in a browser; reads `last_run.json` via `fetch`
