# freightindex CLI Brief

## API Identity
- Domain: Container shipping rates (Shanghai Containerized Freight Index, SCFI). Published weekly by the Shanghai Shipping Exchange (SSE). The SCFI is the de-facto benchmark for spot-market rates from China.
- Source: `https://en.sse.net.cn/indices/scfinew.jsp` is the public web page; the data is loaded via two replayable JSON endpoints discovered in the page JS:
  - `GET https://en.sse.net.cn/currentIndex?indexName=scfi` — current week, no auth, returns `data.lineDataList[]` with 16 rows (Comprehensive + 15 lanes)
  - `GET https://en.sse.net.cn/singleIndex/scfi?date=YYYY-MM-DD HH:MM:SS` — historical by date (login-gated; `/islogin2` check; CSRF token in form)
- Users: ocean-freight forwarders, NVOCCs, BCO procurement teams, shipping analysts.
- Data profile: 16 weekly rows. Each row has `properties.lineName_EN`, `unit_EN` (mostly `USD/TEU` or `USD/FEU`), `weighting_EN`, plus `currentContent`, `lastContent`, `absolute`, `percentage`. Some lanes return null when not published in a given week.

## Reachability Risk
- None for `/currentIndex`. HTTP 200 + `application/json`, no auth, just needs a `User-Agent` header.
- Medium for `/singleIndex/scfi?date=...` — requires session login on en.sse.net.cn. Out of scope for v0.1 (current week is enough for the briefing's needs).

## Top Workflows
1. **Daily morning poll** (08:30) — fetch `/currentIndex?indexName=scfi`, append to local SQLite if new week, render Markdown digest filtered to user's lanes (Mediterranean/Red Sea, Persian Gulf/Middle East, India/Pakistan).
2. **Lane history** — `freightindex history --route 'CN-Mediterranean' --weeks 4` reads the local SQLite to compute 4-week trend (need >=4 weekly snapshots; backfill comes from re-running poll).
3. **Compare to our quote** — `freightindex compare-with-rate-store rates.xlsx` (transcendence; v0.1 stub) — cross-reference our company quote vs the latest SCFI for the same lane.
4. **JSON / Markdown export** — agents and humans can both consume; default `pull` writes JSON to stdout, `report` writes Markdown.

## Table Stakes (what every freight-index tool must do)
Pulled from competitive landscape — all paid/proprietary, no community wrappers exist:
- **Bloomberg Container Index** (paid terminal)
- **Drewry Container Index** (drewry.co.uk paid subscription)
- **S&P Platts / Argus container assessments** (paid)
- **Freightos Baltic Index (FBX)** (fbx.freightos.com, free web page, paid API)
- **Xeneta** (SaaS, paid)
- **CCFI** (sister index from same SSE.net.cn, same shape)

Common features:
- Pull current week index by lane.
- Historical chart / table.
- Week-over-week delta.
- Lane filter.
- CSV/JSON export.
- Email/webhook on threshold change.

## Data Layer
- Primary entities: `scfi_snapshot(week_date PRIMARY KEY, fetched_at, raw_json)`, `scfi_line(week_date, line_name, unit, weighting, current, last, absolute, percentage, PRIMARY KEY (week_date, line_name))`.
- Sync cursor: `currentDate` from response. If unchanged from last persisted row, skip.
- FTS/search: `scfi_line_fts(line_name)` for fuzzy lane search.

## User Vision (from docs/briefings/freightindex.md)
- Run daily 08:30, persist to SQLite, render weekly Markdown digest.
- Three priority lanes: Mediterranean/Red Sea, Persian Gulf/Middle East, India/Pakistan (Nhava Sheva, Karachi).
- `--json` for orchestrator, `--markdown` for the sales digest.
- Transcendence: `compare-with-rate-store <rate-table.xlsx>` — for v0.1 ship as a stub command with a clear "wire rate-store next" message; full impl in a later plan.

## Product Thesis
- Name: `freightindex-pp-cli`.
- Why it should exist: SCFI data is published weekly on a flaky JS-rendered Chinese-government site. There's no Go/Node/Python wrapper. The data is structurally trivial (one JSON endpoint, 16 rows). What's missing is a tiny single-binary CLI that polls daily, persists weekly snapshots, filters to lanes you care about, and outputs JSON for orchestrators.

## Build Priorities
1. **Priority 0 (foundation):** `pull` (fetch current index → SQLite), schema for `scfi_snapshot` + `scfi_line`, `--json` and table output.
2. **Priority 1 (absorb):** `history --route X --weeks N`, `lanes` (list known lane names), `digest` (Markdown weekly report filtered to user's priority lanes), `doctor` (endpoint reachability), `sql` (raw SQL on the local DB).
3. **Priority 2 (transcend):**
   - `compare-with-rate-store <file>` — stub for v0.1.
   - `delta --weeks N` — show week-over-week % change for every lane, sorted by largest move.
   - `route-watch <route> --threshold 5%` — alert when a lane moves by more than threshold (stub the alerter; print to stdout).
   - `priority-digest` — render only the user's priority lanes (Mediterranean/Red Sea, Persian Gulf/Middle East, India/Pakistan) as a one-screen Markdown summary.
   - `lane-trend <route> --json --weeks 12` — emit a tiny time-series JSON suitable for an orchestrator to chart.
