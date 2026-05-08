# UN Comtrade CLI

**A single-binary CLI for UN Comtrade trade-flow data — converts country names and HS descriptions into numeric codes locally, then pulls per-year import volumes for ocean-freight prospect research.**

Comtrade's official portal requires a paid key for the production API, but the public preview endpoint exposes the same trade-flow shape with a 500-record cap. webprofile-pp-cli wraps it for ocean-freight forwarders: country and HS lookups happen offline (cached reference files), and prospect-fit scoring combines import volume, growth, and your route portfolio into a single number.

## Install

The recommended path installs both the `webprofile-pp-cli` binary and the `pp-webprofile` agent skill in one shot:

```bash
npx -y @mvanhorn/printing-press install webprofile
```

For CLI only (no skill):

```bash
npx -y @mvanhorn/printing-press install webprofile --cli-only
```

### Without Node (Go fallback)

If `npx` isn't available (no Node, offline), install the CLI directly via Go (requires Go 1.26.3 or newer):

```bash
go install github.com/mvanhorn/printing-press-library/library/other/webprofile/cmd/webprofile-pp-cli@latest
```

This installs the CLI only — no skill.

### Pre-built binary

Download a pre-built binary for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/webprofile-current). On macOS, clear the Gatekeeper quarantine: `xattr -d com.apple.quarantine <binary>`. On Unix, mark it executable: `chmod +x <binary>`.

<!-- pp-hermes-install-anchor -->
## Install for Hermes

From the Hermes CLI:

```bash
hermes skills install mvanhorn/printing-press-library/cli-skills/pp-webprofile --force
```

Inside a Hermes chat session:

```bash
/skills install mvanhorn/printing-press-library/cli-skills/pp-webprofile --force
```

## Install for OpenClaw

Tell your OpenClaw agent (copy this):

```
Install the pp-webprofile skill from https://github.com/mvanhorn/printing-press-library/tree/main/cli-skills/pp-webprofile. The skill defines how its required CLI can be installed.
```

## Quick Start

```bash
# Resolve a country name to its Comtrade numeric code (cached after first run).
webprofile-pp-cli country Egypt


# Find the HS code for a commodity description.
webprofile-pp-cli hs-search telephone


# Pull 3 years of Egypt's telephone imports from China.
webprofile-pp-cli who-imports Egypt 8517 --years 3 --json


# Score this country/HS as a prospect (0-100).
webprofile-pp-cli fit-score Egypt 8517 --json

```

## Unique Features

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

## Usage

Run `webprofile-pp-cli --help` for the full command reference and flag list.

## Commands

### reporters

Country reference data (numeric Comtrade code, ISO3, name)

- **`webprofile-pp-cli reporters list`** - Fetch the full Comtrade reporter (country) reference file.

### trade

UN Comtrade trade-flow data

- **`webprofile-pp-cli trade preview`** - Fetch annual trade-flow records (no auth, ~500 records max).


## Output Formats

```bash
# Human-readable table (default in terminal, JSON when piped)
webprofile-pp-cli reporters

# JSON for scripting and agents
webprofile-pp-cli reporters --json

# Filter to specific fields
webprofile-pp-cli reporters --json --select id,name,status

# Dry run — show the request without sending
webprofile-pp-cli reporters --dry-run

# Agent mode — JSON + compact + no prompts in one flag
webprofile-pp-cli reporters --agent
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
npx skills add mvanhorn/printing-press-library/cli-skills/pp-webprofile -g
```

Then invoke `/pp-webprofile <query>` in Claude Code. The skill is the most efficient path — Claude Code drives the CLI directly without an MCP server in the middle.

<details>
<summary>Use as an MCP server in Claude Code (advanced)</summary>

If you'd rather register this CLI as an MCP server in Claude Code, install the MCP binary first:

```bash
go install github.com/mvanhorn/printing-press-library/library/other/webprofile/cmd/webprofile-pp-mcp@latest
```

Then register it:

```bash
claude mcp add webprofile webprofile-pp-mcp
```

</details>

## Use with Claude Desktop

This CLI ships an [MCPB](https://github.com/modelcontextprotocol/mcpb) bundle — Claude Desktop's standard format for one-click MCP extension installs (no JSON config required).

To install:

1. Download the `.mcpb` for your platform from the [latest release](https://github.com/mvanhorn/printing-press-library/releases/tag/webprofile-current).
2. Double-click the `.mcpb` file. Claude Desktop opens and walks you through the install.

Requires Claude Desktop 1.0.0 or later. Pre-built bundles ship for macOS Apple Silicon (`darwin-arm64`) and Windows (`amd64`, `arm64`); for other platforms, use the manual config below.

<details>
<summary>Manual JSON config (advanced)</summary>

If you can't use the MCPB bundle (older Claude Desktop, unsupported platform), install the MCP binary and configure it manually.

```bash
go install github.com/mvanhorn/printing-press-library/library/other/webprofile/cmd/webprofile-pp-mcp@latest
```

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "webprofile": {
      "command": "webprofile-pp-mcp"
    }
  }
}
```

</details>

## Health Check

```bash
webprofile-pp-cli doctor
```

Verifies configuration and connectivity to the API.

## Configuration

Config file: `~/.config/webprofile-pp-cli/config.toml`

## Troubleshooting
**Not found errors (exit code 3)**
- Check the resource ID is correct
- Run the `list` command to see available items

### API-specific

- **Empty result for a country/HS pair** — Comtrade preview is rate-limited and capped at 500 records; reduce --years or check the country/HS via  and .
- **Slow first run** — The first call fetches and caches the Reporters/HS reference files (~1.6 MB combined); subsequent calls are instant.

---

Generated by [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)
