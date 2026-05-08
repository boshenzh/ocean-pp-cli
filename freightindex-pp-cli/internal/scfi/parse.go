// Package scfi parses Shanghai Containerized Freight Index responses
// from SSE.net.cn into typed rows usable by transcendence commands.
package scfi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// LineRow is one row from the SCFI lineDataList: either the Comprehensive Index
// or one of the 15 lane sub-indices.
type LineRow struct {
	LineName  string   // English lane name from properties.lineName_EN
	Unit      string   // properties.unit_EN ("USD/TEU" or "USD/FEU"; empty for Comprehensive)
	Weighting string   // properties.weighting_EN ("20.0%"; empty for Comprehensive)
	Current   *float64 // currentContent — nil when SSE didn't publish this lane this week
	Last      *float64 // lastContent
	Absolute  *float64 // currentContent - lastContent
	Percent   *float64 // weekly percent change
}

// Snapshot is one weekly SCFI publication.
type Snapshot struct {
	CurrentDate string
	LastDate    string
	Lines       []LineRow
}

// ParseSnapshot accepts the raw response body from /currentIndex (the wrapper
// envelope `{"data": {...}}`) OR the unwrapped data object. It returns a
// fully decoded Snapshot.
func ParseSnapshot(raw json.RawMessage) (*Snapshot, error) {
	var wrapper struct {
		Data *struct {
			CurrentDate  string            `json:"currentDate"`
			LastDate     string            `json:"lastDate"`
			LineDataList []json.RawMessage `json:"lineDataList"`
		} `json:"data"`
		// Also accept the unwrapped shape (after extractResponseData):
		CurrentDate  string            `json:"currentDate"`
		LastDate     string            `json:"lastDate"`
		LineDataList []json.RawMessage `json:"lineDataList"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, fmt.Errorf("scfi: parse snapshot envelope: %w", err)
	}
	currentDate := wrapper.CurrentDate
	lastDate := wrapper.LastDate
	rows := wrapper.LineDataList
	if wrapper.Data != nil {
		currentDate = wrapper.Data.CurrentDate
		lastDate = wrapper.Data.LastDate
		rows = wrapper.Data.LineDataList
	}
	if currentDate == "" || len(rows) == 0 {
		return nil, fmt.Errorf("scfi: empty snapshot (currentDate=%q, lines=%d)", currentDate, len(rows))
	}
	snap := &Snapshot{CurrentDate: currentDate, LastDate: lastDate}
	for i, rowRaw := range rows {
		row, err := parseLine(rowRaw)
		if err != nil {
			return nil, fmt.Errorf("scfi: line %d: %w", i, err)
		}
		snap.Lines = append(snap.Lines, row)
	}
	return snap, nil
}

func parseLine(raw json.RawMessage) (LineRow, error) {
	var src struct {
		Properties struct {
			LineNameEN  string `json:"lineName_EN"`
			UnitEN      string `json:"unit_EN"`
			WeightingEN string `json:"weighting_EN"`
		} `json:"properties"`
		CurrentContent *float64 `json:"currentContent"`
		LastContent    *float64 `json:"lastContent"`
		Absolute       *float64 `json:"absolute"`
		Percentage     *float64 `json:"percentage"`
	}
	if err := json.Unmarshal(raw, &src); err != nil {
		return LineRow{}, err
	}
	return LineRow{
		LineName:  src.Properties.LineNameEN,
		Unit:      src.Properties.UnitEN,
		Weighting: src.Properties.WeightingEN,
		Current:   src.CurrentContent,
		Last:      src.LastContent,
		Absolute:  src.Absolute,
		Percent:   src.Percentage,
	}, nil
}

// IsComprehensive reports whether a lane row is the headline Comprehensive Index
// (the only row with a blank weighting and unit).
func (l LineRow) IsComprehensive() bool {
	return l.Weighting == "" && l.Unit == ""
}

// WeightingFraction parses "20.0%" -> 0.20. Returns 0 for blank inputs (the
// Comprehensive Index row has no weighting). The caller decides whether a
// zero weighting means "skip this row" or "treat as zero contribution."
func (l LineRow) WeightingFraction() float64 {
	w := strings.TrimSuffix(strings.TrimSpace(l.Weighting), "%")
	if w == "" {
		return 0
	}
	v, err := strconv.ParseFloat(w, 64)
	if err != nil {
		return 0
	}
	return v / 100
}

// Contribution returns weighting% × percent change, the per-lane contribution to
// the comprehensive-index move. Returns 0 when either input is missing. Sign
// follows the percent change.
func (l LineRow) Contribution() float64 {
	if l.Percent == nil {
		return 0
	}
	w := l.WeightingFraction()
	if w == 0 {
		return 0
	}
	return w * (*l.Percent)
}

// MatchesAnyLane reports whether the lane matches any of the given fragments
// (case-insensitive substring). When fragments is empty, every lane matches —
// callers use this for "show everything" semantics.
func (l LineRow) MatchesAnyLane(fragments []string) bool {
	if len(fragments) == 0 {
		return true
	}
	name := strings.ToLower(l.LineName)
	for _, frag := range fragments {
		f := strings.ToLower(strings.TrimSpace(frag))
		if f == "" {
			continue
		}
		if strings.Contains(name, f) {
			return true
		}
	}
	return false
}

// MatchesRoute reports whether the lane name contains the given fragment
// (case-insensitive substring).
func (l LineRow) MatchesRoute(fragment string) bool {
	if fragment == "" {
		return false
	}
	return strings.Contains(strings.ToLower(l.LineName), strings.ToLower(fragment))
}

// NormalizeToFEU converts a lane's current value from its native unit to USD/FEU.
// USD/TEU rows are doubled. Returns (newValue, normalized) where normalized is
// false for rows whose unit cannot be converted (Comprehensive Index, blank
// units, or already FEU).
func (l LineRow) NormalizeToFEU() (float64, bool) {
	if l.Current == nil {
		return 0, false
	}
	switch strings.ToUpper(strings.TrimSpace(l.Unit)) {
	case "USD/TEU":
		return *l.Current * 2, true
	case "USD/FEU":
		return *l.Current, true
	}
	return 0, false
}

// NormalizeToTEU converts a lane's current value from its native unit to USD/TEU.
// USD/FEU rows are halved.
func (l LineRow) NormalizeToTEU() (float64, bool) {
	if l.Current == nil {
		return 0, false
	}
	switch strings.ToUpper(strings.TrimSpace(l.Unit)) {
	case "USD/FEU":
		return *l.Current / 2, true
	case "USD/TEU":
		return *l.Current, true
	}
	return 0, false
}
