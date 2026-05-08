// Hand-written transcendence commands for schedule-pp-cli.
// Combines the public Weiyun APIs (already wired by the generator under
// `ports`, `carriers`, `fleet_routes`, `carrier_services`) with a local
// route registry + HTML scraper for per-week sailing data.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/boshenzh/ocean-pp-cli/schedule-pp-cli/internal/registry"
	"github.com/boshenzh/ocean-pp-cli/schedule-pp-cli/internal/scrape"
	"github.com/boshenzh/ocean-pp-cli/schedule-pp-cli/internal/store"

	"github.com/spf13/cobra"
)

const (
	resourceLane = "schedule_lane"
	uaHeader     = "github.com/boshenzh/ocean-pp-cli/schedule-pp-cli/0.1 (+ocean-pp-cli)"
)

func openLaneStore(ctx context.Context) (*store.Store, error) {
	return store.OpenWithContext(ctx, defaultDBPath("schedule-pp-cli"))
}

// ---------- routes group ----------

func newRoutesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes",
		Short: "Manage the local route registry (alias -> Weiyun routePort URL).",
		Long: `The registry maps your common lanes (e.g. NS-JEDDAH) to the routePort URL
that Weiyun returns when you search the lane in your browser. Once a lane is
registered, all schedule commands accept the alias instead of the encrypted URL.

The registry is seeded with 7 representative lanes on first use; add more
with 'routes add' as you discover them.`,
	}
	cmd.AddCommand(newRoutesListCmd(flags))
	cmd.AddCommand(newRoutesAddCmd(flags))
	cmd.AddCommand(newRoutesRemoveCmd(flags))
	cmd.AddCommand(newRoutesPathCmd(flags))
	return cmd
}

func newRoutesListCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all registered lanes.",
		Example: "  schedule-pp-cli routes list --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			s, err := registry.Open("")
			if err != nil {
				return err
			}
			if added := s.Seed(); added > 0 {
				_ = s.Save()
			}
			rows := s.List()
			out := make([]map[string]any, 0, len(rows))
			for _, r := range rows {
				out = append(out, map[string]any{"name": r.Name, "pol": r.POL, "pod": r.POD, "url": r.URL, "note": r.Note})
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"path": s.Path(), "count": len(rows), "routes": out}, flags)
		},
	}
	return cmd
}

func newRoutesAddCmd(flags *rootFlags) *cobra.Command {
	var pol, pod, note string
	cmd := &cobra.Command{
		Use:   "add [name] [url]",
		Short: "Add or replace a lane in the registry.",
		Long: `Capture a routePort URL from your Chrome address bar and store it under a
short alias. URLs are stable per (POL, POD) pair, so you only need to do this
once per lane.`,
		Example: `  schedule-pp-cli routes add SK-DJIBOUTI 'https://www.weiyun001.com/routePort?st=...&de=...' --pol SHENZHEN --pod DJIBOUTI`,
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return fmt.Errorf("usage: routes add <alias> <url>")
			}
			if dryRunOK(flags) {
				return nil
			}
			s, err := registry.Open("")
			if err != nil {
				return err
			}
			r := registry.Route{Name: args[0], URL: args[1], POL: pol, POD: pod, Note: note}
			if err := s.Add(r); err != nil {
				return err
			}
			if err := s.Save(); err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"added": r.Name, "path": s.Path()}, flags)
		},
	}
	cmd.Flags().StringVar(&pol, "pol", "", "Origin port (free text, e.g. NANSHA)")
	cmd.Flags().StringVar(&pod, "pod", "", "Destination port (free text, e.g. JEDDAH)")
	cmd.Flags().StringVar(&note, "note", "", "Free-text note about this lane")
	return cmd
}

func newRoutesRemoveCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [name]",
		Short: "Remove a lane from the registry.",
		Example: "  schedule-pp-cli routes remove NS-DJIBOUTI",
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			s, err := registry.Open("")
			if err != nil {
				return err
			}
			if !s.Remove(args[0]) {
				return fmt.Errorf("no route named %q", args[0])
			}
			if err := s.Save(); err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"removed": args[0]}, flags)
		},
	}
	return cmd
}

func newRoutesPathCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the on-disk path of the route registry.",
		Example: "  schedule-pp-cli routes path",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			path := registry.DefaultPath()
			if flags.asJSON {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"path": path}, flags)
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}

// ---------- pull ----------

