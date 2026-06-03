#!/usr/bin/env python3
"""
AaraMinds Emerging Companies Monitor
====================================
A practical, single-user watchlist monitor for Indian SME / small-cap / emerging
mid-cap businesses.

What it does
------------
1. Reads your editable watchlist (watchlist.csv).
2. Pulls BSE corporate announcements for each name.
3. Keyword-flags the early-warning events that cause PERMANENT capital loss
   (auditor exits, promoter pledging, rating downgrades) and the positive
   momentum signals (big orders, capacity expansion).
4. Writes a self-contained HTML dashboard (dashboard.html) you open in a browser.

What it deliberately does NOT do
--------------------------------
No scores out of 100, no buy/sell calls, no price prediction. Flags are a
TRIAGE aid that points you at filings to read — your judgment sets the thesis.

Usage
-----
    python monitor.py --demo        # see it work instantly, no network
    python monitor.py               # live: fetch BSE announcements, rebuild dashboard
    python monitor.py --no-fetch    # rebuild dashboard from last cached fetch
    python monitor.py --days 30     # override lookback window
    python monitor.py --open        # open dashboard.html when done

This is a personal research tool. It is not investment advice.
"""

import argparse
import csv
import datetime as dt
import html
import json
import sys
from pathlib import Path

# ======================================================================
# ============================  EDIT HERE  =============================
# ======================================================================

LOOKBACK_DAYS = 21          # how many days of announcements count as "new"
BASE_DIR = Path(__file__).resolve().parent
WATCHLIST_FILE = BASE_DIR / "watchlist.csv"
CACHE_FILE = BASE_DIR / "data" / "announcements_cache.json"
OUTPUT_FILE = BASE_DIR / "dashboard.html"

# BSE corporate-announcements endpoint (public). Headers matter — BSE blocks
# requests without a browser-like User-Agent and a bseindia.com Referer.
BSE_API = "https://api.bseindia.com/BseIndiaAPI/api/AnnSubCategoryGetData/w"
BSE_ATTACH = "https://www.bseindia.com/xml-data/corpfiling/AttachLive/"
BSE_HEADERS = {
    "User-Agent": ("Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
                   "AppleWebKit/537.36 (KHTML, like Gecko) "
                   "Chrome/124.0 Safari/537.36"),
    "Referer": "https://www.bseindia.com/",
    "Accept": "application/json, text/plain, */*",
}

# Keyword flag rules. Lower-case substrings matched against the announcement
# subject + category. Priority: RED > AMBER > GREEN > NEUTRAL. First match wins,
# and the matched phrase is shown on the dashboard so you can tune these lists.
FLAG_RULES = {
    "RED": [  # potential permanent impairment — read the filing today
        "resignation of statutory auditor", "resignation of the statutory auditor",
        "resignation of auditor", "auditor has resigned", "resignation of joint auditor",
        "resignation of chief financial officer", "resignation of cfo", "cfo resign",
        "resignation of company secretary", "resignation of managing director",
        "resignation of whole-time director", "resignation of independent director",
        "pledge", "invocation of pledge", "creation of encumbrance", "encumbrance",
        "rating downgrade", "downgraded", "revised downward", "default",
        "delay in payment of interest", "nclt", "insolvency", "ibc", "cirp",
        "liquidation", "winding up", "fraud", "siphon", "diversion of funds",
        "misappropriation", "sebi order", "show cause notice", "search and seizure",
        "income tax search", "qualified opinion", "adverse opinion",
        "disclaimer of opinion", "going concern", "delay in financial results",
        "suspension of trading", "forensic audit",
    ],
    "AMBER": [  # governance / dilution / watch
        "credit rating", "rating revision", "outlook revised", "rating action",
        "under credit watch", "rating reaffirmed", "related party",
        "preferential allotment", "preferential issue", "warrant", "qip",
        "qualified institutional", "offer for sale", "promoter stake",
        "sale of shares by promoter", "increase in authorised capital",
        "scheme of arrangement", "demerger", "amalgamation", "one time settlement",
        "one-time settlement", "tax demand", "gst", "penalty", "change in auditor",
        "appointment of auditor", "resignation",
    ],
    "GREEN": [  # positive momentum — confirm size before getting excited
        "receipt of order", "received order", "bagging of order", "bagged",
        "award of contract", "awarded", "work order", "letter of award", "loa",
        "purchase order", "order win", "new order", "contract win",
        "capacity expansion", "expansion of capacity", "commissioning",
        "commenced commercial production", "new plant", "greenfield", "brownfield",
        "capital expenditure", "capex", "joint venture", "memorandum of understanding",
        "acquisition", "buyback", "bonus", "record date", "interim dividend",
    ],
}

