"""AskAT&T GenAI client.

Authentication strategy (tried in order):
  1. azure.identity.aio.DefaultAzureCredential  — works in Container Apps with MI.
  2. Client-credentials OAuth2 flow via ASKAT_TOKEN_URL  — local / non-Azure envs.

Token is cached and refreshed 60 s before expiry.

The Authorization header is NEVER emitted to structlog.  The `_redact` processor
in app.py and the explicit guard in _build_headers() enforce this.

Retry: 3 attempts, 2× exponential back-off, capped at 30 s.
On any unrecoverable error: return None (explanation=None).  Never raise.
"""

from __future__ import annotations

import asyncio
import json
import time
from dataclasses import dataclass, field
from typing import Optional

import httpx
import structlog

from explainer.models import FindingInput, LLMSettings

log = structlog.get_logger(__name__)

# ---------------------------------------------------------------------------
# Prompt templates
# ---------------------------------------------------------------------------

_EXPLAIN_PROMPT = """\
You are an Azure network security expert helping AT&T engineers understand \
findings from an automated topology reviewer.

Finding type: {type}
Severity: {severity}
Affected resource: {resource}
Evidence: {evidence}
Traffic reachable: {reachable}

In 2–3 sentences, explain what this finding means in plain English and its \
potential security impact.
Do not restate severity or resource name. \
Do not recommend fixes — that is the RAG layer's job.\
"""

_SUMMARY_PROMPT = """\
Summarise these {n} Azure network security findings in 2 sentences for a \
senior network engineer.
Focus on the most critical patterns, not individual findings.
Findings: {json_findings_list}\
"""

# ---------------------------------------------------------------------------
# Token cache
# ---------------------------------------------------------------------------


@dataclass
class _CachedToken:
    value: str
    expires_at: float  # unix epoch seconds


# ---------------------------------------------------------------------------
# LLM client
# ---------------------------------------------------------------------------


