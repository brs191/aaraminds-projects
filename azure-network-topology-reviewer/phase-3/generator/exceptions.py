"""Custom exceptions for the AT&T network topology generator.

All public exception messages are sanitised — they never contain bearer tokens,
client secrets, or any credential material.
"""

from __future__ import annotations


class LLMClientError(Exception):
    """Raised when the AskAT&T LLM client fails after all retry attempts.

    The message is deliberately stripped of any credential material before
    this exception is raised.  See ``AskATTClient.complete`` for the
    sanitisation contract.
    """


class SpecValidationError(Exception):
    """Raised when the LLM produces output that cannot be parsed into a valid
    ``TopologySpec`` after exhausting all allowed iterations.

    Wraps the underlying ``pydantic.ValidationError`` as ``__cause__`` so
    callers can inspect the schema-violation details without needing to
    catch ``pydantic.ValidationError`` directly.
    """