# ======================================================================
# =========================  END OF EDIT ZONE  =========================
# ======================================================================

FLAG_ORDER = {"RED": 3, "AMBER": 2, "GREEN": 1, "NEUTRAL": 0}


# ---------- Demo data (used by --demo; also what the logic is tested on) ----------
DEMO_ANNOUNCEMENTS = {
    "543428": [
        {"NEWSSUB": "Awarding of Order/Receipt of Order - Receipt of order worth Rs. 120.4 Cr from Ministry of Defence",
         "CATEGORYNAME": "Award of Order / Receipt of Order", "NEWS_DT": "2026-05-30T17:42:00",
         "ATTACHMENTNAME": "demo_dp_order.pdf"},
        {"NEWSSUB": "Board Meeting Outcome - Audited Standalone & Consolidated Financial Results for Q4FY26",
         "CATEGORYNAME": "Result", "NEWS_DT": "2026-05-26T18:05:00", "ATTACHMENTNAME": "demo_dp_results.pdf"},
    ],
    "543573": [
        {"NEWSSUB": "Resignation of Chief Financial Officer (CFO) with effect from 31 May 2026",
         "CATEGORYNAME": "Change in Directors/KMP", "NEWS_DT": "2026-05-31T20:11:00",
         "ATTACHMENTNAME": "demo_syrma_cfo.pdf"},
        {"NEWSSUB": "Disclosure under Reg. 30 - Credit Rating outlook revised to Negative by CRISIL",
         "CATEGORYNAME": "Credit Rating", "NEWS_DT": "2026-05-28T13:20:00", "ATTACHMENTNAME": "demo_syrma_rating.pdf"},
    ],
    "543664": [
        {"NEWSSUB": "Intimation of Capacity Expansion - Commissioning of new SMT lines at Mysuru facility",
         "CATEGORYNAME": "Company Update", "NEWS_DT": "2026-05-29T11:00:00", "ATTACHMENTNAME": "demo_kaynes_capex.pdf"},
        {"NEWSSUB": "Announcement under Regulation 30 - Acquisition of majority stake in OSAT venture",
         "CATEGORYNAME": "Acquisition", "NEWS_DT": "2026-05-27T09:35:00", "ATTACHMENTNAME": "demo_kaynes_acq.pdf"},
    ],
    "544090": [
        {"NEWSSUB": "Creation of Encumbrance - Pledge of shares by promoter group",
         "CATEGORYNAME": "Encumbrance", "NEWS_DT": "2026-05-30T15:48:00", "ATTACHMENTNAME": "demo_azad_pledge.pdf"},
    ],
    "532259": [
        {"NEWSSUB": "Board Meeting Intimation for considering Q4 results and dividend",
         "CATEGORYNAME": "Board Meeting", "NEWS_DT": "2026-05-22T10:15:00", "ATTACHMENTNAME": "demo_apar_bm.pdf"},
        {"NEWSSUB": "Credit Rating reaffirmed at AA-/Stable by ICRA",
         "CATEGORYNAME": "Credit Rating", "NEWS_DT": "2026-05-21T14:02:00", "ATTACHMENTNAME": "demo_apar_rating.pdf"},
    ],
    "540900": [
        {"NEWSSUB": "Press Release - Bagged a large order from a leading public sector bank for low-code platform",
         "CATEGORYNAME": "Press Release", "NEWS_DT": "2026-05-25T16:30:00", "ATTACHMENTNAME": "demo_newgen_order.pdf"},
    ],
}


