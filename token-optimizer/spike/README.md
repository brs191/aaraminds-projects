# Token Optimizer — Option C Spike Kit

A runnable measurement rig for the 2–4 week Option C spike. It self-hosts a
**LiteLLM** proxy with an **LLMLingua-2** compression hook, so you can route
real coding-assistant traffic through it and measure what compression actually
saves. See `SPIKE_PLAN.md` for objective, metrics, and the decision gate.

This is a measurement rig, not a product. Nothing here is the v0.1 blueprint.

## What's in the kit

| File | Purpose |
|---|---|
| `SPIKE_PLAN.md` | Objective, scope, metrics, timeline, decision gate |
| `docker-compose.yml` | Runs the LiteLLM proxy locally |
| `Dockerfile` | LiteLLM image + LLMLingua-2 dependencies |
| `litellm_config.yaml` | Providers, model aliases, hook registration |
| `compression_hook.py` | LLMLingua-2 compression as a LiteLLM pre-call hook |
| `requirements.txt` | Compression dependencies layered onto the image |
| `.env.example` | Provider keys and proxy auth — copy to `.env` |
| `measure.py` | A/B harness — compressed vs raw on identical prompts |
| `summarize.py` | Aggregates metrics against the gate thresholds |
| `fixtures/sample_prompts.jsonl` | Representative coding prompts for the A/B |

## Prerequisites

- Docker + Docker Compose
- An API key for the LLM provider your agents use (Azure OpenAI / Anthropic / OpenAI)
- Python 3.10+ with the `openai` package, to run `measure.py` (`pip install openai`)

## Setup

```bash
cp .env.example .env
# edit .env — set LITELLM_MASTER_KEY and the key(s) for your provider
docker compose up --build
```

First build is slow: it installs `torch` and downloads the LLMLingua-2 model
(a few hundred MB, cached afterwards in the `hf-cache` volume). The proxy is
ready when it serves `http://localhost:4000`.

Quick check:

```bash
curl http://localhost:4000/health -H "Authorization: Bearer $LITELLM_MASTER_KEY"
```

## Point your coding agent at the proxy

The proxy is OpenAI-compatible at `/v1` and Anthropic-compatible at `/v1/messages`.
Set the agent's base URL and API key, then use one of the model aliases from
`litellm_config.yaml` (`gpt-4o`, `claude-sonnet`, ...).

- **Claude Code:** set `ANTHROPIC_BASE_URL=http://localhost:4000` and `ANTHROPIC_API_KEY` to your `LITELLM_MASTER_KEY`.
- **Cursor / Continue / other BYOK editors:** set the OpenAI base URL to `http://localhost:4000/v1` and the key to your `LITELLM_MASTER_KEY`.

Every request now flows through the compression hook, and a metrics line is
written to `metrics/requests.jsonl` on the host.

## Run the A/B measurement

```bash
pip install openai
python measure.py        # sends each fixture through compressed + raw aliases
python summarize.py      # aggregates metrics + checks the decision gate
```

`measure.py` writes `results/ab_results.jsonl` with token counts, latency, and
**both answers** for each fixture. Token reduction is only half the gate —
open that file and review answer quality, especially the `code-heavy` fixture,
which is where compression is most likely to hurt.

Replace `fixtures/sample_prompts.jsonl` with prompts from genuinely
representative AITO work before trusting the numbers.

## Tuning knobs

Set in `docker-compose.yml` (the `litellm` service `environment:` block):

| Variable | Default | Effect |
|---|---|---|
| `COMPRESSION_ENABLED` | `true` | Master on/off for compression |
| `COMPRESSION_RATE` | `0.5` | Fraction of tokens to keep — lower = more aggressive |
| `MIN_CHARS_TO_COMPRESS` | `800` | Messages shorter than this are left alone |
| `LLMLINGUA_MODEL` | bert-base multilingual | The LLMLingua-2 model used |

The `-raw` model aliases (`gpt-4o-raw`, etc.) always skip compression — that is
how `measure.py` does per-request A/B without restarting the proxy.

## Important caveats

- **LLMLingua-2 was trained on prose, not code.** Compressing code blocks may
  degrade answers. The hook is conservative (skips short messages, preserves the
  latest user message), but the quality risk is real — the `code-heavy` fixture
  exists to surface it. This is the most likely cause of a Red gate outcome.
- **Compression must fail open.** On any compression error the hook logs and
  passes the request through uncompressed. Confirm this in Week 1.
- **`[VERIFY]` markers** in the files flag version-sensitive things: pin the
  LiteLLM image to a specific, advisory-checked tag (a 2026 supply-chain
  incident was reported), and confirm the LiteLLM hook API against that version.
- **Budgets** need a database — left off by default. The spike measures
  savings; budget *enforcement* is a later add (see `docker-compose.yml`).
