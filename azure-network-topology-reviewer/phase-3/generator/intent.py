"""Intent capture layer — Phase 3 topology generation pipeline.

Entry point: ``generate_spec(intent, subscription_context, max_iterations,
failing_findings) -> TopologySpec``.

Design references
-----------------
* §1.2  TopologySpec / Go type definitions
* §1.3  JSON Schema (embedded as ``_TOPOLOGY_SPEC_JSON_SCHEMA``)
* §1.4  LLM system prompt constraint (verbatim, non-negotiable)
* §1.5  Worked example — stub spec returned in ``GENERATOR_MODE=stub``
* §4.3  Iterative refinement loop prompt injection
* §6.5  AskAT&T-only guardrail — no external LLM calls

AT&T non-negotiables enforced here
-----------------------------------
* ``# AT&T: AskAT&T only — no external LLM calls``
* Bearer token is NEVER written to any log record at any level.
* ``client_secret`` is NEVER stored as an instance attribute after token
  acquisition; the local variable is deleted before the function returns.
* ``GENERATOR_MODE=stub`` works with zero network calls and zero credentials.
"""

from __future__ import annotations

import asyncio
import json
import logging
import os
import re
import time
from typing import Any

import httpx
from pydantic import ValidationError

from .exceptions import LLMClientError, SpecValidationError
from .models import (
    PeeringPairSpec,
    PrivateEndpointSpec,
    SubnetSpec,
    TopologySpec,
    VNetSpec,
)

# ---------------------------------------------------------------------------
# Module-level logger with bearer-token redaction filter
# ---------------------------------------------------------------------------

_logger = logging.getLogger(__name__)


class _BearerTokenRedactFilter(logging.Filter):
    """Scrub ``Authorization: Bearer <token>`` values from log records.

    Applied at the logger level so that *any* log record emitted through
    ``_logger`` (including records that propagate to the root logger and
    pytest's caplog handler) will have tokens replaced with ``[REDACTED]``
    before reaching any handler.

    The filter modifies the record **in-place** and returns ``True``
    (allowing the record through) so that log messages are still emitted —
    they just never carry credential material.
    """

    _BEARER_RE: re.Pattern[str] = re.compile(
        r"Bearer\s+[A-Za-z0-9\-._~+/]+=*", re.IGNORECASE
    )

    def filter(self, record: logging.LogRecord) -> bool:  # noqa: A003
        # Resolve the message string (handles %-style formatting).
        if record.args:
            try:
                record.msg = record.getMessage()
            except Exception:  # pragma: no cover  # noqa: BLE001
                pass
            record.args = ()

        if isinstance(record.msg, str):
            record.msg = self._BEARER_RE.sub("Bearer [REDACTED]", record.msg)

        return True


# Install the redaction filter on our logger once at module load.
# Also install on the httpx logger to catch any transport-level debug output.
_REDACT_FILTER = _BearerTokenRedactFilter()
_logger.addFilter(_REDACT_FILTER)
logging.getLogger("httpx").addFilter(_REDACT_FILTER)
logging.getLogger("httpcore").addFilter(_REDACT_FILTER)

# ---------------------------------------------------------------------------
# JSON Schema for AskAT&T structured output (§1.3)
# ---------------------------------------------------------------------------
# This schema is embedded in every chat/completions request as
# ``response_format.json_schema.schema`` to constrain the LLM to produce
# exactly TopologySpec JSON.  It is derived verbatim from §1.3.

