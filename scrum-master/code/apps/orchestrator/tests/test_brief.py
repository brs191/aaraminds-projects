"""Pure unit tests for the brief/report builders — no network, no DB."""

from scrum_orchestrator.brief import build_brief, build_report, brief_title

SPRINT = {"id": 101, "name": "CRS Sprint 24", "goal": "Stabilize the API"}
ISSUES = [
    {"key": "CRS-405", "summary": "Audit log", "status": "Done", "assignee": "Priya N.",
     "blocked": False, "timeoriginalestimate": 18000, "timeestimate": 0, "timespent": 17100, "daysInStatus": 1},
    {"key": "CRS-417", "summary": "SOAP faults", "status": "In Progress", "assignee": "Marco D.",
     "blocked": True, "blockReason": "is blocked by CRS-420", "timeoriginalestimate": 14400,
     "timeestimate": 10800, "timespent": 3600, "daysInStatus": 4},
    {"key": "CRS-420", "summary": "Sandbox creds", "status": "Blocked", "assignee": None,
     "blocked": True, "blockReason": "Vendor ticket open", "timeoriginalestimate": 7200,
     "timeestimate": 7200, "timespent": 0, "daysInStatus": 6},
    {"key": "CRS-431", "summary": "Mongo pool", "status": "To Do", "assignee": "Sam K.",
     "blocked": False, "timeoriginalestimate": None, "timeestimate": None, "timespent": 0, "daysInStatus": 2},
]


def test_title():
    assert brief_title(SPRINT) == "Daily Scrum Brief — CRS Sprint 24"


def test_brief_flags_blockers_stale_and_missing_estimate():
    md = build_brief(SPRINT, ISSUES, stale_days=3)
    assert "CRS-420" in md and "CRS-417" in md
    assert "Blockers (2)" in md
    assert "Stale" in md and "6 days in Blocked" in md
    assert "CRS-405" in md
    # time-based summary, never story points
    assert "committed" in md and "remaining" in md
    assert "point" not in md.lower()
    assert "missing time estimate on CRS-431" in md


def test_done_item_not_flagged_stale():
    md = build_brief(SPRINT, ISSUES, stale_days=3)
    stale_section = md.split("## Stale")[1] if "## Stale" in md else ""
    assert "CRS-405" not in stale_section  # Done items never stale


def test_blocked_status_issue_appears_in_body_not_only_blockers():
    # Regression: a "Blocked"-status issue must be bucketed into the body (Other),
    # not silently dropped from the status groups. CRS-420 has status "Blocked".
    md = build_brief(SPRINT, ISSUES, stale_days=3)
    assert "## Other" in md
    other_section = md.split("## Other")[1].split("##")[0]
    assert "CRS-420" in other_section


def test_report_has_toc():
    report = build_report(SPRINT, {"Went well": "shipped audit log", "Risks": "vendor blocker"})
    assert "## Table of contents" in report
    assert "[Went well](#went-well)" in report
    assert "[Risks](#risks)" in report
