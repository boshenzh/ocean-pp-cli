// Hand-written transcendence commands for freightindex-pp-cli.
// These read the local SQLite snapshot store populated by `pull`.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/boshenzh/ocean-pp-cli/freightindex-pp-cli/internal/scfi"
	"github.com/boshenzh/ocean-pp-cli/freightindex-pp-cli/internal/store"

	"github.com/spf13/cobra"
)

const snapshotResourceType = "scfi_snapshot"

// ---------- shared helpers ----------

func openSnapshotStore(ctx context.Context) (*store.Store, error) {
	return store.OpenWithContext(ctx, defaultDBPath("freightindex-pp-cli"))
}

// loadSnapshots returns all stored snapshots, newest first by currentDate.
// limit <= 0 means "all".
func loadSnapshots(ctx context.Context, limit int) ([]*scfi.Snapshot, error) {
	db, err := openSnapshotStore(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.List(snapshotResourceType, 0)
	if err != nil {
		return nil, fmt.Errorf("listing snapshots: %w", err)
	}
	out := make([]*scfi.Snapshot, 0, len(rows))
	for _, raw := range rows {
		snap, err := scfi.ParseSnapshot(raw)
		if err != nil {
			continue
		}
		out = append(out, snap)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CurrentDate > out[j].CurrentDate })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func emit(cmd *cobra.Command, flags *rootFlags, v any) error {
	if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv) {
		return printJSONFiltered(cmd.OutOrStdout(), v, flags)
	}
	if flags.csv {
		return printJSONFiltered(cmd.OutOrStdout(), v, flags)
	}
	// Default: pretty-print as JSON with 2-space indent for human readability.
	return printJSONFiltered(cmd.OutOrStdout(), v, flags)
}

// ---------- pull ----------

func newPullCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Fetch the current weekly SCFI and persist a snapshot to the local store.",
		Long: `Fetch the current weekly SCFI from en.sse.net.cn and append a snapshot to the
local SQLite store keyed by the publication date. Idempotent: re-running between
weekly publications is a no-op (the snapshot for the current week is upserted).

This is the daily 08:30 polling command — wire it into cron and let history
build up over time.`,
		Example: "  freightindex-pp-cli pull --json",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			data, _, err := resolveRead(cmd.Context(), c, flags, "scfi", false, "/currentIndex", map[string]string{"indexName": "scfi"}, nil)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			data = extractResponseData(data)
			snap, err := scfi.ParseSnapshot(data)
			if err != nil {
				return fmt.Errorf("parse snapshot: %w", err)
			}
			db, err := openSnapshotStore(cmd.Context())
			if err != nil {
				return err
			}
			defer db.Close()
			if err := db.Upsert(snapshotResourceType, snap.CurrentDate, data); err != nil {
				return fmt.Errorf("upsert snapshot: %w", err)
			}
			return emit(cmd, flags, map[string]any{
				"persisted":     true,
				"week":          snap.CurrentDate,
				"previous_week": snap.LastDate,
				"lanes":         len(snap.Lines),
			})
		},
	}
	return cmd
}

// ---------- digest ----------

func newDigestCmd(flags *rootFlags) *cobra.Command {
	var asMarkdown bool
	var lanes []string
	cmd := &cobra.Command{
		Use:   "digest",
		Short: "Render the latest weekly snapshot as JSON or Markdown; pass --lane to filter.",
		Long: `Read the most recent snapshot from the local store and render it. Without
--lane, the digest covers every lane SSE published this week (16 rows by default).
Pass --lane <fragment> (repeatable, case-insensitive substring) to filter — e.g.
--lane 'Persian Gulf' --lane 'Mediterranean' for a forwarder focused on the
Middle East and Med, or --lane 'USWC' --lane 'USEC' for a US-bound desk.`,
		Example: `  # Full SCFI table for the latest week
  freightindex-pp-cli digest --markdown

  # Filtered digest for a Middle East / India desk
  freightindex-pp-cli digest --lane 'Persian Gulf' --lane 'India' --markdown

  # JSON for orchestrators
  freightindex-pp-cli digest --json`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			snaps, err := loadSnapshots(cmd.Context(), 1)
			if err != nil {
				return err
			}
			if len(snaps) == 0 {
				return fmt.Errorf("no snapshots in local store; run `freightindex-pp-cli pull` first")
			}
			latest := snaps[0]
			rows := []map[string]any{}
			for _, l := range latest.Lines {
				if !l.MatchesAnyLane(lanes) && !l.IsComprehensive() {
					continue
				}
				row := map[string]any{
					"lane":      l.LineName,
					"unit":      l.Unit,
					"weighting": l.Weighting,
					"current":   l.Current,
					"last":      l.Last,
					"absolute":  l.Absolute,
					"percent":   l.Percent,
				}
				rows = append(rows, row)
			}
			// --json wins over --markdown when both are set (agents expect JSON).
			if asMarkdown && !flags.asJSON {
				return renderDigestMarkdown(cmd, latest, rows, lanes)
			}
			return emit(cmd, flags, map[string]any{
				"week":   latest.CurrentDate,
				"prior":  latest.LastDate,
				"filter": lanes,
				"lanes":  rows,
			})
		},
	}
	cmd.Flags().BoolVar(&asMarkdown, "markdown", false, "Render the digest as a Markdown table")
	cmd.Flags().StringArrayVar(&lanes, "lane", nil, "Filter to lanes whose name contains this fragment (repeatable, case-insensitive)")
	return cmd
}