def load_watchlist(path):
    if not path.exists():
        sys.exit(f"ERROR: watchlist not found at {path}")
    rows = []
    with open(path, newline="", encoding="utf-8-sig") as f:
        for r in csv.DictReader(f):
            code = (r.get("bse_code") or "").strip()
            if not code:
                continue
            rows.append({
                "bse_code": code,
                "nse_symbol": (r.get("nse_symbol") or "").strip(),
                "company": (r.get("company") or "").strip(),
                "sector": (r.get("sector") or "").strip(),
                "thesis_status": (r.get("thesis_status") or "Stable").strip(),
                "conviction": (r.get("conviction") or "").strip(),
                "original_thesis": (r.get("original_thesis") or "").strip(),
                "watch_items": (r.get("watch_items") or "").strip(),
                "position": (r.get("position") or "").strip(),
                "last_reviewed": (r.get("last_reviewed") or "").strip(),
            })
    return rows


def fetch_bse_announcements(scrip_code, from_date, to_date):
    """Live fetch. Runs on YOUR machine; requests is imported lazily so --demo
    works without it installed. Returns [] on any failure (degrade gracefully)."""
    try:
        import requests
    except ImportError:
        print("  ! 'requests' not installed. Run: pip install requests", file=sys.stderr)
        return []
    params = {
        "pageno": 1, "strCat": "-1", "strPrevDate": from_date, "strToDate": to_date,
        "strScrip": scrip_code, "strSearch": "P", "strType": "C",
    }
    try:
        resp = requests.get(BSE_API, params=params, headers=BSE_HEADERS, timeout=20)
        resp.raise_for_status()
        return resp.json().get("Table", []) or []
    except Exception as e:  # noqa: BLE001 - we want to swallow and report any failure
        print(f"  ! fetch failed for {scrip_code}: {e}", file=sys.stderr)
        return []


def classify(text):
    """Return (flag, matched_phrase). RED > AMBER > GREEN > NEUTRAL."""
    t = (text or "").lower()
    for flag in ("RED", "AMBER", "GREEN"):
        for kw in FLAG_RULES[flag]:
            if kw in t:
                return flag, kw
    return "NEUTRAL", None


def parse_date(raw):
    """Best-effort parse of BSE date strings -> (sort_key, display)."""
    raw = (raw or "").strip()
    for fmt in ("%Y-%m-%dT%H:%M:%S", "%Y-%m-%d %H:%M:%S", "%d %b %Y %H:%M:%S",
                "%d %b %Y", "%Y-%m-%d"):
        try:
            d = dt.datetime.strptime(raw, fmt)
            return d.strftime("%Y-%m-%dT%H:%M:%S"), d.strftime("%d %b %Y, %H:%M")
        except ValueError:
            continue
    # fallback: try just the date part
    try:
        d = dt.datetime.fromisoformat(raw)
        return d.strftime("%Y-%m-%dT%H:%M:%S"), d.strftime("%d %b %Y, %H:%M")
    except Exception:  # noqa: BLE001
        return raw, raw or "—"


def normalise(row):
    subject = (row.get("NEWSSUB") or row.get("HEADLINE") or "").strip()
    category = (row.get("CATEGORYNAME") or "").strip()
    sort_key, display = parse_date(row.get("NEWS_DT"))
    attach = (row.get("ATTACHMENTNAME") or "").strip()
    flag, kw = classify(subject + " " + category)
    return {
        "subject": subject, "category": category,
        "date_sort": sort_key, "date": display,
        "attachment": (BSE_ATTACH + attach) if attach else "",
        "flag": flag, "keyword": kw or "",
    }


def within_lookback(date_sort, cutoff):
    try:
        return dt.datetime.fromisoformat(date_sort) >= cutoff
    except Exception:  # noqa: BLE001
        return True  # if we can't parse, surface it rather than hide it


