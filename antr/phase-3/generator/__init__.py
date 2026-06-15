"""AT&T Network Topology Reviewer — Phase 3 generator package.

Public surface: models, exceptions, and the ``generate_spec`` entry point.
"""

from .exceptions import LLMClientError, SpecValidationError
from .intent import AskATTClient, generate_spec
from .models import (
    PeeringPairSpec,
    PrivateEndpointSpec,
    SubnetSpec,
    TopologySpec,
    VNetSpec,
)

__all__ = [
    "AskATTClient",
    "LLMClientError",
    "PeeringPairSpec",
    "PrivateEndpointSpec",
    "SpecValidationError",
    "SubnetSpec",
    "TopologySpec",
    "VNetSpec",
    "generate_spec",
]