func renderDigestMarkdown(cmd *cobra.Command, snap *scfi.Snapshot, rows []map[string]any, lanes []string) error {
	var b strings.Builder
	if len(lanes) == 0 {
		fmt.Fprintf(&b, "# SCFI Digest — %s (vs %s)\n\n", snap.CurrentDate, snap.LastDate)
	} else {
		fmt.Fprintf(&b, "# SCFI Digest — %s (vs %s) — filtered to %s\n\n", snap.CurrentDate, snap.LastDate, strings.Join(lanes, ", "))
	}
	fmt.Fprintln(&b, "| Lane | Unit | Weighting | Current | Last | % WoW |")
	fmt.Fprintln(&b, "|---|---|---|---:|---:|---:|")
	for _, r := range rows {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n",
			r["lane"], r["unit"], r["weighting"],
			fmtNum(r["current"]), fmtNum(r["last"]), fmtPct(r["percent"]))
	}
	_, err := fmt.Fprint(cmd.OutOrStdout(), b.String())
	return err
}

func fmtNum(v any) string {
	if v == nil {
		return "—"
	}
	if f, ok := v.(*float64); ok {
		if f == nil {
			return "—"
		}
		return fmt.Sprintf("%.2f", *f)
	}
	if f, ok := v.(float64); ok {
		return fmt.Sprintf("%.2f", f)
	}
	return fmt.Sprintf("%v", v)
}

func fmtPct(v any) string {
	if v == nil {
		return "—"
	}
	if f, ok := v.(*float64); ok {
		if f == nil {
			return "—"
		}
		return fmt.Sprintf("%+.2f%%", *f)
	}
	if f, ok := v.(float64); ok {
		return fmt.Sprintf("%+.2f%%", f)
	}
	return fmt.Sprintf("%v", v)
}

// ---------- contribution ----------

func newContributionCmd(flags *rootFlags) *cobra.Command {
	var week string
	cmd := &cobra.Command{
		Use:   "contribution",
		Short: "Decompose this week's headline SCFI move into per-lane contributions (weighting% × percent change).",
		Example: "  freightindex-pp-cli contribution --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			snaps, err := loadSnapshots(cmd.Context(), 0)
			if err != nil {
				return err
			}
			if len(snaps) == 0 {
				return fmt.Errorf("no snapshots; run pull first")
			}
			var snap *scfi.Snapshot
			if week == "" {
				snap = snaps[0]
			} else {
				for _, s := range snaps {
					if s.CurrentDate == week {
						snap = s
						break
					}
				}
				if snap == nil {
					return fmt.Errorf("no snapshot found for week %s", week)
				}
			}
			rows := []map[string]any{}
			var headlinePct float64
			for _, l := range snap.Lines {
				if l.IsComprehensive() {
					if l.Percent != nil {
						headlinePct = *l.Percent
					}
					continue
				}
				contrib := l.Contribution()
				rows = append(rows, map[string]any{
					"lane":         l.LineName,
					"weighting":    l.Weighting,
					"percent":      l.Percent,
					"contribution": contrib,
				})
			}
			sort.Slice(rows, func(i, j int) bool {
				return math.Abs(rows[i]["contribution"].(float64)) > math.Abs(rows[j]["contribution"].(float64))
			})
			return emit(cmd, flags, map[string]any{
				"week":             snap.CurrentDate,
				"prior":            snap.LastDate,
				"comprehensive":    headlinePct,
				"contributions":    rows,
			})
		},
	}
	cmd.Flags().StringVar(&week, "week", "", "Specific week (YYYY-MM-DD); defaults to the most recent snapshot")
	return cmd
}

// ---------- divergence ----------

