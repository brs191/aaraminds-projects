from __future__ import annotations

import argparse
import sys
from collections.abc import Sequence
from pathlib import Path

from ba_agent import __version__
from ba_agent.config import RuntimeSettings
from ba_agent.evaluation import run_eval
from ba_agent.fixtures import load_fixture_set
from ba_agent.orchestrator import run_synthetic_standup
from ba_agent.validation import validation_summary_json


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="ba-agent",
        description="BA Agent local/synthetic command surface.",
    )
    parser.add_argument("--version", action="version", version=f"ba-agent {__version__}")

    subparsers = parser.add_subparsers(dest="command")
    subparsers.add_parser("check-config", help="Validate local/synthetic runtime settings.")
    synthetic = subparsers.add_parser("synthetic", help="Run a local synthetic standup case.")
    synthetic.add_argument("case_id", nargs="?", help="Synthetic fixture case ID, for example STD-001.")
    synthetic.add_argument("--list", action="store_true", help="List available synthetic cases.")
    eval_parser = subparsers.add_parser("eval", help="Run local seed evaluation sets.")
    eval_parser.add_argument(
        "eval_set",
        nargs="?",
        choices=["GTS-STANDUP", "GTS-ROUTER", "GTS-GATE", "GTS-PLANNING", "GTS-RETRO", "GTS-HEALTH", "GTS-MVP", "GTS-P2-REQ"],
        help="Evaluation set to run.",
    )
    validate = subparsers.add_parser("validate-mcp", help="Validate local MCP validation-register shape.")
    validate.add_argument("--register", default="docs/development/mcp-validation-register.json", help="Path to local validation register JSON.")
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    if args.command is None:
        parser.print_help()
        return 0

    if args.command == "check-config":
        try:
            settings = RuntimeSettings.from_env()
        except ValueError as exc:
            print(f"configuration rejected: {exc}", file=sys.stderr)
            return 2
        print(settings.model_dump_json())
        return 0

    if args.command == "synthetic":
        if args.list:
            fixture_set = load_fixture_set()
            print("\n".join(fixture_set.manifest.case_ids))
            return 0
        if args.case_id is None:
            print("case_id is required unless --list is used", file=sys.stderr)
            return 2
        try:
            print(run_synthetic_standup(args.case_id))
        except KeyError as exc:
            print(str(exc), file=sys.stderr)
            return 2
        return 0

    if args.command == "eval":
        if args.eval_set is None:
            print("eval_set is required", file=sys.stderr)
            return 2
        result = run_eval(args.eval_set)
        print(result.model_dump_json(indent=2))
        return 0 if result.passed else 1

    if args.command == "validate-mcp":
        print(validation_summary_json(Path(args.register)))
        return 0

    parser.error(f"unsupported command: {args.command}")
    return 2