_TOPOLOGY_SPEC_JSON_SCHEMA: dict[str, Any] = {
    "$schema": "https://json-schema.org/draft/2020-12/schema",
    "$id": "urn:att:nettopo:topology-spec:1.0",
    "title": "TopologySpec",
    "type": "object",
    "required": [
        "specVersion",
        "description",
        "region",
        "vnets",
        "peeringTopology",
        "gatewayType",
        "firewallEnabled",
        "avnmEnabled",
        "tierLabels",
        "tags",
    ],
    "additionalProperties": False,
    "properties": {
        "specVersion": {"type": "string", "const": "1.0"},
        "description": {"type": "string", "minLength": 10},
        "region": {
            "type": "string",
            "description": "Primary Azure region slug, e.g. eastus2, westus3",
        },
        "vnets": {
            "type": "array",
            "minItems": 1,
            "maxItems": 20,
            "items": {"$ref": "#/$defs/VNetSpec"},
        },
        "peeringTopology": {
            "type": "string",
            "enum": ["hub-spoke", "mesh", "none", "custom"],
        },
        "peeringPairs": {
            "type": "array",
            "items": {"$ref": "#/$defs/PeeringPairSpec"},
        },
        "hubVnetName": {"type": "string"},
        "gatewayType": {
            "type": "string",
            "enum": ["vpn", "expressroute", "none"],
        },
        "firewallEnabled": {"type": "boolean"},
        "avnmEnabled": {"type": "boolean"},
        "avnmNetworkGroupId": {"type": "string"},
        "tierLabels": {
            "type": "array",
            "items": {
                "type": "string",
                "enum": [
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
                ],
            },
        },
        "tags": {
            "type": "object",
            "required": ["env", "owner", "costcenter", "appid"],
            "additionalProperties": {"type": "string"},
        },
    },
    "$defs": {
        "VNetSpec": {
            "type": "object",
            "required": ["name", "addressSpace", "subnets"],
            "additionalProperties": False,
            "properties": {
                "name": {"type": "string", "minLength": 1},
                "addressSpace": {
                    "type": "array",
                    "items": {
                        "type": "string",
                        "pattern": r"^\d+\.\d+\.\d+\.\d+/\d+$",
                    },
                },
                "subnets": {
                    "type": "array",
                    "minItems": 1,
                    "items": {"$ref": "#/$defs/SubnetSpec"},
                },
                "isHub": {"type": "boolean"},
            },
        },
        "SubnetSpec": {
            "type": "object",
            "required": [
                "name",
                "addressPrefix",
                "tierLabel",
                "sensitive",
                "nsgIntents",
            ],
            "additionalProperties": False,
            "properties": {
                "name": {"type": "string"},
                "addressPrefix": {
                    "type": "string",
                    "pattern": r"^\d+\.\d+\.\d+\.\d+/\d+$",
                },
                "tierLabel": {"type": "string"},
                "sensitive": {"type": "boolean"},
                "nsgIntents": {
                    "type": "array",
                    "items": {"type": "string"},
                },
                "routeToFirewall": {"type": "boolean"},
                "serviceEndpoints": {
                    "type": "array",
                    "items": {"type": "string"},
                },
                "delegations": {
                    "type": "array",
                    "items": {"type": "string"},
                },
                "privateEndpointSubnet": {"type": "boolean"},
                "privateEndpoints": {
                    "type": "array",
                    "items": {"$ref": "#/$defs/PrivateEndpointSpec"},
                },
            },
        },
        "PrivateEndpointSpec": {
            "type": "object",
            "required": ["name", "groupId", "serviceResourceId"],
            "additionalProperties": False,
            "properties": {
                "name": {"type": "string", "minLength": 1},
                "groupId": {"type": "string", "minLength": 1},
                "serviceResourceId": {"type": "string", "minLength": 1},
            },
        },
        "PeeringPairSpec": {
            "type": "object",
            "required": ["localVnet", "remoteVnet"],
            "additionalProperties": False,
            "properties": {
                "localVnet": {"type": "string"},
                "remoteVnet": {"type": "string"},
                "allowForwardedTraffic": {"type": "boolean"},
                "useRemoteGateways": {"type": "boolean"},
                "allowGatewayTransit": {"type": "boolean"},
            },
        },
    },
}

# ---------------------------------------------------------------------------
# LLM system prompt (§1.4 — verbatim constraint block, non-negotiable)
# ---------------------------------------------------------------------------

