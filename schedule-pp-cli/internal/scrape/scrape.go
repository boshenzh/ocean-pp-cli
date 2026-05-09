// Package scrape extracts vessel and carrier records from Weiyun's
// /routePort SSR HTML. The page embeds Next.js Server Components streamed
// JSON via `self.__next_f.push([1,"..."])`; each escaped chunk contains
// either a vessel record or a carrier+service record we care about.
package scrape

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Vessel is one sailing extracted from the page.
type Vessel struct {
	ETD          string `json:"etd"`           // "05.15 (五)"
	CarrierETD   string `json:"carrier_etd"`   // "2026/05/15(周五)"
	VesselName   string `json:"vessel_name"`   // "CMA CGM APOLLON"
	VoyageNo     string `json:"voyage_no"`     // "0RDNYW1MA"
	ShipCapacity string `json:"ship_capacity"` // "15363TEU"
	ShipFlag     string `json:"ship_flag"`     // "MT"
	ShipYear     string `json:"ship_year"`     // "2021"
	IsWorking    bool   `json:"is_working"`
}

// CarrierService is one (carrier, service) combination on the lane.
type CarrierService struct {
	CarrierCode      string `json:"carrier_code"`       // "CMA"
	CarrierShortName string `json:"carrier_short_name"` // "CMA" or "CUL 中联航运"
	ServiceCode      string `json:"service_code"`       // "REX2"
}

// PageData is the structured extraction from one routePort HTML page.
type PageData struct {
	URL             string           `json:"url"`
	FetchedAt       time.Time        `json:"fetched_at"`
	Vessels         []Vessel         `json:"vessels"`
	CarrierServices []CarrierService `json:"carrier_services"`
}

// vessel records appear as escaped JSON: \"etdFormatter\":\"...\",\"vslName\":\"...\",...
// We unescape one chunk at a time and decode.
var vesselRe = regexp.MustCompile(`\\"etdFormatter\\":\\"[^\\]+\\",\\"vslName\\":\\"[^\\]+\\",\\"voyNo\\":\\"[^\\]+\\"[^}]+isWorking\\":(?:true|false)`)

// carrier records appear as: \"carrierCode\":\"...\",\"carrierShortName\":\"...\",\"svcCode\":\"...\"
var carrierRe = regexp.MustCompile(`\\"carrierCode\\":\\"[^\\]+\\",\\"carrierShortName\\":\\"[^\\]+\\",\\"svcCode\\":\\"[^\\]+\\"`)

// Parse extracts vessel and carrier records from a routePort HTML body.
// Duplicates (same vessel or same carrier+svc combo) are kept as-is — they
// appear once per page render, even when a service has multiple operating
// carriers.
func Parse(html []byte, pageURL string) (*PageData, error) {
	out := &PageData{
		URL:       pageURL,
		FetchedAt: time.Now().UTC(),
	}
	seenV := make(map[string]bool)
	for _, m := range vesselRe.FindAllString(string(html), -1) {
		v, err := decodeVessel(m)
		if err != nil {
			continue
		}
		key := v.VesselName + "|" + v.VoyageNo
		if seenV[key] {
			continue
		}
		seenV[key] = true
		out.Vessels = append(out.Vessels, v)
	}
	seen := make(map[string]bool)
	for _, m := range carrierRe.FindAllString(string(html), -1) {
		c, err := decodeCarrier(m)
		if err != nil {
			continue
		}
		key := c.CarrierCode + "|" + c.ServiceCode
		if seen[key] {
			continue
		}
		seen[key] = true
		out.CarrierServices = append(out.CarrierServices, c)
	}
	if len(out.Vessels) == 0 && len(out.CarrierServices) == 0 {
		return out, fmt.Errorf("no vessel or carrier records found in %d bytes", len(html))
	}
	return out, nil
}

// decodeVessel parses an escaped JSON fragment by re-quoting it and decoding.
func decodeVessel(escaped string) (Vessel, error) {
	// Wrap fragment in {} and unescape backslashes that the SSR streamer added.
	src := "{" + strings.ReplaceAll(escaped, `\"`, `"`) + "}"
	var raw struct {
		ETDFormatter      string  `json:"etdFormatter"`
		CarrierETDFormatter string `json:"carrierETDFormatter"`
		VslName           string  `json:"vslName"`
		VoyNo             string  `json:"voyNo"`
		ShipCapacity      string  `json:"shipCapacity"`
		ShipFlag          string  `json:"shipFlag"`
		ShipYearBuilt     string  `json:"shipYearBuilt"`
		IsWorking         bool    `json:"isWorking"`
	}
	if err := json.Unmarshal([]byte(src), &raw); err != nil {
		return Vessel{}, err
	}
	return Vessel{
		ETD:          raw.ETDFormatter,
		CarrierETD:   raw.CarrierETDFormatter,
		VesselName:   raw.VslName,
		VoyageNo:     raw.VoyNo,
		ShipCapacity: raw.ShipCapacity,
		ShipFlag:     raw.ShipFlag,
		ShipYear:     raw.ShipYearBuilt,
		IsWorking:    raw.IsWorking,
	}, nil
}

func decodeCarrier(escaped string) (CarrierService, error) {
	src := "{" + strings.ReplaceAll(escaped, `\"`, `"`) + "}"
	var raw struct {
		CarrierCode      string `json:"carrierCode"`
		CarrierShortName string `json:"carrierShortName"`
		SvcCode          string `json:"svcCode"`
	}
	if err := json.Unmarshal([]byte(src), &raw); err != nil {
		return CarrierService{}, err
	}
	return CarrierService{
		CarrierCode:      raw.CarrierCode,
		CarrierShortName: raw.CarrierShortName,
		ServiceCode:      raw.SvcCode,
	}, nil
}

// FilterCarriers keeps only carrier services whose CarrierCode is in the
// whitelist (case-insensitive). Empty whitelist returns the input unchanged.
func FilterCarriers(carriers []CarrierService, whitelist []string) []CarrierService {
	if len(whitelist) == 0 {
		return carriers
	}
	allow := make(map[string]bool)
	for _, w := range whitelist {
		allow[strings.ToUpper(strings.TrimSpace(w))] = true
	}
	out := make([]CarrierService, 0, len(carriers))
	for _, c := range carriers {
		if allow[strings.ToUpper(c.CarrierCode)] {
			out = append(out, c)
		}
	}
	return out
}

