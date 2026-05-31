# 02 · Architecture

System design artifacts — the **how**.

## What's here

| File | Purpose |
| --- | --- |
| `Aara_Architecture.drawio` | Editable source. Open in [diagrams.net](https://app.diagrams.net) — four pages, one per view. |
| `Aara_Architecture.pptx` | 7-slide deck for review with stakeholders. Cover, "how to read this pack", four diagrams full-bleed, stack reference. |
| `diagrams/Aara_Architecture_01_System.svg` | **Macro view.** Four trust zones (Users → M365 → Azure → AT&T network). All components, all categories. |
| `diagrams/Aara_Architecture_02_Security.svg` | **Network & Security.** Identity perimeter, network zones, Private Endpoints, ExpressRoute, Purview/DLP, NIST 800-53 R5 mapping. |
| `diagrams/Aara_Architecture_03_DataFlow.svg` | **Sequence view.** End-to-end trace of one digest run: capture → extract → persist → deliver → interact. 9 lifelines, 19 steps. |
| `diagrams/Aara_Architecture_04_Deployment.svg` | **Deployment view.** Azure regions (East US 2 primary, West US 3 DR), specific SKUs, autoscale, subnet layout, CI/CD. |

## What goes here over time

- **`adr/`** — Architecture Decision Records (one ADR per significant decision). Format: `NNNN-title.md` with status / context / decision / consequences.
- Subsystem-specific design docs (e.g., "Adaptive Card schema", "AskAT&T client wrapper")
- API contracts / OpenAPI specs
- Threat models

## Editing diagrams

1. Open `Aara_Architecture.drawio` in [diagrams.net](https://app.diagrams.net).
2. Edit on any of the four pages. The diagrams render from embedded SVG sources.
3. Export back to SVG: `File ▸ Export As ▸ SVG…` → save into `diagrams/` with the same filename.
4. Rebuild the PPTX from the new SVGs (or replace images directly inside PowerPoint).

## Source-build pipeline (optional)

The original diagrams were programmatically generated. The build scripts live in the scratch outputs folder. If you want them in-repo for reproducible regeneration, ask and I'll move them under `build/` here.
