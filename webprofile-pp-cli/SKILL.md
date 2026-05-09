---
name: pp-webprofile
description: "A single-binary CLI for UN Comtrade trade-flow data — converts country names and HS descriptions into numeric... Trigger phrases: `comtrade for egypt`, `trade flow research`, `who imports from china`, `score this prospect`, `use webprofile`, `run webprofile`."
author: "boshenzh"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - webprofile-pp-cli
    install:
      - kind: go
        bins: [webprofile-pp-cli]
        module: github.com/boshenzh/ocean-pp-cli/webprofile-pp-cli/cmd/webprofile-pp-cli
---

# UN Comtrade — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `webprofile-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   go install github.com/boshenzh/ocean-pp-cli/webprofile-pp-cli/cmd/webprofile-pp-cli@latest
   ```
2. Verify: `webprofile-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails (no Node, offline, etc.), fall back to a direct Go install (requires Go 1.26.3 or newer):

```bash
go install github.com/boshenzh/ocean-pp-cli/webprofile-pp-cli/cmd/webprofile-pp-cli@latest
```

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

Comtrade's official portal requires a paid key for the production API, but the public preview endpoint exposes the same trade-flow shape with a 500-record cap. webprofile-pp-cli wraps it for ocean-freight forwarders: country and HS lookups happen offline (cached reference files), and prospect-fit scoring combines import volume, growth, and your route portfolio into a single number.

## When to Use This CLI

Use this CLI when an agent needs trade-flow data for ocean-freight prospect research. The transcendence commands (country, hs-search, who-imports, fit-score) read from the local cache, so they answer prospecting questions a generic Comtrade scraper cannot.

## When Not to Use This CLI

Do not activate this CLI for requests that require creating, updating, deleting, publishing, commenting, upvoting, inviting, ordering, sending messages, booking, purchasing, or changing remote state. This printed CLI exposes read-only commands for inspection, export, sync, and analysis.

## Unique Capabilities

These capabilities aren't available in any other tool for this API.

### Reference resolution
- **`country`** — Fuzzy-search Comtrade country reporters by name, ISO2, ISO3, or numeric code.

  _When an agent needs Egypt's numeric Comtrade code without making the user know the magic number 818._

  ```bash
  webprofile-pp-cli country Egypt --json
  ```
- **`hs-search`** — Fuzzy-search HS6 commodity codes by code prefix or text description.

  _When an agent needs the HS code for a commodity name._

  ```bash
  webprofile-pp-cli hs-search telephone --json
  ```

### Trade-flow research
- **`who-imports`** — Annual import volume for a country/HS pair from a specific partner (default China).

  _When an agent needs concrete trade-flow numbers for a prospect._

  ```bash
  webprofile-pp-cli who-imports Egypt 8517 --years 3 --json
  ```
- **`fit-score`** — 0-100 score combining China-share of imports, year-over-year growth, and route coverage.

  _When an agent needs to rank candidate destinations for an outbound campaign._

  ```bash
  webprofile-pp-cli fit-score Egypt 8517 --json
  ```

## Command Reference

**reporters** — Country reference data (numeric Comtrade code, ISO3, name)

- `webprofile-pp-cli reporters` — Fetch the full Comtrade reporter (country) reference file.

**trade** — UN Comtrade trade-flow data

- `webprofile-pp-cli trade` — Fetch annual trade-flow records (no auth, ~500 records max).


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
webprofile-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Recipes


### Score a candidate country

```bash
webprofile-pp-cli fit-score Egypt 8517 --json --select 'fit_score,import_from_cn,yoy_growth_pct'
```

Compact agent-shaped score with the three components that drove the number.

### Compare imports from China vs world

```bash
webprofile-pp-cli who-imports India 6109 --partner World --years 3
```

Run with partner=World to see total imports, then partner=China to see just China — gap is the China-share.

## Auth Setup

No authentication required.

Run `webprofile-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  webprofile-pp-cli reporters --agent --select id,name,status
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
webprofile-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
webprofile-pp-cli feedback --stdin < notes.txt
webprofile-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.webprofile-pp-cli/feedback.jsonl`. They are never POSTed unless `WEBPROFILE_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `WEBPROFILE_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

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
webprofile-pp-cli profile save briefing --json
webprofile-pp-cli --profile briefing reporters
webprofile-pp-cli profile list --json
webprofile-pp-cli profile show briefing
webprofile-pp-cli profile delete briefing --yes
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

1. **Empty, `help`, or `--help`** → show `webprofile-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

1. Install the MCP server:
   ```bash
   go install github.com/boshenzh/ocean-pp-cli/webprofile-pp-cli/cmd/webprofile-pp-mcp@latest
   ```
2. Register with Claude Code:
   ```bash
   claude mcp add webprofile-pp-mcp -- webprofile-pp-mcp
   ```
3. Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which webprofile-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   webprofile-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `webprofile-pp-cli <command> --help`.
