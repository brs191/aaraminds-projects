# Independent answer keys

The keys under `phase-1/eval/answer-keys/` were "corrected against live engine
output," so precision/recall there measures whether the engine reproduces
*itself* — determinism, not correctness (audit H-2).

These keys are different. Each `expected_findings` entry was derived **by hand from
the fixture's raw inputs** — the NSG effective rules, effective routes, AVNM admin
rules, public-IP attachment, firewall DNAT, and resource tags — by walking antr's
4-gate reachability model on paper, **without consulting engine output**. The
`_derivation` field on each finding records that reasoning so the independence is
auditable. When the engine matches these keys, that is second-source evidence of
*correctness*, not just reproducibility.

Scope: the core reachability fixtures (internet exposure, firewall DNAT multi-hop,
tier segmentation). Run with `make eval-independent`.
