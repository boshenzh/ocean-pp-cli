---
name: pp-freightindex
description: "A single-binary CLI for the Shanghai Containerized Freight Index — pulls the weekly index, persists snapshots, and... Trigger phrases: `scfi this week`, `shanghai freight index`, `container freight rates from china`, `compare scfi vs my rate sheet`, `use freightindex`, `run freightindex`."
author: "boshenzh"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - freightindex-pp-cli
    install:
      - kind: go
        bins: [freightindex-pp-cli]
        module: github.com/mvanhorn/printing-press-library/library/other/freightindex/cmd/freightindex-pp-cli
---

# Shanghai Containerized Freight Index — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `freightindex-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press install freightindex --cli-only
   ```
2. Verify: `freightindex-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails (no Node, offline, etc.), fall back to a direct Go install (requires Go 1.26.3 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/other/freightindex/cmd/freightindex-pp-cli@latest
```

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

Most container-rate intelligence sits behind paid terminals (Bloomberg, Drewry, S&P Platts) or a brittle BeautifulSoup scraper somebody wrote two years ago. freightindex-pp-cli pulls SCFI's public JSON endpoint directly, persists weekly snapshots to local SQLite, and ships compound queries — `digest` (filterable by lane), contribution decomposition, and lane-vs-comprehensive divergence — that the underlying API doesn't answer on its own. Useful for any forwarder, NVOCC, BCO desk, or analyst tracking spot-rate movements out of China.

## When to Use This CLI

Use this CLI when an agent needs ocean-freight rate intelligence for any forwarder, NVOCC, BCO procurement desk, or analyst tracking SCFI lane movements. The novel commands (digest, contribution, divergence, volatility, normalize, nulls) read from the local snapshot store, so they answer questions a generic SCFI scraper cannot.

## When Not to Use This CLI

Do not activate this CLI for requests that require creating, updating, deleting, publishing, commenting, upvoting, inviting, ordering, sending messages, booking, purchasing, or changing remote state. This printed CLI exposes read-only commands for inspection, export, sync, and analysis.

## Unique Capabilities

These capabilities aren't available in any other tool for this API.

### Local state that compounds
- **`digest`** — Render the latest weekly snapshot as JSON or Markdown; optional --lane <fragment> repeatable filter narrows to the lanes the user cares about. No defaults — works for any forwarder regardless of lane portfolio.

  _When an agent needs the Monday morning rate-update for any specific subset of lanes, this is the one command that produces the digest the desk actually broadcasts._

  ```bash
  freightindex-pp-cli digest --lane 'Persian Gulf' --lane 'Mediterranean' --markdown
  ```
- **`divergence`** — Per-week spread between a lane's percent change and the comprehensive index over the last N weeks, with cumulative drift.

  _When an agent needs to flag whether a lane is moving with or against the broader market, this is the multi-week comparison built for that question._

  ```bash
  freightindex-pp-cli divergence --route 'Persian Gulf and Red Sea' --weeks 8 --json
  ```
- **`compare-with-rate-store`** — Joins the user's quote workbook (xlsx) against the latest SCFI per lane and tags every row High/Low/Par with the USD/TEU gap. v0.1 ships a stub; full implementation lands when the rate-store reader does.

  _When an agent needs to check whether the desk's quote sheet is on-market, this is the side-by-side comparison._

  ```bash
  freightindex-pp-cli compare-with-rate-store /path/to/rates.xlsx
  ```

### Index math
- **`contribution`** — Decomposes the headline SCFI move into per-lane contributions by multiplying each lane's weighting percent by its percent change.

  _When an agent needs to explain why the headline index moved this week, this is the table that attributes the move to specific lanes._

  ```bash
  freightindex-pp-cli contribution --json
  ```
- **`volatility`** — Per-lane standard deviation / mean / min / max of weekly percent change over the last N weeks; ranks the most volatile lanes first.

  _When an agent needs to identify which lanes are jumpy enough to warrant fresh quotes, this is the ranking._

  ```bash
  freightindex-pp-cli volatility --weeks 12 --top 5 --json
  ```
- **`normalize`** — Converts every lane to a single unit (USD/TEU or USD/FEU) so cross-lane comparisons aren't apples-to-oranges.

  _When an agent has to feed SCFI rows into a downstream model that expects one unit, run this first._

  ```bash
  freightindex-pp-cli normalize --to FEU --json
  ```

### Reachability mitigation
- **`nulls`** — Lists weeks where SSE returned null for one or more lanes — a service quirk that breaks naive scripts.

  _When an orchestrator pipeline produces unexpected blanks in a lane's chart, this command tells you whether the data was missing upstream._

  ```bash
  freightindex-pp-cli nulls --weeks 8 --json
  ```

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

## Command Reference

**scfi** — Shanghai Containerized Freight Index (current week and weekly snapshots)

- `freightindex-pp-cli scfi` — Fetch the current weekly SCFI (Comprehensive Index plus all 15 lane sub-indices).


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
freightindex-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes


### Daily 08:30 digest cron

```bash
freightindex-pp-cli pull && freightindex-pp-cli digest --markdown
```

Idempotent pull then render the full Monday-morning digest; pipe to mail or post to Slack.

### Lane-filtered digest for a regional desk

```bash
freightindex-pp-cli digest --lane 'Persian Gulf' --lane 'India' --markdown
```

Filter to whatever lane subset matters for this desk; --lane is repeatable and matches case-insensitive substrings.

### Explain this week's headline move

```bash
freightindex-pp-cli contribution --json --select 'lane,weighting,percentage,contribution'
```

Decomposes the comprehensive index into per-lane contributions; pair `--agent` with `--select` so an agent only loads the four columns that matter.

### Did Persian Gulf detach from the market?

```bash
freightindex-pp-cli divergence --route 'Persian Gulf and Red Sea' --weeks 8 --json
```

Cumulative spread between a single lane and the comprehensive index over 8 weeks.

### Rank volatile lanes for re-quoting

```bash
freightindex-pp-cli volatility --weeks 12 --top 5 --json
```

Top 5 lanes by realized volatility — useful before a re-quote cycle.

## Auth Setup

No authentication required.

Run `freightindex-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  freightindex-pp-cli scfi --index-name example-resource --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Read-only** — do not use this CLI for create, update, delete, publish, comment, upvote, invite, order, send, or other mutating requests

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal — piped/agent consumers get pure JSON on stdout.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
freightindex-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
freightindex-pp-cli feedback --stdin < notes.txt
freightindex-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.freightindex-pp-cli/feedback.jsonl`. They are never POSTed unless `FREIGHTINDEX_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `FREIGHTINDEX_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
freightindex-pp-cli profile save briefing --json
freightindex-pp-cli --profile briefing scfi --index-name example-resource
freightindex-pp-cli profile list --json
freightindex-pp-cli profile show briefing
freightindex-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `freightindex-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/mvanhorn/printing-press-library/library/other/freightindex/cmd/freightindex-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add freightindex-pp-mcp -- freightindex-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which freightindex-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   freightindex-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `freightindex-pp-cli <command> --help`.