def build_data(watchlist, raw_by_code, lookback_days):
    cutoff = dt.datetime.now() - dt.timedelta(days=lookback_days)
    companies, alerts = [], []
    summary = {"names": len(watchlist), "red": 0, "amber": 0, "green": 0,
               "neutral": 0, "new_total": 0, "no_data": 0}

    for w in watchlist:
        raw = raw_by_code.get(w["bse_code"], [])
        anns = sorted((normalise(r) for r in raw),
                      key=lambda a: a["date_sort"], reverse=True)
        recent = [a for a in anns if within_lookback(a["date_sort"], cutoff)]
        counts = {"RED": 0, "AMBER": 0, "GREEN": 0, "NEUTRAL": 0}
        for a in recent:
            counts[a["flag"]] += 1
        worst = "NEUTRAL"
        for a in recent:
            if FLAG_ORDER[a["flag"]] > FLAG_ORDER[worst]:
                worst = a["flag"]
        if not raw:
            summary["no_data"] += 1
        summary["new_total"] += len(recent)
        if worst == "RED":
            summary["red"] += 1
        elif worst == "AMBER":
            summary["amber"] += 1
        elif worst == "GREEN":
            summary["green"] += 1

        for a in recent:
            if a["flag"] in ("RED", "AMBER"):
                alerts.append({**a, "company": w["company"],
                               "symbol": w["nse_symbol"], "code": w["bse_code"]})

        companies.append({**w, "worst_flag": worst, "counts": counts,
                          "new_count": len(recent), "has_data": bool(raw),
                          "announcements": anns[:25]})

    alerts.sort(key=lambda a: (FLAG_ORDER[a["flag"]], a["date_sort"]), reverse=True)
    return {
        "generated_at": dt.datetime.now().strftime("%d %b %Y, %H:%M"),
        "lookback_days": lookback_days,
        "source": "BSE corporate announcements",
        "summary": summary, "alerts": alerts, "companies": companies,
    }


def render_html(data, out_path):
    payload = json.dumps(data, ensure_ascii=False)
    html_doc = HTML_TEMPLATE.replace("/*__DATA__*/null", payload)
    out_path.write_text(html_doc, encoding="utf-8")


# ----------------------------------------------------------------------
HTML_TEMPLATE = r"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Emerging Companies Monitor</title>
<style>
  :root{
    --bg:#0f1115; --panel:#171a21; --panel2:#1d2129; --line:#2a2f3a;
    --txt:#e6e9ef; --muted:#9aa3b2; --red:#ff5d5d; --amber:#ffb02e;
    --green:#3ecf8e; --neutral:#5b6472; --accent:#6ea8fe;
  }
  *{box-sizing:border-box}
  body{margin:0;background:var(--bg);color:var(--txt);
    font:14px/1.5 system-ui,-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif}
  .wrap{max-width:1180px;margin:0 auto;padding:24px 20px 60px}
  h1{font-size:20px;margin:0 0 2px} .sub{color:var(--muted);font-size:13px}
  .cards{display:flex;gap:12px;flex-wrap:wrap;margin:20px 0}
  .card{background:var(--panel);border:1px solid var(--line);border-radius:10px;
    padding:14px 16px;min-width:120px;flex:1}
  .card .n{font-size:24px;font-weight:700} .card .l{color:var(--muted);font-size:12px}
  .card.red .n{color:var(--red)} .card.amber .n{color:var(--amber)}
  .card.green .n{color:var(--green)}
  h2{font-size:15px;margin:26px 0 10px;color:var(--muted);text-transform:uppercase;
    letter-spacing:.06em}
  .alerts{display:flex;flex-direction:column;gap:8px}
  .alert{display:flex;gap:12px;align-items:flex-start;background:var(--panel);
    border:1px solid var(--line);border-left:4px solid var(--neutral);
    border-radius:8px;padding:10px 14px}
  .alert.RED{border-left-color:var(--red)} .alert.AMBER{border-left-color:var(--amber)}
  .alert .co{font-weight:600;min-width:150px} .alert .meta{color:var(--muted);font-size:12px}
  .pill{display:inline-block;font-size:11px;font-weight:700;padding:2px 8px;border-radius:20px;
    text-transform:uppercase;letter-spacing:.04em}
  .pill.RED{background:rgba(255,93,93,.15);color:var(--red)}
  .pill.AMBER{background:rgba(255,176,46,.15);color:var(--amber)}
  .pill.GREEN{background:rgba(62,207,142,.15);color:var(--green)}
  .pill.NEUTRAL{background:rgba(91,100,114,.18);color:var(--muted)}
  .toolbar{display:flex;gap:10px;flex-wrap:wrap;align-items:center;margin:8px 0 12px}
  .toolbar input{background:var(--panel2);border:1px solid var(--line);color:var(--txt);
    border-radius:8px;padding:8px 12px;min-width:240px;outline:none}
  .fbtn{background:var(--panel2);border:1px solid var(--line);color:var(--muted);
    border-radius:20px;padding:6px 14px;cursor:pointer;font-size:12px}
  .fbtn.active{color:var(--txt);border-color:var(--accent)}
  table{width:100%;border-collapse:collapse;background:var(--panel);
    border:1px solid var(--line);border-radius:10px;overflow:hidden}
  th,td{text-align:left;padding:11px 14px;border-bottom:1px solid var(--line)}
  th{color:var(--muted);font-size:12px;cursor:pointer;user-select:none;white-space:nowrap}
  th:hover{color:var(--txt)}
  tr.row{cursor:pointer} tr.row:hover{background:var(--panel2)}
  .dot{display:inline-block;width:9px;height:9px;border-radius:50%;margin-right:7px;vertical-align:middle}
  .dot.RED{background:var(--red)} .dot.AMBER{background:var(--amber)}
  .dot.GREEN{background:var(--green)} .dot.NEUTRAL{background:var(--neutral)}
  .status{font-size:12px;font-weight:600}
  .status.Strengthening{color:var(--green)} .status.Stable{color:var(--accent)}
  .status.Watch{color:var(--amber)} .status.Weakening{color:var(--amber)}
  .status.Broken{color:var(--red)}
  .detail{background:var(--panel2);padding:0 16px}
  .detail .inner{padding:14px 0;display:grid;grid-template-columns:1fr 1fr;gap:18px}
  .detail h4{margin:0 0 6px;font-size:12px;color:var(--muted);text-transform:uppercase}
  .thesis p{margin:0 0 8px} .thesis .k{color:var(--muted)}
  .annlist{display:flex;flex-direction:column;gap:6px}
  .ann{font-size:13px;padding:7px 10px;background:var(--panel);border:1px solid var(--line);
    border-radius:7px;border-left:3px solid var(--neutral)}
  .ann.RED{border-left-color:var(--red)} .ann.AMBER{border-left-color:var(--amber)}
  .ann.GREEN{border-left-color:var(--green)}
  .ann .s{display:block} .ann .m{color:var(--muted);font-size:11px;margin-top:2px}
  .ann a{color:var(--accent);text-decoration:none} .ann a:hover{text-decoration:underline}
  .nodata{color:var(--amber);font-size:12px}
  .foot{margin-top:34px;padding-top:16px;border-top:1px solid var(--line);
    color:var(--muted);font-size:12px}
  .hide{display:none}
