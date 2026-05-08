# freightindex Absorb Manifest

## Absorbed (match or beat everything that exists)

The competitive landscape for SCFI scraping is empty in open source. No GitHub repos
or npm/PyPI packages exist; competition is entirely paid SaaS. The "absorbed" rows
below are table-stakes pulled from paid feeds the user might evaluate against.

| # | Feature | Best Source | Our Implementation | Added Value |
|---|---------|------------|-------------------|-------------|
| 1 | Pull current SCFI by lane | Bloomberg Container Index (paid terminal) | `pull` -> JSON + SQLite snapshot | Free, single binary, no terminal subscription |
| 2 | Lane historical chart | Drewry Container Index (paid sub) | `history --route R --weeks N` from local SQLite | Offline, agent-friendly JSON |
| 3 | Week-over-week delta | All paid feeds | `delta --weeks N` sorted by largest move | Free, scriptable |
| 4 | Lane filter | All paid feeds | `--route` flag on every command | Same |
| 5 | CSV/JSON/Markdown export | Most paid feeds | `--json --csv --markdown` everywhere | Better than CSV-only feeds |
| 6 | Threshold alerts | Some paid feeds | `route-watch <route> --threshold 5%` | Free; emit-to-stdout (orchestrator pipes) |
| 7 | Lane listing / fuzzy search | Most feeds | `lanes [--match TEXT]` (FTS5) | Offline + fuzzy |
| 8 | Reachability / health check | Most feeds | `doctor` (endpoint + DB schema + last-fetch-age) | Typed exit codes for cron |
| 9 | Raw SQL / explore | Most feeds | `sql 'SELECT ...'` | JSON/CSV/markdown output |

## Transcendence (only possible with our approach)

Output from the novel-features brainstorm (Pass 3 survivors, all scored >=6/10). See
[novel-features-brainstorm.md](./2026-05-08-151751-novel-features-brainstorm.md) for
the customer model, candidate list, and kill rationale.

| # | Feature | Command | Score | Why Only We Can Do This | Persona |
|---|---------|---------|-------|------------------------|---------|
| 1 | Priority-lane digest | `priority-digest [--markdown]` | 9 | Joins last-two snapshots from local SQLite, filters to user's three priority lanes (Med/Red Sea, Persian Gulf/Mideast, India/Pak), renders one-screen Markdown — replaces the user's manual Saturday WeChat broadcast. | Sales Manager (A) |
| 2 | Comprehensive-index contribution | `contribution [--week W]` | 8 | Decomposes the comprehensive-index move into `weighting% × percent_change` per lane; sorted by absolute contribution. The `weighting_EN` column only matters because of this math; no paid feed ships it. | Shipping Analyst (C) |
| 3 | Lane-vs-comprehensive divergence | `divergence --route R [--weeks N]` | 8 | Cross-week local SQLite join: lane_pct - comprehensive_pct over N weeks with cumulative spread. Surfaces "my lane detached from the headline." | Sales Manager (A), Analyst (C) |
| 4 | Compare with in-house rate sheet (stub in v0.1) | `compare-with-rate-store <rate.xlsx>` | 7 | Cross-source join: in-house quote workbook vs latest SCFI lane index, tagged High/Low/Par with USD/TEU gap. v0.1 ships as a stub printing "rate-store reader not yet wired"; full impl in plan #2. | Sales Manager (A), Procurement (B) |
| 5 | Null-lane gap report | `nulls [--weeks N]` | 7 | Service-specific quirk: SSE returns null for some lanes some weeks. Local-data scan over snapshots returns weeks-with-missing-lanes. Breaks naive scripts otherwise. | Procurement (B), Analyst (C) |
| 6 | Lane volatility ranking | `volatility [--weeks N] [--top K]` | 7 | Per-lane stdev/mean/min/max over the last N weeks of `percentage`. Cross-week aggregate from local snapshots — not derivable from a single API call. | Analyst (C), Sales Manager (A) |
| 7 | Unit normalization | `normalize --to FEU` | 6 | Branches on `unit_EN` (USD/TEU vs USD/FEU), multiplies by 2 to normalize, emits cross-lane comparable table. Mechanical; required by Procurement (B)'s pandas pipeline. | Procurement (B), Analyst (C) |

## Stubs (explicit)

Per Phase 1.5 rules, stub features must be flagged for the gate showcase:

- **`compare-with-rate-store`** — v0.1 stub. Reason: full implementation requires an
  rate-store reader (xlsx parsing of the user's quote workbook) which is in another
  plan. v0.1 ships the command, parses args, prints a clear "rate-store reader not yet
  wired — see plan #2" and exits 0 so orchestrators can wire it once plan #2 lands.

## Kill rationale

The brainstorm dropped 9 candidates. Highlights:

- `streak`, `comove`, `regime` — niche statistical detectors absorbed by `volatility`.
- `priority-watch` — convenience macro over `route-watch`; rolled into `priority-digest`.
- `lane-trend` — thin alias of absorbed `history --json`.
- `lanes`, `doctor`, `sql` — Priority-1 absorb scope, not novel.
- `rebase-check` — fires once every few years; verifiability low.

Full list with one-sentence kill reasons is in the brainstorm artifact.
