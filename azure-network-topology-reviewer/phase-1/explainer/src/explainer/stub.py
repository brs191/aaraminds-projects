"""Stub implementations of LLMClient and RAGClient for local/CI use.

Set EXPLAINER_MODE=stub to activate.  No Azure credentials or live endpoints
are required.  Responses are deterministic and keyed on finding type.

This mode is suitable for:
  - Local development (no AskAT&T credentials yet)
  - CI smoke tests (no Azure Search index yet)
  - Demo environments
  - Integration tests (test_integration.py)
"""

from __future__ import annotations

import json
from typing import Optional, Tuple

import structlog

from explainer.models import FindingInput

log = structlog.get_logger(__name__)

# ---------------------------------------------------------------------------
# Per-finding-type explanations (plain English, 2–3 sentences)
# ---------------------------------------------------------------------------

_EXPLANATIONS: dict[str, str] = {
    "over-permissive NSG (reachable)": (
        "This NIC is directly reachable from the internet because its NSG permits "
        "inbound traffic from a broad source (e.g., Any or Internet), a default route "
        "to the internet exists, and a public IP is attached. An attacker with network "
        "access can initiate connections to the exposed port without any intermediate "
        "inspection layer."
    ),
    "over-permissive NSG (latent)": (
        "The NSG rule is overly permissive but the NIC is not currently reachable "
        "because a mitigating control (no public IP, black-hole route, or AVNM Deny) "
        "blocks the path. If that control is removed — for example, a public IP is "
        "attached — the NIC immediately becomes internet-exposed."
    ),
    "orphaned public endpoint": (
        "A public IP address exists in the subscription but is not attached to any "
        "resource. Orphaned PIPs still incur cost and represent an unclaimed address "
        "that could be mis-assigned or used to infer infrastructure details."
    ),
    "private DNS zone missing": (
        "A Private Endpoint is present but no Private DNS Zone is linked to resolve "
        "its private FQDN. Without DNS resolution, clients fall back to the public "
        "endpoint, bypassing the private link and potentially routing traffic outside "
        "the VNet."
    ),
    "private DNS zone not linked to VNet": (
        "A Private DNS Zone exists for the service but is not linked to the VNet "
        "that hosts the consuming workload. DNS queries from that VNet will not "
        "resolve the private IP, causing traffic to traverse the public endpoint "
        "instead of the Private Link path."
    ),
    "app gateway WAF disabled": (
        "The Application Gateway has a public IP but WAF is completely disabled, "
        "providing no L7 protection against OWASP Top-10 attacks on public-facing "
        "HTTP/S workloads. All inbound web traffic passes without inspection."
    ),
    "app gateway WAF in detection mode": (
        "WAF is enabled but operating in Detection mode, which logs potential attacks "
        "without blocking them. Malicious requests still reach the backend application. "
        "Detection mode is acceptable for initial tuning but should be switched to "
        "Prevention once false-positive baselines are established."
    ),
    "AKS non-private cluster": (
        "The AKS cluster's Kubernetes API server has a public endpoint, meaning the "
        "control plane is accessible from the internet. Credential compromise or "
        "unauthenticated vulnerabilities in the API server could give an attacker "
        "direct cluster access."
    ),
    "cross-subscription peering without firewall": (
        "Two VNets in different subscriptions are peered directly without an Azure "
        "Firewall or NVA in the path. Traffic can flow east-west between the "
        "subscriptions without inspection, violating least-privilege network "
        "segmentation."
    ),
    "internet reachable via load balancer NAT": (
        "A load balancer inbound NAT rule maps a public IP and port directly to a "
        "backend NIC. Even though the NIC itself has no public IP, internet traffic "
        "reaches it through the load balancer frontend. The threat model is "
        "identical to a NIC with a direct public IP."
    ),
    "APIM without VNet isolation": (
        "Azure API Management is deployed without VNet integration, exposing the "
        "management plane and developer portal to the public internet. All API "
        "traffic and administrative operations traverse the public network."
    ),
    "APIM External mode without WAF": (
        "APIM is in External VNet mode (management APIs are VNet-internal) but the "
        "public gateway endpoint has no WAF in front of it. L7 attacks against "
        "published APIs are not inspected before reaching the APIM gateway."
    ),
    "Bastion bypass — direct management port exposed": (
        "Azure Bastion is deployed in this VNet to provide secure RDP/SSH access, "
        "but this NIC also has a direct public IP with port 22 or 3389 open from "
        "the internet. Bastion is intended to be the exclusive management ingress; "
        "the direct public port creates a parallel attack surface that bypasses "
        "Bastion's session recording and MFA enforcement."
    ),
    "Front Door WAF disabled": (
        "Azure Front Door is configured without a WAF policy on this endpoint. "
        "Internet-facing traffic arrives at the origin without L7 DDoS mitigation, "
        "bot protection, or OWASP rule enforcement."
    ),
    "Front Door WAF in detection mode": (
        "Front Door has a WAF policy but it is in Detection mode. Malicious requests "
        "are logged but not blocked, providing visibility without enforcement. "
        "Switch to Prevention mode after tuning to activate blocking."
    ),
    "vWAN hub unsecured — no firewall": (
        "A Virtual WAN hub has no Azure Firewall deployed. All spoke-to-spoke and "
        "spoke-to-internet traffic transits the hub without inspection, making the "
        "vWAN topology equivalent to flat network connectivity."
    ),
    "vWAN hub firewall bypasses private traffic": (
        "An Azure Firewall is deployed in the vWAN hub but routing policy does not "
        "route private (RFC-1918) traffic through it. East-west spoke traffic and "
        "on-premises traffic bypass firewall inspection."
    ),
}