_SYSTEM_PROMPT: str = """\
You are an Azure network topology designer for AT&T.
Your output MUST be valid JSON matching the TopologySpec schema exactly.

CONSTRAINT — SECURITY RULES:
You MUST NOT produce raw NSG security rule objects with fields: priority, access, direction,
destinationPortRange, sourceAddressPrefix. Instead, express desired access as named intents
from the approved vocabulary in the nsgIntents array of each subnet. The renderer translates
intents to rules. Intents outside the approved vocabulary will be rejected.

CONSTRAINT — RESOURCE IDENTIFIERS:
Do not invent Terraform resource IDs, module source paths, or version strings.
The renderer selects modules from the AT&T-approved registry. Your output is TopologySpec JSON only.

APPROVED NSG INTENT VOCABULARY (use ONLY these 16 values in nsgIntents):
  allow-https-from-internet, allow-http-from-internet, allow-internal-vnet,
  deny-internet-inbound, deny-all-inbound, deny-all-inbound-other,
  allow-bastion-rdp-ssh, allow-ssh-from-bastion, allow-rdp-from-bastion,
  allow-appgw-management, allow-internal-lb, allow-azure-monitor,
  allow-azure-loadbalancer, allow-vnet-peering, deny-vnet-inbound,
  allow-sql-from-app

APPROVED TIER LABELS (use ONLY these in tierLabel fields):
  dmz, web, app, data, mgmt, shared, gateway, bastion, pe, aks, appgw

MANDATORY TAGS: every TopologySpec.tags object MUST contain env, owner, costcenter, appid.
"""

# ---------------------------------------------------------------------------
# AskATTClient
# ---------------------------------------------------------------------------

# AT&T: AskAT&T only — no external LLM calls
# The endpoint is sourced from ASKAT_ENDPOINT env var (non-secret).
# Credentials flow: client_credentials OAuth2 → bearer token cached until
# (exp - 60 s).  The client_secret is NEVER stored as an instance attribute.

_RETRY_DELAYS: list[float] = [1.0, 2.0, 4.0]  # seconds before attempt 2, 3


