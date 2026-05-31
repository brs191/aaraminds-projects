# M0-lite — Pilot Cohort Recruitment

**Locked:** 2026-05-27  ·  **Owner:** Raja  ·  **Target:** 3–5 developers committed to a 1-week routing experiment
**Source:** `../tracking/milestones/M0-Spike.md` (to be repurposed) · `../tracking/milestones/M1-Decision-Gate.md` (GREEN verdict, conditional on M0-lite)

## Why this exists

The M1 GREEN verdict (2026-05-27) is conditional on M0-lite measuring **R ≥ 10% incremental** token reduction on real AITO code-heavy prompts. Without that validation, the build doesn't start. M0-lite is 1 week. 3–5 willing developers is the entire input. This file makes recruiting them a 30-minute task.

## What you need from the cohort

| Requirement | Why | How to verify |
|---|---|---|
| **VS Code as primary IDE** | M2-lite has zero IntelliJ coverage. An IntelliJ-only dev in the cohort is wasted spend. | Ask directly: "Do you do 90%+ of your AI-assisted coding in VS Code?" |
| **Daily use of an AI coding assistant** | M0-lite measures real usage, not synthetic. A dev who uses Copilot twice a week produces too thin a sample. | Ask: "How many days per week do you use Claude Code / Copilot / Cursor / similar?" Floor: 4 days/week. |
| **Working on code-heavy tasks during the week** | The GREEN math's tightest constraint is on code-heavy fixtures (3% quality regression cap). Devs writing docs all week don't stress the right surface. | Ask about current sprint work: backend / infra / data-engineering / frontend logic. Avoid pure UI tweaking, pure docs, pure config. |
| **Willing to point their agent at a localhost proxy** | The optimizer interposes. Devs uncomfortable with that won't generate honest data. | Explain the setup is `docker-compose up` + agent config change; reversible by removing the config line. |
| **Willing to share `metrics/requests.jsonl` and answer a 10-minute exit interview** | The data is the deliverable. | Confirm in the recruitment ask. |

## Inclusion screen — copy/paste

Use this as a Slack DM or 5-minute hallway conversation:

> Hey — I'm running a 1-week experiment that measures whether a local prompt compressor saves us meaningful tokens on real coding work. I need 3–5 people for the pilot.
>
> Quick screen:
> 1. Do you use VS Code for 90%+ of your AI-assisted coding?
> 2. How many days a week do you use Claude Code / Copilot / Cursor?
> 3. What kind of work are you doing this/next week — backend, frontend logic, infra, data?
> 4. Are you OK pointing your agent at a localhost proxy for the week? (It's a `docker-compose up` + one agent config line. Reversible.)
> 5. Will you share an anonymised `requests.jsonl` and do a 10-minute debrief at the end?
>
> If you're a yes on all five, I'll send setup docs. Time commitment: ~30 min to set up + however you'd normally code that week.

## The five-yes criterion

Don't recruit on enthusiasm. Recruit on the five questions. A "yes-ish" on question 1 (uses IntelliJ sometimes) or question 3 (mostly UI work this week) introduces noise the 1-week sample can't absorb.

## Target composition

| Slot | Profile | Notes |
|---|---|---|
| 1 | Backend / API work | Tests the compression on long context windows + structured data. |
| 2 | Backend / API work | Second backend dev to break ties on quality calls. |
| 3 | Infra / DevOps with code-heavy weeks | Stresses the prompt mix toward configs, scripts, terraform — different distribution from app code. |
| 4 (optional) | Frontend with real logic | Different prompt shape; reveals if compression hurts JSX/TS-heavy contexts. |
| 5 (optional) | Data engineering / SQL-heavy | Code-adjacent prompts with lots of column-name noise that LLMLingua-2 will try to compress. |

3 is the floor (slots 1–3). 5 is the ceiling — beyond that the 1-week measurement isn't really cheaper than the original 2-4 week M0.

## Disqualifiers — be ruthless

- IntelliJ-primary developers. (Not "uses IntelliJ for X" — primary.)
- Devs who use AI assistants ≤ 3 days/week.
- Devs whose current week is dominated by code review or non-coding work.
- Devs with strong opinions about the experiment's outcome (selection bias for both yes and no answers).

## Setup pack — what each cohort member gets

M0-lite uses the existing `../spike/` kit only — NO VS Code `.vsix` yet (that's M2-lite scope, built only after M0-lite clears the R gate). Each dev gets:

1. Link to the spike kit: `../spike/README.md`.
2. The pinned LiteLLM image tag: `ghcr.io/berriai/litellm:v1.83.14-stable.patch.3` (pinned 2026-05-27 in `../spike/Dockerfile`).
3. Two commands + one config change:
   - Clone the repo, then `docker-compose up --build` in `../spike/`.
   - Point their AI coding agent (Claude Code / Copilot / Cursor) at `http://localhost:4000` instead of its default endpoint. Reversible — just remove the line.
4. A daily check-in (~30 seconds): "Did the proxy stay up today? Any answer quality you noticed?"
5. Exit interview script (separate doc; write closer to end of measurement week).

## When the cohort is secured

Mark task #2 complete with the names committed + dates blocked + IDE confirmation per member. Cohort securing unblocks M0-lite (task #15) along with task #1 (LiteLLM image pin).

## Committed cohort (2026-05-27)

**Protocol:** 2-day PoC → 9-day intermission for fixes → 5-day full measurement.

**PoC dates:** Thu 2026-05-28 → Fri 2026-05-29 (this week — operational gate, R signal is informative not gating)
**Measurement dates:** Mon 2026-06-08 → Fri 2026-06-12 (R gate; full 5-day window with same cohort)
**Conflicts checked:** None (confirmed 2026-05-27)
**Aggregate screen:** All 5 questions passed. Team composition 70/30 backend/frontend.

| # | Name | Role | VS Code primary? | Setup-pack sent | PoC kit running | Measurement complete |
|---|---|---|---|---|---|---|
| 1 | Namratha | TBD | ☐ | ☐ | ☐ | ☐ |
| 2 | Bharat | TBD | ☐ | ☐ | ☐ | ☐ |
| 3 | Mounika | TBD | ☐ | ☐ | ☐ | ☐ |
| 4 | Pranitha | TBD | ☐ | ☐ | ☐ | ☐ |
| 5 | Karthick | TBD | ☐ | ☐ | ☐ | ☐ |
| 6 | Rohit | TBD | ☐ | ☐ | ☐ | ☐ |
| 7 | Dhyan | TBD | ☐ | ☐ | ☐ | ☐ |

**Wait list (from the original 13 willing devs):** Chansi, Pritam, Shankar, Saurabh, Ranjith, Shyla. Available to swap in if any of the committed 7 hit a conflict or are insufficient.

**Per-dev backend/frontend mapping** — fill in the Role column for each. Composition target was 5 backend + 2 frontend per the team's 70/30 split. If the locked 7 land materially off that ratio, surface in the PoC retrospective.

**Per-dev Q3 sanity check** — the aggregate "70/30" describes team composition, not what each named dev is doing during the PoC and measurement windows specifically. Confirm per-dev that the work is substantive coding (not pure CSS, not pure docs, not pure config) during BOTH date windows.
