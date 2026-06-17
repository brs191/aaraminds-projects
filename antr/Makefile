# antr — common tasks
FX ?= phase-4/fixtures/estate-multisub.json

.PHONY: help test twin eval diagram views demo
help:           ## list targets
	@grep -E '^[a-z]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*## /\t/' | sort

test:           ## Go engine tests (needs Go 1.25)
	cd engine/go && go test ./...

twin:           ## Python reference == Go engine (twin-drift)
	cd engine && python3 twin_drift_check.py

eval:           ## engine accuracy gate over the eval corpus
	python3 phase-1/eval/run_eval.py --engine-dir engine/go \
	  --fixtures-dir phase-1/eval/fixtures --answer-keys-dir phase-1/eval/answer-keys --output out/eval.json

diagram:        ## diagram-eval gate (RC checks + severity coverage)
	python3 phase-4/viz/eval_diagram.py --fixtures phase-4/fixtures phase-1/eval/fixtures --report out/diagram_eval.json

views:          ## view-family faithfulness + determinism gate
	python3 phase-4/viz/test_views.py phase-4/fixtures phase-1/eval/fixtures

demo:           ## end-to-end: fixture -> analyze -> views -> report  (FX=<fixture> to override)
	@./demo.sh $(FX)
