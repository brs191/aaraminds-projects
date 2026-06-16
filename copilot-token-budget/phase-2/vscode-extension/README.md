# Copilot Token Budget

Local credit tracker for GitHub Copilot CLI — AT&T Enterprise edition.

Copilot Token Budget reads Copilot CLI session data from disk and surfaces your
credit consumption directly inside VS Code: a Budget Overview tree in the
activity bar, a dashboard, usage forecasting, and optional Microsoft Teams
alerts when you approach your monthly allowance. All processing is local — no
usage data leaves your machine except the Teams alert webhook you configure.

## Features

- **Budget Overview** — a dedicated activity-bar view (credit-card icon) showing
  current spend against your monthly allowance.
- **Dashboard** — a richer breakdown of usage by model and session.
- **Forecasting** — projects month-end spend from recent activity.
- **Usage export** — generate a usage report for sharing or archiving.
- **Threshold alerts** — optional WARNING / CRITICAL alerts via a Microsoft
  Teams incoming webhook.

## Commands

| Command | ID |
| --- | --- |
| Copilot Budget: Show Dashboard | `copilotBudget.showDashboard` |
| Copilot Budget: Refresh Now | `copilotBudget.refresh` |
| Copilot Budget: Open Settings | `copilotBudget.openSettings` |
| Copilot Budget: Export Usage Report | `copilotBudget.exportUsage` |

## Settings

| Setting | Default | Description |
| --- | --- | --- |
| `copilotBudget.monthlyAllowance` | `7000` | Monthly credit allowance. AT&T Enterprise promo: 7,000 cr/month until 2026-09-01. Overrides the pricing config when set. |
| `copilotBudget.workspacePath` | `""` | Workspace root for instruction-file scanning. Defaults to the first workspace folder. |
| `copilotBudget.refreshIntervalSec` | `30` | How often (seconds) to refresh session data from disk. |
| `copilotBudget.teamsWebhookUrl` | `""` | Microsoft Teams incoming webhook URL for budget alerts. |
| `copilotBudget.alertThresholdWarn` | `60` | Usage percentage that triggers a WARNING alert. |
| `copilotBudget.alertThresholdCrit` | `90` | Usage percentage that triggers a CRITICAL alert. |
| `copilotBudget.alertBinaryPath` | `""` | Absolute path to the `copilot-budget-alert` binary. Empty = auto-detect. |
| `copilotBudget.pricingPath` | `""` | Absolute path to a `pricing.json` overriding bundled per-model rates, context windows, and allowance. Partial files merge over defaults. |

## Requirements

- VS Code `^1.85.0`
- GitHub Copilot CLI installed and producing local session data.

## License

Proprietary — internal use only. See [LICENSE](./LICENSE).
