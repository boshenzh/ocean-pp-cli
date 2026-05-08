package scrape

import (
	"testing"
)

const sampleHTML = `<!DOCTYPE html><html><body>
some markup ...
\"etdFormatter\":\"05.15 (五)\",\"vslName\":\"CMA CGM APOLLON\",\"voyNo\":\"0RDNYW1MA\",\"shipCapacity\":\"15363TEU\",\"shipYearBuilt\":\"2021\",\"shipFlagFTPPath\":\"x.jpg\",\"shipFlag\":\"MT\",\"hasWharfActualBerthTime\":false,\"carrierETDFormatter\":\"2026/05/15(周五)\",\"wharfPlanBerthDateFormatter\":\"2026/05/15(周五)\",\"wharfActualBerthDateFormatter\":null,\"wharfPlanLeaveDateFormatter\":null,\"isWorking\":false}\n
\"etdFormatter\":\"05.13 (三)\",\"vslName\":\"MSC FREYA\",\"voyNo\":\"FD619W\",\"shipCapacity\":\"15264TEU\",\"shipYearBuilt\":\"2023\",\"shipFlag\":\"LR\",\"hasWharfActualBerthTime\":false,\"carrierETDFormatter\":\"2026/05/13(周三)\",\"wharfPlanBerthDateFormatter\":null,\"wharfActualBerthDateFormatter\":null,\"wharfPlanLeaveDateFormatter\":null,\"isWorking\":false}\n
\"carrierCode\":\"CMA\",\"carrierShortName\":\"CMA\",\"svcCode\":\"REX2\"\n
\"carrierCode\":\"COSCO\",\"carrierShortName\":\"COSCO\",\"svcCode\":\"RES1\"\n
\"carrierCode\":\"PIL\",\"carrierShortName\":\"PIL\",\"svcCode\":\"RS2\"\n
</body></html>`

func TestParse(t *testing.T) {
	pd, err := Parse([]byte(sampleHTML), "https://example/routePort?st=A&de=B")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := len(pd.Vessels); got != 2 {
		t.Fatalf("vessel count = %d, want 2", got)
	}
	if pd.Vessels[0].VesselName != "CMA CGM APOLLON" {
		t.Errorf("vessel 0 name = %q", pd.Vessels[0].VesselName)
	}
	if pd.Vessels[0].VoyageNo != "0RDNYW1MA" {
		t.Errorf("vessel 0 voyage = %q", pd.Vessels[0].VoyageNo)
	}
	if pd.Vessels[0].CarrierETD != "2026/05/15(周五)" {
		t.Errorf("vessel 0 carrier ETD = %q", pd.Vessels[0].CarrierETD)
	}
	if got := len(pd.CarrierServices); got != 3 {
		t.Fatalf("carrier-service count = %d, want 3", got)
	}
	codes := []string{pd.CarrierServices[0].CarrierCode, pd.CarrierServices[1].CarrierCode, pd.CarrierServices[2].CarrierCode}
	want := []string{"CMA", "COSCO", "PIL"}
	for i := range want {
		if codes[i] != want[i] {
			t.Errorf("carrier %d code = %q, want %q", i, codes[i], want[i])
		}
	}
}

func TestParseEmpty(t *testing.T) {
	if _, err := Parse([]byte("<html>nothing here</html>"), "x"); err == nil {
		t.Error("expected error on empty page")
	}
}

func TestFilterCarriers(t *testing.T) {
	carriers := []CarrierService{
		{CarrierCode: "CMA", ServiceCode: "REX2"},
		{CarrierCode: "OBSCURE", ServiceCode: "X1"},
		{CarrierCode: "COSCO", ServiceCode: "RES1"},
		{CarrierCode: "pil", ServiceCode: "RS2"}, // lowercase to verify case-insensitive
	}
	out := FilterCarriers(carriers, []string{"CMA", "PIL"})
	if len(out) != 2 {
		t.Fatalf("filtered count = %d, want 2", len(out))
	}
	if out[0].CarrierCode != "CMA" || out[1].CarrierCode != "pil" {
		t.Errorf("filtered codes = %+v", out)
	}
	// Empty whitelist returns input unchanged.
	if len(FilterCarriers(carriers, nil)) != 4 {
		t.Error("empty whitelist should pass through")
	}
}
