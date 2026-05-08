// Package comtrade resolves human-friendly country and HS-code lookups
// against the UN Comtrade reference files that ship without auth.
package comtrade

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Reporter is one row in the Comtrade Reporters reference file.
// id is emitted as a number by Comtrade; declare via json.RawMessage so the
// decoder accepts both numeric and stringy shapes — we don't actually use it
// (reporterCode is the canonical numeric key).
type Reporter struct {
	ID                       json.RawMessage `json:"id,omitempty"`
	Text                     string          `json:"text"`
	ReporterCode             int             `json:"reporterCode"`
	ReporterDesc             string          `json:"reporterDesc"`
	ReporterCodeIsoAlpha2    string          `json:"reporterCodeIsoAlpha2"`
	ReporterCodeIsoAlpha3    string          `json:"reporterCodeIsoAlpha3"`
}

// HSEntry is one row in the Comtrade HS reference file.
type HSEntry struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// ParseReporters accepts the response body from /files/v1/app/reference/Reporters.json
// at any nesting depth. The Comtrade response is `{"results":[...]}`, but the
// CLI's read pipeline can wrap it as `{"results":{"results":[...]}}` depending
// on whether extractResponseData ran.
func ParseReporters(raw json.RawMessage) ([]Reporter, error) {
	var bare []Reporter
	if err := json.Unmarshal(raw, &bare); err == nil && len(bare) > 0 {
		return bare, nil
	}
	var single struct {
		Results []Reporter `json:"results"`
		Data    []Reporter `json:"data"`
	}
	if err := json.Unmarshal(raw, &single); err == nil {
		if len(single.Results) > 0 {
			return single.Results, nil
		}
		if len(single.Data) > 0 {
			return single.Data, nil
		}
	}
	var double struct {
		Results struct {
			Results []Reporter `json:"results"`
			Data    []Reporter `json:"data"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &double); err == nil {
		if len(double.Results.Results) > 0 {
			return double.Results.Results, nil
		}
		if len(double.Results.Data) > 0 {
			return double.Results.Data, nil
		}
	}
	return nil, nil
}

// ResolveReporter accepts a numeric code, ISO2 ("EG"), ISO3 ("EGY"), or name
// fragment ("Egypt", "egy") and returns the matching Reporter. Returns nil
// when no match. Matching is case-insensitive.
func ResolveReporter(query string, reporters []Reporter) *Reporter {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	if n, err := strconv.Atoi(q); err == nil {
		for i := range reporters {
			if reporters[i].ReporterCode == n {
				return &reporters[i]
			}
		}
		return nil
	}
	// Pass 1: exact ISO2 / ISO3 match (deterministic, avoids ambiguity).
	for i := range reporters {
		if strings.EqualFold(reporters[i].ReporterCodeIsoAlpha2, q) ||
			strings.EqualFold(reporters[i].ReporterCodeIsoAlpha3, q) {
			return &reporters[i]
		}
	}
	// Pass 2: exact name match.
	for i := range reporters {
		if strings.EqualFold(reporters[i].ReporterDesc, q) ||
			strings.EqualFold(reporters[i].Text, q) {
			return &reporters[i]
		}
	}
	// Pass 3: substring match against name.
	for i := range reporters {
		if strings.Contains(strings.ToLower(reporters[i].ReporterDesc), q) ||
			strings.Contains(strings.ToLower(reporters[i].Text), q) {
			return &reporters[i]
		}
	}
	return nil
}

// SearchReporters returns every reporter whose name contains the fragment
// (case-insensitive). Used by the `country` lookup command.
func SearchReporters(fragment string, reporters []Reporter) []Reporter {
	q := strings.ToLower(strings.TrimSpace(fragment))
	if q == "" {
		return nil
	}
	var out []Reporter
	for i := range reporters {
		if strings.Contains(strings.ToLower(reporters[i].ReporterDesc), q) ||
			strings.Contains(strings.ToLower(reporters[i].Text), q) ||
			strings.EqualFold(reporters[i].ReporterCodeIsoAlpha2, q) ||
			strings.EqualFold(reporters[i].ReporterCodeIsoAlpha3, q) {
			out = append(out, reporters[i])
		}
	}
	return out
}

// SearchHS returns every HS entry whose code or description contains the
// fragment (case-insensitive). The Comtrade HS file uses 6-digit codes.
func SearchHS(fragment string, entries []HSEntry) []HSEntry {
	q := strings.ToLower(strings.TrimSpace(fragment))
	if q == "" {
		return nil
	}
	var out []HSEntry
	for i := range entries {
		if strings.Contains(strings.ToLower(entries[i].ID), q) ||
			strings.Contains(strings.ToLower(entries[i].Text), q) {
			out = append(out, entries[i])
		}
	}
	return out
}

// ParseHS accepts the response body from /files/v1/app/reference/H6.json at
// any nesting depth — same handling as ParseReporters.
func ParseHS(raw json.RawMessage) ([]HSEntry, error) {
	var bare []HSEntry
	if err := json.Unmarshal(raw, &bare); err == nil && len(bare) > 0 {
		return bare, nil
	}
	var single struct {
		Results []HSEntry `json:"results"`
		Data    []HSEntry `json:"data"`
	}
	if err := json.Unmarshal(raw, &single); err == nil {
		if len(single.Results) > 0 {
			return single.Results, nil
		}
		if len(single.Data) > 0 {
			return single.Data, nil
		}
	}
	var double struct {
		Results struct {
			Results []HSEntry `json:"results"`
			Data    []HSEntry `json:"data"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &double); err == nil {
		if len(double.Results.Results) > 0 {
			return double.Results.Results, nil
		}
		if len(double.Results.Data) > 0 {
			return double.Results.Data, nil
		}
	}
	return nil, nil
}

// FitScore computes a 0-100 prospect-fit score for an importer/HS pair.
//
// Inputs:
//
//	importValueUSD  — primaryValue (CIF or fallback) for this country/HS/source
//	totalImportsUSD — country's total imports of this HS from the world
//	yoyGrowthPct    — year-over-year growth in imports of this HS
//	routeBonus      — 0-30 fixed bonus when the importer is on a covered lane
//
// The score is clamped to [0,100]. Components:
//
//	share component (0-50): min(50, importValue/totalImports * 100)
//	growth component (0-20): clamp(yoyGrowthPct, -20, 20)+20 capped at 20
//	route component (0-30): pass-through clamp
func FitScore(importValueUSD, totalImportsUSD, yoyGrowthPct, routeBonus float64) float64 {
	share := 0.0
	if totalImportsUSD > 0 {
		share = (importValueUSD / totalImportsUSD) * 100
	}
	if share > 50 {
		share = 50
	}
	if share < 0 {
		share = 0
	}
	growth := yoyGrowthPct
	if growth > 20 {
		growth = 20
	}
	if growth < -20 {
		growth = -20
	}
	growth += 20
	if growth > 20 {
		growth = 20
	}
	if routeBonus > 30 {
		routeBonus = 30
	}
	if routeBonus < 0 {
		routeBonus = 0
	}
	score := share + growth + routeBonus
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

// IsCoveredRoute reports whether an ISO3 country code is in our headline
// lane portfolio. Hardcoded for v0.1; reads from rate-store in v0.2.
var coveredISO3 = map[string]bool{
	"EGY": true, // Egypt (Suez/Sokhna/Alexandria)
	"SAU": true, // Saudi Arabia (Jeddah/Dammam)
	"ARE": true, // UAE (Jebel Ali/Hamad)
	"IND": true, // India (Nhava Sheva)
	"PAK": true, // Pakistan (Karachi)
	"DJI": true, // Djibouti
	"YEM": true, // Yemen (Aden)
	"QAT": true, // Qatar
	"KWT": true, // Kuwait
	"OMN": true, // Oman
	"BHR": true, // Bahrain
	"JOR": true, // Jordan (Aqaba)
	"IRQ": true, // Iraq
	"IRN": true, // Iran
	"LBN": true, // Lebanon
	"SYR": true, // Syria
}

// IsCoveredRoute reports whether the given ISO3 country code is on a route
// the user already covers. Returns 30 (full route bonus) when true, 0 otherwise.
func IsCoveredRoute(iso3 string) float64 {
	if coveredISO3[strings.ToUpper(strings.TrimSpace(iso3))] {
		return 30
	}
	return 0
}
