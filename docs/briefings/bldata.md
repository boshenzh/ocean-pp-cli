# Briefing: bldata-pp-cli

Notes to paste into the `/printing-press` Claude Code session when the skill asks
for context. Not auto-loaded.

## Pre-run user prep (time-boxed: 7-day trial)

Plan to start on a Monday so the trial covers a full work week. The HAR can
be captured early; the actual generation can stretch into the following week.

1. Register the 7-day trial at https://www.53wmb.com.
2. Confirm `.gitignore` includes `inputs/**/*.har`.
3. Chrome DevTools -> Network tab -> "Preserve log".
4. Run representative queries:
   - Look up one real existing customer by company name.
   - "Nansha -> JEDDAH" bills of lading in the last 90 days.
   - HS 8517 to EG in the last 30 days.
5. Save HAR to `inputs/bldata/52wmb.har`.
6. `git status` self-check.

## Invocation

```
/printing-press --har inputs/bldata/52wmb.har --name bldata
```

## Purpose

Mine bills of lading for prospecting (find new buyers in our lanes) and
follow-up (detect existing customers shipping with competitors). Feeds
Service Scenarios 1 and 4.

## Inputs

- Company name (consignee or shipper).
- HS code.
- POL.
- Date range (default last 12 months).

## Outputs per row

Shipper, consignee, carrier, POL, POD, container type and count, BL date.

## Storage

Local SQLite under `~/.ocean-pp-cli/bldata.db`.

## Transcendence commands

- `lost-customer-radar` — given a list of our active customers (read from
  rate-store in v0.2; for v0.1 accept a `--customers <file>` arg with a
  newline-delimited company list), flag any whose BL data shows shipments in
  the last 30 days that did **not** route through us.
- `prospect-rank <country>` — for a target country, rank consignees by
  total TEU received from China in the last 6 months.

## Auth

Cookie-based via captured HAR. Same `secret-protection` flow as schedule
if the trial session rotates.

## Acceptance

- `printing-press scorecard` >= 85.
- `bldata-pp-cli company "<real existing customer>"` returns BLs from the
  last 90 days.

## Trial-window risk

If the trial expires before generation finishes, the HAR may still work
for some endpoints (depends on 52WMB's session policy). If endpoints 401,
revisit subscription before promoting to production.