</style>
</head>
<body>
<div class="wrap">
  <h1>Emerging Companies Monitor</h1>
  <div class="sub" id="subline"></div>

  <div class="cards" id="cards"></div>

  <h2>Early-Warning Alerts <span id="alertwin" class="sub"></span></h2>
  <div class="alerts" id="alerts"></div>

  <h2>Watchlist</h2>
  <div class="toolbar">
    <input id="search" placeholder="Search company / symbol / sector…">
    <button class="fbtn active" data-f="ALL">All</button>
    <button class="fbtn" data-f="RED">Red</button>
    <button class="fbtn" data-f="AMBER">Amber</button>
    <button class="fbtn" data-f="GREEN">Green</button>
  </div>
  <table>
    <thead><tr>
      <th data-k="company">Company</th>
      <th data-k="sector">Sector</th>
      <th data-k="thesis_status">Thesis</th>
      <th data-k="conviction">Conviction</th>
      <th data-k="new_count">New</th>
      <th data-k="worst_flag">Flag</th>
      <th data-k="last_reviewed">Reviewed</th>
    </tr></thead>
    <tbody id="tbody"></tbody>
  </table>

  <div class="foot" id="foot"></div>
</div>

<script>
const DATA = /*__DATA__*/null;
const el = (t, c, txt) => { const e = document.createElement(t); if(c) e.className=c;
  if(txt!=null) e.textContent=txt; return e; };
let filter = "ALL", sortKey = "worst_flag", sortDir = -1, query = "";
const FORDER = {RED:3, AMBER:2, GREEN:1, NEUTRAL:0};

