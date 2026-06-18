# Caveman Prompt Optimization

This demo is the **before/after playbook** for prompt compression.

## Quick numbers

| File | Approx tokens | Role |
|---|---:|---|
| `workspace.instructions.md` | ~630 | verbose baseline |
| `project.instructions.md` | ~152 | caveman rewrite |
| Difference | ~478 saved | smaller always-loaded prompt |

## Caveman rules

| Technique | What to do | Why |
|---|---|---|
| Remove filler | cut hedging and repeated reminders | keeps only meaning |
| Merge duplicates | keep one canonical rule | avoids paying twice |
| Tables over prose | turn long explanations into rows | easier to scan and cheaper to keep |
| Move detail out | put examples in a README or doc | keeps the prompt lean |

## What to review

- **Baseline:** `examples/token-optimization-demo/.github/instructions/workspace.instructions.md`
- **Caveman version:** `examples/token-optimization-demo/app/.github/instructions/project.instructions.md`
- **Outcome:** the dashboard should favor the shorter version when you copy its style back into the workspace file

## How to test it

1. Open the demo workspace in VS Code.
2. Run **Copilot Budget: Show Dashboard**.
3. Read the **Token Optimization Plan** block.
4. Replace repeated prose in the workspace file with compact rules.
5. Rerun the dashboard and compare the token estimate.

## Success criteria

- the rules still make sense to a new engineer
- repeated text disappears
- the workspace file becomes easier to skim
- the plan shows a lower always-loaded token count

If you only change one thing, make the workspace-root file shorter while keeping the same decisions and guardrails.
