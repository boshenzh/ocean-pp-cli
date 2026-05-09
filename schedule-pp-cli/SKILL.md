---
name: pp-schedule
description: "Discovers carriers, fleet routes, and weekly sailings for ocean-freight forwarders; combines Weiyun's public..."
author: "boshenzh"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - schedule-pp-cli
    install:
      - kind: go
        bins: [schedule-pp-cli]
        module: github.com/boshenzh/ocean-pp-cli/schedule-pp-cli/cmd/schedule-pp-cli
---

# Schedule — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `schedule-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   go install github.com/boshenzh/ocean-pp-cli/schedule-pp-cli/cmd/schedule-pp-cli@latest
   ```
2. Verify: `schedule-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails (no Node, offline, etc.), fall back to a direct Go install (requires Go 1.26.3 or newer):

```bash
go install github.com/boshenzh/ocean-pp-cli/schedule-pp-cli/cmd/schedule-pp-cli@latest
```

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

Container shipping schedule intelligence sourced from Weiyun's public APIs
(port catalog, carriers per port, fleet routes per carrier) plus per-lane
schedule data scraped from routePort URLs the user has registered.

The 4 public APIs at wywapi.weiyun001.com require no auth and return clean
JSON: port catalog, carrier list per port, service lines per (port,carrier),
and continent-grouped service maps. They cover lane discovery and prospecting.

Per-week schedule data (ETD dates, transit days, carrier services with
10-day-window sailings) comes from /routePort URLs the user captures once
in their browser; the CLI scrapes the HTML on demand. Both layers are
pure HTTP replay — no resident browser, no encryption reverse-engineering.


## When Not to Use This CLI

Do not activate this CLI for requests that require creating, updating, deleting, publishing, commenting, upvoting, inviting, ordering, sending messages, booking, purchasing, or changing remote state. This printed CLI exposes read-only commands for inspection, export, sync, and analysis.

## Unique Capabilities

These capabilities aren't available in any other tool for this API.

### Registry
- **`routes list`** — List all registered Weiyun lane URLs (alias -> URL).

  _When an agent needs to know which lanes the user has captured._

  ```bash
  schedule-pp-cli routes list --json
  ```

### Schedule data
- **`pull`** — Fetch routePort HTML for one or all registered lanes and persist to local SQLite.

  _When an agent needs the latest weekly sailings for the user's lane portfolio._

  ```bash
  schedule-pp-cli pull --route NS-JEDDAH
  ```
- **`query`** — Read the most recent scraped schedule for a registered lane from local SQLite.

  _When an agent needs to filter sailings by carrier whitelist._

  ```bash
  schedule-pp-cli query NS-JEDDAH --carrier PIL --carrier COSCO --json
  ```

### Schedule transcendence
- **`next-cls`** — Returns the lane's nearest future ETD vessel name + voyage + capacity + flag.

  _When an agent needs to answer 'what's the next ship out?' without scanning all rows._

  ```bash
  schedule-pp-cli next-cls NS-JEDDAH --json
  ```
- **`compare-carriers`** — Group lane vessels by ETD with carrier+service context.

  _When an agent needs to see all carriers' sailings on one lane in chronological order._

  ```bash
  schedule-pp-cli compare-carriers NS-JEDDAH --json
  ```

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

## Command Reference

**carrier-services** — Services (continent-grouped) a carrier offers from a port

- `schedule-pp-cli carrier-services` — Continent → service map for a (port, carrier) pair.

**carriers** — Carriers serving a given start port

- `schedule-pp-cli carriers` — Carriers (船公司) that depart from the named port.

**fleet_routes** — Fleet routes (services) per carrier from a given port

- `schedule-pp-cli fleet_routes` — Service lines (e.g. REX2, BEX, MEX) that a carrier runs out of the given port, grouped by destination region.

**ports** — Container ship start-port catalog

- `schedule-pp-cli ports` — Full list of ~200 ports Weiyun knows about (English/Chinese names, port code, pinyin).


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
schedule-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Auth Setup

No authentication required.

Run `schedule-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  schedule-pp-cli carrier-services --port-code example-value --agent --select id,name,status
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
schedule-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
schedule-pp-cli feedback --stdin < notes.txt
schedule-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.schedule-pp-cli/feedback.jsonl`. They are never POSTed unless `SCHEDULE_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `SCHEDULE_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

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
schedule-pp-cli profile save briefing --json
schedule-pp-cli --profile briefing carrier-services --port-code example-value
schedule-pp-cli profile list --json
schedule-pp-cli profile show briefing
schedule-pp-cli profile delete briefing --yes
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

1. **Empty, `help`, or `--help`** → show `schedule-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/boshenzh/ocean-pp-cli/schedule-pp-cli/cmd/schedule-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add schedule-pp-mcp -- schedule-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which schedule-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   schedule-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `schedule-pp-cli <command> --help`.
