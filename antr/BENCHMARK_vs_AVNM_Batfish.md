# antr Reachability Engine — Benchmark vs AVNM Network Verifier & Batfish

**Status:** ADOPT-11 (adoption roadmap, Wave 3) · **Date:** 2026-06-16
**Audience:** Principal/staff network-security engineers evaluating whether antr earns a place alongside native Azure and OSS tooling.
**Scope:** Azure network reachability and exposure analysis only — not a general network-validation comparison.
**Sources:** Every external claim is cited inline to Microsoft Learn, the Batfish GitHub repository, or Batfish docs. Claims about antr describe its own engine.

---

## 0. TL;DR verdict

- **AVNM Network Verifier** — the right tool for a *deployed* estate, a *single* source→destination intent, where you can satisfy its "running VM in the subnet" requirement. Authoritative on the live control plane; models AVNM admin rules, peering, private endpoints, vWAN. **Not** pre-deploy, **not** a fleet sweep, **no** severity ranking, and models **Azure Firewall as static L4 only** ([MS Learn — *What is network verifier?*](https://learn.microsoft.com/en-us/azure/virtual-network-manager/concept-virtual-network-verifier)).
- **Batfish** — best-in-class for **multi-vendor physical networks and AWS**, with real pre-deploy strength. Its **Azure support is shallow**: VNets/subnets/NSGs/NICs/public-IPs/NAT-gateways only; its own source says it **does not support UDRs** and has **no knowledge of VNet peering** ([`Subnet.java`](https://github.com/batfish/batfish/blob/master/projects/batfish/src/main/java/org/batfish/representation/azure/Subnet.java)). No Azure Firewall, Private Endpoint, WAF, Bastion, vWAN, or AVNM.
- **antr** — occupies the gap both leave open on Azure: deterministic, **pre-deploy-capable** 4-gate reachability (AVNM admin → NSG → effective routes incl. `None` black-hole → public IP), **firewall DNAT depth**, **fleet sweep**, **severity-ranked findings with the exact path as evidence**, license-free, **MCP-embeddable**, and it does **not** require running resources. Defer to Verifier for authoritative single-intent answers on a live estate; defer to Batfish for non-Azure / multi-vendor.

---

## 1. AVNM Network Verifier

### 1.1 What it does
A feature of Azure Virtual Network Manager: create a verifier workspace, define a reachability intent (source, destination, protocol, ports), run a **static analysis** that "checks if various resources and policy configurations in the network manager's scope preserve reachability" and returns all/some/no-packets-reached with a per-step path explanation ([MS Learn — concept](https://learn.microsoft.com/en-us/azure/virtual-network-manager/concept-virtual-network-verifier); [how-to](https://learn.microsoft.com/en-us/azure/virtual-network-manager/how-to-verify-reachability-with-virtual-network-verifier)).

**Supported features:** NSG rules, ASG rules, **AVNM security admin rules** (its unique authoritative strength), mesh topology, VNet peering, route tables, service endpoints/ACLs, private endpoints, Virtual WAN, **Azure Firewall (static L4 only)** ([MS Learn — concept](https://learn.microsoft.com/en-us/azure/virtual-network-manager/concept-virtual-network-verifier)).

### 1.2 Documented limitations antr exploits (verbatim from Microsoft)
- **(a) Needs a running VM.** *"Subnets selected as the source and/or destination … must have at least one running virtual machine for a reachability analysis result to be provided."* → unusable for empty/scale-to-zero/not-yet-provisioned subnets. antr analyzes config/plan, no running resource needed.
- **(b) Firewall = static L4 only.** Does not trace **DNAT chains** (public frontend IP:port → internal private target) — the most common internet→internal exposure path. antr models DNAT depth as a first-class finding.
- **(c) One intent per analysis.** *"A reachability analysis can only be run on a single reachability analysis intent."* No estate-wide enumeration of "every internet-reachable endpoint." antr runs a fleet sweep.
- **(d) No severity ranking.** Binary-with-path only; doesn't tell you which of N reachable paths is the breach. antr ranks by severity.
- **(e) Live control plane, not pre-deploy.** Evaluates resources "currently in place"; can't answer "will this Terraform change open a path?" antr runs against the plan, shifting left into the PR.

> **Concession:** Verifier is the **only** tool that models AVNM **admin-rule precedence authoritatively** (Microsoft owns the evaluation order). antr implements it as Gate 1 from documented behavior; on a live estate, where they disagree, **Verifier is the source of truth.**

## 2. Batfish

Vendor-config analysis engine; headline strength is **pre-deployment validation** with no device access, excellent for multi-vendor fabrics and **AWS** ([README](https://github.com/batfish/batfish)). **Azure support is shallow and recent** — input is Azure resource JSON only (ARM templates unsupported) ([formats](https://pybatfish.readthedocs.io/en/latest/formats.html#azure)).

Modeled Azure types (from the [representation directory](https://github.com/batfish/batfish/tree/master/projects/batfish/src/main/java/org/batfish/representation/azure)): VNet, Subnet, NSG, SecurityRule, NetworkInterface, PublicIp/Prefix, NatGateway, VM, ContainerGroup, Postgres. **Absent:** RouteTable/UDR, VNet peering, Azure Firewall, Private Endpoint, App Gateway/Front Door/WAF, Bastion, vWAN, AVNM. Its own `Subnet.java` states in-code: *"Do not support UDR"* (line 31) and *"no knowledge of Vnet … peering"* (line 107) ([Subnet.java](https://github.com/batfish/batfish/blob/master/projects/batfish/src/main/java/org/batfish/representation/azure/Subnet.java)).

**Consequence:** Batfish fits **Azure NSG** analysis only. The routing/peering/admin-rule half of Azure reachability — forced-tunnel UDRs, `None` black-hole routes, transitive peering, firewall DNAT — is unmodeled. "We already run Batfish for Azure" covers NSGs and nothing else.

> **Credit:** Batfish's differential change-analysis across vendors is broader/more mature than antr's. The gap is **breadth of Azure modeling**, not the pre-deploy idea.

## 3. Capability matrix

| Dimension | antr | AVNM Verifier | Batfish (Azure) |
|---|---|---|---|
| Deterministic / fixture-testable | Yes | Yes | Yes |
| Needs running/deployed resources | **No** | **Yes** (≥1 running VM) | No |
| Severity-ranked findings | **Yes** | **No** | No |
| Fleet / estate-wide sweep | **Yes** | **No** (1 intent) | Partial |
| Pre-deploy / Terraform-plan input | **Yes** | **No** (live only) | Yes (general), **not** Azure UDR/peering |
| Firewall DNAT depth | **Yes** | **No** (static L4) | **No** (firewall not modeled) |
| AVNM admin-rule precedence | Yes (defer to Verifier as truth) | **Yes — authoritative** | **No** |
| UDR / effective-route incl. `None` black-hole | **Yes** | Yes | **No** ("Do not support UDR") |
| Transitive-peering exposure | **Yes** | Yes | **No** ("no knowledge of peering") |
| License | Free | Azure service billing | Apache-2.0 |
| Embeddable / MCP | **Yes** (MCP) | No (portal/CLI/API) | SDK/Docker, not MCP |
| Evidence as exact reachable path | **Yes** | **Yes** | Partial |

## 4. Honest verdict

**antr is genuinely better at:** pre-deploy Azure reachability/exposure; estate-wide sweep with severity; Azure Firewall DNAT depth + transitive peering + `None` black-hole route; no-running-resource/empty-subnet analysis; MCP-embeddable + license-free + deterministic.

**Defer to the incumbent for:** AVNM admin-rule precedence on a live estate (→ Verifier, authoritative); authoritative single-intent answer on a deployed estate (→ Verifier); non-Azure / multi-vendor / AWS (→ Batfish); mature differential change-analysis across heterogeneous fabrics (→ Batfish).

**The 2–3 cases that justify antr alongside both:**
1. **Internet → internal via Azure Firewall DNAT, evaluated pre-deploy** — Verifier is static-L4 + live-only; Batfish doesn't model the firewall. antr is the only one tracing the DNAT chain before apply.
2. **Transitive-peering lateral exposure (spoke-to-spoke)** — Batfish has no peering model; Verifier can't sweep the estate with severity. antr does both, pre-deploy.
3. **`None` black-hole / forced-tunnel routing correctness, fleet-wide** — Batfish has no UDR; Verifier is one-intent, post-deploy, unranked. antr's Gate 3 evaluates effective routes estate-wide as ranked findings.

## Sources
- [MS Learn — What is network verifier? (features, Limits)](https://learn.microsoft.com/en-us/azure/virtual-network-manager/concept-virtual-network-verifier)
- [MS Learn — Verify resource reachability with Virtual Network Verifier](https://learn.microsoft.com/en-us/azure/virtual-network-manager/how-to-verify-reachability-with-virtual-network-verifier)
- [Batfish — GitHub README](https://github.com/batfish/batfish)
- [Batfish — Azure representation directory](https://github.com/batfish/batfish/tree/master/projects/batfish/src/main/java/org/batfish/representation/azure)
- [Batfish — Subnet.java ("Do not support UDR"; "no knowledge of Vnet peering")](https://github.com/batfish/batfish/blob/master/projects/batfish/src/main/java/org/batfish/representation/azure/Subnet.java)
- [pybatfish — formats (Azure JSON-only; AWS)](https://pybatfish.readthedocs.io/en/latest/formats.html)
