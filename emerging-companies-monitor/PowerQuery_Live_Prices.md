# Live prices in Excel via Power Query

A self-contained query that pulls the latest price for every name (ValueEducator +
SOIC + your Raja holdings) from Yahoo Finance into a **live, refreshable table**.
Refresh with **Data -> Refresh All** (or on a timer) - no script, no closing the file.

> **Requires Microsoft Excel (Windows / Microsoft 365).** Quotes lag ~15 min. NSE-Emerge
> SME names may come back blank (Yahoo doesn't carry them); those keep their fallback.

82 of 83 names carry a ticker. A wrong/uncovered ticker returns blank -
edit its `Ticker` in the M table and refresh. Use with **...Watchlist_v6.xlsx**.

---

## Setup (one time, ~3 min)

1. Open the v6 workbook in **Excel**.
2. **Data -> Get Data -> From Other Sources -> Blank Query**.
3. **Home -> Advanced Editor**, select-all, delete, paste the M below, **Done**.
4. Rename the query (left panel) to **LivePrices**.
5. If asked how to connect to `query1.finance.yahoo.com`, choose **Anonymous**.
6. If you see *"Information is required about data privacy"*: **File -> Options and
   settings -> Query Options -> Privacy -> Ignore the Privacy Levels**, then Refresh Preview.
7. **Home -> Close & Load To... -> Table -> New worksheet**, named exactly **LivePrices**.
8. Format the **P/L %** column as Percentage.

## Refresh

- Manual: **Data -> Refresh All**.
- Automatic: **Data -> Queries & Connections** -> right-click **LivePrices** ->
  **Properties** -> tick **Refresh every 15 minutes** and **Refresh when opening the file**.

## CMP is already wired

Every **CMP** cell across Combined, Emerging, Tiny, SOIC, Portfolio and Raja sheets pulls
from this query via `=IFERROR(XLOOKUP(<name>, LivePrices!$A:$A, LivePrices!$D:$D), <fallback>)`.
Until the query loads, CMP shows the fallback (entry / snapshot / your broker LTP); once
**LivePrices** exists it goes live and every P/L recomputes on refresh.

## Notes
- A blank batch = Yahoo briefly throttled; just refresh again.
- Personal-use data from Yahoo Finance. Not investment advice.

---

## The M script - paste into Advanced Editor

```m
let
    // ===== Watchlist (ValueEducator + SOIC + Raja): Name, Yahoo ticker, Entry price =====
    Watchlist = #table(
        type table [Name = text, Ticker = text, Entry = number],
        {
            {"Techno Electric & Engineering Ltd","TECHNOE.NS",1125},
            {"Oswal Pumps Ltd","OSWALPUMPS.NS",362},
            {"PNGS Reva Diamond Jewellery Limited","PNGSREVA.NS",379},
            {"HBL Engineering","HBLENGINE.NS",666},
            {"Centum Electronics Ltd","CENTUM.NS",2685},
            {"SJS","SJS.NS",1635},
            {"Narayana Hrudayalaya Ltd","NH.NS",1862},
            {"Force Motors","FORCEMOT.NS",19000},
            {"Samhi Hotels","SAMHI.NS",189.5},
            {"Gravita India Limited","GRAVITA.NS",1736},
            {"Timex Group","TIMEX.NS",335},
            {"Federal bank financial services","FEDFINA.NS",159},
            {"Praveg Ltd","PRAVEG.NS",315},
            {"Navkar Corporation Ltd","NAVKARCORP.NS",96.4},
            {"Indo Tech Transformers Ltd","INDOTECH.NS",1759},
            {"Arkade Developers Ltd","ARKADE.NS",179},
            {"Sambhv Steel Tubes Ltd","SAMBHV.NS",114.5},
            {"India Shelter Finance","INDIASHLTR.NS",889},
            {"Kilburn Engineering","522101.BO",454},
            {"PSP Projects","PSPPROJECT.NS",637},
            {"Frontier Spring","FRONTSP.NS",761},
            {"CCL Products (India) Ltd","CCL.NS",554},
            {"Tara Chand Infra Solution","TARACHAND.NS",58.3},
            {"Mac Power CNC Machines Ltd","MACPOWER.NS",689},
            {"Paushak Ltd","532742.BO",521},
            {"P N Gadgil Jewellers Ltd","PNGJL.NS",543.6},
            {"Shivalik Bi Metals Ltd","SBCL.NS",474},
            {"S G Finserve Ltd","SGFIN.NS",343},
            {"Maiden Forging","543874.BO",74.5},
            {"PNGS Gargi Fashion Jewellery Ltd","543709.BO",968},
            {"Patil Automation Ltd","",162},
            {"Airfloa Rail Technology Ltd","AIRFLOA.NS",295},
            {"Sat Kartar Shopping Ltd","SATKARTAR.NS",160},
            {"Z-Tech","ZTECH.NS",515},
            {"Sealmatic India Ltd","543782.BO",395.8},
            {"CFF Fluid Control Ltd","543920.BO",617},
            {"Sathlokhar Synergys E&C","SSEGL.NS",465},
            {"Aluwind Infra-Tech Ltd","ALUWIND.NS",87},
            {"Danish Power Ltd","DANISH.NS",897},
            {"Wise Travel India Ltd","WTICAB.NS",170},
            {"Infollion Research Services Ltd","INFOLLION.NS",440},
            {"Narayana Hrudayalaya","NH.NS",1021},
            {"Goodluck India","GOODLUCK.NS",680},
            {"GPIL","GPIL.NS",122},
            {"Nuvama Wealth","NUVAMA.NS",676},
            {"Arman Finance","ARMANFIN.NS",2487},
            {"Interarch Building Products","INTERARCH.NS",1250},
            {"Time Technoplast","TIMETECHNO.NS",206},
            {"Aarti Pharma","AARTIPHARM.NS",731},
            {"Pondy Oxides","PONDYOXIDE.NS",809},
            {"Fedbank Financial Services","FEDFINA.NS",131},
            {"Deepak Fertilizers","DEEPAKFERT.NS",1592},
            {"Vishnu Chemicals","VISHNU.NS",550},
            {"Adani Port and SEZ","ADANIPORTS.NS",1496},
            {"One97 Communications (PayTm)","PAYTM.NS",1352},
            {"Sansera Engineering","SANSERA.NS",1703},
            {"Privi Speciality","PRIVISCL.NS",2824},
            {"Garware Hi-Tech","GRWRHITECH.NS",4148},
            {"Sai Life Science","SAILIFE.NS",921},
            {"Nippon India ETF Nifty 50 BeES","NIFTYBEES.NS",272},
            {"Venus Pipes & Tubes","VENUSPIPES.NS",1186},
            {"Knowledge Marine & Engineering Works","KMEW.NS",1752},
            {"Hindustan Zinc","HINDZINC.NS",606},
            {"RR Kabel","RRKABEL.NS",1650},
            {"Motilal Oswal Financial Services","MOTILALOFS.NS",829},
            {"Jeena Sikho Lifecare","JSLL.NS",755},
            {"KDDL","KDDL.NS",2614},
            {"Viyash Scientific","VIYASH.NS",257},
            {"Thyrocare","THYROCARE.NS",503},
            {"Apollo Micro Systems","APOLLO.NS",186.69},
            {"Astra Microwave Products","ASTRAMICRO.NS",898.4},
            {"Cosmic CRF","COSMICCRF.NS",254.98},
            {"E to E Transportation Infra","E2ERAIL.NS",174.5},
            {"Elecon Engineering","ELECON.NS",217},
            {"Garden Reach Shipbuilders","GRSE.NS",2311.2},
            {"HealthCare Global","HCG.NS",504.39},
            {"ISGEC Heavy Engineering","ISGEC.NS",900},
            {"Krishna Defence & Allied","KRISHNADEF.NS",841.55},
            {"Pennar Industries","PENIND.NS",141.23},
            {"Shriram Finance","SHRIRAMFIN.NS",984.3},
            {"Suyog Telematics","SUYOG.NS",592.95},
            {"Vinyas Innovative Technologies","VINYAS.NS",708.16},
            {"Websol Energy System","WEBELSOLAR.NS",65.85}
        }
    ),
    GetPrice = (ticker as text) as nullable number =>
        if ticker = "" then null
        else
            let
                Response = try Json.Document(
                    Web.Contents("https://query1.finance.yahoo.com",
                        [ RelativePath = "v8/finance/chart/" & ticker,
                          Query = [ range = "1d", interval = "1d" ],
                          Headers = [ #"User-Agent" = "Mozilla/5.0" ] ])) otherwise null,
                Price = try Number.Round(Response[chart][result]{0}[meta][regularMarketPrice], 2) otherwise null
            in Price,
    WithCMP  = Table.AddColumn(Watchlist, "CMP",   each GetPrice([Ticker]), type nullable number),
    WithPL   = Table.AddColumn(WithCMP,  "P/L %", each if [CMP] = null or [Entry] = 0 then null else Number.Round(([CMP] - [Entry]) / [Entry], 4), type nullable number),
    WithAsOf = Table.AddColumn(WithPL,   "As Of", each DateTime.LocalNow(), type datetime)
in
    WithAsOf
```
