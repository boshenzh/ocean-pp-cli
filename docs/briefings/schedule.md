# Briefing: schedule-pp-cli

Notes to paste into the `/printing-press` Claude Code session when the skill asks
for context. Not auto-loaded.

## Pre-run user prep

1. Register a free account at https://www.weiyun001.com/.
2. Confirm `git status` is clean and `.gitignore` already contains
   `inputs/**/*.har` (it does — see repo root).
3. Open Chrome DevTools, Network tab, tick "Preserve log".
4. In the Weiyun web UI run 3-5 representative queries spanning our lanes:
   - SK -> JEDDAH
   - NS -> SOKHNA
   - SK -> NHAVA SHEVA
   - NS -> KARACHI
   - SK -> DJIBOUTI
5. Right-click any row in the Network panel -> Save all as HAR with content ->
   save to `inputs/schedule/weiyun.har`.
6. Run `git status` and confirm the HAR is **not** in the staged list.

## Invocation

```
/printing-press --har inputs/schedule/weiyun.har --name schedule
```

## Purpose

Given POL/POD, return all sailings in the next 14 days. Powers the daily sales
briefing and the auto-quote scenario.

## Required fields per sailing

- Carrier (船司)
- Vessel name (船名)
- Voyage number (航次)
- CLS / cut-off time (截关)
- ETD
- ETA
- Direct or transshipment flag
- Transshipment port (if any)

## Carrier whitelist

PIL, COSCO, KMTC, CMA, WHL, CUL, EMC, OOCL, HPL, RCL, TSL, 锦江, SLS, ESL,
WAN HAI, MAERSK.

Other carriers can be parsed but the default `query` view filters to this list.

## Outputs

- `--json` for orchestrator.
- Default human output: table sorted by ETD.
- SQLite under `~/.ocean-pp-cli/schedule.db` for cross-run history.

## Transcendence commands

- `next-cls <POL> <POD>` — the soonest cut-off we can still hit, accounting
  for current local time.
- `compare-carriers <POL> <POD> <date>` — same lane, all carriers, ETA gap
  vs the earliest, sorted ascending.

## Auth

Cookie-based session via the captured HAR. If Weiyun rotates sessions or
adds anti-bot, follow the skill's `secret-protection` flow to inject cookies
from an env var rather than baking them into the binary.

## Acceptance

- `printing-press scorecard` >= 85.
- `schedule-pp-cli query SK JEDDAH --next-7-days` returns real sailings
  including at least three of: PIL, COSCO, KMTC.

## Multi-source note

Maersk Schedule API and CMA-CGM eBusiness API are interesting but go into
v0.2 as hand-merged sources. Do not try to fold them into this run.
