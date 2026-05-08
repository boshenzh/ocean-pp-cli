package scfi

import (
	"encoding/json"
	"math"
	"testing"
)

const sampleResponse = `{"data":{"currentDate":"2026-05-08","lastDate":"2026-04-30","lineDataList":[
  {"properties":{"lineName_EN":"Comprehensive Index","unit_EN":"","weighting_EN":""},"currentContent":1954.21,"lastContent":1911.40,"absolute":42.81,"percentage":2.24,"dataItemTypeName":"SCFI_T"},
  {"properties":{"lineName_EN":"Europe (Base port)","unit_EN":"USD/TEU","weighting_EN":"20.0%"},"currentContent":2000,"lastContent":1900,"absolute":100,"percentage":5.26,"dataItemTypeName":"SCFI_L1"},
  {"properties":{"lineName_EN":"Mediterranean (Base port)","unit_EN":"USD/TEU","weighting_EN":"10.0%"},"currentContent":null,"lastContent":null,"absolute":null,"percentage":null,"dataItemTypeName":"SCFI_L2"},
  {"properties":{"lineName_EN":"Persian Gulf and Red Sea (Dubai)","unit_EN":"USD/TEU","weighting_EN":"7.0%"},"currentContent":1500,"lastContent":1450,"absolute":50,"percentage":3.45,"dataItemTypeName":"SCFI_L3"},
  {"properties":{"lineName_EN":"India/Pakistan","unit_EN":"USD/FEU","weighting_EN":"5.0%"},"currentContent":2400,"lastContent":2300,"absolute":100,"percentage":4.35,"dataItemTypeName":"SCFI_L4"}
]}}`

func TestParseSnapshot_Wrapped(t *testing.T) {
	snap, err := ParseSnapshot(json.RawMessage(sampleResponse))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if snap.CurrentDate != "2026-05-08" {
		t.Errorf("CurrentDate = %q, want 2026-05-08", snap.CurrentDate)
	}
	if got, want := len(snap.Lines), 5; got != want {
		t.Fatalf("lines = %d, want %d", got, want)
	}
	if !snap.Lines[0].IsComprehensive() {
		t.Errorf("row 0 should be comprehensive (blank weighting, blank unit)")
	}
	if snap.Lines[1].LineName != "Europe (Base port)" {
		t.Errorf("row 1 name = %q", snap.Lines[1].LineName)
	}
	if snap.Lines[2].Current != nil {
		t.Errorf("row 2 (Mediterranean) should have null currentContent in this fixture")
	}
}

func TestParseSnapshot_RejectsEmpty(t *testing.T) {
	if _, err := ParseSnapshot(json.RawMessage(`{"data":{}}`)); err == nil {
		t.Error("expected empty-snapshot error, got nil")
	}
}

func TestWeightingFraction(t *testing.T) {
	cases := map[string]float64{
		"20.0%": 0.2,
		"5%":    0.05,
		"":      0,
		"junk":  0,
	}
	for in, want := range cases {
		got := LineRow{Weighting: in}.WeightingFraction()
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("WeightingFraction(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestContribution(t *testing.T) {
	pct := 5.26
	row := LineRow{Weighting: "20.0%", Percent: &pct}
	if got, want := row.Contribution(), 0.20*5.26; math.Abs(got-want) > 1e-9 {
		t.Errorf("Contribution = %v, want %v", got, want)
	}
	if (LineRow{Weighting: "20.0%"}).Contribution() != 0 {
		t.Error("nil Percent should yield 0 contribution")
	}
	if (LineRow{Percent: &pct}).Contribution() != 0 {
		t.Error("blank weighting should yield 0 contribution")
	}
}

func TestMatchesAnyLane(t *testing.T) {
	row := LineRow{LineName: "Persian Gulf and Red Sea (Dubai)"}
	if !row.MatchesAnyLane(nil) {
		t.Error("nil fragment list should match all (used for 'show everything')")
	}
	if !row.MatchesAnyLane([]string{}) {
		t.Error("empty fragment list should match all")
	}
	if !row.MatchesAnyLane([]string{"Persian Gulf"}) {
		t.Error("should match exact fragment")
	}
	if !row.MatchesAnyLane([]string{"unrelated", "red sea"}) {
		t.Error("should match if any fragment matches")
	}
	if row.MatchesAnyLane([]string{"Mediterranean", "USEC"}) {
		t.Error("should not match unrelated fragments")
	}
	if !row.MatchesAnyLane([]string{"  ", "Persian"}) {
		t.Error("blank entries are skipped, not treated as match-all")
	}
}

func TestMatchesRoute(t *testing.T) {
	row := LineRow{LineName: "Persian Gulf and Red Sea (Dubai)"}
	if !row.MatchesRoute("Persian Gulf") {
		t.Error("should match 'Persian Gulf'")
	}
	if !row.MatchesRoute("red sea") {
		t.Error("should match case-insensitively")
	}
	if row.MatchesRoute("Mediterranean") {
		t.Error("should not match unrelated lane")
	}
	if row.MatchesRoute("") {
		t.Error("empty fragment should not match")
	}
}

func TestNormalizeToFEU(t *testing.T) {
	teuVal := 2000.0
	teu := LineRow{Unit: "USD/TEU", Current: &teuVal}
	if got, ok := teu.NormalizeToFEU(); !ok || got != 4000 {
		t.Errorf("USD/TEU 2000 -> FEU got=%v ok=%v, want 4000 true", got, ok)
	}
	feuVal := 3500.0
	feu := LineRow{Unit: "USD/FEU", Current: &feuVal}
	if got, ok := feu.NormalizeToFEU(); !ok || got != 3500 {
		t.Errorf("USD/FEU 3500 -> FEU got=%v ok=%v, want 3500 true", got, ok)
	}
	if _, ok := (LineRow{Unit: "", Current: &teuVal}.NormalizeToFEU()); ok {
		t.Error("blank unit should not normalize")
	}
	if _, ok := (LineRow{Unit: "USD/TEU"}.NormalizeToFEU()); ok {
		t.Error("nil Current should not normalize")
	}
}

func TestNormalizeToTEU(t *testing.T) {
	feuVal := 3500.0
	feu := LineRow{Unit: "USD/FEU", Current: &feuVal}
	if got, ok := feu.NormalizeToTEU(); !ok || got != 1750 {
		t.Errorf("USD/FEU 3500 -> TEU got=%v ok=%v, want 1750 true", got, ok)
	}
}