class LLMClient:
    """Async AskAT&T GenAI client with dual-auth, token caching, and retry."""

    def __init__(self, settings: LLMSettings) -> None:
        self._settings = settings
        self._http = httpx.AsyncClient(
            timeout=httpx.Timeout(connect=10.0, read=60.0, write=10.0, pool=5.0),
        )
        self._token: _CachedToken | None = None
        self._token_lock = asyncio.Lock()
        self._auth_mode: str = "unknown"  # set on first successful token fetch

    # ------------------------------------------------------------------
    # Public interface
    # ------------------------------------------------------------------

    async def explain(self, finding: FindingInput) -> str | None:
        """Return a 2–3 sentence plain-English explanation, or None on failure."""
        prompt = _EXPLAIN_PROMPT.format(
            type=finding.type,
            severity=finding.severity,
            resource=finding.resource,
            evidence=finding.evidence,
            reachable=finding.reachable,
        )
        return await self._complete(prompt, span_name="explain.llm")

    async def summarise(self, explained: list[dict]) -> str | None:
        """Return a 2-sentence summary of all findings, or None on failure."""
        # Omit evidence/resource — may contain sensitive IPs; pass only
        # type, severity, explanation for the synthesis prompt.
        safe_list = [
            {"type": f["type"], "severity": f["severity"], "explanation": f.get("explanation")}
            for f in explained
        ]
        prompt = _SUMMARY_PROMPT.format(
            n=len(explained),
            json_findings_list=json.dumps(safe_list, indent=None),
        )
        return await self._complete(prompt, span_name="explain.summary")

    async def aclose(self) -> None:
        await self._http.aclose()

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    async def _complete(self, prompt: str, span_name: str) -> str | None:
        """Call the LLM with retry; return text or None."""
        try:
            from opentelemetry import trace as otel_trace

            tracer = otel_trace.get_tracer(__name__)
        except Exception:
            tracer = None

        async def _attempt() -> str:
            headers = await self._build_headers()
            payload = {
                "model": self._settings.model,
                "messages": [{"role": "user", "content": prompt}],
            }
            resp = await self._http.post(
                f"{self._settings.endpoint}/chat/completions",
                headers=headers,
                json=payload,
            )
            if resp.status_code == 429 or resp.status_code >= 500:
                resp.raise_for_status()  # triggers retry
            resp.raise_for_status()
            data = resp.json()
            return data["choices"][0]["message"]["content"].strip()

        result = await self._with_retry(_attempt)
        return result

    async def _with_retry(
        self,
        fn,
        max_retries: int = 3,
    ) -> str | None:
        """Run *fn* with exponential back-off; return None on exhaustion."""
        delay = 2.0
        for attempt in range(max_retries + 1):
            try:
                return await fn()
            except (httpx.TimeoutException, httpx.HTTPStatusError) as exc:
                retryable = (
                    isinstance(exc, httpx.TimeoutException)
                    or (
                        isinstance(exc, httpx.HTTPStatusError)
                        and exc.response.status_code in (429, 500, 502, 503, 504)
                    )
                )
                if not retryable or attempt == max_retries:
                    log.warning(
                        "llm.request.failed",
                        attempt=attempt,
                        error=str(exc),
                    )
                    return None
                sleep_for = min(delay, 30.0)
                log.info("llm.request.retry", attempt=attempt, sleep_s=sleep_for)
                await asyncio.sleep(sleep_for)
                delay *= 2
            except Exception as exc:
                log.warning("llm.request.unexpected_error", error=str(exc))
                return None
        return None  # unreachable but satisfies mypy

    async def _build_headers(self) -> dict[str, str]:
        """Return HTTP headers with Bearer token.  Authorization is NEVER logged."""
        token = await self._get_token()
        # Do NOT pass this dict to any logger.
        return {
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
        }

    # ------------------------------------------------------------------
    # Token management
    # ------------------------------------------------------------------

    async def _get_token(self) -> str:
        """Return a valid token, refreshing if within 60 s of expiry."""
        async with self._token_lock:
            if self._token and time.monotonic() < self._token.expires_at - 60:
                return self._token.value
            cached = await self._fetch_token()
            self._token = cached
            log.info("llm.token.refreshed", auth_mode=self._auth_mode)
            return cached.value

    async def _fetch_token(self) -> _CachedToken:
        """Try MI (DefaultAzureCredential) first; fall back to client-creds."""
        try:
            return await self._fetch_via_managed_identity()
        except Exception as mi_exc:
            log.info(
                "llm.auth.mi_failed_falling_back",
                reason=str(mi_exc),
            )
            return await self._fetch_via_client_credentials()

    async def _fetch_via_managed_identity(self) -> _CachedToken:
        from azure.identity.aio import DefaultAzureCredential  # type: ignore[import]
        from azure.core.exceptions import ClientAuthenticationError  # type: ignore[import]

        credential = DefaultAzureCredential()
        try:
            token = await credential.get_token(self._settings.scope)
            self._auth_mode = "managed_identity"
            return _CachedToken(
                value=token.token,
                expires_at=time.monotonic() + (token.expires_on - time.time()),
            )
        except ClientAuthenticationError:
            raise
        finally:
            await credential.close()

    async def _fetch_via_client_credentials(self) -> _CachedToken:
        """OAuth2 client-credentials grant against ASKAT_TOKEN_URL."""
        s = self._settings
        if not s.token_url or not s.client_id or not s.client_secret:
            raise RuntimeError(
                "ASKAT_TOKEN_URL / ASKAT_CLIENT_ID / ASKAT_CLIENT_SECRET "
                "must be set when not running with Managed Identity"
            )
        resp = await self._http.post(
            s.token_url,
            data={
                "grant_type": "client_credentials",
                "client_id": s.client_id,
                "client_secret": s.client_secret,
                "scope": s.scope,
            },
            headers={"Content-Type": "application/x-www-form-urlencoded"},
        )
        resp.raise_for_status()
        body = resp.json()
        expires_in: int = int(body.get("expires_in", 3600))
        self._auth_mode = "client_credentials"
        return _CachedToken(
            value=body["access_token"],
            expires_at=time.monotonic() + expires_in,
        )
