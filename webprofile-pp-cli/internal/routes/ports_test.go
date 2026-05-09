package routes

import "testing"

func TestPortToISO3(t *testing.T) {
	cases := []struct {
		in       string
		wantISO3 string
		wantOK   bool
	}{
		{"JEDDAH", "SAU", true},
		{"jeddah", "SAU", true},
		{" JEDDAH ", "SAU", true},
		{"Jeddah", "SAU", true},
		{"JEBEL ALI", "ARE", true},
		{"NHAVA SHEVA", "IND", true},
		{"DJIBOUTI", "DJI", true},
		{"ROTTERDAM", "NLD", true},
		{"LOS ANGELES", "USA", true},
		{"UNKNOWN PORT", "", false},
		{"", "", false},
		{"   ", "", false},
	}
	for _, tc := range cases {
		gotISO3, gotOK := PortToISO3(tc.in)
		if gotISO3 != tc.wantISO3 || gotOK != tc.wantOK {
			t.Errorf("PortToISO3(%q) = (%q, %v), want (%q, %v)",
				tc.in, gotISO3, gotOK, tc.wantISO3, tc.wantOK)
		}
	}
}

func TestPortToISO3CoverageSampling(t *testing.T) {
	// Sanity: spot-check that core ports across all 5 of the user's seeded
	// schedule lanes resolve. If any of these regress, import-from-schedule
	// will silently drop a real lane.
	required := map[string]string{
		"JEDDAH":      "SAU",
		"SOKHNA":      "EGY",
		"KARACHI":     "PAK",
		"JEBEL ALI":   "ARE",
		"NHAVA SHEVA": "IND",
		"DJIBOUTI":    "DJI",
	}
	for port, want := range required {
		got, ok := PortToISO3(port)
		if !ok || got != want {
			t.Errorf("PortToISO3(%q) = (%q, %v), want (%q, true)", port, got, ok, want)
		}
	}
}
