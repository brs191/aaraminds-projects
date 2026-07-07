from __future__ import annotations

from typing import Sequence


async def narrate_with_litellm(
    *,
    model: str,
    prompt: str,
    citations: Sequence[str],
    api_key: str | None = None,
) -> str:
    if not citations:
        raise RuntimeError("Narrative generation requires at least one source_ref citation")

    try:
        from litellm import acompletion
    except Exception:
        return _fallback_narrative(prompt, citations)

    try:
        response = await acompletion(
            model=model,
            api_key=api_key,
            messages=[
                {
                    "role": "system",
                    "content": "You are a code intelligence agent. Every claim must include at least one source_ref citation.",
                },
                {"role": "user", "content": f"{prompt}\n\nCitations:\n" + "\n".join(f"- {c}" for c in citations[:12])},
            ],
            temperature=0,
        )
        content = response.choices[0].message.content if response and response.choices else None
        if isinstance(content, str) and content.strip():
            text = content.strip()
            if not any(ref in text for ref in citations):
                return _fallback_narrative(prompt, citations)
            return text
        return _fallback_narrative(prompt, citations)
    except Exception:
        return _fallback_narrative(prompt, citations)


def _fallback_narrative(prompt: str, citations: Sequence[str]) -> str:
    top = ", ".join(citations[:5])
    return f"{prompt} Citations: {top}."
