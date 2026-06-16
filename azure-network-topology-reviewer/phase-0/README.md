# Azure Network Topology Reviewer — Phase 0

**Status:** ✅ ACCEPTED — 2026-06-03

The analysis engine is proven. See `FINDINGS_MEMO.md` for the full exit document.

## What's here

- `FINDINGS_MEMO.md` — Phase 0 acceptance document
- `fixtures/` — symlink / copy destination for `engine/go/testdata/` (shared with engine)
- `evalset/` — reserved for Phase 1 eval harness expansion (10 new fixtures land here)

## Golden corpus (in engine/go/testdata/)

| Fixture | Key scenario |
|---|---|
| `fixture-1-internet-exposure.json` | NIC with PIP + open NSG + internet route → Critical |
| `fixture-2-segmentation-peering.json` | Transitive peering + missing tier segmentation |
| `fixture-3-cidr-avnm.json` | CIDR overlap + AVNM AlwaysAllow override |
| `fixture-h1-dnat-multihop.json` | Azure Firewall DNAT → private NIC |
| `fixture-h2-blackhole-tags.json` | Black-hole route + AzureCloud tag → latent |

## Phase 1 starts here

```bash
# Verify engine is green before starting Phase 1
cd ../engine/go
go test ./...
```

All 5 fixtures must pass. Then start Step 1.1 from `../IMPLEMENTATION_PLAYBOOK.md`.
