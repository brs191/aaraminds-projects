from __future__ import annotations

from pathlib import Path

import pytest
from pydantic import ValidationError

from ba_agent.fixtures import load_fixture_set
from ba_agent.models import StandupFixtureSet


def test_load_fixture_set_validates_manifest_and_cases() -> None:
    fixture_set = load_fixture_set()

    assert fixture_set.manifest.fixture_version == "synthetic-standup-v1"
    assert "STD-001" in fixture_set.manifest.case_ids
    assert fixture_set.get_case("STD-002").git_activity == []


def test_unknown_case_raises_key_error() -> None:
    fixture_set = load_fixture_set()

    with pytest.raises(KeyError, match="unknown fixture case"):
        fixture_set.get_case("MISSING")


def test_fixture_rejects_non_synthetic_evidence() -> None:
    raw = Path("tests/fixtures/standup_cases.json").read_text(encoding="utf-8")
    bad = raw.replace("jira:synthetic:SYN/SYN-1", "jira:live:SYN/SYN-1")

    with pytest.raises(ValidationError, match="synthetic prefix"):
        StandupFixtureSet.model_validate_json(bad)


def test_fixture_rejects_bad_checksum(tmp_path: Path) -> None:
    raw = Path("tests/fixtures/standup_cases.json").read_text(encoding="utf-8")
    bad = raw.replace("sha256:3dda618a9c198fb30864ea7588b9d8c72da63fbc22586d127e5ca52765ef7d05", "sha256:bad")
    path = tmp_path / "bad_checksum.json"
    path.write_text(bad, encoding="utf-8")

    with pytest.raises(ValueError, match="checksum"):
        load_fixture_set(path)


def test_fixture_checksum_detects_content_tampering(tmp_path: Path) -> None:
    raw = Path("tests/fixtures/standup_cases.json").read_text(encoding="utf-8")
    bad = raw.replace("Complete local command scaffold", "Tampered summary")
    path = tmp_path / "tampered.json"
    path.write_text(bad, encoding="utf-8")

    with pytest.raises(ValueError, match="checksum"):
        load_fixture_set(path)
