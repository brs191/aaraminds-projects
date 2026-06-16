"""Pydantic models for the AT&T TopologySpec intermediate representation.

Field names follow Python ``snake_case`` convention; every field carries a
``alias=`` matching the Go ``json:"..."`` tag defined in
``phase-3/design/GENERATION_MODEL.md §1.2`` so that JSON round-trips through
the Go MCP server remain lossless.

``model_config = ConfigDict(populate_by_name=True)`` is set on every model so
that both the Python snake_case name *and* the Go camelCase alias are accepted
as constructor kwargs — this is essential for test construction and for
``model_validate_json`` when the LLM returns camelCase JSON.
"""

from __future__ import annotations

import re
from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, field_validator

# ---------------------------------------------------------------------------
# Approved vocabulary constants
# ---------------------------------------------------------------------------

#: The 16-value closed NSG intent vocabulary (task spec §SubnetSpec.nsgIntents).
#: Any intent string not in this set MUST raise ``ValueError`` at parse time.
VALID_NSG_INTENTS: frozenset[str] = frozenset(
    [
        "allow-https-from-internet",
        "allow-http-from-internet",
        "allow-internal-vnet",
        "deny-internet-inbound",
        "deny-all-inbound",
        "deny-all-inbound-other",
        "allow-bastion-rdp-ssh",
        "allow-ssh-from-bastion",
        "allow-rdp-from-bastion",
        "allow-appgw-management",
        "allow-internal-lb",
        "allow-azure-monitor",
        "allow-azure-loadbalancer",
        "allow-vnet-peering",
        "deny-vnet-inbound",
        "allow-sql-from-app",
    ]
)

#: Standard AT&T tier labels for subnets (§1.3 tierLabels enum).
VALID_TIER_LABELS: frozenset[str] = frozenset(
    [
        "dmz",
        "web",
        "app",
        "data",
        "mgmt",
        "shared",
        "gateway",
        "bastion",
        "pe",
        "aks",
        "appgw",
    ]
)

#: AT&T-mandated tag keys that MUST be present on every ``TopologySpec``
#: (§1.3 tags.required).  [VERIFY] AT&T mandatory tag policy — confirm keys.
MANDATORY_TAG_KEYS: frozenset[str] = frozenset(["env", "owner", "costcenter", "appid"])

_CIDR_RE: re.Pattern[str] = re.compile(
    r"^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/\d{1,2}$"
)


def _assert_cidr(value: str) -> str:
    """Return *value* unchanged if it looks like a CIDR block, else raise."""
    if not _CIDR_RE.match(value):
        raise ValueError(
            f"Expected CIDR notation (e.g. '10.0.0.0/24'), got: {value!r}"
        )
    return value


# ---------------------------------------------------------------------------
# PrivateEndpointSpec
# ---------------------------------------------------------------------------


class PrivateEndpointSpec(BaseModel):
    """Describes one generated private endpoint (§1.2 PrivateEndpointSpec).

    ``group_id`` maps to ``groupId`` (Go JSON tag).
    ``service_resource_id`` maps to ``serviceResourceId`` (Go JSON tag).
    """

    model_config = ConfigDict(populate_by_name=True)

    name: str = Field(min_length=1)
    group_id: str = Field(alias="groupId", min_length=1)
    service_resource_id: str = Field(alias="serviceResourceId", min_length=1)


# ---------------------------------------------------------------------------
# SubnetSpec
# ---------------------------------------------------------------------------


class SubnetSpec(BaseModel):
    """Describes one subnet to generate (§1.2 SubnetSpec).

    ``nsg_intents`` is validated against the closed 16-value vocabulary; any
    unknown intent raises ``ValueError`` immediately — the renderer would
    reject it anyway (``ErrUnknownNSGIntent``) but catching it here keeps the
    error as close to the boundary as possible.
    """

    model_config = ConfigDict(populate_by_name=True)

    name: str
    address_prefix: str = Field(alias="addressPrefix")
    tier_label: str = Field(alias="tierLabel")
    sensitive: bool
    nsg_intents: list[str] = Field(alias="nsgIntents")
    route_to_firewall: bool | None = Field(default=None, alias="routeToFirewall")
    service_endpoints: list[str] | None = Field(
        default=None, alias="serviceEndpoints"
    )
    delegations: list[str] | None = Field(default=None, alias="delegations")
    private_endpoint_subnet: bool | None = Field(
        default=None, alias="privateEndpointSubnet"
    )
    private_endpoints: list[PrivateEndpointSpec] | None = Field(
        default=None, alias="privateEndpoints"
    )

    @field_validator("address_prefix", mode="before")
    @classmethod
    def _validate_address_prefix(cls, v: object) -> str:
        if not isinstance(v, str):
            raise ValueError("addressPrefix must be a string")
        return _assert_cidr(v)

    @field_validator("tier_label", mode="before")
    @classmethod
    def _validate_tier_label(cls, v: object) -> str:
        if not isinstance(v, str):
            raise ValueError("tierLabel must be a string")
        if v not in VALID_TIER_LABELS:
            raise ValueError(
                f"tierLabel {v!r} is not in the approved vocabulary: "
                f"{sorted(VALID_TIER_LABELS)}"
            )
        return v

    @field_validator("nsg_intents", mode="before")
    @classmethod
    def _validate_nsg_intents(cls, v: object) -> list[str]:
        """Reject any intent that is not in the approved 16-value vocabulary.

        This is the primary guard that prevents the LLM (or a test) from
        injecting out-of-vocabulary intents that the Go renderer would reject
        with ``ErrUnknownNSGIntent``.
        """
        if not isinstance(v, list):
            raise ValueError("nsgIntents must be an array")
        for item in v:
            if not isinstance(item, str):
                raise ValueError(
                    f"nsgIntents entries must be strings, got {item!r}"
                )
            if item not in VALID_NSG_INTENTS:
                raise ValueError(
                    f"NSG intent {item!r} is not in the approved 16-value vocabulary. "
                    f"Approved intents: {sorted(VALID_NSG_INTENTS)}"
                )
        result: list[str] = list(v)
        return result