class AskATTClient:
    """AskAT&T structured-output LLM client.

    Credentials sourced from environment variables:

    ``ASKAT_ENDPOINT``
        Base URL of the AskAT&T API (e.g. ``https://askatt.example.att.com``).
    ``ASKAT_TOKEN_URL``
        OAuth2 token endpoint for the client-credentials flow.
    ``ASKAT_CLIENT_ID``
        Client ID (non-secret; safe to log *schema only*, never the value).
    ``ASKAT_CLIENT_SECRET``
        Client secret read from env at token-acquisition time only.
        [VERIFY] V-11: replace with Azure Key Vault secret fetch via managed
        identity.  The KV secret name is ``ASKAT_SECRET_NAME``.  Until V-11
        is resolved, the secret is read from ``ASKAT_CLIENT_SECRET`` env var
        and deleted from local scope immediately after use.

    Token caching
    ~~~~~~~~~~~~~
    The bearer token is cached in ``_cached_token`` with expiry tracked in
    ``_token_expiry_ts`` (Unix timestamp).  Before reuse, a 60-second safety
    margin is applied so the token is refreshed before the server rejects it.

    Security invariants
    ~~~~~~~~~~~~~~~~~~~
    * Token is NEVER logged at any level (DEBUG, INFO, WARNING, ERROR).
      The ``_BearerTokenRedactFilter`` installed on the module logger provides
      defence-in-depth, but the primary control is: we never pass the token
      to any logging call.
    * ``client_secret`` is NEVER stored as an instance attribute after token
      acquisition.  It is a local variable that goes out of scope (and is
      explicitly ``del``-d) within ``_get_token``.
    """

    def __init__(self) -> None:
        # AT&T: AskAT&T only — no external LLM calls
        self._endpoint: str = os.environ["ASKAT_ENDPOINT"].rstrip("/")
        self._token_url: str = os.environ["ASKAT_TOKEN_URL"]
        self._client_id: str = os.environ["ASKAT_CLIENT_ID"]
        # [VERIFY] V-11: replace with Key Vault fetch (secret name in ASKAT_SECRET_NAME)
        # For now: client secret lives in ASKAT_CLIENT_SECRET env var.
        # It is NOT read here — it is read inside _get_token and deleted on exit.
        self._cached_token: str | None = None
        self._token_expiry_ts: float = 0.0  # Unix epoch seconds

    async def _get_token(self) -> str:
        """Fetch (or return cached) bearer token via client-credentials flow.

        The ``client_secret`` is read from the environment inside this method,
        used in the POST body, and then explicitly ``del``-d so it does not
        persist in the local frame beyond the HTTP call.  It is NEVER stored
        on ``self``.
        """
        now = time.time()
        if self._cached_token is not None and now < self._token_expiry_ts - 60.0:
            return self._cached_token

        _logger.debug(
            "Fetching AskAT&T bearer token from token endpoint (client_id=%s)",
            # Log the client_id schema only — the value itself is benign
            # (it's a non-secret), but we log just the fact of the call.
            "present" if self._client_id else "missing",
        )

        # [VERIFY] V-11: replace os.environ read with Azure Key Vault secret fetch
        # using DefaultAzureCredential + SecretClient, keyed by ASKAT_SECRET_NAME.
        client_secret = os.environ["ASKAT_CLIENT_SECRET"]

        try:
            async with httpx.AsyncClient(timeout=30.0) as http:
                resp = await http.post(
                    self._token_url,
                    data={
                        "grant_type": "client_credentials",
                        "client_id": self._client_id,
                        "client_secret": client_secret,
                    },
                )
                resp.raise_for_status()
                token_data: dict[str, Any] = resp.json()
        except httpx.HTTPStatusError as exc:
            # Sanitise: do NOT include the client_secret in the message.
            raise LLMClientError(
                f"Token endpoint returned HTTP {exc.response.status_code}"
            ) from exc
        except httpx.HTTPError as exc:
            raise LLMClientError(
                "Token endpoint unreachable — check ASKAT_TOKEN_URL"
            ) from exc
        finally:
            # Ensure client_secret is not retained in the local frame.
            del client_secret

        access_token: str = str(token_data["access_token"])
        expires_in: float = float(token_data.get("expires_in", 3600))

        self._cached_token = access_token
        self._token_expiry_ts = now + expires_in

        _logger.debug("AskAT&T bearer token acquired (expires_in=%.0fs)", expires_in)
        # NOTE: token value is intentionally NOT logged here or anywhere below.
        return access_token

    async def complete(
        self,
        messages: list[dict[str, str]],
    ) -> str:
        """POST to ``{ASKAT_ENDPOINT}/chat/completions`` with the TopologySpec
        JSON Schema embedded as ``response_format``.

        Retries up to 3 total attempts on HTTP 429 (rate limit) or 503
        (service unavailable) with exponential back-off: 1 s → 2 s → 4 s.
        Any other HTTP error fails immediately.

        Raises
        ------
        LLMClientError
            After all retry attempts are exhausted, or on a non-retryable
            HTTP error.  The exception message is sanitised (no token, no
            secret).
        """
        # AT&T: AskAT&T only — no external LLM calls
        token = await self._get_token()

        payload: dict[str, Any] = {
            "messages": messages,
            "response_format": {
                "type": "json_schema",
                "json_schema": {
                    "name": "TopologySpec",
                    "strict": True,
                    "schema": _TOPOLOGY_SPEC_JSON_SCHEMA,
                },
            },
        }

        url = f"{self._endpoint}/chat/completions"
        last_exc: Exception | None = None

        for attempt in range(3):
            if attempt > 0:
                delay = _RETRY_DELAYS[attempt - 1]
                _logger.debug(
                    "AskAT&T call retry %d/3 — sleeping %.0fs", attempt + 1, delay
                )
                await asyncio.sleep(delay)

            _logger.debug(
                "AskAT&T chat/completions attempt %d/3", attempt + 1
            )

            try:
                async with httpx.AsyncClient(timeout=120.0) as http:
                    resp = await http.post(
                        url,
                        headers={
                            # The Authorization header value is assembled here
                            # but NEVER passed to any logger.
                            "Authorization": f"Bearer {token}",
                            "Content-Type": "application/json",
                        },
                        json=payload,
                    )
            except httpx.HTTPError as exc:
                last_exc = exc
                _logger.debug("AskAT&T HTTP transport error on attempt %d", attempt + 1)
                if attempt == 2:
                    raise LLMClientError(
                        f"AskAT&T chat/completions unreachable after {attempt + 1} attempts"
                    ) from exc
                continue

            if resp.status_code in (429, 503):
                _logger.debug(
                    "AskAT&T returned %d on attempt %d — will retry",
                    resp.status_code,
                    attempt + 1,
                )
                last_exc = httpx.HTTPStatusError(
                    message=f"status {resp.status_code}",
                    request=resp.request,
                    response=resp,
                )
                if attempt == 2:
                    raise LLMClientError(
                        f"AskAT&T returned HTTP {resp.status_code} after 3 attempts"
                    ) from last_exc
                continue

            try:
                resp.raise_for_status()
            except httpx.HTTPStatusError as exc:
                # Non-retryable error — fail immediately, sanitised message.
                raise LLMClientError(
                    f"AskAT&T returned unexpected HTTP {resp.status_code}"
                ) from exc

            # Parse response — extract choices[0].message.content
            data: dict[str, Any] = resp.json()
            choices = data.get("choices")
            if not isinstance(choices, list) or not choices:
                raise LLMClientError(
                    "AskAT&T response missing 'choices' array"
                )
            first: Any = choices[0]
            if not isinstance(first, dict):
                raise LLMClientError(
                    "AskAT&T response choices[0] is not an object"
                )
            message: Any = first.get("message")
            if not isinstance(message, dict):
                raise LLMClientError(
                    "AskAT&T response choices[0].message is not an object"
                )
            content: Any = message.get("content")
            if not isinstance(content, str):
                raise LLMClientError(
                    "AskAT&T response choices[0].message.content is not a string"
                )

            _logger.debug("AskAT&T response received (content_length=%d)", len(content))
            return content

        # Should be unreachable — the loop always returns or raises.
        raise LLMClientError("AskAT&T call failed after all retry attempts")