func newPullCmd(flags *rootFlags) *cobra.Command {
	var route string
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Fetch the latest schedule HTML for one or all registered lanes and persist to local SQLite.",
		Long: `Iterates registered lanes (or just the one named via --route), GETs each
routePort URL with no auth, scrapes the embedded vessel + carrier records, and
upserts each lane's snapshot into the local store keyed by alias.`,
		Example: `  schedule-pp-cli pull
  schedule-pp-cli pull --route NS-JEDDAH --json`,
		Annotations: map[string]string{"mcp:read-only": "false"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			s, err := registry.Open("")
			if err != nil {
				return err
			}
			if added := s.Seed(); added > 0 {
				_ = s.Save()
			}
			targets := s.List()
			if route != "" {
				r := s.Get(route)
				if r == nil {
					return routeNotInRegistryError(route, s)
				}
				targets = []registry.Route{*r}
			}
			db, err := openLaneStore(cmd.Context())
			if err != nil {
				return err
			}
			defer db.Close()

			results := make([]map[string]any, 0, len(targets))
			for _, r := range targets {
				pd, err := fetchAndScrape(cmd.Context(), r.URL)
				rec := map[string]any{"name": r.Name, "pol": r.POL, "pod": r.POD}
				if err != nil {
					rec["error"] = err.Error()
					results = append(results, rec)
					continue
				}
				rec["vessels"] = len(pd.Vessels)
				rec["carrier_services"] = len(pd.CarrierServices)
				rec["fetched_at"] = pd.FetchedAt
				blob, _ := json.Marshal(map[string]any{
					"name":             r.Name,
					"pol":              r.POL,
					"pod":              r.POD,
					"url":              r.URL,
					"fetched_at":       pd.FetchedAt,
					"vessels":          pd.Vessels,
					"carrier_services": pd.CarrierServices,
				})
				if err := db.Upsert(resourceLane, r.Name, blob); err != nil {
					rec["store_error"] = err.Error()
				}
				results = append(results, rec)
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"pulled": len(targets), "results": results}, flags)
		},
	}
	cmd.Flags().StringVar(&route, "route", "", "Pull only this registered lane (default: all)")
	return cmd
}

func fetchAndScrape(ctx context.Context, url string) (*scrape.PageData, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", uaHeader)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return scrape.Parse(body, url)
}

func routeNotInRegistryError(name string, s *registry.Store) error {
	return fmt.Errorf(`route %q not in registry.

To add it (one-time, ~30 seconds):
  1. Open https://www.weiyun001.com in Chrome (logged in)
  2. Search the lane (e.g. NANSHA -> JEDDAH)
  3. Copy the URL from the address bar
  4. Run:
     schedule-pp-cli routes add %s '<paste URL here>' --pol POL --pod POD

Registry: %s`, name, name, s.Path())
}

// ---------- query ----------

func newQueryCmd(flags *rootFlags) *cobra.Command {
	var carrierFilter []string
	var nextDays int
	cmd := &cobra.Command{
		Use:   "query [route]",
		Short: "Read the most recent scraped schedule for a registered lane from the local store.",
		Example: `  schedule-pp-cli query NS-JEDDAH --json
  schedule-pp-cli query NS-JEDDAH --carrier PIL --carrier COSCO`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			lane, err := loadLane(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			result := map[string]any{
				"name":             lane.Name,
				"pol":              lane.POL,
				"pod":              lane.POD,
				"fetched_at":       lane.FetchedAt,
				"vessels":          filterByDays(lane.Vessels, nextDays),
				"carrier_services": scrape.FilterCarriers(lane.CarrierServices, carrierFilter),
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringArrayVar(&carrierFilter, "carrier", nil, "Filter carrier-services by carrier code (repeatable)")
	cmd.Flags().IntVar(&nextDays, "next-days", 0, "Only show vessels with ETD within N days from today (0 = all)")
	return cmd
}

// ---------- next-cls ----------

func newNextClsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "next-cls [route]",
		Short: "Soonest sailing on the lane that hasn't departed yet.",
		Long: `Reads the local store for the named lane and returns the vessel with the
nearest future ETD. The Weiyun page lists per-vessel ETD as carrierETDFormatter
(e.g. "2026/05/15(周五)") which we parse and compare against today.`,
		Example: "  schedule-pp-cli next-cls NS-JEDDAH --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			lane, err := loadLane(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			best := pickNextSailing(lane.Vessels)
			if best == nil {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"name":   lane.Name,
					"status": "no upcoming sailings",
					"hint":   fmt.Sprintf("run `schedule-pp-cli pull --route %s` to refresh", lane.Name),
				}, flags)
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
				"name":          lane.Name,
				"next_etd":      best.CarrierETD,
				"vessel":        best.VesselName,
				"voyage":        best.VoyageNo,
				"capacity":      best.ShipCapacity,
				"flag":          best.ShipFlag,
				"days_from_now": daysFromNow(best.CarrierETD),
			}, flags)
		},
	}
	return cmd
}

// ---------- compare-carriers ----------

func newCompareCarriersCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare-carriers [route]",
		Short: "Group the lane's vessels by carrier+service so you can spot which carrier sails when.",
		Example: "  schedule-pp-cli compare-carriers NS-JEDDAH --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			lane, err := loadLane(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			byCarrier := map[string][]map[string]any{}
			for _, cs := range lane.CarrierServices {
				key := cs.CarrierCode + "/" + cs.ServiceCode
				byCarrier[key] = append(byCarrier[key], map[string]any{
					"carrier_code":       cs.CarrierCode,
					"carrier_short_name": cs.CarrierShortName,
					"service_code":       cs.ServiceCode,
				})
			}
			vessels := append([]scrape.Vessel(nil), lane.Vessels...)
			sort.Slice(vessels, func(i, j int) bool {
				return vessels[i].CarrierETD < vessels[j].CarrierETD
			})
			rows := make([]map[string]any, 0, len(vessels))
			for _, v := range vessels {
				rows = append(rows, map[string]any{
					"etd":      v.CarrierETD,
					"vessel":   v.VesselName,
					"voyage":   v.VoyageNo,
					"capacity": v.ShipCapacity,
					"flag":     v.ShipFlag,
				})
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
				"name":             lane.Name,
				"pol":              lane.POL,
				"pod":              lane.POD,
				"fetched_at":       lane.FetchedAt,
				"carrier_count":    len(byCarrier),
				"vessels_by_etd":   rows,
				"carrier_services": lane.CarrierServices,
			}, flags)
		},
	}
	return cmd
}

// ---------- shared loaders ----------

type laneSnapshot struct {
	Name            string                  `json:"name"`
	POL             string                  `json:"pol"`
	POD             string                  `json:"pod"`
	URL             string                  `json:"url"`
	FetchedAt       time.Time               `json:"fetched_at"`
	Vessels         []scrape.Vessel         `json:"vessels"`
	CarrierServices []scrape.CarrierService `json:"carrier_services"`
}

func loadLane(ctx context.Context, name string) (*laneSnapshot, error) {
	s, err := registry.Open("")
	if err != nil {
		return nil, err
	}
	if added := s.Seed(); added > 0 {
		_ = s.Save()
	}
	r := s.Get(name)
	if r == nil {
		return nil, routeNotInRegistryError(name, s)
	}
	db, err := openLaneStore(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	raw, err := db.Get(resourceLane, r.Name)
	if err != nil || len(raw) == 0 {
		return nil, fmt.Errorf("no scraped data for %q yet — run `schedule-pp-cli pull --route %s` first", name, r.Name)
	}
	var snap laneSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, fmt.Errorf("decode lane snapshot: %w", err)
	}
	return &snap, nil
}

func filterByDays(vessels []scrape.Vessel, withinDays int) []scrape.Vessel {
	if withinDays <= 0 {
		return vessels
	}
	cutoff := time.Now().AddDate(0, 0, withinDays)
	out := make([]scrape.Vessel, 0, len(vessels))
	for _, v := range vessels {
		t := parseChineseETD(v.CarrierETD)
		if t.IsZero() || (t.After(time.Now().AddDate(0, 0, -1)) && t.Before(cutoff)) {
			out = append(out, v)
		}
	}
	return out
}

func pickNextSailing(vessels []scrape.Vessel) *scrape.Vessel {
	now := time.Now()
	var best *scrape.Vessel
	var bestT time.Time
	for i := range vessels {
		t := parseChineseETD(vessels[i].CarrierETD)
		if t.IsZero() || t.Before(now) {
			continue
		}
		if best == nil || t.Before(bestT) {
			best = &vessels[i]
			bestT = t
		}
	}
	return best
}

func parseChineseETD(s string) time.Time {
	// Format examples: "2026/05/15(周五)" or "05.15 (五)" — only the first is unambiguous.
	cut := strings.IndexAny(s, "(")
	if cut > 0 {
		s = s[:cut]
	}
	s = strings.TrimSpace(s)
	if t, err := time.Parse("2006/01/02", s); err == nil {
		return t
	}
	return time.Time{}
}

func daysFromNow(s string) int {
	t := parseChineseETD(s)
	if t.IsZero() {
		return -1
	}
	return int(time.Until(t).Hours() / 24)
}

// ---------- registrar ----------

func registerNovelCommands(rootCmd *cobra.Command, flags *rootFlags) {
	rootCmd.AddCommand(newRoutesCmd(flags))
	rootCmd.AddCommand(newPullCmd(flags))
	rootCmd.AddCommand(newQueryCmd(flags))
	rootCmd.AddCommand(newNextClsCmd(flags))
	rootCmd.AddCommand(newCompareCarriersCmd(flags))
}
