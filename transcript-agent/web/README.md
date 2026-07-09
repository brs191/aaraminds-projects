# Podcast Transcript Agent — Review UI

Minimal React review UI for the Podcast Transcript Agent MVP (PRD v1.5). Covers file/YouTube submission,
job list, review editor, approval, exports, summary, quality report, and audit trail.

## Stack

- Vite + React 19 + TypeScript
- @tanstack/react-query v5 (data fetching + 2s polling of active jobs)
- react-router-dom v7
- Plain CSS (`src/styles.css`), no CSS framework

## Run

```bash
npm install
npm run dev     # http://localhost:5173, proxies /api → http://localhost:8080
npm run build   # tsc + vite build
```

The dev server proxies `/api` to the Go backend on `localhost:8080` (see `vite.config.ts`).
Start the backend first; without it every screen shows a `NETWORK_ERROR` state.

## Dev identity

There is no real auth in the MVP. The header dropdown switches between
`producer-1/producer`, `reviewer-1/reviewer`, and `admin-1/admin`; the choice is stored in
localStorage and sent as `X-User-Id` / `X-User-Role` headers on every request. Only
reviewer/admin see the Approve button (server must still enforce).

## Layout

```
src/
├── api/
│   ├── types.ts     # API contract types + status enums
│   ├── client.ts    # fetch wrapper, headers, structured ApiError
│   └── hooks.ts     # TanStack Query hooks for every endpoint
├── identity.tsx     # dev identity switcher (localStorage + context)
├── pages/           # Submit, Jobs list, Job detail (tabbed)
├── tabs/            # Overview, Review, Summary, Quality, Exports, Audit
├── components/ui.tsx# badges, error box, formatting helpers
└── styles.css
```

## Contract notes

- `GET /jobs/{id}/summary` and `GET /jobs/{id}/quality-report` 404s are treated as
  "not yet available", not errors.
- Export downloads use the signed `download_url` returned by the backend in a plain anchor
  (the request has no auth headers, but does require the token in the URL).
- "Rename speaker everywhere" issues one `PATCH` per matching segment (no bulk endpoint
  in the contract).
