# Schedule CLI

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


Learn more at [Schedule](https://wywapi.weiyun001.com).

## Install

The recommended path installs both the `schedule-pp-cli` binary and the `pp-schedule` agent skill in one shot:

```bash
npx -y @mvanhorn/printing-press install schedule
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press install schedule --cli-only
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.3 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/other/schedule/cmd/schedule-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/schedule-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-schedule --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-schedule --force
```

## Install for OpenClaw

Tell your OpenClaw agent (copy this):

```
Install the pp-schedule skill from https://github.com/mvanhorn/printing-press-library/tree/main/cli-skills/pp-schedule. The skill defines how its required CLI can be installed.
```

## Quick Start

### 1. Install

See [Install](#install) above.

### 2. Verify Setup

```bash
schedule-pp-cli doctor
```

This checks your configuration.

### 3. Try Your First Command

```bash
schedule-pp-cli carrier-services --port-code example-value
```

## Unique Features

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

## Usage

Run `schedule-pp-cli --help` for the full command reference and flag list.

## Commands

### carrier-services

Services (continent-grouped) a carrier offers from a port

- **`schedule-pp-cli carrier-services list`** - Continent → service map for a (port, carrier) pair.

### carriers

Carriers serving a given start port

- **`schedule-pp-cli carriers by_port`** - Carriers (船公司) that depart from the named port.

### fleet_routes

Fleet routes (services) per carrier from a given port

- **`schedule-pp-cli fleet_routes list`** - Service lines (e.g. REX2, BEX, MEX) that a carrier runs out of the given port, grouped by destination region.

### ports

Container ship start-port catalog

- **`schedule-pp-cli ports list`** - Full list of ~200 ports Weiyun knows about (English/Chinese names, port code, pinyin).


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
schedule-pp-cli carrier-services --port-code example-value

# JSON for scripting and agents
schedule-pp-cli carrier-services --port-code example-value --json

# Filter to specific fields
schedule-pp-cli carrier-services --port-code example-value --json --select id,name,status

# Dry run — show the request without sending
schedule-pp-cli carrier-services --port-code example-value --dry-run

# Agent mode — JSON + compact + no prompts in one flag
schedule-pp-cli carrier-services --port-code example-value --agent
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
npx skills add mvanhorn/printing-press-library/cli-skills/pp-schedule -g
```

Then invoke `/pp-schedule <query>` in Claude Code. The skill is the most efficient path — Claude Code drives the CLI directly without an MCP server in the middle.

<details>
<summary>Use as an MCP server in Claude Code (advanced)</summary>

If you'd rather register this CLI as an MCP server in Claude Code, install the MCP binary first:

```bash
go install github.com/mvanhorn/printing-press-library/library/other/schedule/cmd/schedule-pp-mcp@latest
```

Then register it:

```bash
claude mcp add schedule schedule-pp-mcp
```

</details>

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/schedule-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.

```bash
go install github.com/mvanhorn/printing-press-library/library/other/schedule/cmd/schedule-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "schedule": {
      "command": "schedule-pp-mcp"
    }
  }
}
```

</details>

## Health Check

```bash
schedule-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Config file: `~/.config/schedule-pp-cli/config.toml`

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

## HTTP Transport

This CLI uses Chrome-compatible HTTP transport for browser-facing endpoints. It does not require a resident browser process for normal API calls.

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
