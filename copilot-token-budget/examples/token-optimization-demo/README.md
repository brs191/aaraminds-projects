# Token Optimization Demo Workspace

This is a hands-on demo of **Caveman Prompt Optimization**.

## What this teaches

- how repeated rules inflate prompt cost
- how a compact rewrite preserves meaning
- how the dashboard's Token Optimization Plan prioritizes the biggest win first

## Compare these files

- `./.github/instructions/workspace.instructions.md` = verbose baseline
- `./app/.github/instructions/project.instructions.md` = caveman-style rewrite

## Quick test loop

1. Open this folder in VS Code.
2. Run **Copilot Budget: Show Dashboard**.
3. Read the **Token Optimization Plan** block.
4. Open the two instruction files side by side.
5. Edit the verbose file, then rerun the dashboard and compare the plan.

## What a good result looks like

- fewer repeated lines
- the same rules, but in shorter form
- a lower **Always Loaded Tokens** number
- the same meaning with less prompt overhead

## Suggested experiment

1. Copy the compact rules from `project.instructions.md` into the workspace file.
2. Remove duplicate paragraphs from the workspace file.
3. Rerun the dashboard.
4. Note how much the optimization plan changes.

The files are local-only and safe to edit or delete.
