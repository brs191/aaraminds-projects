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

## Upload flow (Submit page)

For `source_type=upload` the primary affordance is a real file input
(`.mp3, .m4a, .wav, .mp4, .mov`). On submit the UI:

1. `POST /api/v1/uploads` — multipart/form-data, field `file`, auth headers, no manual
   `Content-Type` (the browser sets the boundary). Shows an indeterminate progress banner.
2. `POST /api/v1/jobs` with the returned `upload_uri` as `source_uri`.

`400 UNSUPPORTED_FORMAT` and `413 REQUEST_TOO_LARGE` are mapped to friendly messages.
An "Advanced: paste an upload URI" toggle accepts `mock://` / `upload://` URIs directly for
demos. The ownership attestation checkbox gates submission in both modes.

## Audio playback (Review tab)

The Review tab mints a signed audio link (`POST /signed-links {kind:"audio", id:jobID}`)
and renders a sticky `<audio controls>` bar above the segment list (`GET
/jobs/{id}/audio?token=…`, Range-capable). Each segment has a ▶ button that seeks to
`start_ms` and plays; the segment containing the current playback time is highlighted via
`timeupdate`. A 404 (`AUDIO_NOT_AVAILABLE`) — e.g. the caption-reuse path — shows a quiet
"No audio available" note instead of a player. If playback errors (typically token expiry
after 15 min) the link is re-minted once automatically.

## Signed export downloads

`GET /exports/{id}/download` requires a token now, so plain anchors would 401. The
Download button mints a link via `POST /signed-links {kind:"export", id:exportID}` and
opens the returned tokenised, site-relative URL in a new tab. The button is disabled while
minting and errors are shown inline.

## Contract notes

- `GET /jobs/{id}/summary` and `GET /jobs/{id}/quality-report` 404s are treated as
  "not yet available", not errors.
- Signed links (`POST /api/v1/signed-links`) are valid for 15 minutes and embed
  `?token=` in a site-relative URL; the follow-up GET needs no auth headers.
- `409 STATUS_CONFLICT` on approve is surfaced inside the Approve dialog
  ("Job state changed — refresh") and triggers a job refetch. The confirm button is also
  disabled while any segment `PATCH` is still in flight.
- "Rename speaker everywhere" issues one `PATCH` per matching segment (no bulk endpoint
  in the contract).