# ---------------------------------------------------------------------------
# VNetSpec
# ---------------------------------------------------------------------------


class VNetSpec(BaseModel):
    """Describes one virtual network to generate (§1.2 VNetSpec)."""

    model_config = ConfigDict(populate_by_name=True)

    name: str = Field(min_length=1)
    address_space: list[str] = Field(alias="addressSpace", min_length=1)
    subnets: list[SubnetSpec] = Field(min_length=1)
    is_hub: bool | None = Field(default=None, alias="isHub")

    @field_validator("address_space", mode="before")
    @classmethod
    def _validate_address_space(cls, v: object) -> list[str]:
        if not isinstance(v, list):
            raise ValueError("addressSpace must be an array")
        result: list[str] = []
        for cidr in v:
            if not isinstance(cidr, str):
                raise ValueError(
                    f"addressSpace entries must be strings, got {cidr!r}"
                )
            result.append(_assert_cidr(cidr))
        return result


# ---------------------------------------------------------------------------
# PeeringPairSpec
# ---------------------------------------------------------------------------


class PeeringPairSpec(BaseModel):
    """Describes one explicit VNet peering pair (§1.2 PeeringPairSpec).

    Used only when ``TopologySpec.peering_topology == "custom"``.
    """

    model_config = ConfigDict(populate_by_name=True)

    local_vnet: str = Field(alias="localVnet")
    remote_vnet: str = Field(alias="remoteVnet")
    allow_forwarded_traffic: bool | None = Field(
        default=None, alias="allowForwardedTraffic"
    )
    use_remote_gateways: bool | None = Field(default=None, alias="useRemoteGateways")
    allow_gateway_transit: bool | None = Field(
        default=None, alias="allowGatewayTransit"
    )


# ---------------------------------------------------------------------------
# TopologySpec (root document)
# ---------------------------------------------------------------------------


class TopologySpec(BaseModel):
    """The structured intermediate representation between architect intent and Terraform.

    Produced by the LLM (AskAT&T), consumed by ``RenderTerraform``.
    Intentionally omits all raw security-rule details — those are the renderer's
    domain, driven by the ``nsgIntents`` vocabulary (§3.3).

    ``spec_version`` is a ``Literal["1.0"]`` so schema evolution is versioned.
    """

    model_config = ConfigDict(populate_by_name=True)

    spec_version: Literal["1.0"] = Field(alias="specVersion")
    description: str = Field(min_length=10)
    region: str
    vnets: list[VNetSpec] = Field(min_length=1, max_length=20)
    peering_topology: Literal["hub-spoke", "mesh", "none", "custom"] = Field(
        alias="peeringTopology"
    )
    peering_pairs: list[PeeringPairSpec] | None = Field(
        default=None, alias="peeringPairs"
    )
    hub_vnet_name: str | None = Field(default=None, alias="hubVnetName")
    gateway_type: Literal["vpn", "expressroute", "none"] = Field(alias="gatewayType")
    firewall_enabled: bool = Field(alias="firewallEnabled")
    avnm_enabled: bool = Field(alias="avnmEnabled")
    avnm_network_group_id: str | None = Field(
        default=None, alias="avnmNetworkGroupId"
    )
    tier_labels: list[str] = Field(alias="tierLabels")
    tags: dict[str, str]

    @field_validator("tier_labels", mode="before")
    @classmethod
    def _validate_tier_labels(cls, v: object) -> list[str]:
        if not isinstance(v, list):
            raise ValueError("tierLabels must be an array")
        for label in v:
            if not isinstance(label, str):
                raise ValueError(
                    f"tierLabels entries must be strings, got {label!r}"
                )
            if label not in VALID_TIER_LABELS:
                raise ValueError(
                    f"tierLabel {label!r} is not in the approved vocabulary: "
                    f"{sorted(VALID_TIER_LABELS)}"
                )
        result: list[str] = list(v)
        return result

    @field_validator("tags", mode="before")
    @classmethod
    def _validate_mandatory_tags(cls, v: object) -> dict[str, str]:
        if not isinstance(v, dict):
            raise ValueError("tags must be an object")
        missing = MANDATORY_TAG_KEYS - set(v.keys())
        if missing:
            raise ValueError(
                f"tags is missing AT&T-mandatory keys: {sorted(missing)}. "
                f"All required: {sorted(MANDATORY_TAG_KEYS)}"
            )
        result: dict[str, str] = {str(k): str(val) for k, val in v.items()}
        return result
