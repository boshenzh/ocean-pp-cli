# Shanghai Containerized Freight Index CLI

**A single-binary CLI for the Shanghai Containerized Freight Index — pulls the weekly index, persists snapshots, and answers ocean-freight questions no single API call can.**

Most container-rate intelligence sits behind paid terminals (Bloomberg, Drewry, S&P Platts) or a brittle BeautifulSoup scraper somebody wrote two years ago. freightindex-pp-cli pulls SCFI's public JSON endpoint directly, persists weekly snapshots to local SQLite, and ships compound queries — `digest` (filterable by lane), contribution decomposition, and lane-vs-comprehensive divergence — that the underlying API doesn't answer on its own. Useful for any forwarder, NVOCC, BCO desk, or analyst tracking spot-rate movements out of China.

Learn more at [Shanghai Containerized Freight Index](https://en.sse.net.cn).

## Install

The recommended path installs both the `freightindex-pp-cli` binary and the `pp-freightindex` agent skill in one shot:

```bash
npx -y @mvanhorn/printing-press install freightindex
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press install freightindex --cli-only
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.3 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/other/freightindex/cmd/freightindex-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/freightindex-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-freightindex --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-freightindex --force
```

## Install for OpenClaw

Tell your OpenClaw agent (copy this):

```
Install the pp-freightindex skill from https://github.com/mvanhorn/printing-press-library/tree/main/cli-skills/pp-freightindex. The skill defines how its required CLI can be installed.
```

## Quick Start

```bash
# Pull the current week and append a snapshot.
freightindex-pp-cli pull --json


# Render the latest snapshot for every lane SSE published this week.
freightindex-pp-cli digest --markdown


# Filter to a forwarder's specific lanes (repeatable, case-insensitive substring match).
freightindex-pp-cli digest --lane 'Persian Gulf' --lane 'Mediterranean' --markdown


# Which lanes drove this week's comprehensive-index move.
freightindex-pp-cli contribution --json


# How that lane has moved relative to the headline over 8 weeks.
freightindex-pp-cli divergence --route 'Persian Gulf and Red Sea' --weeks 8


# Rank the most volatile lanes.
freightindex-pp-cli volatility --weeks 12 --top 5

```

## Unique Features

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

## Usage

Run `freightindex-pp-cli --help` for the full command reference and flag list.

## Commands

### scfi

Shanghai Containerized Freight Index (current week and weekly snapshots)

- **`freightindex-pp-cli scfi current`** - Fetch the current weekly SCFI (Comprehensive Index plus all 15 lane sub-indices).


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
freightindex-pp-cli scfi --index-name example-resource

# JSON for scripting and agents
freightindex-pp-cli scfi --index-name example-resource --json

# Filter to specific fields
freightindex-pp-cli scfi --index-name example-resource --json --select id,name,status

# Dry run — show the request without sending
freightindex-pp-cli scfi --index-name example-resource --dry-run

# Agent mode — JSON + compact + no prompts in one flag
freightindex-pp-cli scfi --index-name example-resource --agent
```

## Agent Usage

This CLI is designed for AI agent consumption:

- **Non-interactive** - never prompts, every input is a flag
- **Pipeable** - `--json` output to stdout, errors to stderr
- **Filterable** - `--select id,name` returns only fields you need
- **Previewable** - `--dry-run` shows the request without sending
- **Read-only by default** - this CLI does not create, update, delete, publish, send, or mutate remote resources
- **Offline-friendly** - sync/search commands can use the local SQLite store when available
- **Agent-safe by default** - no colors or formatting unless `--human-friendly` is set

Exit codes: `0` success, `2` usage error, `3` not found, `5` API error, `7` rate limited, `10` config error.

## Use with Claude Code

Install the focused skill — it auto-installs the CLI on first invocation:

```bash
npx skills add mvanhorn/printing-press-library/cli-skills/pp-freightindex -g
```

Then invoke `/pp-freightindex <query>` in Claude Code. The skill is the most efficient path — Claude Code drives the CLI directly without an MCP server in the middle.

<details>
<summary>Use as an MCP server in Claude Code (advanced)</summary>

If you'd rather register this CLI as an MCP server in Claude Code, install the MCP binary first:

```bash
go install github.com/mvanhorn/printing-press-library/library/other/freightindex/cmd/freightindex-pp-mcp@latest
```

Then register it:

```bash
claude mcp add freightindex freightindex-pp-mcp
```

</details>

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/freightindex-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.

```bash
go install github.com/mvanhorn/printing-press-library/library/other/freightindex/cmd/freightindex-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "freightindex": {
      "command": "freightindex-pp-mcp"
    }
  }
}
```

</details>

## Health Check

```bash
freightindex-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Config file: `~/.config/freightindex-pp-cli/config.toml`

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific

- **Empty JSON output for a specific lane** — Run `freightindex-pp-cli nulls --weeks 4` — SSE publishes some lanes intermittently and the empty value is upstream, not a bug here.
- **Pull returns the same week as last run** — SCFI is published weekly (Friday). Re-running between Fridays is a no-op; check `freightindex-pp-cli doctor` for last-fetch age.
- **Cross-lane comparisons mix USD/TEU and USD/FEU** — Pipe through `freightindex-pp-cli normalize --to FEU` first; the upstream response mixes units per lane.

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