func newDivergenceCmd(flags *rootFlags) *cobra.Command {
	var route string
	var weeks int
	cmd := &cobra.Command{
		Use:   "divergence",
		Short: "Per-week spread between a lane and the comprehensive index over N weeks, with cumulative drift.",
		Example: "  freightindex-pp-cli divergence --route 'Persian Gulf and Red Sea' --weeks 8 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if route == "" {
				return fmt.Errorf("--route is required")
			}
			snaps, err := loadSnapshots(cmd.Context(), weeks)
			if err != nil {
				return err
			}
			if len(snaps) == 0 {
				return fmt.Errorf("no snapshots; run pull first")
			}
			weeksOut := []map[string]any{}
			cumulative := 0.0
			// Iterate from oldest to newest so cumulative drift is a forward sum.
			for i := len(snaps) - 1; i >= 0; i-- {
				snap := snaps[i]
				var lanePct, headlinePct *float64
				for j := range snap.Lines {
					l := snap.Lines[j]
					if l.IsComprehensive() {
						headlinePct = l.Percent
					} else if l.MatchesRoute(route) && lanePct == nil {
						lanePct = l.Percent
					}
				}
				rec := map[string]any{
					"week":          snap.CurrentDate,
					"lane_pct":      lanePct,
					"headline_pct":  headlinePct,
				}
				if lanePct != nil && headlinePct != nil {
					spread := *lanePct - *headlinePct
					cumulative += spread
					rec["spread"] = spread
					rec["cumulative_drift"] = cumulative
				} else {
					rec["spread"] = nil
					rec["cumulative_drift"] = cumulative
				}
				weeksOut = append(weeksOut, rec)
			}
			return emit(cmd, flags, map[string]any{
				"route":         route,
				"weeks":         weeksOut,
				"final_drift":   cumulative,
			})
		},
	}
	cmd.Flags().StringVar(&route, "route", "", "Lane-name fragment (case-insensitive substring), e.g. 'Persian Gulf' or 'Mediterranean'")
	cmd.Flags().IntVar(&weeks, "weeks", 8, "How many recent weeks to compare; uses min(weeks, snapshots-on-hand)")
	return cmd
}

// ---------- nulls ----------

func newNullsCmd(flags *rootFlags) *cobra.Command {
	var weeks int
	cmd := &cobra.Command{
		Use:   "nulls",
		Short: "List weeks where SSE returned null for one or more lanes (a service quirk that breaks naive scripts).",
		Example: "  freightindex-pp-cli nulls --weeks 8 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			snaps, err := loadSnapshots(cmd.Context(), weeks)
			if err != nil {
				return err
			}
			out := []map[string]any{}
			for _, s := range snaps {
				var nulls []string
				for _, l := range s.Lines {
					if l.Current == nil {
						nulls = append(nulls, l.LineName)
					}
				}
				if len(nulls) == 0 {
					continue
				}
				out = append(out, map[string]any{
					"week":        s.CurrentDate,
					"null_lanes":  nulls,
					"null_count":  len(nulls),
				})
			}
			return emit(cmd, flags, map[string]any{
				"weeks_inspected": len(snaps),
				"weeks_with_nulls": out,
			})
		},
	}
	cmd.Flags().IntVar(&weeks, "weeks", 8, "How many recent weeks to inspect")
	return cmd
}

// ---------- volatility ----------

func newVolatilityCmd(flags *rootFlags) *cobra.Command {
	var weeks int
	var top int
	cmd := &cobra.Command{
		Use:   "volatility",
		Short: "Per-lane mean/stdev of weekly percent change over N weeks; ranks the most volatile lanes.",
		Example: "  freightindex-pp-cli volatility --weeks 12 --top 5 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			snaps, err := loadSnapshots(cmd.Context(), weeks)
			if err != nil {
				return err
			}
			if len(snaps) < 2 {
				// Return exit 0 with a clear "insufficient_data" payload — volatility is
				// inherently a multi-week stat. Agents can detect the empty `ranked`
				// list and fall back to a single-week view.
				return emit(cmd, flags, map[string]any{
					"weeks":            len(snaps),
					"ranked":           []any{},
					"status":           "insufficient_data",
					"reason":           "volatility requires at least 2 weekly snapshots",
					"hint":             "run `freightindex-pp-cli pull` weekly so history accumulates",
				})
			}
			// Aggregate per-lane percent changes across snapshots.
			byLane := map[string][]float64{}
			for _, s := range snaps {
				for _, l := range s.Lines {
					if l.Percent == nil || l.IsComprehensive() {
						continue
					}
					byLane[l.LineName] = append(byLane[l.LineName], *l.Percent)
				}
			}
			rows := []map[string]any{}
			for lane, samples := range byLane {
				if len(samples) < 2 {
					continue
				}
				mean, stdev, mn, mx := stats(samples)
				rows = append(rows, map[string]any{
					"lane":     lane,
					"n":        len(samples),
					"mean_pct": mean,
					"stdev":    stdev,
					"min":      mn,
					"max":      mx,
				})
			}
			sort.Slice(rows, func(i, j int) bool {
				return rows[i]["stdev"].(float64) > rows[j]["stdev"].(float64)
			})
			if top > 0 && len(rows) > top {
				rows = rows[:top]
			}
			return emit(cmd, flags, map[string]any{
				"weeks":   len(snaps),
				"ranked":  rows,
			})
		},
	}
	cmd.Flags().IntVar(&weeks, "weeks", 12, "How many recent weeks to include")
	cmd.Flags().IntVar(&top, "top", 5, "Return only the top K most volatile lanes (0 = all)")
	return cmd
}