_DEFAULT_EXPLANATION = (
    "This finding indicates a network configuration that deviates from AT&T "
    "security baseline requirements. Review the evidence field for specific "
    "resource identifiers and consult the AT&T Azure Network Standard for "
    "remediation guidance."
)

# ---------------------------------------------------------------------------
# Per-finding-type RAG recommendations
# Keyed on finding type; value is (clause, recommendation_text, document_title)
# ---------------------------------------------------------------------------

_RAG: dict[str, tuple[str, str, str]] = {
    "over-permissive NSG (reachable)": (
        "4.2.1",
        "Remove Any/Internet source rules from NSGs on non-DMZ subnets. "
        "Replace with application-specific source IP prefixes or ASGs. "
        "Deploy Azure Firewall or NVA as the sole internet ingress point.",
        "AT&T Azure Network Security Baseline v3.1",
    ),
    "over-permissive NSG (latent)": (
        "4.2.2",
        "Tighten the overly broad NSG rule now, before a public IP or route change "
        "makes the latent exposure live. Apply least-privilege source restrictions.",
        "AT&T Azure Network Security Baseline v3.1",
    ),
    "orphaned public endpoint": (
        "6.1.4",
        "Delete unattached public IP addresses. Implement Azure Policy to alert on "
        "PIPs that are unassociated for more than 7 days.",
        "AT&T Azure Cost and Hygiene Standard v2.0",
    ),
    "private DNS zone missing": (
        "5.3.1",
        "Create a Private DNS Zone for the service (e.g., privatelink.blob.core.windows.net) "
        "and link it to all VNets that consume the Private Endpoint. "
        "Validate DNS resolution with nslookup from within the VNet.",
        "AT&T Private Link DNS Architecture Standard v1.2",
    ),
    "private DNS zone not linked to VNet": (
        "5.3.2",
        "Add a VNet link for the consuming VNet to the existing Private DNS Zone. "
        "Verify that `enableAutoRegistration` is set appropriately for workload DNS.",
        "AT&T Private Link DNS Architecture Standard v1.2",
    ),
    "app gateway WAF disabled": (
        "4.5.1",
        "Enable WAF_v2 SKU on the Application Gateway and set it to Prevention mode "
        "with the OWASP 3.2 rule set. Enable bot protection and rate limiting.",
        "AT&T Web Application Firewall Policy Standard v2.3",
    ),
    "app gateway WAF in detection mode": (
        "4.5.2",
        "Transition WAF from Detection to Prevention mode. Review detection logs for "
        "false positives, add exclusions where necessary, then switch to Prevention.",
        "AT&T Web Application Firewall Policy Standard v2.3",
    ),
    "AKS non-private cluster": (
        "7.1.1",
        "Enable the AKS private cluster feature (`--enable-private-cluster`). "
        "Access the API server exclusively via Private Endpoint or VNet-integrated "
        "tooling. Restrict authorized IP ranges as an interim measure.",
        "AT&T Kubernetes Security Standard v1.4",
    ),
    "cross-subscription peering without firewall": (
        "3.4.2",
        "Insert an Azure Firewall or approved NVA in the hub between peered "
        "cross-subscription VNets. Use UDRs to force all inter-VNet traffic through "
        "the firewall. Disable AllowForwardedTraffic on spoke peerings.",
        "AT&T Hub-Spoke Network Architecture Standard v2.0",
    ),
    "internet reachable via load balancer NAT": (
        "4.3.1",
        "Remove inbound NAT rules that expose backend NICs directly. Route inbound "
        "traffic through Azure Firewall DNAT or Application Gateway instead. "
        "Use load balancer backend pools without direct internet NAT for internal services.",
        "AT&T Azure Network Security Baseline v3.1",
    ),
    "APIM without VNet isolation": (
        "8.2.1",
        "Deploy APIM in Internal or External VNet integration mode. For Internal mode, "
        "front all public traffic with Application Gateway + WAF. Restrict management "
        "plane access to the corporate IP range.",
        "AT&T API Gateway Security Standard v1.1",
    ),
    "APIM External mode without WAF": (
        "8.2.2",
        "Place an Application Gateway with WAF_v2 in front of the APIM External "
        "endpoint. Configure WAF in Prevention mode with OWASP 3.2 rules. "
        "Set APIM's backend certificate validation to enforce mTLS.",
        "AT&T API Gateway Security Standard v1.1",
    ),
    "Bastion bypass — direct management port exposed": (
        "4.4.1",
        "Remove the public IP from management NICs when Azure Bastion is deployed. "
        "Delete NSG rules permitting port 22/3389 from Internet. "
        "Enforce this via Azure Policy deny assignment on public management ports.",
        "AT&T Secure Remote Access Standard v2.2",
    ),
    "Front Door WAF disabled": (
        "4.6.1",
        "Associate a WAF policy (Prevention mode, OWASP 3.2, bot protection enabled) "
        "with every Front Door endpoint. Enable rate limiting for DDoS mitigation.",
        "AT&T Web Application Firewall Policy Standard v2.3",
    ),
    "Front Door WAF in detection mode": (
        "4.6.2",
        "Promote the Front Door WAF policy from Detection to Prevention mode. "
        "Monitor WAF logs for 7–14 days post-switchover and add tuned exclusions.",
        "AT&T Web Application Firewall Policy Standard v2.3",
    ),
    "vWAN hub unsecured — no firewall": (
        "3.5.1",
        "Deploy Azure Firewall in each secured vWAN hub. Set routing intent to route "
        "both internet and private traffic through the hub firewall. "
        "Enable Firewall Manager policy inheritance for consistent rule management.",
        "AT&T Virtual WAN Security Standard v1.0",
    ),
    "vWAN hub firewall bypasses private traffic": (
        "3.5.2",
        "Update vWAN routing policy to include `PrivateTraffic` in the next-hop "
        "firewall configuration. Verify that all spoke route tables inherit the "
        "hub firewall as the next hop for RFC-1918 destinations.",
        "AT&T Virtual WAN Security Standard v1.0",
    ),
}