# ---------------------------------------------------------------------------
# Refinement prompt builder (§4.3)
# ---------------------------------------------------------------------------


def _build_refinement_block(
    iteration: int,
    max_iterations: int,
    failing_findings: list[dict[str, Any]],
) -> str:
    """Return the verbatim refinement context block defined in §4.3.

    Injected as a prefix to the user message on iteration 2 and above.
    """
    findings_json = json.dumps(failing_findings, indent=2)
    return (
        f"VALIDATION FAILURE — iteration {iteration} of {max_iterations}.\n"
        f"The previous TopologySpec produced the following blocking security findings:\n\n"
        f"{findings_json}\n\n"
        f"You MUST revise the TopologySpec to eliminate these findings. Common fixes:\n"
        f'- Add "deny-internet-inbound" to sensitive subnet nsgIntents\n'
        f"- Set routeToFirewall: true on subnets with sensitive: true\n"
        f'- Remove allow-https-from-internet from data/app tier subnets\n'
        f'- Pair allow-internal-vnet with deny-all-inbound-other on sensitive subnets\n'
        f"- For `private DNS zone missing` / `private DNS zone not linked to VNet`, add or\n"
        f"  correct the relevant privateEndpoints[] declaration\n"
        f"- For Bastion findings, replace direct internet management intents with "
        f"allow-bastion-rdp-ssh\n"
        f"\nDo NOT change the intent described by the architect. Only change NSG intents,\n"
        f"routeToFirewall flags, subnet labelling, and other generator-owned fields to "
        f"satisfy the security gate.\n"
    )


# ---------------------------------------------------------------------------
# Stub spec (§1.5 worked example — GENERATOR_MODE=stub)
# ---------------------------------------------------------------------------