func stats(xs []float64) (mean, stdev, min, max float64) {
	if len(xs) == 0 {
		return 0, 0, 0, 0
	}
	min = xs[0]
	max = xs[0]
	var sum float64
	for _, x := range xs {
		sum += x
		if x < min {
			min = x
		}
		if x > max {
			max = x
		}
	}
	mean = sum / float64(len(xs))
	var sq float64
	for _, x := range xs {
		d := x - mean
		sq += d * d
	}
	stdev = math.Sqrt(sq / float64(len(xs)))
	return
}

// ---------- normalize ----------

func newNormalizeCmd(flags *rootFlags) *cobra.Command {
	var to string
	cmd := &cobra.Command{
		Use:   "normalize",
		Short: "Convert every lane to a single unit (USD/TEU or USD/FEU) so cross-lane comparisons are comparable.",
		Example: "  freightindex-pp-cli normalize --to FEU --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			target := strings.ToUpper(strings.TrimSpace(to))
			if target != "TEU" && target != "FEU" {
				return fmt.Errorf("--to must be TEU or FEU (got %q)", to)
			}
			snaps, err := loadSnapshots(cmd.Context(), 1)
			if err != nil {
				return err
			}
			if len(snaps) == 0 {
				return fmt.Errorf("no snapshots; run pull first")
			}
			snap := snaps[0]
			rows := []map[string]any{}
			for _, l := range snap.Lines {
				if l.IsComprehensive() {
					continue
				}
				var val float64
				var ok bool
				if target == "FEU" {
					val, ok = l.NormalizeToFEU()
				} else {
					val, ok = l.NormalizeToTEU()
				}
				rec := map[string]any{
					"lane":          l.LineName,
					"native_unit":   l.Unit,
					"native_value":  l.Current,
					"normalized":    nil,
				}
				if ok {
					rec["normalized"] = val
				}
				rows = append(rows, rec)
			}
			return emit(cmd, flags, map[string]any{
				"week":   snap.CurrentDate,
				"to":     "USD/" + target,
				"lanes":  rows,
			})
		},
	}
	cmd.Flags().StringVar(&to, "to", "FEU", "Target unit: TEU or FEU")
	return cmd
}

// ---------- compare-with-rate-store (stub) ----------

func newCompareWithRateStoreCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare-with-rate-store [rates.xlsx]",
		Short: "Cross-reference your in-house rate sheet against the latest SCFI per lane (v0.1 stub).",
		Long: `STUB IN v0.1. The full implementation reads an xlsx workbook, joins each
row by lane name against the latest SCFI snapshot, and tags each row High /
Low / Par with the USD/TEU gap. The xlsx reader (rate-store) lands in plan #2.

For now this command parses its argument, persists nothing, and prints a
clear "not yet wired" payload so orchestrators can detect the gap.`,
		Example: "  freightindex-pp-cli compare-with-rate-store ./rates.xlsx",
		Annotations: map[string]string{"mcp:read-only": "true", "pp:typed-exit-codes": "0"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			path := ""
			if len(args) > 0 {
				path = args[0]
			}
			payload := map[string]any{
				"status":        "stub",
				"path":          path,
				"reason":        "rate-store reader not yet wired",
				"next_plan":     "plan #2 ships the xlsx reader; this command will then tag each lane High / Low / Par with USD/TEU gap vs the latest SCFI snapshot.",
				"hint":          "until then, render the latest snapshot via `freightindex-pp-cli priority-digest --markdown` and compare manually.",
			}
			return emit(cmd, flags, payload)
		},
	}
	return cmd
}

// ---------- registrar ----------

// registerNovelCommands wires the hand-written transcendence commands into the
// root command. Called from root.go.
func registerNovelCommands(rootCmd *cobra.Command, flags *rootFlags) {
	rootCmd.AddCommand(newPullCmd(flags))
	rootCmd.AddCommand(newDigestCmd(flags))
	rootCmd.AddCommand(newContributionCmd(flags))
	rootCmd.AddCommand(newDivergenceCmd(flags))
	rootCmd.AddCommand(newNullsCmd(flags))
	rootCmd.AddCommand(newVolatilityCmd(flags))
	rootCmd.AddCommand(newNormalizeCmd(flags))
	rootCmd.AddCommand(newCompareWithRateStoreCmd(flags))
}

// keep encoding/json reachable for potential future use
var _ = json.Unmarshal
