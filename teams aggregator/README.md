# Aara — Teams Aggregator

A Microsoft Teams bot that listens across designated Aara chats and `@Aara` mentions, then produces a single, citation-backed digest of **summary, action items, decisions, path forward, and follow-ups**. Hosted on Azure, powered by **AskAT&T**, and integrated across the Microsoft 365 estate (Teams, Graph, Planner, Loop, SharePoint, Outlook, Purview).

## Project layout

```
Teams Aggregator/
├── README.md                       this file — the map of the project
├── 01-product/                     what we're building and why
│   ├── Aara_Teams_Aggregator_PRD.docx          v0.2 PRD
│   ├── Aara_Decisions_Needed.docx              12 open questions, owners, due dates
│   └── Aara_Stakeholder_Review_Brief.pptx      8-slide v0.2 review deck
├── 02-architecture/                how we're building it
│   ├── Aara_Architecture.drawio    editable source (open in diagrams.net)
│   ├── Aara_Architecture.pptx      review/share deck (7 slides)
│   └── diagrams/                   rendered SVG exports for embedding
│       ├── Aara_Architecture_01_System.svg
│       ├── Aara_Architecture_02_Security.svg
│       ├── Aara_Architecture_03_DataFlow.svg
│       └── Aara_Architecture_04_Deployment.svg
├── 03-engineering/                 tech specs, ADRs, runbooks, code
├── 04-design/                      UX mockups, prototypes, design system
├── 05-rollout/                     GTM, comms, training, launch plan
└── 99-archive/                     superseded versions
```

## Where things live

| If you need... | Look in... |
| --- | --- |
| The product spec (problem, goals, scope, M365 surface, AskAT&T LLM choice, risks, rollout, open questions) | `01-product/Aara_Teams_Aggregator_PRD.docx` |
| The 12 decisions blocking forward motion — owners, dates, recommendations | `01-product/Aara_Decisions_Needed.docx` |
| Deck for the v0.2 stakeholder review meeting | `01-product/Aara_Stakeholder_Review_Brief.pptx` |
| The macro view (4 trust zones, all components) | `02-architecture/diagrams/Aara_Architecture_01_System.svg` |
| The IT/Security review artifact (identity, perimeters, NIST mapping) | `02-architecture/diagrams/Aara_Architecture_02_Security.svg` |
| End-to-end digest run (sequence, 19 steps) | `02-architecture/diagrams/Aara_Architecture_03_DataFlow.svg` |
| Azure regions, SKUs, scaling, DR, CI/CD | `02-architecture/diagrams/Aara_Architecture_04_Deployment.svg` |
| To **edit** any diagram | `02-architecture/Aara_Architecture.drawio` — open in diagrams.net |
| To present to stakeholders | `02-architecture/Aara_Architecture.pptx` |

## Status

- **Version:** 0.2 (Draft — LLM swapped to AskAT&T, M365 leverage made explicit)
- **Owner:** Raja (Product)
- **Updated:** May 2026
- **Stage:** Pending stakeholder review

## Open questions (extract from PRD §15)

- AskAT&T model pinning, rate limits, JSON-mode support, embeddings endpoint, network connectivity (ExpressRoute / Private Link / AT&T peering — owner & lead time)
- Initial Aara channel list — needs the explicit list from the PM
- IT owner for Teams app registration and RSC approval
- Planner write-back: bidirectional in v1, or one-way in v1.1?
- Retention policy on raw messages vs. extracted items
- Authorization model for `/Aara channels add`
- Teams app catalog approval timeline
- Non-English language support in v1

See the PRD for the full list and the answers we already have.