function cards(){
  const s = DATA.summary, box = document.getElementById("cards");
  const defs = [["names","Names tracked",""],["red","Red flags","red"],
    ["amber","Amber flags","amber"],["green","Positive signals","green"],
    ["new_total","New filings",""],["no_data","No data / check code",""]];
  defs.forEach(([k,l,c])=>{ const card=el("div","card "+c);
    card.append(el("div","n",s[k]), el("div","l",l)); box.append(card); });
}

function alerts(){
  const box = document.getElementById("alerts");
  document.getElementById("alertwin").textContent =
    "(RED & AMBER in last "+DATA.lookback_days+" days)";
  if(!DATA.alerts.length){ box.append(el("div","sub","No red or amber events in the window. Quiet is good.")); return; }
  DATA.alerts.forEach(a=>{
    const row = el("div","alert "+a.flag);
    const left = el("div"); left.append(el("div","co",a.company || a.code));
    left.append(el("span","pill "+a.flag, a.flag));
    const body = el("div"); body.style.flex="1";
    body.append(el("div","", a.subject));
    const meta = el("div","meta");
    meta.textContent = a.date + (a.category? "  ·  "+a.category : "") +
      (a.keyword? "  ·  matched: \""+a.keyword+"\"" : "");
    body.append(meta);
    if(a.attachment){ const lk=el("a",null,"open filing ↗"); lk.href=a.attachment;
      lk.target="_blank"; lk.style.cssText="color:var(--accent);font-size:12px"; body.append(lk); }
    row.append(left, body); box.append(row);
  });
}

function sortVal(c){
  if(sortKey==="worst_flag") return FORDER[c.worst_flag];
  if(sortKey==="new_count") return c.new_count;
  return (c[sortKey]||"").toString().toLowerCase();
}

function table(){
  const tb = document.getElementById("tbody"); tb.innerHTML="";
  let rows = DATA.companies.filter(c=>{
    if(filter!=="ALL" && c.worst_flag!==filter) return false;
    if(query){ const h=(c.company+" "+c.nse_symbol+" "+c.sector).toLowerCase();
      if(!h.includes(query)) return false; }
    return true;
  });
  rows.sort((a,b)=>{ const x=sortVal(a),y=sortVal(b);
    return (x>y?1:x<y?-1:0)*sortDir; });

  rows.forEach(c=>{
    const tr = el("tr","row");
    const c1 = el("td"); c1.append(el("span","dot "+c.worst_flag));
    c1.append(document.createTextNode(c.company + (c.nse_symbol? "  ("+c.nse_symbol+")":"")));
    tr.append(c1);
    tr.append(el("td",null,c.sector));
    const st=el("td"); st.append(el("span","status "+c.thesis_status, c.thesis_status)); tr.append(st);
    tr.append(el("td",null,c.conviction||"—"));
    tr.append(el("td",null,String(c.new_count)));
    const fl=el("td"); fl.append(el("span","pill "+c.worst_flag, c.worst_flag)); tr.append(fl);
    tr.append(el("td",null,c.last_reviewed||"—"));

    const drow = el("tr","detail hide"); const dcell=el("td"); dcell.colSpan=7;
    dcell.append(detail(c)); drow.append(dcell);
    tr.onclick=()=>drow.classList.toggle("hide");
    tb.append(tr, drow);
  });
  if(!rows.length) tb.append(el("tr")).append(el("td","sub","No names match."));
}

function detail(c){
  const wrap=el("div","inner");
  const th=el("div","thesis");
  th.append(el("h4",null,"Living thesis"));
  const add=(k,v)=>{ const p=el("p"); p.append(el("span","k",k+": "));
    p.append(document.createTextNode(v||"—")); th.append(p); };
  add("Original", c.original_thesis); add("Watch items", c.watch_items);
  add("Position", c.position); add("Conviction", c.conviction);
  add("Last reviewed", c.last_reviewed);
  wrap.append(th);

  const ann=el("div");
  ann.append(el("h4",null,"Recent announcements ("+(c.has_data?c.announcements.length:"no data")+")"));
  if(!c.has_data){ ann.append(el("div","nodata",
    "No announcements returned — verify the BSE scrip code for this name.")); }
  const list=el("div","annlist");
  c.announcements.forEach(a=>{ const d=el("div","ann "+a.flag);
    d.append(el("span","s", a.subject));
    const m=el("span","m"); m.textContent=a.date+(a.category?"  ·  "+a.category:"")+
      (a.flag!=="NEUTRAL"?"  ·  "+a.flag:"");
    d.append(m);
    if(a.attachment){ d.append(document.createElement("br"));
      const lk=el("a",null,"open filing ↗"); lk.href=a.attachment; lk.target="_blank"; d.append(lk); }
    list.append(d); });
  ann.append(list); wrap.append(ann);
  return wrap;
}

