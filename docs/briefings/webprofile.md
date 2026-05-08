# Briefing: webprofile-pp-cli

Notes to paste into the `/printing-press` Claude Code session when the skill asks
for context. Not auto-loaded.

## Invocation

```
/printing-press https://comtradeapi.un.org/
```

If the skill asks for a more specific spec URL, point at the public Comtrade v1
endpoints documentation.

## Scope decision (important)

Generate the **Comtrade-only** CLI in this run. Hunter.io and generic-website
scraping are interesting secondary sources but cannot be merged into one
binary by printing-press in a single run. They land in v0.2 as hand additions.

## Purpose

Customer prospecting: given a country and HS code, return the country's import
volume from China, seasonality, and top importers. Feeds the lead pipeline
(Service Scenario 1).

## Inputs

- ISO country code (importer side, e.g. `EG`, `SA`, `AE`, `IN`, `PK`).
- HS code (commodity, e.g. `8517`, `8703`, `6109`).
- Optional company name (for later cross-reference with bldata).

## Outputs

- Monthly trade volumes for the last 24 months.
- Seasonality summary (peak months, off-season).
- Top importers in the destination country for that HS code (ranked by value).
- `--json` and a compact human table.

## Storage

Local SQLite under `~/.ocean-pp-cli/webprofile.db`. Comtrade rate-limits
aggressively, so cache-on-write is required; default TTL 7 days.

## Transcendence command

`fit-score <country> <hs-code>` — combine import volume, seasonality stability,
and our route coverage (red-sea / mideast / india-pak) into a 0-100 score.
Route coverage is a hardcoded list for v0.1 (see hint below); it will read
from rate-store in a later version.

## Auth

Comtrade has a free tier with API key. Read the key from `COMTRADE_API_KEY`
env var. The skill should follow its `secret-protection` flow when wiring this.

## Acceptance

- `printing-press scorecard` >= 85.
- `webprofile-pp-cli profile EG 8517` returns 24 months of data.
- Re-running the same command within 7 days hits the cache (no API call).

## Route coverage hint (for fit-score, v0.1 hardcoded)

Origins: South China ports (SK Shekou, NS Nansha, YT Yantian).
Destinations: JEDDAH, SOKHNA, AQABA, DJIBOUTI, ADEN, NHAVA SHEVA, KARACHI,
HAMAD, JEBEL ALI, DAMMAM.
