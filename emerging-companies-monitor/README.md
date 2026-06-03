# Emerging Companies Monitor

A practical, single-user watchlist monitor for Indian SME / small-cap / emerging mid-cap
businesses. It watches **BSE corporate announcements** for your names, flags the
early-warning events that cause permanent capital loss (auditor exits, promoter pledging,
rating downgrades) and the positive momentum signals (large orders, capacity expansion),
and writes a self-contained `dashboard.html` you open in a browser.

It is deliberately small. No scores out of 100, no buy/sell calls, no price prediction.
The flags tell you **which filing to read** — your judgment sets the thesis.

---

## Quick start

```bash
# 1. (optional) install the one dependency for live data
pip install -r requirements.txt

# 2. see it work immediately — bundled sample data, no network
python monitor.py --demo --open

# 3. when your watchlist codes are verified, pull live data
python monitor.py --open
```

`--open` launches `dashboard.html` when done. Drop it to just regenerate the file.

| Command | What it does |
|---|---|
| `python monitor.py --demo` | Uses bundled sample announcements. No network, no `requests` needed. |
| `python monitor.py` | Live: fetches BSE announcements for every watchlist name, caches them, rebuilds the dashboard. |
| `python monitor.py --no-fetch` | Rebuilds the dashboard from the **last** cached fetch (fast, offline). |
| `python monitor.py --days 30` | Override the "new" lookback window (default 21 days). |

---

## The watchlist (`watchlist.csv`)

This is the one file you edit regularly. Open it in Excel or any editor.

| Column | Meaning |
|---|---|
| `bse_code` | **BSE scrip code** (numeric) — this is what the fetcher uses. See below. |
| `nse_symbol` | NSE ticker, display only. |
| `company` | Display name. |
| `sector` | Your bucket (used for search/filter). |
| `thesis_status` | One of: `Strengthening`, `Stable`, `Watch`, `Weakening`, `Broken`. Drives the colour. |
| `conviction` | `High` / `Medium` / `Low` — your call. |
| `original_thesis` | Why you bought / are watching it. The anchor you re-test against. |
| `watch_items` | What would break the thesis. |
| `position` | Your sizing (e.g. `2.5%`) or blank for a watch-only name. |
| `last_reviewed` | `YYYY-MM-DD` of your last real review. |

`thesis_status` is a **judgement you set by hand** — not computed. That is intentional: a
summed score would manufacture precision the evidence doesn't support. The dashboard gives
you the events; you move the status.

---

## ⚠️ Verify the BSE scrip codes

The codes shipped in `watchlist.csv` are **best-guess seeds** for six thematically relevant
names. Verify each before trusting live data — it takes ~10 seconds:

1. Go to **bseindia.com** and search the company.
2. Open its quote page. The 6-digit number in the URL / "Security Code" is the `bse_code`.
   (e.g. Reliance is `500325`.)
3. Paste it into `watchlist.csv`.

If a code is wrong or missing, that name shows **"No data / check code"** on the dashboard
rather than failing silently — so you'll notice.

> Only BSE is wired up. Most SME and small-cap names trade on BSE and its announcement feed
> is the most scrape-friendly. NSE can be added later (see Roadmap).

---

## How the flags work

Each announcement's subject + category is matched against keyword lists in `monitor.py`
(look for the `EDIT HERE` block). Priority is **RED > AMBER > GREEN > NEUTRAL**, first match
wins, and the matched phrase is shown on the dashboard so you can audit and tune it.

- 🔴 **RED** — potential permanent impairment: auditor / CFO / KMP resignation, pledging,
  rating downgrade, default, insolvency, fraud, SEBI action, qualified opinion, results delay.
- 🟠 **AMBER** — governance / dilution / watch: rating changes, related-party, preferential
  allotment, QIP, promoter selling, schemes of arrangement, tax demands.
- 🟢 **GREEN** — positive momentum: order wins, capacity expansion, capex, JV, buyback.
- ⚪ **NEUTRAL** — everything else (board-meeting intimations, results, investor calls).

These are **triage heuristics, not truth.** "Resignation" might be a junior clerk; an "order"
might be tiny. Always open the actual filing — the dashboard links straight to the BSE PDF.

To tune: edit the `FLAG_RULES` lists in `monitor.py` and re-run with `--no-fetch`.

---

## Run it weekly (Windows Task Scheduler)

1. Task Scheduler → **Create Basic Task** → trigger **Weekly** (e.g. Saturday 8am).
2. Action → **Start a program**:
   - Program: `python`
   - Arguments: `monitor.py`
   - Start in: `C:\aaraminds-projects\emerging-companies-monitor`
3. It refreshes `dashboard.html` in place; open the file whenever you want to check.

---

## What this is / isn't

This is a **personal research tool**. It is **not investment advice**, gives no buy/sell
calls and no price targets. It surfaces public exchange disclosures and flags them for your
own analysis. Confirm everything against primary filings before you act.

---

## Roadmap (only if it earns its place)

In rough priority — add one, prove it's useful, then add the next:

1. **NSE announcements** feed (needs cookie-primed session; more fragile than BSE).
2. **Shareholding-pattern deltas** quarter-on-quarter (promoter %, pledge %, institutional %).
3. **Concall / results PDF** ingestion → LLM summary into the thesis note.
4. **Rating-agency** pages (CRISIL / ICRA / CARE) scraped directly, not just via announcements.

Resist building all of it. The monitor above is the part with real ROI: it watches the
boring bad-news feed you'd otherwise have to check by hand across every name.

---

## Live price refresh (refresh_prices.py)

Updates the **CMP** column across all sheets with the latest market price; the
**Live P/L %** columns then recompute when you open the file in Excel.

```bash
pip install requests          # one-time
python refresh_prices.py      # CLOSE the workbook in Excel first (Excel locks it)
python refresh_prices.py --open   # refresh, then open the file
```

Ticker resolution order: NSE Symbol -> `SYMBOL.NS`, else BSE Code -> `CODE.BO`,
else a Yahoo Finance name search. Resolved tickers are written back into the
*Watchlist (Combined)* sheet so you can verify/correct them — for any name
reported "not found," fill its NSE Symbol or BSE Code and re-run. Prices are from
Yahoo Finance (personal use; may lag ~15 min; the thinnest SME names may be
unavailable). Not investment advice.

Schedule it like the monitor via Task Scheduler (Program `python`, Arguments
`refresh_prices.py`, Start in the project folder).