def _stub_spec() -> TopologySpec:
    """Return the AT&T BCLM 3-tier hub-spoke TopologySpec from §1.5.

    Used when ``GENERATOR_MODE=stub`` is set.  No AskAT&T credentials are
    needed in stub mode — no network calls are made.

    Note: ``allow-app-tier-only`` from the §1.5 example is replaced with
    ``allow-sql-from-app`` which is in the approved 16-value vocabulary.
    """
    # Use Python snake_case field names throughout (not Go-alias camelCase) so
    # that mypy --strict resolves the correct constructor signatures.
    return TopologySpec(
        spec_version="1.0",
        description=(
            "AT&T BCLM payment processing platform — hub-spoke, East US 2, "
            "Azure Firewall, VPN gateway, 3 spoke tiers (web/app/data), "
            "data tier sensitive."
        ),
        region="eastus2",
        peering_topology="hub-spoke",
        hub_vnet_name="hub-vnet-bclm-prod",
        gateway_type="vpn",
        firewall_enabled=True,
        avnm_enabled=True,
        avnm_network_group_id=(
            "/subscriptions/XXXX/resourceGroups/rg-avnm/providers/"
            "Microsoft.Network/networkManagers/nm-att-prod/"
            "networkGroups/ng-bclm-prod"
        ),
        tier_labels=["gateway", "mgmt", "web", "app", "data", "pe", "appgw"],
        tags={
            "env": "prod",
            "owner": "network-team",
            "costcenter": "BCLM-NET",
            "appid": "PAY-001",
        },
        vnets=[
            # Hub VNet -------------------------------------------------------
            VNetSpec(
                name="hub-vnet-bclm-prod",
                address_space=["10.0.0.0/16"],
                is_hub=True,
                subnets=[
                    SubnetSpec(
                        name="AzureFirewallSubnet",
                        address_prefix="10.0.0.0/26",
                        tier_label="mgmt",
                        sensitive=False,
                        nsg_intents=[],
                    ),
                    SubnetSpec(
                        name="GatewaySubnet",
                        address_prefix="10.0.1.0/27",
                        tier_label="gateway",
                        sensitive=False,
                        nsg_intents=[],
                    ),
                    SubnetSpec(
                        name="AzureBastionSubnet",
                        address_prefix="10.0.2.0/26",
                        tier_label="mgmt",
                        sensitive=False,
                        nsg_intents=[
                            "allow-https-from-internet",
                            "deny-all-inbound-other",
                        ],
                    ),
                    SubnetSpec(
                        name="snet-mgmt",
                        address_prefix="10.0.3.0/24",
                        tier_label="mgmt",
                        sensitive=False,
                        nsg_intents=["allow-bastion-rdp-ssh", "deny-internet-inbound"],
                        route_to_firewall=True,
                    ),
                ],
            ),
            # Spoke: web tier ------------------------------------------------
            VNetSpec(
                name="spoke-vnet-web-prod",
                address_space=["10.1.0.0/16"],
                subnets=[
                    SubnetSpec(
                        name="snet-web",
                        address_prefix="10.1.1.0/24",
                        tier_label="web",
                        sensitive=False,
                        nsg_intents=[
                            "allow-https-from-internet",
                            "allow-http-from-internet",
                            "deny-all-inbound-other",
                        ],
                        route_to_firewall=True,
                    ),
                    SubnetSpec(
                        name="snet-appgw",
                        address_prefix="10.1.2.0/24",
                        tier_label="appgw",
                        sensitive=False,
                        nsg_intents=[
                            "allow-appgw-management",
                            "allow-https-from-internet",
                        ],
                    ),
                ],
            ),
            # Spoke: app tier ------------------------------------------------
            VNetSpec(
                name="spoke-vnet-app-prod",
                address_space=["10.2.0.0/16"],
                subnets=[
                    SubnetSpec(
                        name="snet-app",
                        address_prefix="10.2.1.0/24",
                        tier_label="app",
                        sensitive=False,
                        nsg_intents=["allow-internal-vnet", "deny-internet-inbound"],
                        route_to_firewall=True,
                    ),
                ],
            ),
            # Spoke: data tier (sensitive) ------------------------------------
            VNetSpec(
                name="spoke-vnet-data-prod",
                address_space=["10.3.0.0/16"],
                subnets=[
                    SubnetSpec(
                        name="snet-data",
                        address_prefix="10.3.1.0/24",
                        tier_label="data",
                        sensitive=True,
                        # allow-sql-from-app replaces allow-app-tier-only from §1.5;
                        # allow-app-tier-only is not in the approved 16-value vocab.
                        nsg_intents=[
                            "allow-sql-from-app",
                            "deny-internet-inbound",
                            "deny-all-inbound-other",
                        ],
                        route_to_firewall=True,
                        service_endpoints=["Microsoft.Storage", "Microsoft.KeyVault"],
                    ),
                    SubnetSpec(
                        name="snet-pe",
                        address_prefix="10.3.2.0/24",
                        tier_label="pe",
                        sensitive=True,
                        nsg_intents=["deny-internet-inbound"],
                        private_endpoint_subnet=True,
                        route_to_firewall=True,
                    ),
                ],
            ),
        ],
    )


# ---------------------------------------------------------------------------
# generate_spec — public entry point
# ---------------------------------------------------------------------------


