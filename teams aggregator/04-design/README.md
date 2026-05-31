# 04 · Design

UX artifacts — the bot's surface across Teams.

## Suggested layout (create as needed)

- `mockups/` — static mocks of Adaptive Cards, in-channel digests, the admin setup card
- `prototypes/` — Figma links, interactive prototypes
- `flows/` — flowcharts for first-time install, on-demand summary, cross-functional `@Aara` mention, scheduled Monday digest
- `system/` — design tokens (colors, typography, spacing) for the Adaptive Cards
- `copy/` — microcopy: button labels, error messages, empty states, slash-command help

## What v1 needs

Per PRD §6 and §7:

- **Cards:** digest card with TL;DR + action items, action-item edit modal, "Mark complete" / "Reassign" interactions
- **Onboarding:** the 60-second setup card the bot DMs the installer
- **Empty states:** what the digest looks like when there's no significant activity in the window
- **Error states:** failed AskAT&T call, Graph rate-limit, permission denied
- **Email:** the exec-stakeholder weekly digest in Outlook (HTML email template)
