"""Pure brief/report builders — no I/O, stdlib only, so they are trivially testable.

Estimation is TIME-BASED per the locked decision (scrum-master/planning/Open_Questions.md):
we read Jira time-tracking fields (timeoriginalestimate / timeestimate / timespent,
all in seconds) — never story points.
"""

from __future__ import annotations

from typing import Any

Issue = dict[str, Any]


def _hours(seconds: Any) -> str:
    if seconds is None:
        return "—"
    return f"{seconds / 3600:.1f}h"


def _sum(issues: list[Issue], field: str) -> int:
    return sum(int(i.get(field) or 0) for i in issues)


def brief_title(sprint: dict[str, Any]) -> str:
    return f"Daily Scrum Brief — {sprint.get('name', 'Active Sprint')}"


def build_brief(sprint: dict[str, Any], issues: list[Issue], stale_days: int = 3) -> str:
    """Markdown daily brief: grouped by status, blockers + stale + missing-estimate
    flagged, every line cites its issue key, time summary is time-based."""
    done = [i for i in issues if i.get("status", "").lower() == "done"]
    in_progress = [i for i in issues if i.get("status", "").lower() == "in progress"]
    todo = [i for i in issues if i.get("status", "").lower() in ("to do", "todo", "backlog")]
    # Catch-all: any issue not in the three buckets above (e.g. status "Blocked",
    # "In Review") still appears in the brief body, not only in the Blockers section.
    other = [i for i in issues if i not in done and i not in in_progress and i not in todo]
    blocked = [i for i in issues if i.get("blocked")]
    stale = [
        i
        for i in issues
        if i.get("daysInStatus", 0) >= stale_days and i.get("status", "").lower() != "done"
    ]
    missing_estimate = [
        i for i in issues if i.get("timeoriginalestimate") is None and i.get("status", "").lower() != "done"
    ]

    lines: list[str] = []
    lines.append(f"# {brief_title(sprint)}")
    if sprint.get("goal"):
        lines.append(f"_Sprint goal: {sprint['goal']}_")
    lines.append(f"_{len(issues)} issues · committed {_hours(_sum(issues, 'timeoriginalestimate'))} · "
                 f"remaining {_hours(_sum(issues, 'timeestimate'))} · logged {_hours(_sum(issues, 'timespent'))}_")
    lines.append("")

    lines.append(f"## Done since yesterday ({len(done)})")
    if done:
        lines.extend(_done_line(i) for i in done)
    else:
        lines.append("- _nothing_")
    lines.append("")

    lines.append(f"## In progress ({len(in_progress)})")
    if in_progress:
        lines.extend(_wip_line(i) for i in in_progress)
    else:
        lines.append("- _nothing_")
    lines.append("")

    if other:
        lines.append(f"## Other ({len(other)})")
        lines.extend(_wip_line(i) for i in other)
        lines.append("")

    if blocked:
        lines.append(f"## Blockers ({len(blocked)})")
        lines.extend(_blocked_line(i) for i in blocked)
        lines.append("")

    if stale:
        lines.append(f"## Stale (no movement >= {stale_days} days)")
        lines.extend(
            f"- **{i['key']}** — {i.get('daysInStatus')} days in {i.get('status')}" for i in stale
        )
        lines.append("")

    if todo:
        lines.append(f"## To do ({len(todo)})")
        lines.extend(_todo_line(i) for i in todo)
        lines.append("")

    if missing_estimate:
        keys = ", ".join(i["key"] for i in missing_estimate)
        lines.append(f"> Hygiene: missing time estimate on {keys}")

    return "\n".join(lines).rstrip() + "\n"


def _assignee(i: Issue) -> str:
    return i.get("assignee") or "unassigned"


def _done_line(i: Issue) -> str:
    return f"- **{i['key']}** — {i.get('summary', '')} ({_assignee(i)})"


def _wip_line(i: Issue) -> str:
    flag = " — BLOCKED" if i.get("blocked") else ""
    return f"- **{i['key']}** — {i.get('summary', '')} ({_assignee(i)}) · {_hours(i.get('timeestimate'))} left{flag}"


def _blocked_line(i: Issue) -> str:
    reason = i.get("blockReason", "blocked")
    age = i.get("daysInStatus")
    age_str = f", {age} days in {i.get('status')}" if age is not None else ""
    return f"- **{i['key']}** — {i.get('summary', '')} ({_assignee(i)}{age_str}) — {reason}"


def _todo_line(i: Issue) -> str:
    est = "" if i.get("timeoriginalestimate") is not None else " · no estimate"
    return f"- **{i['key']}** — {i.get('summary', '')} ({_assignee(i)}){est}"


# --- Retro report with table of contents (locked decision: emit Report.md w/ TOC).
# Used by the Sprint Closing / Retro feature (P1). Included here so the contract
# exists; the P0 demo path uses build_brief above.

def build_report(sprint: dict[str, Any], sections: dict[str, str]) -> str:
    """Assemble a Report.md with a navigable table of contents from {heading: body}."""
    title = f"Sprint Report — {sprint.get('name', 'Sprint')}"
    toc = ["## Table of contents"]
    body: list[str] = []
    for heading, content in sections.items():
        anchor = heading.lower().replace(" ", "-")
        toc.append(f"- [{heading}](#{anchor})")
        body.append(f"## {heading}\n\n{content}\n")
    return f"# {title}\n\n" + "\n".join(toc) + "\n\n" + "\n".join(body)
