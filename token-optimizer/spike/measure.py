"""
A/B measurement harness — Token Optimizer Option C spike.

For each fixture prompt, sends the SAME messages through the proxy twice:
  - the compressed model alias  (compression applied)
  - the "-raw" model alias       (compression skipped)
and records token usage, latency, and both answers for quality review.

Usage:
    pip install openai
    python measure.py

Environment (all optional — defaults shown):
    PROXY_BASE_URL   http://localhost:4000
    LITELLM_MASTER_KEY   sk-spike-local-change-me
    COMPRESSED_MODEL     gpt-4o
    RAW_MODEL            gpt-4o-raw
    FIXTURES             fixtures/sample_prompts.jsonl
    RESULTS              results/ab_results.jsonl

Then review results/ab_results.jsonl and run:  python summarize.py
"""

import os
import json
import time

from openai import OpenAI

PROXY = os.getenv("PROXY_BASE_URL", "http://localhost:4000")
KEY = os.getenv("LITELLM_MASTER_KEY", "sk-spike-local-change-me")
COMPRESSED_MODEL = os.getenv("COMPRESSED_MODEL", "gpt-4o")
RAW_MODEL = os.getenv("RAW_MODEL", "gpt-4o-raw")
FIXTURES = os.getenv("FIXTURES", "fixtures/sample_prompts.jsonl")
RESULTS = os.getenv("RESULTS", "results/ab_results.jsonl")

client = OpenAI(base_url=f"{PROXY}/v1", api_key=KEY)


def call(model, messages):
    t0 = time.perf_counter()
    resp = client.chat.completions.create(model=model, messages=messages)
    latency_ms = (time.perf_counter() - t0) * 1000.0
    usage = resp.usage
    return {
        "model": model,
        "latency_ms": round(latency_ms, 1),
        "prompt_tokens": usage.prompt_tokens,
        "completion_tokens": usage.completion_tokens,
        "answer": resp.choices[0].message.content,
    }


def main():
    if os.path.dirname(RESULTS):
        os.makedirs(os.path.dirname(RESULTS), exist_ok=True)

    with open(FIXTURES, encoding="utf-8") as f:
        fixtures = [json.loads(line) for line in f if line.strip()]

    print(f"Running A/B over {len(fixtures)} fixtures "
          f"({RAW_MODEL} vs {COMPRESSED_MODEL})\n")

    with open(RESULTS, "w", encoding="utf-8") as out:
        for fx in fixtures:
            messages = fx["messages"]
            try:
                raw = call(RAW_MODEL, messages)
                compressed = call(COMPRESSED_MODEL, messages)
            except Exception as e:
                print(f"  {fx.get('id')}: ERROR — {e}")
                continue

            saved = raw["prompt_tokens"] - compressed["prompt_tokens"]
            reduction = (round(100.0 * saved / raw["prompt_tokens"], 1)
                         if raw["prompt_tokens"] else 0.0)
            record = {
                "id": fx.get("id"),
                "category": fx.get("category"),
                "prompt_tokens_raw": raw["prompt_tokens"],
                "prompt_tokens_compressed": compressed["prompt_tokens"],
                "prompt_tokens_saved": saved,
                "reduction_pct": reduction,
                "latency_delta_ms": round(
                    compressed["latency_ms"] - raw["latency_ms"], 1),
                "raw": raw,
                "compressed": compressed,
            }
            out.write(json.dumps(record) + "\n")
            print(f"  {fx.get('id'):24s} -{reduction:5.1f}% tokens   "
                  f"Δlatency {record['latency_delta_ms']:+.0f} ms")

    print(f"\nWrote {RESULTS}")
    print("Next: review answers for quality, then run  python summarize.py")


if __name__ == "__main__":
    main()