function wire(){
  document.getElementById("subline").textContent =
    "Generated "+DATA.generated_at+"  ·  source: "+DATA.source+
    "  ·  lookback "+DATA.lookback_days+" days";
  document.querySelectorAll(".fbtn").forEach(b=> b.onclick=()=>{
    document.querySelectorAll(".fbtn").forEach(x=>x.classList.remove("active"));
    b.classList.add("active"); filter=b.dataset.f; table(); });
  document.getElementById("search").oninput=e=>{ query=e.target.value.toLowerCase().trim(); table(); };
  document.querySelectorAll("th[data-k]").forEach(h=> h.onclick=()=>{
    const k=h.dataset.k; if(sortKey===k) sortDir*=-1; else {sortKey=k; sortDir=1;}
    table(); });
  document.getElementById("foot").innerHTML =
    "Personal research tool — <b>not investment advice</b>, no buy/sell calls, no price targets. "+
    "Flags are keyword heuristics to triage which filings to read; always confirm against the "+
    "primary BSE/NSE filing and the company's disclosures before acting. Keyword rules are "+
    "editable in monitor.py.";
}

cards(); alerts(); wire(); table();
</script>
</body>
</html>"""


def main():
    ap = argparse.ArgumentParser(description="AaraMinds Emerging Companies Monitor")
    ap.add_argument("--demo", action="store_true",
                    help="use bundled sample announcements (no network)")
    ap.add_argument("--no-fetch", action="store_true",
                    help="rebuild dashboard from last cached fetch")
    ap.add_argument("--days", type=int, default=LOOKBACK_DAYS,
                    help=f"lookback window in days (default {LOOKBACK_DAYS})")
    ap.add_argument("--open", action="store_true",
                    help="open the dashboard in your browser when done")
    args = ap.parse_args()

    watchlist = load_watchlist(WATCHLIST_FILE)
    print(f"Loaded {len(watchlist)} names from {WATCHLIST_FILE.name}")

    if args.demo:
        raw_by_code = DEMO_ANNOUNCEMENTS
        print("DEMO mode — using bundled sample announcements (no network).")
    elif args.no_fetch:
        if not CACHE_FILE.exists():
            sys.exit("No cache found. Run once without --no-fetch first.")
        raw_by_code = json.loads(CACHE_FILE.read_text(encoding="utf-8"))["by_code"]
        print(f"Loaded cached announcements from {CACHE_FILE}")
    else:
        today = dt.date.today()
        frm = (today - dt.timedelta(days=max(args.days, 30))).strftime("%Y%m%d")
        to = today.strftime("%Y%m%d")
        raw_by_code = {}
        for w in watchlist:
            print(f"  fetching {w['company']} ({w['bse_code']}) …")
            raw_by_code[w["bse_code"]] = fetch_bse_announcements(w["bse_code"], frm, to)
        CACHE_FILE.parent.mkdir(parents=True, exist_ok=True)
        CACHE_FILE.write_text(json.dumps(
            {"fetched_at": dt.datetime.now().isoformat(), "by_code": raw_by_code},
            ensure_ascii=False), encoding="utf-8")
        print(f"Cached raw announcements to {CACHE_FILE}")

    data = build_data(watchlist, raw_by_code, args.days)
    render_html(data, OUTPUT_FILE)
    s = data["summary"]
    print(f"\nDashboard written: {OUTPUT_FILE}")
    print(f"  {s['red']} red · {s['amber']} amber · {s['green']} positive · "
          f"{s['new_total']} new filings · {s['no_data']} no-data")

    if args.open:
        import webbrowser
        webbrowser.open(OUTPUT_FILE.as_uri())


if __name__ == "__main__":
    main()
