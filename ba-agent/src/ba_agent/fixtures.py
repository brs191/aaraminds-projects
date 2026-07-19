from __future__ import annotations

import json
from pathlib import Path
from hashlib import sha256
from typing import Any, cast

from ba_agent.models import StandupFixtureCase, StandupFixtureSet

DEFAULT_FIXTURE_PATH = Path("tests/fixtures/standup_cases.json")


def load_fixture_set(path: Path = DEFAULT_FIXTURE_PATH) -> StandupFixtureSet:
    data = cast(dict[str, Any], json.loads(path.read_text(encoding="utf-8")))
    _validate_manifest_checksum(data)
    return StandupFixtureSet.model_validate(data)


def load_fixture_case(case_id: str, path: Path = DEFAULT_FIXTURE_PATH) -> tuple[StandupFixtureSet, StandupFixtureCase]:
    fixture_set = load_fixture_set(path)
    return fixture_set, fixture_set.get_case(case_id)


def _validate_manifest_checksum(data: dict[str, Any]) -> None:
    manifest = cast(dict[str, Any], data["manifest"])
    payload = {
        "fixture_version": manifest["fixture_version"],
        "case_ids": manifest["case_ids"],
        "source_files": manifest["source_files"],
        "cases": data["cases"],
    }
    expected = "sha256:" + sha256(json.dumps(payload, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
    if manifest["checksum"] != expected:
        raise ValueError("manifest checksum does not match deterministic checksum")
