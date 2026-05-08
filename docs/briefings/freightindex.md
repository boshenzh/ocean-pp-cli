# Briefing: freightindex-pp-cli

Notes to paste into the `/printing-press` Claude Code session when the skill asks
for context. Not auto-loaded.

## Invocation

```
/printing-press https://en.sse.net.cn/indices/scfinew.jsp
```

## Purpose

Pull SCFI (Shanghai Containerized Freight Index) once a day at 08:30 local,
persist to SQLite, render a weekly markdown digest the sales team reads
alongside our own quote table.

## Primary user

Ocean-freight forwarder selling Red Sea / Middle East / India-Pakistan lanes
out of South China.

## Data we care about

- SCFI columns from China to:
  - Mediterranean / Red Sea
  - Persian Gulf / Middle East
  - India / Pakistan (Nhava Sheva, Karachi)
- Other columns parse-and-store fine, but the digest highlights the three above.

## Required output modes

- `--json` for machine consumers (orchestrator).
- `--markdown` for the weekly sales digest.
- Default human output: a compact stdout table.

## Storage

Local SQLite under `~/.ocean-pp-cli/freightindex.db`. Schema must keep history
across runs so we can compute week-over-week deltas.

## Transcendence command (insight beyond endpoint mirroring)

`compare-with-rate-store <rate-table.xlsx>` — read a local rate workbook
(structure TBD; for v0.1 generate a stub that accepts the path and reports
"not yet wired"), match each lane against the latest SCFI point, label every
row High / Low / Par with the gap in USD/TEU.

The actual rate-store reader ships in a later plan. For this run, just create
the command and CLI plumbing; the body can be a stub that prints a clear TODO.

## Auth

None. Public site.

## Acceptance

- `printing-press scorecard` >= 85.
- `freightindex-pp-cli pull` writes one new SQLite row.
- `freightindex-pp-cli history --route 'CN-Mediterranean' --weeks 4` returns
  the last four weekly points.

## Carrier whitelist (for later filters; not needed at index level)

PIL, COSCO, KMTC, CMA, WHL, CUL, EMC, OOCL, HPL, RCL, TSL, 锦江, SLS, ESL,
WAN HAI, MAERSK.
