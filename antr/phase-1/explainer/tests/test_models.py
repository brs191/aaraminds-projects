"""Unit tests for Pydantic models.

Tests
-----
- TestFindingInputValidation  : valid model creation; invalid severity rejected.
- TestTopologyReportRagPct    : rag_grounded_pct matches grounded/total.
- TestExplainRequestEmpty     : zero-finding request → derived fields are 0/0.0.
"""

from __future__ import annotations

from datetime import datetime, timezone

import pytest
from pydantic import ValidationError

from explainer.models import (
    ExplainRequest,
    ExplainedFinding,
    FindingInput,
    TopologyReport,
)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

_NOW = datetime.now(tz=timezone.utc)


def _finding(
    *,
    type: str = "INTERNET_EXPOSURE",
    severity: str = "High",
    resource: str = "vnet-prod",
    evidence: str = "NSG rule allows 0.0.0.0/0",
    reachable: bool = True,
) -> FindingInput:
    return FindingInput(
        type=type,
        severity=severity,
        resource=resource,
        evidence=evidence,
        reachable=reachable,
    )


def _explained(
    finding: FindingInput | None = None,
    *,
    rag_grounded: bool = False,
    explanation: str | None = "Some explanation",
) -> ExplainedFinding:
    f = finding or _finding()
    return ExplainedFinding(
        **f.model_dump(),
        explanation=explanation,
        rag_grounded=rag_grounded,
    )


def _report(findings: list[ExplainedFinding]) -> TopologyReport:
    n = len(findings)
    high_critical = sum(
        1 for f in findings if f.severity in ("Critical", "High")
    )
    grounded = sum(1 for f in findings if f.rag_grounded)
    rag_pct = round(grounded / n, 4) if n > 0 else 0.0
    return TopologyReport(
        subscription_id="sub-001",
        analyzed_at=_NOW,
        findings=findings,
        high_critical_count=high_critical,
        rag_grounded_pct=rag_pct,
    )


# ---------------------------------------------------------------------------
# TestFindingInputValidation
# ---------------------------------------------------------------------------


class TestFindingInputValidation:
    def test_valid_critical(self):
        f = _finding(severity="Critical")
        assert f.severity == "Critical"
        assert f.reachable is True

    def test_valid_informational(self):
        f = _finding(severity="Informational")
        assert f.severity == "Informational"

    def test_invalid_severity_rejected(self):
        with pytest.raises(ValidationError) as exc_info:
            FindingInput(
                type="X",
                severity="UNKNOWN",    # not a valid Literal
                resource="r",
                evidence="e",
                reachable=False,
            )
        errors = exc_info.value.errors()
        assert any(e["loc"] == ("severity",) for e in errors)

    def test_missing_required_field_rejected(self):
        with pytest.raises(ValidationError):
            FindingInput(
                type="X",
                severity="High",
                # resource missing
                evidence="e",
                reachable=False,
            )

    @pytest.mark.parametrize(
        "severity", ["Critical", "High", "Medium", "Informational"]
    )
    def test_all_valid_severities(self, severity: str):
        f = _finding(severity=severity)
        assert f.severity == severity


# ---------------------------------------------------------------------------
# TestTopologyReportRagPct
# ---------------------------------------------------------------------------


class TestTopologyReportRagPct:
    def test_all_grounded(self):
        findings = [
            _explained(_finding(severity="High"), rag_grounded=True),
            _explained(_finding(severity="Critical"), rag_grounded=True),
        ]
        r = _report(findings)
        assert r.rag_grounded_pct == 1.0

    def test_none_grounded(self):
        findings = [
            _explained(_finding(), rag_grounded=False),
            _explained(_finding(), rag_grounded=False),
        ]
        r = _report(findings)
        assert r.rag_grounded_pct == 0.0

    def test_partial_grounded(self):
        findings = [
            _explained(_finding(), rag_grounded=True),
            _explained(_finding(), rag_grounded=False),
            _explained(_finding(), rag_grounded=False),
            _explained(_finding(), rag_grounded=False),
        ]
        r = _report(findings)
        assert r.rag_grounded_pct == 0.25

    def test_high_critical_count_correct(self):
        findings = [
            _explained(_finding(severity="Critical"), rag_grounded=False),
            _explained(_finding(severity="High"), rag_grounded=False),
            _explained(_finding(severity="Medium"), rag_grounded=False),
            _explained(_finding(severity="Informational"), rag_grounded=False),
        ]
        r = _report(findings)
        assert r.high_critical_count == 2

    def test_mismatched_rag_pct_rejected(self):
        """Validator rejects TopologyReport with wrong rag_grounded_pct."""
        findings = [_explained(_finding(), rag_grounded=True)]
        with pytest.raises(ValidationError):
            TopologyReport(
                subscription_id="sub-001",
                analyzed_at=_NOW,
                findings=findings,
                high_critical_count=0,
                rag_grounded_pct=0.0,  # wrong — should be 1.0
            )

    def test_mismatched_high_critical_rejected(self):
        """Validator rejects TopologyReport with wrong high_critical_count."""
        findings = [_explained(_finding(severity="Critical"), rag_grounded=False)]
        with pytest.raises(ValidationError):
            TopologyReport(
                subscription_id="sub-001",
                analyzed_at=_NOW,
                findings=findings,
                high_critical_count=0,  # wrong — should be 1
                rag_grounded_pct=0.0,
            )


# ---------------------------------------------------------------------------
# TestExplainRequestEmpty
# ---------------------------------------------------------------------------


class TestExplainRequestEmpty:
    def test_empty_request_parses(self):
        req = ExplainRequest(subscription_id="sub-empty", findings=[])
        assert req.findings == []

    def test_empty_report_derived_fields(self):
        r = _report([])
        assert r.high_critical_count == 0
        assert r.rag_grounded_pct == 0.0
        assert r.findings == []

    def test_empty_report_no_error_by_default(self):
        r = _report([])
        assert r.error is None