_DEFAULT_RAG = (
    "9.0.1",
    "Review this finding against the AT&T Azure Security Baseline and apply the "
    "least-privilege network access principle. Engage the AT&T Cloud Network "
    "Architecture team for remediation guidance specific to your workload.",
    "AT&T Azure Security Baseline v3.1",
)


# ---------------------------------------------------------------------------
# Stub clients
# ---------------------------------------------------------------------------


class StubLLMClient:
    """Returns canned explanations keyed on finding type.  No Azure calls."""

    async def explain(self, finding: FindingInput) -> str | None:
        log.info("stub.llm.explain", finding_type=finding.type)
        return _EXPLANATIONS.get(finding.type, _DEFAULT_EXPLANATION)

    async def summarise(self, explained: list[dict]) -> str | None:
        n = len(explained)
        hc = sum(1 for f in explained if f.get("severity") in ("Critical", "High"))
        log.info("stub.llm.summarise", finding_count=n, high_critical=hc)
        if n == 0:
            return "No findings detected in this subscription."
        return (
            f"Analysis identified {n} findings across this Azure subscription, "
            f"including {hc} High or Critical severity issues requiring immediate "
            f"attention. Prioritise remediation of internet-reachable resources and "
            f"WAF gaps before addressing medium-severity configuration drift."
        )

    async def aclose(self) -> None:
        pass


class StubRAGClient:
    """Returns canned AT&T standard recommendations keyed on finding type."""

    async def search(
        self, finding: FindingInput
    ) -> tuple[str | None, str | None, bool]:
        log.info("stub.rag.search", finding_type=finding.type)
        clause, rec_text, doc_title = _RAG.get(finding.type, _DEFAULT_RAG)
        recommendation = (
            f"Per AT&T Network Standard §{clause}: {rec_text}. Source: {doc_title}."
        )
        return recommendation, doc_title, True

    async def aclose(self) -> None:
        pass