async def generate_spec(
    intent: str,
    subscription_context: dict[str, Any],
    max_iterations: int,
    failing_findings: list[dict[str, Any]],
) -> TopologySpec:
    """Translate architect natural-language intent into a validated TopologySpec.

    Parameters
    ----------
    intent:
        The architect's natural-language description of the desired topology.
        Passed verbatim to the LLM system prompt.
    subscription_context:
        Read-only subscription context fetched by the MCP handler (AVNM state,
        existing firewalls, etc.).  Serialised as JSON in the user message so
        the LLM can respect the subscription's existing posture.
    max_iterations:
        Maximum number of LLM calls.  Clamped to [1, 3] — never 0 (at least
        one attempt required), never >3 (infinite-loop prevention per §4.3).
    failing_findings:
        Blocking findings (Critical / High) from ``ValidateBeforeEmit`` on a
        *previous* iteration, passed in by the refinement loop caller.
        Empty list on the first call.  On iterations 2+, injected into the
        user message as the §4.3 refinement block.

    Returns
    -------
    TopologySpec
        The last successfully parsed ``TopologySpec``.  Always the Pydantic-
        validated object — the caller's security gate (``ValidateBeforeEmit``)
        determines whether the *security posture* is acceptable.

    Raises
    ------
    SpecValidationError
        If every iteration produces LLM output that cannot be parsed into a
        valid ``TopologySpec`` (Pydantic schema violation).
    LLMClientError
        If the AskAT&T HTTP client fails permanently (auth error, all retries
        exhausted).

    Stub mode
    ~~~~~~~~~
    If ``os.environ["GENERATOR_MODE"] == "stub"``, returns the §1.5 worked-
    example spec immediately — no network calls, no credentials required.

    Iteration mechanics
    ~~~~~~~~~~~~~~~~~~~
    Iteration 1:  system prompt (§1.4) + architect intent (no failing findings).
    Iteration 2+: system prompt + architect intent + §4.3 refinement block
                  (using the ``failing_findings`` argument).
    The loop exits as soon as the LLM response parses into a valid
    ``TopologySpec``.  If parsing fails, the loop retries up to
    ``max_iterations`` times.  ``failing_findings`` are injected from
    iteration 2 onwards to give the LLM context on what went wrong.
    """
    # Clamp to [1, 3] — do not raise, per spec.
    max_iterations = max(1, min(3, max_iterations))

    # ---- Stub mode: no LLM call, no credentials ----------------------------
    if os.environ.get("GENERATOR_MODE") == "stub":
        _logger.debug("GENERATOR_MODE=stub — returning hardcoded §1.5 spec")
        return _stub_spec()

    # ---- Live mode: call AskAT&T -------------------------------------------
    # AT&T: AskAT&T only — no external LLM calls
    client = AskATTClient()
    last_exc: BaseException | None = None

    for iteration in range(1, max_iterations + 1):
        # Build user message -------------------------------------------------
        ctx_json = json.dumps(subscription_context, indent=2)
        user_content = (
            f"Intent: {intent}\n\n"
            f"Subscription context:\n{ctx_json}"
        )

        # On iteration 2+, prepend the §4.3 refinement block.
        # The block is always added if failing_findings is non-empty, so the
        # LLM has full context on what the security gate rejected.
        if iteration >= 2 and failing_findings:
            refinement = _build_refinement_block(
                iteration, max_iterations, failing_findings
            )
            user_content = refinement + "\n\n" + user_content

        messages: list[dict[str, str]] = [
            {"role": "system", "content": _SYSTEM_PROMPT},
            {"role": "user", "content": user_content},
        ]

        _logger.debug(
            "Calling AskAT&T — iteration %d/%d", iteration, max_iterations
        )

        try:
            raw_content = await client.complete(messages)
        except LLMClientError:
            raise  # Propagate HTTP-level errors immediately.

        # Parse and validate the LLM response as a TopologySpec --------------
        try:
            spec = TopologySpec.model_validate_json(raw_content)
            _logger.debug(
                "TopologySpec parsed successfully on iteration %d", iteration
            )
            return spec
        except (ValueError, ValidationError) as exc:
            last_exc = exc
            _logger.debug(
                "TopologySpec parse failed on iteration %d (%s: %s)",
                iteration,
                type(exc).__name__,
                # Log the exception TYPE and first line only — never raw
                # LLM output that might contain sensitive data.
                str(exc).splitlines()[0] if str(exc) else "(no detail)",
            )
            # Continue to next iteration (if any remain).

    # All iterations exhausted without a valid parse.
    raise SpecValidationError(
        f"Failed to produce a valid TopologySpec after {max_iterations} "
        f"iteration(s). Last error: {type(last_exc).__name__}"
    ) from last_exc
