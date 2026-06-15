# Simulation and Cost Model — Phase 2

**Document:** `phase-2/design/SIMULATION_MODEL.md`
**Date:** 2026-06-12
**Status:** DRAFT — Phase 2 design. Phase 1 acceptance: ACCEPTED WITH CONDITIONS (commit `8d6aac3`).
**Scope:** Defines the typed delta schema, apply-delta semantics, security-posture diffing, cost
estimation model, Azure Cost MCP integration boundary, and Phase 2 placeholder reconciliation
for all `// Phase 2` fields identified in TOPOLOGY_MODEL.md §7.

**Upstream references:**
- `engine/go/internal/graph/model.go` — `graph.Fixture`, `graph.Fixture`, `graph.SecRule`, `graph.Route`, `graph.Peering`, `graph.Subnet`
- `engine/go/internal/analyze/analyze.go` — `Analyze(*graph.Fixture) []Finding`
- `phase-1/design/TOPOLOGY_MODEL.md` §7 — Phase 2 placeholder fields
- `phase-1/PHASE_1_ACCEPTANCE_MEMO.md` — Gate verdicts, P2-C3 condition

---

## Contents

1. [Scope and Out-of-Scope Boundary](#1-scope-and-out-of-scope-boundary)
2. [Delta Schema — `TopologyDelta`](#2-delta-schema)
3. [Apply-Delta Function](#3-apply-delta-function)
4. [Effective-Rule Projection in Simulation Mode](#4-effective-rule-projection-in-simulation-mode)
5. [Security Delta — `SecurityDelta`](#5-security-delta)
6. [Cost Model — Fixed Costs](#6-cost-model--fixed-costs)
7. [Cost Model — Variable Costs](#7-cost-model--variable-costs)
8. [`CostForecast` Output Schema](#8-costforecast-output-schema)
9. [Azure Cost MCP Integration Boundary](#9-azure-cost-mcp-integration-boundary)
10. [Phase 2 Placeholder Reconciliation](#10-phase-2-placeholder-reconciliation)
11. [Phase 2 Adapter Extension Plan](#11-phase-2-adapter-extension-plan)

---

## 1. Scope and Out-of-Scope Boundary

### 1.1 What Phase 2 adds

| Capability | MCP tool | Input | Output |
|---|---|---|---|
| Topology what-if | `simulate_change` | subscription + `TopologyDelta` | `SecurityDelta` (risks added / mitigated) |
| Cost impact forecast | `forecast_cost` | subscription + `TopologyDelta` | `CostForecast` (fixed exact, variable band) |

Both tools share the same execution path:

```
FetchFixture(ctx, cred, sub)
  → ApplyDelta(fixture, delta) → simulatedFixture
  → Analyze(simulatedFixture) → simulatedFindings
  → diff(Analyze(fixture), simulatedFindings) → SecurityDelta
  → estimateCost(fixture, simulatedFixture, delta) → CostForecast
```

### 1.2 In-scope delta operations (Phase 2)

Five operations are in scope:

| Operation | Rationale | Gate coverage |
|---|---|---|
| `AddSubnet` / `RemoveSubnet` | Structural topology change; cost-only delta for Phase 2 simulation (see SR-002 note below) | None — no NICs in new subnet; CIDR overlap is VNet-level `AddressSpace`, not subnet CIDR |
| `AddNSGRule` / `RemoveNSGRule` | Directly changes Gate 2 (declared → projected effective rules); most common what-if scenario | **Gate 2** |
| `AddPeering` / `RemovePeering` | Modifies `VNet.Peerings[]`; exercises CIDR overlap + segmentation checks once Phase 2 engine rules are wired (see SR-003 note below) | **Phase 1 limitation** — no current Phase 1 rule reads `VNet.Peerings[]` for intra-subscription topology; primarily useful with Phase 2 analysis rules |
| `AddPublicIP` / `RemovePublicIP` | Directly toggles Gate 4 (`nic.PublicIP != nil`); highest-impact single-resource change | **Gate 4** |
| `ModifyRoute` | Changes Gate 3 (`0.0.0.0/0 → NextHopType`); models adding/removing forced tunnelling, NVA insertion | **Gate 3** |

**Why only these five?** They are the minimal closure over the engine's four-gate reachability analysis. Every gate — AVNM (Gate 1), NSG effective rules (Gate 2), default route (Gate 3), public IP (Gate 4) — has at least one delta type that modifies its inputs. This makes the simulation meaningful without requiring NW re-query for every change.

> **SR-002 — AddSubnet security simulation limitation:** `AddSubnet` creates a subnet with no NICs (adding NICs is out of scope). The Phase 1 engine only fires findings for NICs, so `AddSubnet` alone produces zero `SecurityDelta`. CIDR overlap detection is VNet `AddressSpace`-level — adding a subnet does not change `VNet.AddressSpace`. `AddSubnet` is retained in Phase 2 because it is the trigger for NAT Gateway variable cost estimation (§7.1) and models the structural prerequisite for future NIC additions. Callers should expect `SecurityDelta.AddedRisks = []` and `MitigatedRisks = []` for pure `AddSubnet` operations. This limitation is lifted when `AddNIC` is introduced in Phase 3.

> **SR-003 — AddPeering security simulation limitation:** `AddPeeringOp` writes to `VNet.Peerings[]` within `ResourceGraph.VirtualNetworks`. The Phase 1 engine's `checkCrossSubPeeringExposure` reads `Fixture.CrossSubscriptionPeerings` (a separate top-level field, not `VNet.Peerings[]`). Consequently, intra-subscription `AddPeering`/`RemovePeering` deltas produce zero `SecurityDelta` against the Phase 1 engine. Phase 2 analysis rules that read `VNet.Peerings[]` for intra-VNet segmentation are planned; until those rules exist, `AddPeering`/`RemovePeering` is useful primarily for `forecast_cost` (cross-region peering egress cost, §7.1). Cross-subscription peering changes (`CrossSubscriptionPeerings`) are explicitly out of scope (§1.3).

### 1.3 Explicitly out-of-scope for Phase 2

The following changes are **not** modelled in Phase 2:

| Out-of-scope change | Reason |
|---|---|
| Add/remove NIC or VM | Requires new NW effective rule/route calls; cannot be simulated without live data or a full NW model |
| Add/remove NSG association (subnet↔NSG) | Correct simulation requires re-deriving effective rules; safe to defer to Phase 3 (see §4 note) |
| AVNM Admin Rule changes | AVNM rules require the full REST walk to resolve Network Group membership; in-scope for Phase 3 |
| Azure Firewall rule changes | Policy-based firewalls have hundreds of rules; Phase 3 with Firewall Policy diff model |
| VNet address space resize | CIDR overlap detection works on existing `AddressSpace` slices; resize changes CIDR topology globally |
| Cross-subscription peering changes | `CrossSubscriptionPeerings` have no live NW backing — no safe simulation path |
| VM SKU change | Affects bandwidth caps for variable cost only; not a security-gate input; Phase 3 |
| AKS private cluster toggle | Requires cluster API server re-provisioning; not a network delta |

All out-of-scope operations MUST return a structured `UnsupportedDeltaError` — never silently ignore them.

---

## 2. Delta Schema

### 2.1 `TopologyDelta`

```go
// TopologyDelta is the input to ApplyDelta. It describes a single proposed
// change to a topology. All fields are optional; exactly one operation field
// must be non-nil (enforced by Validate()). Multiple changes are not batched
// into a single delta — callers issue one delta per change and diff the results.
//
// Rationale for single-change-per-delta: the engine is deterministic and fast
// (~1ms). Multiple changes are better modelled as sequential applications
// (delta1 → fixture2, delta2 → fixture3) so callers see the incremental risk
// impact of each change. Batching hides which individual change caused a new finding.
type TopologyDelta struct {
    // AddSubnet adds a new subnet to an existing VNet.
    // The VNet named by TargetVNet must already exist in the fixture.
    // NSG and RouteTable may be empty strings (subnet with no NSG/RT = wide open).
    AddSubnet *AddSubnetOp `json:"addSubnet,omitempty"`

    // RemoveSubnet removes a named subnet from an existing VNet.
    // NICs that reference the removed subnet are not removed — they become
    // orphaned (Subnet field no longer resolvable). The engine will
    // still analyse them; callers should be aware of this.
    RemoveSubnet *RemoveSubnetOp `json:"removeSubnet,omitempty"`

    // AddNSGRule adds a declared security rule to an existing NSG.
    // Existing effective rules in NetworkWatcher are not modified;
    // the new rule is projected into simulated effective rules by §4.
    AddNSGRule *AddNSGRuleOp `json:"addNsgRule,omitempty"`

    // RemoveNSGRule removes a declared security rule from an existing NSG
    // by name. The rule is also removed from projected effective rules.
    RemoveNSGRule *RemoveNSGRuleOp `json:"removeNsgRule,omitempty"`

    // AddPeering adds a new VNet peering from LocalVNet to RemoteVNet.
    // The engine checks CIDR overlap and segmentation across peered VNets.
    // Both VNets must already exist in the fixture.
    AddPeering *AddPeeringOp `json:"addPeering,omitempty"`

    // RemovePeering removes an existing VNet peering by remote VNet name.
    RemovePeering *RemovePeeringOp `json:"removePeering,omitempty"`

    // AddPublicIP attaches a named PIP to a NIC. The NIC must exist.
    // After application, nic.PublicIP becomes non-nil → Gate 4 may fire.
    AddPublicIP *AddPublicIPOp `json:"addPublicIP,omitempty"`

    // RemovePublicIP detaches the PIP from a NIC.
    // After application, nic.PublicIP becomes nil → Gate 4 will not fire.
    RemovePublicIP *RemovePublicIPOp `json:"removePublicIP,omitempty"`

    // ModifyRoute changes the next hop type of a named route in a named
    // route table. Modifying the 0.0.0.0/0 default route is the primary use
    // case — e.g., changing Internet → VirtualAppliance to model NVA insertion.
    // ApplyDelta updates both RouteTable.Routes and the projected effective routes
    // for all NICs in subnets associated with the modified route table (see §4).
    ModifyRoute *ModifyRouteOp `json:"modifyRoute,omitempty"`
}
```

### 2.2 Operation structs

```go
type AddSubnetOp struct {
    VNetName      string `json:"vnetName"`      // must match an existing VNet.Name
    Name          string `json:"name"`           // new subnet name; must not already exist in VNet
    AddressPrefix string `json:"addressPrefix"`  // CIDR, e.g. "10.1.5.0/24"
    NSGName       string `json:"nsgName"`        // bare NSG name; "" = no NSG
    RouteTableName string `json:"routeTableName"` // bare RT name; "" = no route table
}

type RemoveSubnetOp struct {
    VNetName   string `json:"vnetName"`   // must match existing VNet.Name
    SubnetName string `json:"subnetName"` // must match existing Subnet.Name in that VNet
}

type AddNSGRuleOp struct {
    NSGName string        `json:"nsgName"` // must match existing NSG.Name
    Rule    graph.SecRule `json:"rule"`    // rule to add; Name must be unique within NSG
}

type RemoveNSGRuleOp struct {
    NSGName  string `json:"nsgName"`  // must match existing NSG.Name
    RuleName string `json:"ruleName"` // must match SecRule.Name in NSG.SecurityRules
}

type AddPeeringOp struct {
    LocalVNet             string `json:"localVnet"`              // must exist in fixture
    RemoteVNet            string `json:"remoteVnet"`             // must exist in fixture
    State                 string `json:"state"`                  // "Connected" | "Initiated"
    AllowForwardedTraffic bool   `json:"allowForwardedTraffic"`
    AllowGatewayTransit   bool   `json:"allowGatewayTransit"`
    UseRemoteGateways     bool   `json:"useRemoteGateways"`
}

type RemovePeeringOp struct {
    LocalVNet  string `json:"localVnet"`  // must exist in fixture
    RemoteVNet string `json:"remoteVnet"` // peering to this VNet is removed
}

type AddPublicIPOp struct {
    NICName   string `json:"nicName"`   // must match existing NIC.Name
    PIPName   string `json:"pipName"`   // new PIP resource name
    IPAddress string `json:"ipAddress"` // simulated IP address (e.g. "20.10.10.50")
}

type RemovePublicIPOp struct {
    NICName string `json:"nicName"` // must match existing NIC.Name; PIP is detached
}

type ModifyRouteOp struct {
    RouteTableName   string `json:"routeTableName"`   // must match existing RouteTable.Name
    RouteName        string `json:"routeName"`         // must match existing Route.Name in that RT
    NewNextHopType   string `json:"newNextHopType"`    // e.g. "VirtualAppliance" | "Internet" | "None"
    NewNextHopIP     string `json:"newNextHopIp"`      // required when NewNextHopType == "VirtualAppliance"
}
```

### 2.3 Delta validation

```go
// Validate returns an error if the delta is structurally invalid:
// - exactly one operation field is non-nil
// - required string fields are non-empty
// - CIDR prefix parses correctly (AddSubnet)
// - NextHopType is a known value (ModifyRoute)
// Validate does NOT check that named resources exist — that is ApplyDelta's responsibility.
func (d TopologyDelta) Validate() error
```

---

## 3. Apply-Delta Function

### 3.1 Signature

```go
// ApplyDelta applies a single topology delta to a fixture and returns a new,
// independent fixture. The original fixture is never mutated.
//
// Returns an error if:
//   - delta.Validate() fails
//   - the target resource (NSG, NIC, VNet, RouteTable) does not exist in the fixture
//   - adding a resource would create a duplicate name (AddSubnet, AddNSGRule, AddPeering)
//
// After applying the structural change, ApplyDelta calls projectEffectiveRules and
// projectEffectiveRoutes to update NetworkWatcher fields for affected NICs (see §4).
func ApplyDelta(fixture *graph.Fixture, delta TopologyDelta) (*graph.Fixture, error)
```

### 3.2 Immutability contract

`ApplyDelta` must **never mutate the original fixture**. This is required because:

1. The caller computes `Analyze(original)` and `Analyze(simulated)` and diffs them. If the original
   is mutated in place, the diff is meaningless.
2. Multiple concurrent `simulate_change` calls from different MCP clients may share a cached fixture.
   Mutation would cause a data race.

**Copy strategy — JSON round-trip:**

```go
func deepCopy(src *graph.Fixture) (*graph.Fixture, error) {
    b, err := json.Marshal(src)
    if err != nil {
        return nil, err
    }
    var dst graph.Fixture
    if err := json.Unmarshal(b, &dst); err != nil {
        return nil, err
    }
    return &dst, nil
}
```

The JSON round-trip is chosen over manual struct copying for correctness and maintainability.
`graph.Fixture` has deeply nested slices, pointer fields (`*string`, `*Firewall`, `*Enrichment`),
and maps (`map[string][]SecRule`, `map[string][]Route`) — any manual copy that misses a field
produces a subtle alias bug. JSON round-trip is O(size of fixture) but fixtures are bounded
(~50–500 KB for large subscriptions), making the cost negligible compared to the NW API latency
already incurred to fetch the original.

**Alternative considered — sync.Mutex cache + copy-on-write:**
A copy-on-write pool would avoid re-serialisation for each `simulate_change` call. Deferred to
Phase 3 if profiling shows JSON round-trip is a bottleneck (expected only for subscriptions with
>5,000 NICs).

### 3.3 Per-operation apply logic

#### AddSubnetOp
1. Find `VNet` by `VNetName`; error if not found.
2. Check that no existing subnet in that VNet has `Name == op.Name`; error if duplicate.
3. Append `graph.Subnet{Name, AddressPrefix, NSGName, RouteTableName}` to `VNet.Subnets`.
4. No NICs reference the new subnet yet → no effective rule/route projection needed.

#### RemoveSubnetOp
1. Find `VNet` by `VNetName`; error if not found.
2. Find subnet index by `SubnetName`; error if not found.
3. Remove from `VNet.Subnets` slice (order-preserving removal).
4. Log a warning for any NICs whose `Subnet` field matches `"{VNetName}/{SubnetName}"` — they
   become orphaned. Do not remove them (engine still analyses them; callers see the orphan).

#### AddNSGRuleOp
1. Find `NSG` by `NSGName`; error if not found.
2. Check that no existing rule has `Name == op.Rule.Name`; error if duplicate.
3. Append `op.Rule` to `NSG.SecurityRules`.
4. Call `projectEffectiveRules(simFixture, nsgName)` to update `NetworkWatcher.EffectiveSecurityRules`
   for all NICs that the NSG governs (see §4).

#### RemoveNSGRuleOp
1. Find `NSG` by `NSGName`; error if not found.
2. Find rule index by `RuleName`; error if not found.
3. Remove from `NSG.SecurityRules`.
4. Call `projectEffectiveRules(simFixture, nsgName)`.

#### AddPeeringOp
1. Find both `LocalVNet` and `RemoteVNet`; error if either not found.
2. Check that `LocalVNet.Peerings` does not already contain a peering to `RemoteVNet`; error if duplicate.
3. Append `graph.Peering{RemoteVnet: op.RemoteVNet, State: op.State, ...}` to `LocalVNet.Peerings`.
4. No effective rule/route changes — peerings affect CIDR overlap and segmentation check logic in `Analyze`,
   which reads `Fixture.ResourceGraph.VirtualNetworks[].Peerings` directly. No NW projection needed.

#### RemovePeeringOp
1. Find `LocalVNet`; error if not found.
2. Find peering index where `RemoteVnet == op.RemoteVNet`; error if not found.
3. Remove from `LocalVNet.Peerings`.

#### AddPublicIPOp
1. Find `NIC` by `NICName`; error if not found.
2. Check `NIC.PublicIP == nil`; error if already has a PIP (callers must RemovePublicIP first).
3. Set `NIC.PublicIP = &op.PIPName`.
4. Append `graph.PublicIP{Name: op.PIPName, IPAddress: op.IPAddress, IPConfiguration: &op.NICName}`
   to `ResourceGraph.PublicIPAddresses` (prevents spurious orphaned-PIP finding on the new IP).
5. No NW projection needed — Gate 4 reads `nic.PublicIP` directly.

#### RemovePublicIPOp
1. Find `NIC` by `NICName`; error if not found.
2. Record the current `*NIC.PublicIP` name; set `NIC.PublicIP = nil`.
3. If the detached PIP exists in `ResourceGraph.PublicIPAddresses`, set its `IPConfiguration = nil`
   (turns it into an orphaned-PIP finding in the simulation — correct behaviour, the PIP is now floating).

#### ModifyRouteOp
1. Find `RouteTable` by `RouteTableName`; error if not found.
2. Find `Route` by `RouteName` within `RouteTable.Routes`; error if not found.
3. Update `Route.NextHopType = op.NewNextHopType` and `Route.NextHopIPAddress = op.NewNextHopIP`.
4. Call `projectEffectiveRoutes(simFixture, routeTableName)` for all affected NICs (see §4).

---

## 4. Effective-Rule Projection in Simulation Mode

### 4.1 The challenge

The Phase 1 engine reads **evaluated truth** from `NetworkWatcher.EffectiveSecurityRules` and
`NetworkWatcher.EffectiveRoutes` — data produced by live Azure NW API calls. These are per-NIC
snapshots captured at fetch time.

When a delta modifies `NSG.SecurityRules` or `RouteTable.Routes` (declared config), the effective
data in `NetworkWatcher` does not automatically update — it still reflects the pre-delta state.
Without projection, `Analyze(simulatedFixture)` would produce identical results to `Analyze(fixture)`
for NSG and Route delta operations, making the simulation useless.

**Projection** is the process of deriving simulated effective rules/routes from the modified
declared config, so the engine sees the correct post-delta evaluated state.

### 4.2 Effective-rule projection (`projectEffectiveRules`)

```
projectEffectiveRules(fixture *graph.Fixture, nsgName string)
```

**Algorithm:**

1. Find all NICs governed by `nsgName`. A NIC is governed by an NSG if:
   a. The NIC's subnet (`NIC.Subnet = "{vnetName}/{subnetName}"`) resolves to a subnet with
      `Subnet.NetworkSecurityGroup == nsgName` — subnet-level NSG association.
   b. `NIC.NetworkSecurityGroup != nil && *NIC.NetworkSecurityGroup == nsgName` — NIC-level NSG association.

2. Build the set of **declared rule names** for this NSG from the _original_ `NSG.SecurityRules`:
   ```go
   declaredNames := map[string]bool{}
   for _, r := range originalNSG.SecurityRules {
       declaredNames[r.Name] = true
   }
   ```

3. For each affected NIC, rebuild simulated effective rules:
   a. **Start with the current effective rules** from `NetworkWatcher.EffectiveSecurityRules[nicName]`
      as the base (captures system defaults: AllowVnetInBound priority 65000, DenyAllInBound priority 65500, etc.)
   b. **Remove** any effective rule that satisfies BOTH conditions:
      - `rule.Name` ∈ `declaredNames` (the rule came from this NSG's declared set), AND
      - `rule.Priority < 65000` (user-defined range — system defaults are ≥ 65000).
      This avoids stripping system-default rules whose names coincidentally match a user-defined
      rule name. Azure ARM prevents duplicate names within an NSG but does not prevent user rule
      names from matching system default names (e.g. a user rule named "AllowVnetInBound" at
      priority 200 would coexist with the system default of the same name at priority 65000 in
      the effective set). Stripping by name alone would incorrectly remove the system default.
   c. **Re-add** all rules from the _modified_ `NSG.SecurityRules`.
      (Injects the post-delta declared rules.)
   d. **Sort** the merged set by `Priority` ascending (lower priority number = higher precedence).
      Rules with the same priority retain their original relative order (stable sort).
   e. Write the result back to `NetworkWatcher.EffectiveSecurityRules[nicName]`.

**Correctness guarantee:**
This projection is correct for declared NSG rule changes. It preserves system defaults and
the effective rules contributed by other NSGs (subnet NSG or NIC NSG — whichever was _not_ modified).

**Known approximation:**
Azure evaluates both subnet-level and NIC-level NSG rules, producing a merged effective set.
If a NIC has both a subnet NSG and a NIC-level NSG, and only one is modified, the projection
correctly preserves the other NSG's rules (they remain in the base effective set from step 2a).
However, the projection does **not** account for AVNM Admin Rules being re-evaluated against
the modified NSG — AVNM override logic runs in `adminVerdict()` against the admin rule list,
which is unchanged (AVNM deltas are out of scope). This is acceptable because `adminVerdict()`
reads `AVNM.SecurityAdminRules` independently of `EffectiveSecurityRules`.

### 4.3 Effective-route projection (`projectEffectiveRoutes`)

```
projectEffectiveRoutes(fixture *graph.Fixture, routeTableName string)
```

**Algorithm:**

1. Find all subnets associated with `routeTableName` via `RouteTable.AssociatedSubnets`.
2. Find all NICs in those subnets: `NIC.Subnet == "{vnetName}/{subnetName}"` for each associated subnet.
3. For each affected NIC, rebuild simulated effective routes:
   a. Start with the current effective routes from `NetworkWatcher.EffectiveRoutes[nicName]` as base.
   b. For each route in the modified `RouteTable.Routes`, find any existing effective route with the same
      `AddressPrefix` and replace it (declared UDRs take precedence over system routes for the same prefix).
      If no existing effective route has the same prefix, append.
   c. Write the result back to `NetworkWatcher.EffectiveRoutes[nicName]`.

**Critical path — `0.0.0.0/0`:**
The engine's Gate 3 check is:
```go
if r.AddressPrefix == "0.0.0.0/0" { defaultHop = r.NextHopType }
```
A `ModifyRoute` on the `0.0.0.0/0` route modifies `defaultHop` for all NICs in the associated
subnets. The projection correctly replaces the existing `0.0.0.0/0` effective route entry.

**Known limitation — BGP-learned routes:**
Live effective routes include BGP-learned routes from VPN/ER gateways. These are not in
`RouteTable.Routes` (declared UDRs). The projection does not modify BGP-learned routes —
they remain as fetched. If a proposed change would affect BGP route propagation
(e.g., `disableBgpRoutePropagation` toggle), that operation is out of scope for Phase 2.

---

## 5. Security Delta

### 5.1 `SecurityDelta` struct

```go
// SecurityDelta is the result of comparing Analyze(original) with Analyze(simulated).
// It makes the security impact of the delta explicit and actionable.
type SecurityDelta struct {
    // AddedRisks contains findings that appear in the simulated topology but
    // not in the original. These are new risks introduced by the change.
    AddedRisks []analyze.Finding `json:"addedRisks"`

    // MitigatedRisks contains findings that appear in the original topology but
    // not in the simulated. These are risks resolved by the change.
    MitigatedRisks []analyze.Finding `json:"mitigatedRisks"`

    // Unchanged contains findings present in both. Populated only when the
    // caller passes ?include_unchanged=true; omitted by default.
    Unchanged []analyze.Finding `json:"unchanged,omitempty"`

    // OriginalFindingCount is the total number of findings before the change.
    OriginalFindingCount int `json:"originalFindingCount"`

    // SimulatedFindingCount is the total number of findings after the change.
    SimulatedFindingCount int `json:"simulatedFindingCount"`

    // RiskVector summarises the net change in severity buckets.
    RiskVector RiskVector `json:"riskVector"`
}

// RiskVector captures net changes in finding severity counts.
// Positive = more findings in that severity after the change (risk increase).
// Negative = fewer findings in that severity after the change (risk reduction).
type RiskVector struct {
    CriticalDelta      int `json:"criticalDelta"`
    HighDelta          int `json:"highDelta"`
    MediumDelta        int `json:"mediumDelta"`
    LowDelta           int `json:"lowDelta"`
    InformationalDelta int `json:"informationalDelta"`
}
```

### 5.2 Diff algorithm

Finding equality for diffing: two findings are **equal** if `Type + Resource + Severity` all match.
Evidence is **excluded** from the diff key because Evidence strings are dynamically constructed in
`analyze.go` and can legitimately differ between two analyses of the same topology (e.g. a latent
finding's reason text changes when `ModifyRoute` alters the `defaultHop` value — "route 0.0.0.0/0→VirtualAppliance"
vs "route 0.0.0.0/0→None" — without changing the finding's severity or the underlying risk). Including
Evidence in the diff key would produce spurious MitigatedRisk + AddedRisk pairs for the same finding
with changed contextual text — a false positive diff.

Severity **is** included in the equality key because a change that escalates a finding from High to Critical
appears as: one MitigatedRisk (original High) + one AddedRisk (new Critical). This correctly surfaces
severity escalation as an added risk.

Evidence is retained in the `Finding` struct (available for display) but is not used for equality.

```go
func DiffFindings(original, simulated []analyze.Finding) SecurityDelta
```

**Implementation:** Build a `map[string]analyze.Finding` keyed by `Type+"|"+Resource+"|"+Severity`
for each set, then compute set difference. O(n) in finding count.

---

## 6. Cost Model — Fixed Costs

### 6.1 Principle

Fixed costs use the **Azure Retail Prices API** — a public, unauthenticated REST endpoint that
returns list prices by SKU. The agent fetches prices at analysis time and caches them for the
duration of the session (not persisted — prices change monthly).

**API endpoint:**
```
GET https://prices.azure.com/api/retail/prices?api-version=2023-01-01-preview
    &$filter=<OData filter>
```

**Response schema (relevant fields):**
```json
{
  "Items": [
    {
      "currencyCode": "USD",
      "retailPrice": 1234.56,
      "unitPrice": 1234.56,
      "unitOfMeasure": "1 Month",
      "skuName": "VpnGw1 Gateway",
      "productName": "VPN Gateway",
      "serviceName": "VPN Gateway",
      "armRegionName": "eastus",
      "type": "Consumption"
    }
  ],
  "NextPageLink": null
}
```

Use `unitPrice` (list price in USD, monthly). Filter `type == 'Consumption'` and
`unitOfMeasure == '1 Month'` for gateway/firewall SKUs.

### 6.2 Fixed-cost resources and Retail Prices API queries

#### VPN Gateway

**Trigger:** `ModifyRoute` that sets `NewNextHopType = "VirtualNetworkGateway"` (introducing forced
tunnelling through a gateway), OR `AddPeering` where `AllowGatewayTransit = true` or
`UseRemoteGateways = true` (gateway transit peering). A simple VNet-to-VNet `AddPeering` within the
same subscription does **not** provision a gateway and does not incur gateway costs — gateway cost
only applies when the delta explicitly introduces a gateway transit path.

> **SR-003 reminder:** intra-subscription `AddPeering` without `AllowGatewayTransit`/`UseRemoteGateways`
> has zero gateway cost impact. Cross-subscription or on-premises VPN/ER gateway costs require the
> `VirtualNetworkGateway` struct to be populated in the fixture (TMR-001).

**Retail Prices API filter:**
```
serviceName eq 'VPN Gateway'
and armRegionName eq '{region}'
and skuName eq '{SKUName} Gateway'
and type eq 'Consumption'
and unitOfMeasure eq '1 Month'
```

**SKU name examples:** `"Basic Gateway"`, `"VpnGw1 Gateway"`, `"VpnGw2 Gateway"`, `"VpnGw3 Gateway"`,
`"VpnGw1AZ Gateway"`, `"VpnGw2AZ Gateway"`, `"VpnGw3AZ Gateway"`, `"VpnGw4AZ Gateway"`, `"VpnGw5AZ Gateway"`.

**SKU source:** `VirtualNetworkGateway.SKU` — populated by Phase 2 adapter extension (see §11).

**Active-active multiplier:** If `VirtualNetworkGateway.ActiveActive == true`, multiply cost ×2
(two gateway instances billed separately).

**Cost in delta:** `fixed_delta_usd = (new gateway price) - (removed gateway price)`.
If no gateway is being added or removed, this component is $0.

---

#### ExpressRoute Gateway

**Trigger:** same as VPN Gateway but `GatewayType == "ExpressRoute"`.

**Retail Prices API filter:**
```
serviceName eq 'ExpressRoute'
and productName eq 'ExpressRoute Gateway'
and armRegionName eq '{region}'
and skuName eq '{SKUName} Gateway'
and type eq 'Consumption'
and unitOfMeasure eq '1 Month'
```

**SKU name examples:** `"ErGw1AZ Gateway"`, `"ErGw2AZ Gateway"`, `"ErGw3AZ Gateway"`,
`"UltraPerformance Gateway"`.

---

#### Azure Firewall

**Trigger:** No delta directly creates/destroys firewalls (out of scope in Phase 2). However,
`forecast_cost` reports the **existing** firewall cost as context — not a delta cost.

**Retail Prices API filter (existing firewall cost reporting):**
```
serviceName eq 'Azure Firewall'
and armRegionName eq '{region}'
and skuName eq '{SKUTier}'
and type eq 'Consumption'
and unitOfMeasure eq '1 Month'
```

**SKU tier examples:** `"Basic"` (≈$300/mo), `"Standard"` (≈$1,500/mo), `"Premium"` (≈$2,500/mo).

**Source:** `Fixture.AzureFirewall.SKUTier` — populated by Phase 2 adapter extension (see §11, TMR-004).

---

#### Public IP Address

**Trigger:** `AddPublicIPOp` (cost increase) or `RemovePublicIPOp` (cost decrease).

**Retail Prices API filter:**
```
serviceName eq 'Virtual Network'
and productName eq 'IP Addresses'
and armRegionName eq '{region}'
and skuName eq '{SKUAndAllocation}'
and type eq 'Consumption'
and unitOfMeasure eq '1 Month'
```

**SKU name examples:**
- `"Basic Dynamic IPv4"` → $0/mo when attached (free when allocated)
- `"Basic Static IPv4"` → ≈$3.00/mo
- `"Standard Static IPv4"` → ≈$3.65/mo
- Zone-redundant: `"Standard Static IPv4 Zone Redundant"` → ≈$5.84/mo

**Source:** `PublicIP.AllocationMethod` + `PublicIP.SKU` — populated by Phase 2 adapter extension
(see §11, TMR-005).

---

#### Private Endpoint

**Trigger:** Not a delta operation in Phase 2 (AddPrivateEndpoint is out of scope). However,
existing Private Endpoints have a fixed monthly charge that `forecast_cost` reports as context.

**Retail Prices API filter:**
```
serviceName eq 'Azure Private Link'
and productName eq 'Private Endpoint'
and armRegionName eq '{region}'
and type eq 'Consumption'
and unitOfMeasure eq '744 Hours'
```

**Current price:** ≈$7.30/mo per endpoint (≈$0.01/hr × 730 hours).

**Source:** `Fixture.ResourceGraph.PrivateEndpoints` count — existing Phase 1 data.

---

### 6.3 Price caching

```go
// PriceCache stores Retail Prices API responses for a session.
// TTL: session-scoped (cache cleared on process restart).
// Invalidation: not implemented in Phase 2 (prices change monthly, not hourly).
type PriceCache struct {
    mu    sync.RWMutex
    items map[string]float64 // key: "serviceName|skuName|region" → monthly USD price
}
```

Prices are fetched lazily on first request and cached. A `price_source_date` string is stored
alongside each price entry (from the API response `effectiveStartDate`) and propagated to
`CostForecast.PriceSourceDate`.

---

## 7. Cost Model — Variable Costs

### 7.1 What variable costs cover

Variable costs are **data transfer charges** that change when the topology changes. They are
estimated from VNet Flow Log traffic volumes, not from a fixed price schedule.

Three categories are modelled:

| Category | Trigger delta | Azure pricing basis |
|---|---|---|
| Azure Firewall data processing | `ModifyRoute` that routes traffic through a Firewall | Per GB processed; Standard ≈$0.016/GB, Premium ≈$0.016/GB |
| Cross-region VNet peering transfer | `AddPeering` where `IsGlobalPeering == true` | Per GB, varies by region pair; ≈$0.02–$0.05/GB |
| NAT Gateway data processing | `AddSubnet` with a NAT Gateway association | Per GB processed; ≈$0.045/GB |

### 7.2 Traffic volume estimation from VNet Flow Logs

**Source:** `Fixture.Enrichment.FlowLogStatuses` + Traffic Analytics workspace data.

If Flow Logs are enabled for the affected NSG/VNet, the variable cost estimate uses the observed
average daily byte count from the last 30 days of Traffic Analytics data.

**Traffic Analytics API:**
```
POST https://management.azure.com/subscriptions/{sub}/resourceGroups/{la-rg}/providers/
     Microsoft.OperationalInsights/workspaces/{workspace}/api/query
     ?api-version=2017-01-01-preview
Body:
{
  "query": "AzureNetworkAnalytics_CL | where TimeGenerated > ago(30d) | summarize TotalBytes = sum(BytesSent_d) by NSGName_s"
}
```

If Flow Logs are **not** enabled (detected via `FlowLogSummary.Enabled == false`), the estimate
falls back to a **subscription-average heuristic**:

```
EstimatedMonthlyGB = avgNICCountInSubnet × 50 GB/NIC/month
```

50 GB/NIC/month is the AT&T AT&T estate empirical baseline (P50 across workloads, from 2025
Traffic Analytics data). This heuristic is the primary driver of uncertainty — see §7.4.

### 7.3 Variable cost formula

```
VariableMonthlyCost_USD = EstimatedMonthlyGB × PricePerGB × ProcessingFactor
```

Where:
- `EstimatedMonthlyGB`: from flow log or heuristic (see §7.2)
- `PricePerGB`: from Retail Prices API (firewall processing, peering egress, or NAT GW)
- `ProcessingFactor`: 1.0 for single-path; 2.0 for active-active gateway configurations

### 7.4 Uncertainty band and driving factors

**Tolerance band: ±30%** of the point estimate.

```
variable_delta_usd_low  = point_estimate × 0.70
variable_delta_usd_high = point_estimate × 1.30
```

| Uncertainty factor | Magnitude | Notes |
|---|---|---|
| Traffic volume seasonality | ±20% | Monthly variance for enterprise workloads |
| Flow Log gap (disabled segments) | Up to ±50% | Blind segments use heuristic; actual may differ materially |
| Pricing tier (committed use vs pay-go) | ±15% | AT&T may have EA/MCA discounts not reflected in Retail Prices API |
| New subnet/peering ramp-up | Variable | Traffic to a new subnet may start near zero and grow; model assumes steady-state |

When `FlowLogSummary.Enabled == false` for the affected segment, the band widens to **±50%**
and a caveat is added to `CostForecast.Caveats`:
`"Variable cost estimated from subscription heuristic — Flow Logs not enabled for {resource}; enable Flow Logs for a tighter estimate."`

---

## 8. `CostForecast` Output Schema

```go
// CostForecast is the output of forecast_cost for a given TopologyDelta.
// Fixed costs are exact (Retail Prices API list price); variable costs are a band.
//
// All amounts are monthly USD. Positive = cost increase. Negative = cost decrease.
type CostForecast struct {
    // FixedDeltaUSD is the exact monthly cost change from fixed-price resources
    // (gateway SKU changes, PIP additions/removals). Derived from Retail Prices API
    // list prices. Value is exact within the list-price definition; EA/MCA contract
    // discounts are not applied.
    FixedDeltaUSD float64 `json:"fixedDeltaUsd"`

    // VariableDeltaUSDLow is the lower bound of the variable cost change band.
    // Computed as: point_estimate × (1 - UncertaintyFactor).
    VariableDeltaUSDLow float64 `json:"variableDeltaUsdLow"`

    // VariableDeltaUSDHigh is the upper bound of the variable cost change band.
    // Computed as: point_estimate × (1 + UncertaintyFactor).
    VariableDeltaUSDHigh float64 `json:"variableDeltaUsdHigh"`

    // ConfidenceBandPct is the uncertainty factor as a percentage (e.g., 30 or 50).
    // 30 = ±30% band (flow log data available).
    // 50 = ±50% band (heuristic estimate, no flow logs).
    ConfidenceBandPct int `json:"confidenceBandPct"`

    // PriceSourceDate is the effectiveStartDate of the Retail Prices API entries
    // used for this forecast. Format: "YYYY-MM-DD". Populated from the oldest
    // effectiveStartDate across all price entries fetched.
    PriceSourceDate string `json:"priceSourceDate"`

    // ExistingFixedMonthlyUSD is the current monthly fixed cost of the topology
    // (firewalls, PIPs, Private Endpoints) before the delta.
    // List price only — derived from Azure Retail Prices API; EA/MCA contract
    // discounts and commitment savings are not applied. Do not present this value
    // as actual billed spend. See PriceSourceDate and CostLineItem.PriceSource.
    // Informational context only — not part of the delta forecast.
    ExistingFixedMonthlyUSD float64 `json:"existingFixedMonthlyUsd"`

    // LineItems breaks down the fixed delta into per-resource components.
    // Each item represents one billable resource change.
    LineItems []CostLineItem `json:"lineItems"`

    // Caveats is a list of machine-readable advisory strings that explain
    // estimate limitations. Surfaced in Markdown and Draw.io outputs.
    Caveats []string `json:"caveats"`
}

// CostLineItem is one billable item in the fixed cost breakdown.
type CostLineItem struct {
    Resource    string  `json:"resource"`     // resource name (e.g. "pip-new-web", "vpngw-hub")
    ResourceType string `json:"resourceType"` // "PublicIP" | "VPNGateway" | "ExpressRouteGateway" | "AzureFirewall" | "PrivateEndpoint"
    ChangeType  string `json:"changeType"`   // "Add" | "Remove" | "Existing"
    SKU         string `json:"sku"`          // e.g. "Standard Static IPv4", "VpnGw1 Gateway"
    MonthlyUSD  float64 `json:"monthlyUsd"`  // positive = cost added, negative = cost removed
    Region      string  `json:"region"`
    PriceSource string  `json:"priceSource"` // "retail-prices-api"
}
```

### 8.1 Mandatory caveats

The following caveats are **always** appended regardless of input:

1. `"Fixed costs reflect Azure Retail Prices API list prices; EA/MCA contract discounts are not applied."`
2. `"Variable costs are estimated from traffic volume data and may not reflect actual billing."`
3. `"Costs reflect the change introduced by the delta; they do not represent the total topology cost."`

---

## 9. Azure Cost MCP Integration Boundary

### 9.1 Two separate data sources

| Source | What it provides | Used in |
|---|---|---|
| **Azure Retail Prices API** (this agent) | List prices for new/changed resources | `CostForecast.FixedDeltaUSD`, `CostLineItem.MonthlyUSD` |
| **Azure Cost MCP** (`azure-ops` skill) | Actual billed costs for existing resources (Azure Cost Management API) | Actuals reconciliation — separate tool invocation |

These two sources **must not be conflated** in any output, report, or UI:

- The Retail Prices API returns **list prices** — pre-discount, pre-reservation, no commitment discounts.
- Azure Cost Management returns **actual charges** — post-discount, post-reservation, after EA/MCA terms.
- A topology change that costs $500/mo at list price may cost $300/mo under AT&T's EA agreement.

### 9.2 Actuals reconciliation flow

Actuals reconciliation is a **separate tool call**, not part of `forecast_cost`.

```
Step 1: forecast_cost(subscriptionID, delta)
         → CostForecast { fixedDeltaUsd: 365.00, ... priceSource: "retail-prices-api" }

Step 2 (separate, human-initiated): call Azure Cost MCP tool
         → ActualCostSummary { resourceGroup: "nettopo-rg", lastMonthUSD: 11,200.00 }

Step 3 (human comparison): apply forecast delta to actual baseline
         → Expected new monthly cost: 11,200 + 365 = 11,565 (at list)
         → Actual new cost will differ by EA discount factor (AT&T-specific)
```

### 9.3 Fields that must be labelled

Every field carrying a price value must include its source in the field name or a sibling field:

| Field | Source label |
|---|---|
| `CostForecast.FixedDeltaUSD` | `CostForecast.PriceSourceDate` + "retail-prices-api" in each `CostLineItem.PriceSource` |
| `CostForecast.ExistingFixedMonthlyUSD` | Same — list price only |
| Any actuals from Azure Cost MCP | Must be labelled `"source": "azure-cost-management"` in that tool's output |

The MCP tool `format_report` (Phase 1) must add a disclaimer when `CostForecast` is included in
Markdown or Draw.io output:
> **Cost estimates are based on Azure Retail Prices list rates. Actual billed amounts under EA/MCA agreements will differ. Contact the AT&T FinOps team for commitment-adjusted cost projections.**

### 9.4 What the Azure Cost MCP is used for (Phase 2)

| Use case | Tool | Notes |
|---|---|---|
| Baseline actual monthly spend for comparison | Azure Cost MCP `get_cost_summary` | Provides last-N-months actual by resource group or subscription |
| Anomaly detection (unexpected cost spike after a topology change) | Azure Cost MCP `get_cost_anomalies` | Triggered by the explainer layer post-change |
| Reserved Instance coverage check | Azure Cost MCP `get_ri_coverage` | Informs whether a new gateway is likely to be RI-covered |

---

## 10. Phase 2 Placeholder Reconciliation

The following table resolves every `// Phase 2` field identified in `phase-1/design/TOPOLOGY_MODEL.md §7`
and every `// Phase 2` annotation in `engine/go/internal/graph/model.go`.

| Reference | Field | Phase 2 disposition | Adapter action required |
|---|---|---|---|
| TMR-001 (§7.1) | `VirtualNetworkGateway.SKU`, `ActiveActive` | **Populated in Phase 2** — required for gateway fixed-cost forecast (§6.2). Note: TOPOLOGY_MODEL.md §7.1 refers to this struct as `VNetGateway`; the actual struct in `model.go` is `VirtualNetworkGateway`. | Populate `SKU` and `EnableForcedTunneling` fields from the existing VNG Resource Graph query |
| TMR-002 (§7.2) | `PrivateEndpoint` struct | **Already populated in Phase 1** — `PrivateEndpoint` struct exists in model.go and is collected by the Phase 1 adapter (`resourcegraph.go`). No action required. | None |
| TMR-003 (§7.3) | `NIC.VMSize` | **Deferred to Phase 3** — variable cost band (§7.3) uses per-subnet average, not per-NIC VM SKU, for Phase 2. VMSize adds a separate VM Resource Graph query; value is marginal vs band uncertainty. | Defer |
| TMR-004 (§7.4) | `Firewall.SKUTier` | **Populated in Phase 2** — already captured in §2.7 KQL (`sku = properties.sku.tier`) but not mapped to struct. | Add `SKUTier string` field to `graph.Firewall` struct; populate in `adapter/resourcegraph.go` |
| TMR-005 (§7.5) | `PublicIP.AllocationMethod`, `PublicIP.SKU` | **Populated in Phase 2** — already captured in §2.4 KQL but not mapped to struct. | Add `AllocationMethod string` and `SKU string` fields to `graph.PublicIP`; populate in adapter |
| TMR-006 (§7.6) | `Peering.RemoteVnetRegion`, `Peering.IsGlobalPeering` | **Populated in Phase 2** — computed from already-collected VNet location data (pure join, no new API calls). | Add fields to `graph.Peering`; compute in `adapter/azure.go` Step F assembly |
| TMR-007 (§7.7) | `RouteTable.DisableBgpRoutePropagation` | **Populated in Phase 2** — already captured in §2.3 KQL but not mapped to struct. Needed for `ModifyRoute` simulation correctness (BGP propagation affects effective route set). | Add `DisableBgpRoutePropagation bool` field to `graph.RouteTable`; populate in adapter |
| TMR-008 (§1.4) | `Peering.AllowForwardedTraffic`, `AllowGatewayTransit`, `UseRemoteGateways` | **Already populated in Phase 1** — per TOPOLOGY_MODEL.md §1.4 adapter note, these fields are collected in Phase 1 but not consumed by analysis rules. `AddPeeringOp` and `RemovePeeringOp` use these fields in Phase 2. | None — already in struct and populated |
| model.go comment | `DNSPrivateResolvers` | **Analysis rule deferred to Phase 3** — data collected by Phase 1 adapter, but hybrid DNS analysis (on-prem → Azure PE resolution path) is Phase 3 scope | None for Phase 2 |
| model.go comment | `AzureRouteServers` | **Analysis rule deferred to Phase 3** — Route Server BGP propagation affects effective routes; simulation model cannot safely handle without full BGP path modelling | None for Phase 2 |
| model.go comment | `DDoSProtectionPlans` | **Analysis rule deferred to Phase 3** — informational finding only; no impact on simulation delta | None for Phase 2 |
| model.go comment | `LocalNetworkGateways` | **Shadow-routing analysis deferred to Phase 3** — on-prem CIDR overlap with Azure VNets is a meaningful finding but not a simulation input | None for Phase 2 |
| model.go comment | `Enrichment.FlowLogStatuses` | **Consumed in Phase 2** — required for variable cost estimation (§7.2). Adapter must populate when `enrich=true` (already conditional in Phase 1 adapter). | Ensure `enrich=true` flow log collection is wired for `forecast_cost` calls |
| model.go comment | `AzureFrontDoors` | **Phase 1 analysis rule already active — model.go `// Phase 2` comment is stale.** `checkFrontDoorExposure` (rule 13) was added in Phase 1 and analyses `AzureFrontDoor.WAFEnabled`. The `// Phase 2` comment in `model.go` predates Phase 1 completion and must be removed in the Phase 2 model.go cleanup pass. No Phase 2 work required. | Remove stale `// Phase 2` comment from `AzureFrontDoors` in `model.go` |
| model.go comment | `Enrichment.DefenderAssessments`, `Enrichment.PolicyFindings` | **Deferred to Phase 3** — model.go documents Phase 2 intent for attack-path cross-correlation (e.g. internet-reachable NIC + Defender "MFA not enforced" → Critical escalation). This cross-correlation engine rule is not part of Phase 2 scope. Phase 2 only uses `FlowLogStatuses` for variable cost estimation. | Defer |

### 10.1 Model.go changes required for Phase 2

The following additions to `engine/go/internal/graph/model.go` are required before Phase 2
implementation begins. These are **additive-only** — no existing field is renamed or removed
(backwards compatible with all Phase 1 fixtures and tests):

```go
// Additions to graph.Firewall:
SKUTier string `json:"skuTier,omitempty"` // "Basic" | "Standard" | "Premium"

// Additions to graph.PublicIP:
AllocationMethod string `json:"allocationMethod,omitempty"` // "Static" | "Dynamic"
SKU              string `json:"sku,omitempty"`               // "Basic" | "Standard"

// Additions to graph.RouteTable:
DisableBgpRoutePropagation bool `json:"disableBgpRoutePropagation,omitempty"`

// Additions to graph.Peering (already has AllowForwardedTraffic etc.):
RemoteVnetRegion string `json:"remoteVnetRegion,omitempty"`
IsGlobalPeering  bool   `json:"isGlobalPeering,omitempty"`
```

All new fields are `omitempty` — Phase 1 fixtures and tests are unaffected by their absence.

---

## 11. Phase 2 Adapter Extension Plan

The following adapter extensions are required to populate the new fields in §10.1. Extensions
are additive — no existing adapter function is modified; new fields are appended in Step F assembly.

| Extension | File | Change |
|---|---|---|
| Populate `Firewall.SKUTier` | `adapter/resourcegraph.go` | Map `sku` column already present in §2.7 KQL to `Firewall.SKUTier` |
| Populate `PublicIP.AllocationMethod` + `SKU` | `adapter/resourcegraph.go` | Map `allocationMethod` and `sku` columns already present in §2.4 KQL |
| Populate `RouteTable.DisableBgpRoutePropagation` | `adapter/resourcegraph.go` | Map `disableBgpRoutePropagation` column already present in §2.3 KQL |
| Populate `Peering.RemoteVnetRegion` + `IsGlobalPeering` | `adapter/azure.go` Step F | After VNet list is assembled: for each peering, look up remote VNet location from `rg.vnetsById` map; compute `IsGlobalPeering = (localLocation != remoteLocation)` |
| Populate `VirtualNetworkGateway.SKU` + `EnableForcedTunneling` | `adapter/resourcegraph.go` | `VirtualNetworkGateway` struct already in model.go; ensure `SKU` and `EnableForcedTunneling` fields are populated from the existing VNG query |
| Ensure `FlowLogStatuses` populated for `forecast_cost` | `adapter/azure.go` | When the caller passes `Options.Enrich = true`, ensure the flow log enrichment path is executed before the fixture is returned |

No new Azure API calls are required for TMR-004, TMR-005, TMR-006, or TMR-007 — all data is
already retrieved by Phase 1 queries and was left un-mapped to structs. The only new API call
is the VNet Flow Log/Traffic Analytics query for variable cost estimation (§7.2), which is
conditional on `forecast_cost` being invoked.

---

---

## 12. Rubber-Duck Review Findings (Step 2.2)

> **Reviewer:** Copilot rubber-duck review (Step 2.2), applied inline.
> **Date:** 2026-06-12. All findings are resolved in this document revision.

| ID | Severity | Section | Finding | Resolution |
|---|---|---|---|---|
| SR-001 | **High** | §4.2 | `projectEffectiveRules` step 2b stripped by `Name` alone. Azure ARM prevents duplicate rule names _within_ an NSG but not naming a user rule the same as a system default (e.g. a rule named "AllowVnetInBound" at priority 200 would coexist with the system default at priority 65000 in the effective set). Stripping by name alone removed the system default. | Fixed in §4.2: strip criterion now requires BOTH `Name ∈ declaredNames` AND `Priority < 65000`. System defaults (priority ≥ 65000) are never stripped. |
| SR-002 | **Medium** | §1.2 | `AddSubnet` rationale claimed "segmentation gaps (AllowVnetInBound check)" but the Phase 1 engine only fires findings for NICs. A new subnet has no NICs → zero `SecurityDelta`. CIDR overlap is VNet `AddressSpace`-level, not subnet CIDR. | Fixed in §1.2: rationale corrected; explicit SR-002 note added documenting AddSubnet as cost-only for Phase 2 simulation. |
| SR-003 | **High** | §1.2, §3.3 | `AddPeering`/`RemovePeering` modifies `VNet.Peerings[]` but no Phase 1 analysis rule reads `VNet.Peerings[]` for intra-subscription topology. `checkCrossSubPeeringExposure` reads the separate `Fixture.CrossSubscriptionPeerings` field. These deltas produce zero `SecurityDelta` against the Phase 1 engine. | Fixed in §1.2: table now shows Phase 1 gate coverage limitation; SR-003 note added. §6.2 VPN Gateway trigger corrected (see SR-004 below). Intra-sub peering analysis rules are a Phase 2 engine addition. |
| SR-004 | **High** | §5.2, §6.2 | §5.2 diff key was `Type+Resource+Evidence`. Evidence strings are dynamically constructed in `analyze.go` (e.g. latent finding reason includes `"route 0.0.0.0/0→VirtualAppliance"` vs `"route 0.0.0.0/0→None"` after `ModifyRoute`). This produces spurious MitigatedRisk + AddedRisk pairs for the same finding with changed reason text. §6.2 VPN Gateway trigger incorrectly cited `AddPeering` to on-premises VNet — within a subscription, VNet-to-VNet peering does not provision a gateway. | Fixed in §5.2: diff key changed to `Type+Resource+Severity`; Evidence excluded from equality (retained for display). Fixed in §6.2: gateway trigger rewritten to require `NewNextHopType = "VirtualNetworkGateway"` (ModifyRoute) or `AllowGatewayTransit`/`UseRemoteGateways` = true (AddPeering gateway transit). |
| SR-005 | **Medium** | §8 | `CostForecast.ExistingFixedMonthlyUSD` field comment said "Informational only" but did not say "list price only". Could be misread as actual billed spend — exactly the conflation §9 warns against. | Fixed in §8: comment now explicitly states "List price only — derived from Azure Retail Prices API; EA/MCA contract discounts and commitment savings are not applied." |
| SR-006 | **Low** | §10 | Three gaps in placeholder reconciliation table: (1) `AzureFrontDoors` has a stale `// Phase 2` comment in model.go — `checkFrontDoorExposure` was added as Phase 1 rule 13 and the comment was never removed. (2) `Enrichment.DefenderAssessments` + `PolicyFindings` are tagged for Phase 2 attack-path cross-correlation in model.go but were missing from §10. (3) TMR-001 referenced `VNetGateway.SKU` but the actual struct in model.go is `VirtualNetworkGateway`. | Fixed in §10: AzureFrontDoors row added (stale comment flagged for removal); Enrichment.DefenderAssessments/PolicyFindings row added (deferred to Phase 3); TMR-001 struct name corrected to `VirtualNetworkGateway`. |

---



```
Tool name:   simulate_change
Description: Apply a proposed topology delta and return the security-posture change.
             Deterministic — no LLM in the path.

Input parameters:
  subscription_id  string  (required) Azure subscription GUID
  delta            object  (required) TopologyDelta JSON object (see §2)

Output (JSON):
  {
    "subscription": "...",
    "delta_summary": "AddNSGRule nsg-web-a: allow-https (Allow Inbound 443)",
    "security_delta": SecurityDelta,
    "simulated_finding_count": 5,
    "original_finding_count": 3
  }
```

## Appendix B — `forecast_cost` MCP Tool Specification

```
Tool name:   forecast_cost
Description: Estimate the monthly cost impact of a proposed topology delta.
             Fixed costs are exact (Retail Prices list); variable costs are a band.
             IMPORTANT: Outputs list prices only. EA/MCA discounts are not applied.

Input parameters:
  subscription_id  string  (required) Azure subscription GUID
  delta            object  (required) TopologyDelta JSON object (see §2)
  region           string  (optional) Override region for price lookup (default: infer from fixture)
  include_existing bool    (optional) Include existing fixed costs for context (default: false)

Output (JSON): CostForecast (see §8)
```

---

*End of SIMULATION_MODEL.md*
