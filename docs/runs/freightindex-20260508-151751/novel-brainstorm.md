## Customer model

### Persona A — Lin Wei, Sales Manager, NVOCC in Shenzhen

**Today.** Lin runs a 6-person sales desk at a mid-size NVOCC in Shenzhen quoting FCL out of Yantian and Shekou. Three of his salespeople focus on Mediterranean / Red Sea, two on Persian Gulf / Middle East, one on India / Pakistan (Nhava Sheva, Karachi). He owns the in-house quote sheet — an Excel workbook updated once a week — and signs off on every quote above USD 1,500 / FEU.

**Weekly ritual.** Friday night the SCFI updates. Saturday morning Lin opens the en.sse.net.cn page in a browser, manually copies four numbers (comprehensive index + his three priority lanes) into a WeChat broadcast to the desk, then re-prices the quote sheet by Monday. On Monday 08:30 he wants a one-screen Markdown digest in his inbox so he can re-broadcast without opening the SSE site.

**Frustration.** The SSE page is JS-rendered, slow, and sometimes returns null for one of his lanes (Persian Gulf in particular). He has no way to see "the comprehensive index moved +6% but my lanes only moved +1.2% — am I leaving money on the table?" without doing arithmetic in his head.

### Persona B — Priya Shah, Procurement Analyst, BCO (textile importer)

**Today.** Priya works for a Mumbai textile importer that ships 200 TEU/month from South China to Nhava Sheva. She negotiates contract rates twice a year and uses spot-rate intelligence to pressure carriers when contracts come up. She doesn't read Mandarin and refuses to log in to a Chinese-government site.

**Weekly ritual.** She runs an orchestrator (LangChain + cron) that pulls SCFI weekly, joins it against carrier quotes she gets via email, and emails her boss any week the India-Pakistan SCFI lane moves >5% or detaches from the comprehensive index. She wants every output as JSON so her orchestrator can pipe it into pandas.

**Frustration.** No Go/Node/Python wrapper exists. She wrote a brittle BeautifulSoup scraper two years ago that breaks every six months. She wants a single binary on her laptop and her ops box that returns JSON, with no Python deps and no Chinese login.

### Persona C — Tom Becker, Shipping Analyst, freight-research desk

**Today.** Tom writes a weekly market note for paying subscribers covering container-shipping spot rates. He reads SCFI, FBX, Drewry, and CCFI side-by-side. Subscribers want "which lane drove the comprehensive index this week" and "are Mediterranean and Red Sea still co-moving."

**Weekly ritual.** Friday evening he downloads the SCFI rows, opens Excel, multiplies each lane's weighting% by its percent change to attribute the headline move, then writes 200 words. He archives an XLS per week to a folder going back to 2023.

**Frustration.** Excel doesn't carry history forward cleanly, weighting% changes occasionally (rebases), and he keeps re-deriving the same lane-contribution math every week. He'd take a single CLI command that emits the contribution table as JSON and Markdown.

---

## Candidates (pre-cut)

> Sources legend: (a) persona-driven, (b) service-specific content, (c) cross-entity local query, (e) user briefing.
>
> The absorb manifest already covers `pull`, `history --route X --weeks N`, `delta --weeks N`, lane filter, `--json/--csv/--markdown`, and `route-watch <route> --threshold 5%`. Those are NOT re-proposed below.

