# ocean-pp-cli

`ocean-pp-cli` is a set of script-friendly CLIs for ocean-freight forwarders covering market index, customer profiling, and vessel schedules.

It is built for terminals, shell scripts, CI, and coding agents:

- predictable `--json` and `--select` output on stdout
- human hints and progress on stderr
- offline-first: every CLI keeps a local SQLite snapshot store under `~/.local/share/`
- agent-native: every command supports `--agent` (alias for `--json --compact --no-input --no-color --yes`)
- typed exit codes (`0` ok, `2` usage, `3` not-found, `5` API error, `7` rate-limited, `10` config)
- generated docs and `doctor` health check on every CLI

The CLIs are independent Go modules in this repo. They share a workspace (`go.work`) and pattern, but each can be `go install`'d and used on its own. They communicate through their local SQLite stores; an orchestrator that joins them is out of scope here.

## Install

### Pre-built binaries (recommended)

Download the archive for your OS/arch from the [latest release](https://github.com/boshenzh/ocean-pp-cli/releases/latest) and extract anywhere on `$PATH`. Each release ships three independent tarballs (`freightindex-pp-cli_*`, `schedule-pp-cli_*`, `webprofile-pp-cli_*`) for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`.

```bash
# Example: macOS arm64
VERSION=0.1.0
ARCH=darwin_arm64
for cli in freightindex-pp-cli schedule-pp-cli webprofile-pp-cli; do
  curl -sSL "https://github.com/boshenzh/ocean-pp-cli/releases/download/v${VERSION}/${cli}_${VERSION}_${ARCH}.tar.gz" \
    | tar xz -C /usr/local/bin
done
```

### `go install` (Go 1.26.3+)

Each CLI is a standalone Go module:

```bash
go install github.com/boshenzh/ocean-pp-cli/freightindex-pp-cli/cmd/freightindex-pp-cli@latest
go install github.com/boshenzh/ocean-pp-cli/webprofile-pp-cli/cmd/webprofile-pp-cli@latest
go install github.com/boshenzh/ocean-pp-cli/schedule-pp-cli/cmd/schedule-pp-cli@latest
```

### Build from source

```bash
git clone git@github.com:boshenzh/ocean-pp-cli.git
cd ocean-pp-cli
for m in freightindex-pp-cli schedule-pp-cli webprofile-pp-cli; do
  (cd "$m" && go install ./cmd/...)
done
```

The repo is a Go workspace (`go.work`) registering each CLI as a member module. `go test ./...` from each module's directory runs that module's tests.

### Verify

```bash
freightindex-pp-cli --version
webprofile-pp-cli --version
schedule-pp-cli --version

# Each CLI ships a doctor command that confirms config + connectivity
freightindex-pp-cli doctor
webprofile-pp-cli doctor
schedule-pp-cli doctor
```

## Quick Start

```bash
# Pull the current SCFI weekly index and persist a snapshot.
freightindex-pp-cli pull --json

# Render the index for the lanes you care about.
freightindex-pp-cli digest --lane 'Persian Gulf' --lane 'Mediterranean' --markdown

# Score a candidate prospect by country + HS code.
webprofile-pp-cli fit-score Egypt 8517 --json

# List carriers serving Nansha; pick one and inspect its services.
schedule-pp-cli carriers --start-port-code CNNSA --json
schedule-pp-cli fleet-routes --start-port-code CNNSA --carrier CMA --json

# Pull the latest schedule HTML for the seeded lanes; query the soonest sailing.
schedule-pp-cli pull
schedule-pp-cli next-cls NS-JEDDAH --json
```

The CLIs are independent. Run any one without the others.

## Daily Examples

### freightindex

```bash
# Daily 08:30 cron: pull the latest SCFI and render a Monday-morning digest.
freightindex-pp-cli pull && freightindex-pp-cli digest --markdown

# Filter to a regional desk's lanes (repeatable, case-insensitive substring).
freightindex-pp-cli digest --lane 'Persian Gulf' --lane 'India' --markdown

# Decompose this week's headline-index move into per-lane contributions.
freightindex-pp-cli contribution --json --select 'comprehensive,contributions'

# Lane vs comprehensive index over 8 weeks.
freightindex-pp-cli divergence --route 'Persian Gulf and Red Sea' --weeks 8 --json

# Rank the most volatile lanes over 12 weeks.
freightindex-pp-cli volatility --weeks 12 --top 5 --json

# Lanes SSE didn't publish this week (data-shape sanity check).
freightindex-pp-cli nulls --weeks 4 --json

# Convert all lanes to a single unit so cross-lane comparisons are apples-to-apples.
freightindex-pp-cli normalize --to FEU --json

# Compare an in-house rate sheet vs the latest SCFI per lane (v0.1 stub).
freightindex-pp-cli compare-with-rate-store ./rates.xlsx
```

### webprofile

```bash
# Resolve a country name to its Comtrade numeric code (cached after first run).
webprofile-pp-cli country Egypt --json

# Find the HS code for a commodity description.
webprofile-pp-cli hs-search telephone --json

# Annual import volume for Egypt importing telephones from China, last 3 years.
webprofile-pp-cli who-imports Egypt 8517 --years 3 --json

# Same lane, total imports from any source.
webprofile-pp-cli who-imports Egypt 8517 --partner World --years 3 --json

# Composite 0-100 prospect-fit score (China-share + YoY growth + route coverage).
webprofile-pp-cli fit-score Egypt 8517 --json

# Compact agent-shaped score with the three components that drove the number.
webprofile-pp-cli fit-score IND 6109 --json --select 'fit_score,import_from_cn,yoy_growth_pct'
```

### schedule

The schedule CLI has two layers — public APIs (no auth) and a per-lane registry (one-time browser capture per lane).

```bash
# Public API layer: ports, carriers, services. No registration needed.
schedule-pp-cli ports --json
schedule-pp-cli carriers --start-port-code CNNSA --json
schedule-pp-cli fleet-routes --start-port-code CNNSA --carrier MAERSK --json
schedule-pp-cli carrier-services --port-code CNNSA --carrier-code CMA --json

# Registry layer: starts empty. Run `routes seed` to load 7 example
# South-China → Red Sea / Mideast / India-Pak lanes, or skip and use
# `routes add` to register your own.
schedule-pp-cli routes seed
schedule-pp-cli routes list --json

# Add a new lane after capturing the routePort URL once in your browser.
schedule-pp-cli routes add SK-DAMMAM 'https://www.weiyun001.com/routePort?st=...' --pol SHEKOU --pod DAMMAM

# Refresh schedule data for one or all registered lanes.
schedule-pp-cli pull --route NS-JEDDAH
schedule-pp-cli pull

# Soonest future sailing on a lane (vessel + voyage + ETD + capacity + flag).
schedule-pp-cli next-cls NS-JEDDAH --json

# All carriers' sailings on a lane in chronological order.
schedule-pp-cli compare-carriers NS-JEDDAH --json

# Filter by carrier whitelist.
schedule-pp-cli query NS-JEDDAH --carrier PIL --carrier COSCO --json
```

To register a new lane, search it once at `https://www.weiyun001.com` in your browser, copy the resulting `/routePort?...` URL from the address bar, and feed it to `routes add`. URLs are stable per (POL, POD) pair — register once, use forever.

## Output and Automation

Every CLI honors the same flags:

```bash
# JSON for scripts and agents.
freightindex-pp-cli digest --json

# Comma-separated field selection (works on JSON output).
schedule-pp-cli compare-carriers NS-JEDDAH --json --select 'name,vessels_by_etd'

# Compact mode keeps only key fields (id, name, status, timestamps).
webprofile-pp-cli fit-score Egypt 8517 --json --compact

# Agent mode flips on every agent-friendly default at once.
schedule-pp-cli pull --agent

# Dry-run shows what would happen without making the API call.
freightindex-pp-cli pull --dry-run

# Markdown for human-readable digests.
freightindex-pp-cli digest --markdown
```

Exit codes are typed:

| Code | Meaning |
|---|---|
| `0` | success |
| `2` | usage error (bad flag, missing required arg) |
| `3` | not found (record / route / route-name) |
| `5` | API error (upstream returned non-2xx) |
| `7` | rate-limited (Comtrade preview tier exhausted, SCFI 429) |
| `10` | config error (bad TOML, missing env var) |

Run `<cli> doctor` on any CLI to verify upstream reachability and local store health:

```bash
freightindex-pp-cli doctor
webprofile-pp-cli doctor
schedule-pp-cli doctor
```

## Auth and Accounts

| CLI | Upstream | Auth |
|---|---|---|
| `freightindex-pp-cli` | SCFI (`en.sse.net.cn/currentIndex?indexName=scfi`) | None — public weekly JSON. |
| `webprofile-pp-cli` | UN Comtrade public preview (`comtradeapi.un.org/public/v1/preview`) | None — rate-limited free tier (~500 records/call). For higher-volume work, request a Comtrade API key and the CLI will use it via `COMTRADE_API_KEY`. |
| `schedule-pp-cli` | Weiyun (`wywapi.weiyun001.com/api/`, `www.weiyun001.com/routePort`) | None for the API layer. The Registry layer reads `/routePort` HTML; URLs are captured by the user in their browser (one-time per lane). |

No CLI requires a long-lived account. The schedule registry stores routePort URLs in `~/.config/schedule-pp-cli/routes.json` (path-only, no cookies).

## Services

What each CLI covers:

### `freightindex-pp-cli`

Shanghai Containerized Freight Index (SCFI), the de-facto benchmark for China-origin spot rates. Pulls the weekly index, persists snapshots, decomposes the headline move into per-lane contributions, ranks lanes by realized volatility, and converts mixed-unit rows to a single unit (USD/TEU or USD/FEU).

Source: `https://en.sse.net.cn/indices/scfinew.jsp` (16 lanes including Comprehensive Index, Mediterranean, Persian Gulf/Red Sea, USWC, USEC, Europe, Australia, South America, Africa, Japan, Korea).

### `webprofile-pp-cli`

UN Comtrade trade-flow data for prospect research. Resolves country names + HS commodity codes to Comtrade numeric codes (offline, after first fetch), pulls annual import volumes per (country, partner, HS), and computes a composite 0-100 fit score combining import volume, YoY growth, and route coverage.

Source: `https://comtradeapi.un.org/public/v1/preview/C/A/HS` plus the Reporters/H6 reference files. Free tier; ~500 records per call; first run downloads the ~1.6 MB H6 reference and caches.

### `schedule-pp-cli`

Container shipping schedule intelligence. The API layer wraps Weiyun's four public endpoints (port catalog, carriers per port, fleet routes per carrier, continent-grouped service map) — useful for prospecting and lane discovery. The Registry layer adds per-week sailing data (vessel name, voyage, ETD, capacity, flag) for lanes the user has registered, scraped from `/routePort` HTML.

Source: `https://wywapi.weiyun001.com/api/` (port/carrier APIs) and `https://www.weiyun001.com/routePort?...` (per-lane HTML). The registry starts empty; run `routes seed` to load 7 bundled example lanes (NANSHA/SHENZHEN → JEDDAH, SOKHNA, KARACHI, JEBEL ALI, NHAVA SHEVA, DJIBOUTI), or use `routes add` to register your own from a Weiyun browser session.

### Roadmap

`bldata-pp-cli` (52WMB bills-of-lading mining) — pending the user's 7-day trial registration. Same pattern as `schedule`: probe for public APIs first, fall back to URL registry + HTML scrape.

## For Agents

Each CLI ships three layers of agent support — pick whichever fits your host.

### Layer 1: `--agent` flag (lowest)

Every command accepts `--agent`, which sets `--json --compact --no-input --no-color --yes`. An agent that can spawn shell commands needs nothing else:

```bash
webprofile-pp-cli fit-score Egypt 8517 --agent
schedule-pp-cli next-cls NS-JEDDAH --agent
freightindex-pp-cli digest --lane 'Persian Gulf' --agent
```

### Layer 2: SKILL.md (markdown)

Each module ships a `SKILL.md` with trigger phrases, command recipes, output schemas, and error codes. Hosts that read markdown skills (Claude Code, OpenClaw, others) can be pointed at this repo as a plugin:

```bash
# OpenClaw — install from GitHub
openclaw plugins install --marketplace boshenzh/ocean-pp-cli
# equivalent: openclaw plugins install --marketplace https://github.com/boshenzh/ocean-pp-cli

# OpenClaw — install from a local clone (development)
openclaw plugins install -l ~/Projects/ocean-pp-cli   # -l symlinks instead of copying

# Claude Code — load skills from a directory
claude --plugin-dir ~/Projects/ocean-pp-cli
```

After install, OpenClaw warns if the plugin isn't on the trust list:
`plugins.allow is empty; discovered non-bundled plugins may auto-load: ocean-pp-cli`. Add it explicitly so loads are intentional, not incidental:

```bash
openclaw config set plugins.allow '["ocean-pp-cli"]'
# Config lives at ~/.openclaw/openclaw.json — see `openclaw config file`.
```

The repo root carries `.claude-plugin/plugin.json` (Claude-compatible) and `skills/pp-{freightindex,schedule,webprofile}/SKILL.md` for plugin auto-discovery. The positional form of `openclaw plugins install <arg>` only accepts paths/archives/npm specs/marketplace names — git URLs must go through `--marketplace`.

### Layer 3: MCP servers (typed RPC)

Each CLI ships a paired `<cli>-pp-mcp` binary that exposes the same commands as MCP tools over stdio:

```bash
go install github.com/boshenzh/ocean-pp-cli/freightindex-pp-cli/cmd/freightindex-pp-mcp@latest
go install github.com/boshenzh/ocean-pp-cli/schedule-pp-cli/cmd/schedule-pp-mcp@latest
go install github.com/boshenzh/ocean-pp-cli/webprofile-pp-cli/cmd/webprofile-pp-mcp@latest

# Register with Claude Code
claude mcp add freightindex-pp-mcp -- freightindex-pp-mcp
claude mcp add schedule-pp-mcp     -- schedule-pp-mcp
claude mcp add webprofile-pp-mcp   -- webprofile-pp-mcp
```

## Documentation

Each CLI ships its own README and SKILL.md inside its module directory:

```
freightindex-pp-cli/README.md   freightindex-pp-cli/SKILL.md
webprofile-pp-cli/README.md     webprofile-pp-cli/SKILL.md
schedule-pp-cli/README.md       schedule-pp-cli/SKILL.md
```

Per-run research artifacts (briefs, absorb manifests, scorecard reports, dogfood acceptance markers) are archived under `docs/runs/<cli>-<run-id>/`:

```
docs/runs/freightindex-20260508-151751/
docs/runs/webprofile-20260508-171333/
docs/runs/schedule-20260509-001818/
```

These are the audit trail of how each CLI was generated by [`cli-printing-press`](https://github.com/mvanhorn/cli-printing-press) — useful when you regenerate a CLI months later and want to see the original briefing decisions.

## Development

```bash
# From repo root: build everything in the workspace.
go build ./...

# Test a single module (run from inside the module's directory).
cd freightindex-pp-cli && go test ./...

# Regenerate a CLI from its captured spec (requires printing-press v4.0.5+).
cd $HOME/printing-press/.runstate/<scope>/runs/<run-id>
printing-press generate --spec research/<api>-spec.yaml --output working/<api>-pp-cli --force --lenient

# Verify a regenerated CLI before promoting back to the project.
printing-press shipcheck --dir working/<api>-pp-cli --spec research/<api>-spec.yaml
printing-press dogfood --live --dir working/<api>-pp-cli --level full
```

The hand-written transcendence packages (e.g. `internal/scfi`, `internal/comtrade`, `internal/registry`, `internal/scrape`) live alongside the generator output and survive regeneration. Spec-derived files (`internal/cli/promoted_*.go`, `internal/client/`, `internal/store/`) are overwritten on regen — do not hand-edit them.

Inputs the printing-press skill consumed during generation are checked in under `docs/briefings/<cli>.md`. They are reference notes, not part of the generator contract.

`inputs/` is reserved for capture artifacts that must never be committed (HAR files, cookie exports). The `.gitignore` excludes `inputs/**/*.har` and `inputs/**/*.har.zip` by default.

## Credits

Generated by [`cli-printing-press`](https://github.com/mvanhorn/cli-printing-press) v4.0.5. The README structure follows [`steipete/gogcli`](https://github.com/steipete/gogcli)'s convention.

## License

Apache-2.0. Each CLI ships its own `LICENSE` file inside the module directory.
