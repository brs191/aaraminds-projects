# Code Intelligence Factory (CIF)

An ecosystem that reverse-engineers an existing codebase, produces traceable requirements and design docs, tracks implementation through to defects, and answers two hard questions: **"is this defect a requirement gap or a developer miss?"** and **"trace this defect back to the BRD."**

CIF is **not five agents** — it is a **traceability platform** with five specialized agents around a shared spine:

```
BRD → HLD → LLD → User Story → PR → Test → Defect → Gap   (bidirectional, typed links)
```

## Components

1. **Reverse Engineering** — codebase → structured System Model (Spring Boot · React · MongoDB).
2. **Business Analyst** — System Model → BRD / HLD / LLD / User Stories, each ID'd and linked.
3. **Scrum Master** — planning, status, and traceability-leak detection (no code generation).
4. **QA** — test plan + requirement coverage map (executed by your CI, not the agent).
5. **Gap Analysis** — defect → spine walk → classification (requirement / design / dev / test gap).

## Locked decisions

- **Tracker:** Jira Cloud (via a **custom Go MCP server** we build over the REST API) · **Code:** GitHub
- **Lifecycle:** hybrid — one-time reverse-engineering now, continuous-ready
- **Autonomy:** gated (humans approve at G1 System Model, G2 BRD, G3 Gap classification)
- **Runtime:** LangGraph (Python) orchestration — durable Postgres checkpointer + `interrupt()` gates; Go retained for the MCP servers

## Layout

```
code-intelligence-factory/
├── README.md
├── docs/
│   └── ARCHITECTURE.md            # the full blueprint — start here
├── .github/
│   └── PULL_REQUEST_TEMPLATE.md   # requirements-traceable PR template
└── examples/
    └── traceability-sample.yaml   # worked example of the spine end-to-end
```

## Status

Architecture blueprint, draft for review (2026-05-29). Build starts at **Phase 0 — the spine** (see `docs/ARCHITECTURE.md` §10). Nothing else is real until the traceability service exists.

## Pilot repo & first golden fixture — clear-cortex

The pilot repo (`docs/ARCHITECTURE.md` §11 open question (a)) is **`apm0045942-credit-routing-service`** — Spring Boot + MongoDB, already on-stack. It is being comprehended **by hand** in the sibling project [`../clear-cortex`](../clear-cortex), which produces, manually, exactly CIF's Phase-1 outputs: a System Model (Code Briefing), an inferred BRD (Inferred Product Spec), and an HLD.

That hand-built HLD is CIF's **first golden-HLD fixture** — the ground truth that validates the Reverse-Engineering (G1) and Business-Analyst (G2) agents on a real repo. Phase 0 (the spine) has no dependency on it and can start now; **Phase 1's evaluation is gated on clear-cortex reaching P3.** Full handoff and sequencing: [`../clear-cortex/planning/CIF_Bridge_Roadmap.md`](../clear-cortex/planning/CIF_Bridge_Roadmap.md).