| # | Name | Command | One-line description | Persona | Source |
|---|------|---------|----------------------|---------|--------|
| C1 | priority-digest | `freightindex priority-digest [--markdown]` | One-screen Markdown digest filtered to Med/Red-Sea, Persian Gulf/Mideast, India-Pak with WoW deltas + comprehensive index. | A | (a)(e) |
| C2 | compare-with-rate-store | `freightindex compare-with-rate-store rates.xlsx` | Cross-source join: in-house quote sheet vs latest SCFI per lane; tags High/Low/Par + USD/TEU gap. v0.1 stub. | A,B | (c)(e) |
| C3 | contribution | `freightindex contribution [--week W]` | Decomposes the comprehensive index move into weighting% × percent change per lane; sorted by absolute contribution. | C | (b) |
| C4 | divergence | `freightindex divergence --route R [--weeks N]` | For each priority lane, % change vs comprehensive index over N weeks; surfaces "my lane detached from the headline." | A,C | (b)(c) |
| C5 | nulls | `freightindex nulls [--weeks N]` | Lists lanes that returned null (no publication) per week — SSE quirk that breaks naive scripts. | B,C | (b) |
| C6 | volatility | `freightindex volatility [--weeks N] [--top K]` | Stdev of weekly percent change per lane over N weeks; ranks lanes by realized volatility. | C | (b)(c) |
| C7 | streak | `freightindex streak [--route R]` | Consecutive weeks each lane moved in the same direction (up/down); "Med up 6 weeks running." | A,C | (c) |
| C8 | comove | `freightindex comove --route A --route B [--weeks N]` | Pearson correlation of weekly percent change between two lanes from local SQLite. | C | (c) |
| C9 | normalize-feu | `freightindex normalize --to FEU` | Converts every lane to a single unit (USD/FEU) so cross-lane comparisons aren't apples-to-oranges. | B,C | (b) |
| C10 | regime | `freightindex regime --route R [--weeks N] [--sigma 2]` | Flags weeks where a lane's percent change exceeded N-week-rolling-mean ± k·stdev (regime change). | C | (b)(c) |
| C11 | lane-trend | `freightindex lane-trend <route> --weeks 12 --json` | Tiny time-series JSON for one lane suitable for an orchestrator to chart. | B | (e) |
| C12 | lanes | `freightindex lanes` | Lists known lane names from the local DB (FTS-backed fuzzy search via `--match`). | A,B,C | (a) |
| C13 | doctor | `freightindex doctor` | Endpoint reachability + DB schema + last-fetch-age check; exits non-zero on stale data. | B | (a) |
| C14 | sql | `freightindex sql 'SELECT ...'` | Raw SQL on the local DB; emits JSON/CSV/markdown. | B,C | (c) |
| C15 | priority-watch | `freightindex priority-watch [--threshold 5%]` | Like `route-watch` but auto-applied to the user's three priority lanes; emits one-line alerts to stdout. | A,B | (a)(e) |
| C16 | rebase-check | `freightindex rebase-check [--weeks N]` | Detects weeks where a lane's `weighting_EN` changed (SSE rebases the comprehensive basket occasionally). | C | (b)(c) |

### Inline kill/keep notes (applied during generation)

- C2 (compare-with-rate-store): scope-creep risk neutralized by shipping a v0.1 stub per the brief.
- C5 (nulls): is a local-data report, not a fake API call — passes reimplementation check.
- C9 (normalize-feu): mechanical (multiply by 2 for TEU-priced lanes when target is FEU; verify against unit_EN per row); no LLM needed.
- C13 (doctor) and C14 (sql) are foundation/Priority-1 in the brief — ALREADY covered by Priority-1 absorb scope. They are ritual table-stakes, not novel transcendence — they will be killed in Pass 3.
- C12 (lanes) similarly is Priority-1 absorb scope — kill in Pass 3.
- C11 (lane-trend) is suspiciously close to `history --route X --weeks N --json` from the absorb manifest. Probable kill in Pass 3.

---

## Survivors and kills

### Survivors

