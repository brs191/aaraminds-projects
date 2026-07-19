PYTHON ?= python3
export PYTHONPATH := src

.PHONY: test typecheck check no-live cli-help synthetic-help eval-help validate-mcp

test:
	$(PYTHON) -m pytest

typecheck:
	$(PYTHON) -m mypy src tests

no-live:
	$(PYTHON) -m ba_agent check-config

cli-help:
	$(PYTHON) -m ba_agent --help

synthetic-help:
	$(PYTHON) -m ba_agent synthetic --help

eval-help:
	$(PYTHON) -m ba_agent eval --help

validate-mcp:
	$(PYTHON) -m ba_agent validate-mcp --register docs/development/mcp-validation-register.json

synthetic-demo:
	$(PYTHON) -m ba_agent synthetic STD-001

eval-standup:
	$(PYTHON) -m ba_agent eval GTS-STANDUP

eval-router:
	$(PYTHON) -m ba_agent eval GTS-ROUTER

eval-gate:
	$(PYTHON) -m ba_agent eval GTS-GATE

eval-planning:
	$(PYTHON) -m ba_agent eval GTS-PLANNING

eval-retro:
	$(PYTHON) -m ba_agent eval GTS-RETRO

eval-health:
	$(PYTHON) -m ba_agent eval GTS-HEALTH

eval-mvp:
	$(PYTHON) -m ba_agent eval GTS-MVP

eval-p2-req:
	$(PYTHON) -m ba_agent eval GTS-P2-REQ

check: test typecheck no-live cli-help synthetic-help eval-help validate-mcp eval-standup eval-router eval-gate eval-planning eval-retro eval-health eval-mvp eval-p2-req
