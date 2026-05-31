# P0 — Foundations

**Goal:** a clean, reproducible starting point. **Effort:** ~0.5–1 day.

## Deliverables
- Pinned commit recorded in `evaluation/HLD.md` Document Control (`e17fe410`).
- A compiling working copy of the repo (generated code present).
- Existing-doc facts captured into `evaluation/Code_Briefing.md` §0–§1.

## Tasks
- [ ] Take a read-only working copy of the repo at `e17fe410` (do not modify the original).
- [ ] `docker-compose up -d` (local Mongo) → `./mvnw clean compile`; confirm `target/generated-sources` populated (MapStruct, SOAP, OpenAPI).
- [ ] Skim & extract from `README.md`, `.github/copilot-instructions.md`, `Credit.yaml`, `application.yml` into the Briefing's raw-material sections.
- [ ] Confirm `HLD_Template.md` + `Evaluation_Rubric.md` adaptations are accepted.

## Gate
SHA pinned · repo compiles · existing-doc facts captured · adapted template + rubric accepted.