| # | Feature | Command | Score | How It Works | Evidence | Weekly use | Sibling killed |
|---|---------|---------|-------|--------------|----------|------------|----------------|
| 1 | Priority-lane digest | `freightindex priority-digest [--markdown]` | 9/10 | Reads latest two `scfi_snapshot` rows from local SQLite, filters `scfi_line` to the user's three priority lanes (Mediterranean/Red Sea, Persian Gulf/Middle East, India/Pakistan) plus the comprehensive index, computes WoW absolute and percent deltas, renders a one-screen Markdown table with the comprehensive index as a reference line. | User briefing explicitly names these three lanes and the 08:30 daily-digest workflow; brief Build Priorities §2 lists `digest` and §3 lists `priority-digest`. Persona A's Saturday WeChat broadcast is exactly this. | Yes — Persona A reads it every Monday 08:30 (the briefing's headline workflow). | C15 priority-watch (rolled into the digest output rather than a separate alerter). |
| 2 | Comprehensive-index contribution | `freightindex contribution [--week W]` | 8/10 | For each row in `scfi_line` for week W, multiply `weighting_EN` (parsed from "12.34%" string) by `percentage` (already a number) to get each lane's contribution to the comprehensive move; sort by absolute contribution; emit JSON/Markdown table. | Service-specific column semantic — `weighting_EN` exists in the response and only matters because of this math. Persona C does this in Excel weekly. No paid feed (Drewry/FBX/Bloomberg) ships this decomposition for SCFI specifically. | Yes — Persona C re-derives the contribution table by hand every Friday and would replace that with this command. | C16 rebase-check (overlaps the same `weighting_EN` parse logic but fires too rarely to keep). |
| 3 | Lane-vs-comprehensive divergence | `freightindex divergence --route R [--weeks N]` | 8/10 | Joins `scfi_line` rows for the named route against the `scfi_line` row where `line_name` matches "Comprehensive Index" over the last N week-dates from `scfi_snapshot`; emits per-week (route_pct - comprehensive_pct) with cumulative spread. | Cross-week local SQLite join. Persona A's "the headline moved +6% but my lane is flat" frustration. Pure local data; no API call beyond the existing snapshots. | Yes — Persona A checks his three priority lanes against the headline weekly; Persona C runs it for the market note. | C8 comove (lane-vs-comprehensive is the one comparison that always matters; pairwise comove is niche). |
| 4 | Compare with in-house rate sheet (stub) | `freightindex compare-with-rate-store <rate.xlsx>` | 7/10 | v0.1 stub: parses CLI args, prints "rate-store reader not yet wired — see plan #2" and exits 0; full impl reads xlsx, joins rows by lane name against latest `scfi_line`, tags High/Low/Par, emits USD/TEU gap. | User briefing names this verbatim as the "Transcendence command." Brief §3 ships it as a stub. Cross-source join with a non-API local artifact. | Yes — Persona A re-prices the in-house quote sheet weekly against the new SCFI; full impl is the headline weekly action. | (none — no sibling proposed; xlsx-join is unique to this command.) |
| 5 | Null-lane gap report | `freightindex nulls [--weeks N]` | 7/10 | Scans `scfi_line` rows for the last N weeks where `current` IS NULL or row missing; emits per-week list of unpublished lanes. | Service quirk: SSE returns null for some lanes some weeks (brief calls this out explicitly). Breaks naive scripts. Local-data query — not a fake API call. | Yes — Persona B's orchestrator needs to skip null lanes cleanly; she runs this as a sanity check after every pull. | C13 doctor (doctor checks reachability; nulls checks data-completeness — kept the data-shape one). |
| 6 | Lane volatility ranking | `freightindex volatility [--weeks N] [--top K]` | 7/10 | For each lane in `scfi_line`, compute stdev of `percentage` over the last N week-dates, rank descending, return top K with mean/stdev/min/max. | Cross-week aggregate that requires the local snapshots — not a single API call. Persona C's research-desk job; differentiator vs Drewry/FBX which only show point-in-time charts. | Yes — Persona C ranks volatile lanes weekly for the market note; Persona A uses it to decide which lanes need re-quoting. | C7 streak and C10 regime (volatility subsumes both as a more robust statistic). |
| 7 | Unit normalization | `freightindex normalize --to FEU` | 6/10 | For each lane in the latest snapshot, branch on `unit_EN` (USD/TEU vs USD/FEU), multiply TEU rows by 2 when target is FEU (and vice versa), emit normalized table. | Service-specific column quirk that paid feeds present but don't normalize. Persona B's pandas pipeline needs single-unit input. Mechanical, no LLM. | Yes — Persona B's weekly pandas pipeline requires single-unit input; she runs it on every pull. | C11 lane-trend (lane-trend is a thin alias of absorbed `history --json`; normalization is the genuinely novel bit on the same data). |

### Killed candidates

| # | Killed feature | Closest surviving sibling | Kill reason |
|---|----------------|---------------------------|-------------|
| C7 | streak | C6 volatility | Streak length is a single statistic that adds little over the volatility ranking and is brittle when null-lane weeks interrupt the count. |
| C8 | comove (Pearson correlation between two lanes) | C6 volatility | Niche analyst tool; Persona C runs it in pandas already. Doesn't service Personas A or B and verifying correctness in dogfood is hard. |
| C10 | regime (rolling z-score outlier flag) | C6 volatility / C3 contribution | Statistical detector with arbitrary sigma threshold; overlaps `delta` (already absorbed) and `volatility`; weak weekly use without a tuned threshold. |
| C11 | lane-trend | (kept in absorb: `history --json`) | Thin alias of `history --route X --weeks N --json`; no novel transcendence above the absorbed command. |
| C12 | lanes | (Priority-1 absorb scope) | Foundation/listing command, already in the brief's Priority-1 absorb list, not a novel transcendence feature. |
| C13 | doctor | (Priority-1 absorb scope) | Foundation reachability check, already in Priority-1 absorb list. |
| C14 | sql | (Priority-1 absorb scope) | Already in Priority-1 absorb list. Useful escape hatch but not a novel transcendence feature. |
| C15 | priority-watch | C1 priority-digest + absorbed `route-watch` | Convenience macro over `route-watch` (already absorbed) applied three times; no new mechanism. |
| C16 | rebase-check | C3 contribution | Edge-case detector for an event that fires once every few years; weak weekly use; verifiability low without a known historical rebase to test against. |
