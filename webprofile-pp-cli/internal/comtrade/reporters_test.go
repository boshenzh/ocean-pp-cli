package comtrade

import (
	"encoding/json"
	"math"
	"testing"
)

const sampleReporters = `{"results": [
  {"id":"818","text":"Egypt","reporterCode":818,"reporterDesc":"Egypt","reporterCodeIsoAlpha2":"EG","reporterCodeIsoAlpha3":"EGY"},
  {"id":"156","text":"China","reporterCode":156,"reporterDesc":"China","reporterCodeIsoAlpha2":"CN","reporterCodeIsoAlpha3":"CHN"},
  {"id":"842","text":"USA, Puerto Rico and US Virgin Islands","reporterCode":842,"reporterDesc":"USA","reporterCodeIsoAlpha2":"US","reporterCodeIsoAlpha3":"USA"}
]}`

const sampleHS = `{"results": [
  {"id":"8517","text":"Telephones for cellular networks; Other apparatus for transmission or reception of voice"},
  {"id":"6109","text":"T-shirts, singlets and other vests, knitted or crocheted"},
  {"id":"8703","text":"Motor cars and other motor vehicles principally designed for the transport of persons"}
]}`

func TestParseReporters(t *testing.T) {
	rs, err := ParseReporters(json.RawMessage(sampleReporters))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rs) != 3 {
		t.Fatalf("count = %d, want 3", len(rs))
	}
	if rs[0].ReporterCode != 818 {
		t.Errorf("first code = %d", rs[0].ReporterCode)
	}
}

func TestResolveReporter(t *testing.T) {
	rs, _ := ParseReporters(json.RawMessage(sampleReporters))
	cases := map[string]int{
		"818":          818,
		"Egypt":        818,
		"egypt":        818,
		"EG":           818,
		"EGY":          818,
		"egy":          818,
		"China":        156,
		"chin":         156, // substring
		"USA":          842,
		"missing-name": 0,
	}
	for q, want := range cases {
		r := ResolveReporter(q, rs)
		got := 0
		if r != nil {
			got = r.ReporterCode
		}
		if got != want {
			t.Errorf("ResolveReporter(%q) = %d, want %d", q, got, want)
		}
	}
}

func TestSearchReporters(t *testing.T) {
	rs, _ := ParseReporters(json.RawMessage(sampleReporters))
	results := SearchReporters("us", rs)
	// "USA" appears in both reporterDesc and Text — both rows containing it should match
	found := false
	for _, r := range results {
		if r.ReporterCode == 842 {
			found = true
		}
	}
	if !found {
		t.Errorf("'us' should match USA: results=%v", results)
	}
}

func TestSearchHS(t *testing.T) {
	hs, _ := ParseHS(json.RawMessage(sampleHS))
	results := SearchHS("telephone", hs)
	if len(results) != 1 || results[0].ID != "8517" {
		t.Errorf("telephone search: %v", results)
	}
	results = SearchHS("8703", hs)
	if len(results) != 1 || results[0].ID != "8703" {
		t.Errorf("8703 search: %v", results)
	}
}

func TestFitScore(t *testing.T) {
	cases := []struct {
		name    string
		import_ float64
		total   float64
		growth  float64
		route   float64
		want    float64
	}{
		{"perfect: high share + high growth + covered", 50, 100, 30, 30, 100},
		{"low share, low growth, uncovered", 1, 100, 0, 0, 21}, // share 1 + growth 20 (0+20) + 0 = 21
		{"covered route boost only", 0, 1, -20, 30, 30},
		{"clamps growth above 20", 100, 100, 999, 0, 70}, // share 50 + growth 20 + 0 = 70
		{"zero total imports", 1000, 0, 0, 0, 20},        // share 0 + growth 20 + 0 = 20
	}
	for _, tc := range cases {
		got := FitScore(tc.import_, tc.total, tc.growth, tc.route)
		if math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("%s: FitScore = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestIsCoveredRoute(t *testing.T) {
	if IsCoveredRoute("EGY") != 30 {
		t.Error("EGY should be covered (Egypt - Suez/Sokhna)")
	}
	if IsCoveredRoute("egy") != 30 {
		t.Error("case-insensitive: egy should be covered")
	}
	if IsCoveredRoute("USA") != 0 {
		t.Error("USA is not in our covered portfolio")
	}
	if IsCoveredRoute("") != 0 {
		t.Error("empty should be 0")
	}
}
