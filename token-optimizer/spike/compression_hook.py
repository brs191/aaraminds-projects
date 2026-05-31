"""
LLMLingua-2 compression hook for the LiteLLM proxy — Token Optimizer Option C spike.

Registered in litellm_config.yaml as:

    litellm_settings:
      callbacks: ["compression_hook.proxy_handler_instance"]

Behaviour
---------
- `async_pre_call_hook` runs inside the proxy after auth, before the request
  leaves for the provider. It can mutate `data` (the request payload).
- Eligible message content is compressed with LLMLingua-2.
- The latest `user` message (the current ask) is preserved verbatim.
- Messages shorter than MIN_CHARS_TO_COMPRESS are skipped — compression
  overhead is not worth it on small messages.
- Compression is FAIL-OPEN: any error logs and passes the request through raw.
  The hook must never break a developer's request.
- One JSONL metrics record is written per request to METRICS_PATH.

A/B switch
----------
Per-request A/B without restarts: if the requested model alias contains
"-raw", compression is skipped for that request. Define both a compressed
alias and a `<name>-raw` alias in litellm_config.yaml pointing at the same
backend. `measure.py` uses this to compare identical prompts.

[VERIFY] The `CustomLogger` import path and the `async_pre_call_hook`
signature can change between LiteLLM versions. Confirm against the version
pinned in the Dockerfile: https://docs.litellm.ai/docs/proxy/call_hooks
"""

import os
import json
import time
import datetime

from litellm.integrations.custom_logger import CustomLogger
import litellm

# --- Configuration (all overridable via environment) -----------------------

COMPRESSION_ENABLED = os.getenv("COMPRESSION_ENABLED", "true").lower() == "true"
COMPRESSION_RATE = float(os.getenv("COMPRESSION_RATE", "0.5"))          # keep ~50% of tokens
MIN_CHARS_TO_COMPRESS = int(os.getenv("MIN_CHARS_TO_COMPRESS", "800"))  # skip small messages
RAW_ALIAS_MARKER = os.getenv("RAW_ALIAS_MARKER", "-raw")               # model aliases containing this skip compression
METRICS_PATH = os.getenv("METRICS_PATH", "/app/metrics/requests.jsonl")
LLMLINGUA_MODEL = os.getenv(
    "LLMLINGUA_MODEL",
    "microsoft/llmlingua-2-bert-base-multilingual-cased-meetingbank",
)
# Token used only for counting/metrics — not the model the request is sent to.
COUNTER_MODEL = os.getenv("COUNTER_MODEL", "gpt-4o")

# Structural tokens LLMLingua-2 must not drop — protects formatting/code structure.
FORCE_TOKENS = ["\n", ".", "!", "?", ",", ":", ";", "(", ")", "{", "}", "[", "]"]

# --- Lazy LLMLingua-2 loader (model downloads on first use, ~hundreds of MB) -

_compressor = None


def _get_compressor():
    global _compressor
    if _compressor is None:
        from llmlingua import PromptCompressor
        _compressor = PromptCompressor(model_name=LLMLINGUA_MODEL, use_llmlingua2=True)
    return _compressor


def _count_tokens(text: str) -> int:
    """Best-effort token count for metrics."""
    try:
        return litellm.token_counter(model=COUNTER_MODEL, text=text)
    except Exception:
        return max(1, len(text) // 4)  # rough fallback


# --- The hook ---------------------------------------------------------------


class CompressionHook(CustomLogger):
    async def async_pre_call_hook(self, user_api_key_dict, cache, data, call_type):
        # Only touch chat/completion calls; leave embeddings, images, etc. alone.
        if call_type not in ("completion", "acompletion", "text_completion"):
            return data

        messages = data.get("messages")
        if not messages:
            return data

        model = data.get("model", "") or ""
        compress = COMPRESSION_ENABLED and (RAW_ALIAS_MARKER not in model)

        started = time.perf_counter()
        original_tokens = _sum_tokens(messages)
        failures = 0

        if compress:
            last_user_idx = _last_user_index(messages)
            compressor = _get_compressor()
            for i, msg in enumerate(messages):
                if i == last_user_idx:
                    continue  # preserve the current ask verbatim
                content = msg.get("content")
                if not isinstance(content, str) or len(content) < MIN_CHARS_TO_COMPRESS:
                    continue
                try:
                    result = compressor.compress_prompt(
                        content,
                        rate=COMPRESSION_RATE,
                        force_tokens=FORCE_TOKENS,
                        drop_consecutive=True,
                    )
                    msg["content"] = result["compressed_prompt"]
                except Exception:
                    failures += 1  # fail open — keep the original content

        compressed_tokens = _sum_tokens(messages)
        hook_ms = (time.perf_counter() - started) * 1000.0

        self._write_metric({
            "ts": datetime.datetime.utcnow().isoformat() + "Z",
            "model": model,
            "call_type": call_type,
            "compression_applied": compress,
            "original_tokens": original_tokens,
            "compressed_tokens": compressed_tokens,
            "tokens_saved": original_tokens - compressed_tokens,
            "reduction_pct": (
                round(100.0 * (original_tokens - compressed_tokens) / original_tokens, 1)
                if original_tokens else 0.0
            ),
            "hook_latency_ms": round(hook_ms, 1),
            "compression_failures": failures,
        })
        return data

    @staticmethod
    def _write_metric(record: dict):
        try:
            os.makedirs(os.path.dirname(METRICS_PATH), exist_ok=True)
            with open(METRICS_PATH, "a", encoding="utf-8") as f:
                f.write(json.dumps(record) + "\n")
        except Exception:
            pass  # metrics must never break a request


def _sum_tokens(messages) -> int:
    return sum(
        _count_tokens(m["content"])
        for m in messages
        if isinstance(m.get("content"), str)
    )


def _last_user_index(messages) -> int:
    return max(
        (i for i, m in enumerate(messages) if m.get("role") == "user"),
        default=-1,
    )


# Instance referenced by litellm_config.yaml callbacks list.
proxy_handler_instance = CompressionHook()
