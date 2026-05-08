// Hand-written transcendence commands for webprofile-pp-cli.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/boshenzh/ocean-pp-cli/webprofile-pp-cli/internal/client"
	"github.com/boshenzh/ocean-pp-cli/webprofile-pp-cli/internal/comtrade"
	"github.com/boshenzh/ocean-pp-cli/webprofile-pp-cli/internal/store"

	"github.com/spf13/cobra"
)

const (
	resourceReporters = "comtrade_reporters"
	resourceHS        = "comtrade_hs"
	resourceTrade     = "comtrade_trade"
	defaultPartnerCN  = 156
)

func openCacheStore(ctx context.Context) (*store.Store, error) {
	return store.OpenWithContext(ctx, defaultDBPath("webprofile-pp-cli"))
}

// ---------- shared helpers ----------

func loadReporters(ctx context.Context, c *client.Client, flags *rootFlags, refresh bool) ([]comtrade.Reporter, error) {
	db, err := openCacheStore(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	// Try the per-row locally-synced cache first — populated by the generator's
	// `sync` command, which stores one row per reporter under type=reporters.
	if !refresh {
		if rows, err := db.List("reporters", 0); err == nil && len(rows) > 0 {
			out := make([]comtrade.Reporter, 0, len(rows))
			for _, r := range rows {
				var rep comtrade.Reporter
				if json.Unmarshal(r, &rep) == nil && rep.ReporterCode > 0 {
					out = append(out, rep)
				}
			}
			if len(out) > 0 {
				return out, nil
			}
		}
	}
	// Fall back to a live fetch.
	data, _, err := resolveRead(ctx, c, flags, "reporters", false, "/files/v1/app/reference/Reporters.json", nil, nil)
	if err != nil {
		return nil, err
	}
	data = extractResponseData(data)
	rs, err := comtrade.ParseReporters(data)
	if err != nil {
		return nil, err
	}
	for _, rep := range rs {
		raw, err := json.Marshal(rep)
		if err != nil {
			continue
		}
		_ = db.Upsert("reporters", strconv.Itoa(rep.ReporterCode), raw)
	}
	return rs, nil
}

func loadHS(ctx context.Context, c *client.Client, flags *rootFlags, refresh bool) ([]comtrade.HSEntry, error) {
	db, err := openCacheStore(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if !refresh {
		if raw, err := db.Get(resourceHS, "all"); err == nil && len(raw) > 0 {
			hs, _ := comtrade.ParseHS(raw)
			if len(hs) > 0 {
				return hs, nil
			}
		}
	}
	data, _, err := resolveRead(ctx, c, flags, "hs", false, "/files/v1/app/reference/H6.json", nil, nil)
	if err != nil {
		return nil, err
	}
	data = extractResponseData(data)
	hs, err := comtrade.ParseHS(data)
	if err != nil {
		return nil, err
	}
	_ = db.Upsert(resourceHS, "all", data)
	return hs, nil
}

// fetchTradeYear queries the public-preview endpoint for one (country, partner, HS, year)
// and returns the parsed primaryValue, netWgt aggregates plus raw rows.
type yearAgg struct {
	Year         int
	Records      int
	PrimaryValue float64
	NetWeight    float64
}

func fetchTradeYear(ctx context.Context, c *client.Client, flags *rootFlags, reporterCode, partnerCode int, hs string, year int) (*yearAgg, error) {
	params := map[string]string{
		"reporterCode": strconv.Itoa(reporterCode),
		"partnerCode":  strconv.Itoa(partnerCode),
		"period":       strconv.Itoa(year),
		"flowCode":     "M",
		"motCode":      "0",
		"customsCode":  "C00",
		"maxRecords":   "500",
	}
	if hs != "" {
		params["cmdCode"] = hs
	}
	data, _, err := resolveRead(ctx, c, flags, "trade", false, "/public/v1/preview/C/A/HS", params, nil)
	if err != nil {
		return nil, err
	}
	data = extractResponseData(data)
	var resp struct {
		Count int               `json:"count"`
		Data  []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse trade response: %w", err)
	}
	agg := &yearAgg{Year: year, Records: resp.Count}
	for _, r := range resp.Data {
		var row struct {
			PrimaryValue float64 `json:"primaryValue"`
			NetWgt       float64 `json:"netWgt"`
		}
		if err := json.Unmarshal(r, &row); err != nil {
			continue
		}
		agg.PrimaryValue += row.PrimaryValue
		agg.NetWeight += row.NetWgt
	}
	return agg, nil
}

// ---------- country (fuzzy lookup) ----------

func newCountryCmd(flags *rootFlags) *cobra.Command {
	var refresh bool
	cmd := &cobra.Command{
		Use:   "country [fragment]",
		Short: "Fuzzy-search Comtrade country reporters by name, ISO2, ISO3, or numeric code.",
		Example: `  # Find Egypt's numeric code
  webprofile-pp-cli country Egypt

  # ISO3 -> code
  webprofile-pp-cli country EGY --json`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			rs, err := loadReporters(cmd.Context(), c, flags, refresh)
			if err != nil {
				return err
			}
			matches := comtrade.SearchReporters(args[0], rs)
			if len(matches) == 0 {
				return fmt.Errorf("no Comtrade reporter matches %q", args[0])
			}
			out := make([]map[string]any, 0, len(matches))
			for _, r := range matches {
				out = append(out, map[string]any{
					"code":     r.ReporterCode,
					"iso2":     r.ReporterCodeIsoAlpha2,
					"iso3":     r.ReporterCodeIsoAlpha3,
					"name":     r.ReporterDesc,
				})
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Force re-fetch the reporters reference file (default: cached)")
	return cmd
}

// ---------- hs-search (fuzzy lookup) ----------

func newHSSearchCmd(flags *rootFlags) *cobra.Command {
	var refresh bool
	cmd := &cobra.Command{
		Use:   "hs-search [fragment]",
		Short: "Fuzzy-search HS6 commodity codes by code prefix or text description.",
		Example: `  # Find HS code for telephones
  webprofile-pp-cli hs-search telephone

  # All chapter 87 (vehicles)
  webprofile-pp-cli hs-search 87`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			hs, err := loadHS(cmd.Context(), c, flags, refresh)
			if err != nil {
				return err
			}
			matches := comtrade.SearchHS(args[0], hs)
			if len(matches) == 0 {
				return fmt.Errorf("no HS code matches %q", args[0])
			}
			if len(matches) > 50 {
				matches = matches[:50]
			}
			out := make([]map[string]any, 0, len(matches))
			for _, e := range matches {
				out = append(out, map[string]any{
					"code":        e.ID,
					"description": e.Text,
				})
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Force re-fetch the HS reference file")
	return cmd
}

// ---------- who-imports ----------

func newWhoImportsCmd(flags *rootFlags) *cobra.Command {
	var partner string
	var years int
	cmd := &cobra.Command{
		Use:   "who-imports [country] [hs-code]",
		Short: "Annual import volume for a country/HS pair from a specific partner (default: China). Writes per-year aggregates.",
		Long: `Resolves the country argument (numeric code, ISO2, ISO3, or name fragment) and the
HS commodity code, then queries Comtrade's public-preview endpoint for the last N
annual records of that country importing that HS from the named partner.

The default partner is China (156); pass --partner to override (e.g. --partner Vietnam,
--partner VNM, or --partner 704). Pass --partner 0 (or "World") for total imports
from the world.`,
		Example: `  # Egypt importing telephones (HS 8517) from China — last 3 years
  webprofile-pp-cli who-imports Egypt 8517 --json

  # India importing T-shirts (HS 6109) from world (any source)
  webprofile-pp-cli who-imports India 6109 --partner World --years 5 --json`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return fmt.Errorf("who-imports requires both country and hs-code (got %d arg)", len(args))
			}
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			rs, err := loadReporters(cmd.Context(), c, flags, false)
			if err != nil {
				return err
			}
			country := comtrade.ResolveReporter(args[0], rs)
			if country == nil {
				return fmt.Errorf("unknown country %q (try `webprofile-pp-cli country %s`)", args[0], args[0])
			}
			hs := strings.TrimSpace(args[1])
			partnerCode := defaultPartnerCN
			if strings.EqualFold(partner, "World") || partner == "0" {
				partnerCode = 0
			} else if partner != "" {
				if p := comtrade.ResolveReporter(partner, rs); p != nil {
					partnerCode = p.ReporterCode
				} else {
					return fmt.Errorf("unknown partner %q", partner)
				}
			}
			endYear := time.Now().Year() - 1
			startYear := endYear - years + 1
			yearly := make([]map[string]any, 0, years)
			for y := endYear; y >= startYear; y-- {
				agg, err := fetchTradeYear(cmd.Context(), c, flags, country.ReporterCode, partnerCode, hs, y)
				if err != nil {
					yearly = append(yearly, map[string]any{"year": y, "error": err.Error()})
					continue
				}
				yearly = append(yearly, map[string]any{
					"year":          y,
					"records":       agg.Records,
					"primary_value": agg.PrimaryValue,
					"net_weight_kg": agg.NetWeight,
				})
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
				"country": map[string]any{
					"code": country.ReporterCode,
					"iso3": country.ReporterCodeIsoAlpha3,
					"name": country.ReporterDesc,
				},
				"hs":           hs,
				"partner_code": partnerCode,
				"years":        yearly,
			}, flags)
		},
	}
	cmd.Flags().StringVar(&partner, "partner", "China", "Partner country (default 'China'); pass 'World' or 0 to query total imports from any source")
	cmd.Flags().IntVar(&years, "years", 3, "How many recent annual records to pull (default 3)")
	return cmd
}

// ---------- fit-score ----------

func newFitScoreCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fit-score [country] [hs-code]",
		Short: "Composite 0-100 prospect-fit score: import volume from China × YoY growth × route coverage.",
		Long: `Pulls two annual records (latest year and prior year) of the country importing
the HS code from China, plus the country's total imports of that HS from the
world for the latest year, and combines them into a single 0-100 score:

  share component  (0-50): China-share of country's total HS imports
  growth component (0-20): year-over-year growth in imports from China, clamped
  route component  (0-30): fixed bonus when the country is on a covered lane

The route component is a hardcoded portfolio for v0.1 (Red Sea / Mideast /
India-Pak); v0.2 reads from rate-store.`,
		Example: `  webprofile-pp-cli fit-score Egypt 8517 --json
  webprofile-pp-cli fit-score IND 6109 --json`,
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return fmt.Errorf("fit-score requires both country and hs-code (got %d arg)", len(args))
			}
			if dryRunOK(flags) {
				return nil
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			rs, err := loadReporters(cmd.Context(), c, flags, false)
			if err != nil {
				return err
			}
			country := comtrade.ResolveReporter(args[0], rs)
			if country == nil {
				return fmt.Errorf("unknown country %q", args[0])
			}
			hs := strings.TrimSpace(args[1])
			endYear := time.Now().Year() - 1
			priorYear := endYear - 1
			fromCN, err := fetchTradeYear(cmd.Context(), c, flags, country.ReporterCode, defaultPartnerCN, hs, endYear)
			if err != nil {
				return fmt.Errorf("import-from-china: %w", err)
			}
			fromCNPrior, _ := fetchTradeYear(cmd.Context(), c, flags, country.ReporterCode, defaultPartnerCN, hs, priorYear)
			fromWorld, err := fetchTradeYear(cmd.Context(), c, flags, country.ReporterCode, 0, hs, endYear)
			if err != nil {
				return fmt.Errorf("import-from-world: %w", err)
			}
			yoy := 0.0
			if fromCNPrior != nil && fromCNPrior.PrimaryValue > 0 {
				yoy = ((fromCN.PrimaryValue - fromCNPrior.PrimaryValue) / fromCNPrior.PrimaryValue) * 100
			}
			route := comtrade.IsCoveredRoute(country.ReporterCodeIsoAlpha3)
			score := comtrade.FitScore(fromCN.PrimaryValue, fromWorld.PrimaryValue, yoy, route)
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
				"country": map[string]any{
					"code": country.ReporterCode,
					"iso3": country.ReporterCodeIsoAlpha3,
					"name": country.ReporterDesc,
				},
				"hs":                  hs,
				"latest_year":         endYear,
				"prior_year":          priorYear,
				"import_from_cn":      fromCN.PrimaryValue,
				"import_from_world":   fromWorld.PrimaryValue,
				"yoy_growth_pct":      yoy,
				"route_covered":       route > 0,
				"route_bonus":         route,
				"fit_score":           score,
			}, flags)
		},
	}
	return cmd
}

// ---------- registrar ----------

func registerNovelCommands(rootCmd *cobra.Command, flags *rootFlags) {
	rootCmd.AddCommand(newCountryCmd(flags))
	rootCmd.AddCommand(newHSSearchCmd(flags))
	rootCmd.AddCommand(newWhoImportsCmd(flags))
	rootCmd.AddCommand(newFitScoreCmd(flags))
}
