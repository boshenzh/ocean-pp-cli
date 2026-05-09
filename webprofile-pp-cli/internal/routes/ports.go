package routes

import "strings"

// portToISO3 maps a free-text ocean-port name (as it appears in
// schedule-pp-cli's registry) to an ISO 3166-1 alpha-3 country code. The
// list is intentionally limited to ~80 globally significant container
// ports — enough to make `routes import-from-schedule` useful out of the
// box. Ports the table doesn't recognize are reported back to the user
// rather than silently dropped, so they can extend the list or add the
// ISO3 manually with `routes add`.
//
// Keys are normalized: uppercase, trimmed, and stored without any internal
// punctuation. PortToISO3 normalizes inputs the same way before lookup.
var portToISO3 = map[string]string{
	// Saudi Arabia
	"JEDDAH": "SAU", "DAMMAM": "SAU", "JUBAIL": "SAU", "YANBU": "SAU", "KING ABDULLAH": "SAU",
	// Egypt
	"SOKHNA": "EGY", "PORT SAID": "EGY", "ALEXANDRIA": "EGY", "DAMIETTA": "EGY", "AIN SOKHNA": "EGY",
	// UAE
	"JEBEL ALI": "ARE", "DUBAI": "ARE", "ABU DHABI": "ARE", "SHARJAH": "ARE", "FUJAIRAH": "ARE", "KHALIFA": "ARE",
	// Pakistan
	"KARACHI": "PAK", "PORT QASIM": "PAK", "QASIM": "PAK",
	// India
	"NHAVA SHEVA": "IND", "MUMBAI": "IND", "MUNDRA": "IND", "CHENNAI": "IND", "COCHIN": "IND",
	"KOCHI": "IND", "TUTICORIN": "IND", "KOLKATA": "IND", "ENNORE": "IND", "VIZAG": "IND",
	"VISAKHAPATNAM": "IND", "HAZIRA": "IND", "PIPAVAV": "IND",
	// Djibouti / Yemen / Qatar / Kuwait / Oman / Bahrain
	"DJIBOUTI": "DJI",
	"ADEN":     "YEM", "HODEIDAH": "YEM",
	"DOHA":  "QAT", "HAMAD": "QAT",
	"KUWAIT": "KWT", "SHUWAIKH": "KWT", "SHUAIBA": "KWT",
	"SOHAR": "OMN", "SALALAH": "OMN", "MUSCAT": "OMN",
	"BAHRAIN": "BHR",
	"AQABA":   "JOR",
	// Iran / Iraq / Lebanon / Syria
	"BANDAR ABBAS": "IRN", "BUSHEHR": "IRN",
	"UMM QASR": "IRQ", "BASRA": "IRQ",
	"BEIRUT":   "LBN", "TRIPOLI": "LBN",
	"LATAKIA":  "SYR", "TARTUS": "SYR",
	// East Africa
	"MOMBASA":       "KEN",
	"DAR ES SALAAM": "TZA",
	// South Asia / SE Asia
	"COLOMBO": "LKA", "HAMBANTOTA": "LKA",
	"CHITTAGONG": "BGD",
	"YANGON":     "MMR",
	"PORT KLANG": "MYS", "KLANG": "MYS", "PASIR GUDANG": "MYS", "PENANG": "MYS", "TANJUNG PELEPAS": "MYS",
	"SINGAPORE":  "SGP",
	"JAKARTA":    "IDN", "TANJUNG PRIOK": "IDN", "SURABAYA": "IDN",
	"MANILA":     "PHL", "CEBU": "PHL",
	"BANGKOK":    "THA", "LAEM CHABANG": "THA",
	"HO CHI MINH": "VNM", "HAIPHONG": "VNM", "DA NANG": "VNM", "CAI MEP": "VNM",
	// North Asia
	"BUSAN":  "KOR", "INCHEON": "KOR", "GWANGYANG": "KOR",
	"TOKYO":  "JPN", "YOKOHAMA": "JPN", "OSAKA": "JPN", "KOBE": "JPN", "NAGOYA": "JPN",
	"TAIPEI": "TWN", "KAOHSIUNG": "TWN", "KEELUNG": "TWN", "TAICHUNG": "TWN",
	"HONG KONG": "HKG",
	// Europe
	"HAMBURG":    "DEU", "BREMERHAVEN": "DEU",
	"ROTTERDAM":  "NLD",
	"ANTWERP":    "BEL", "ZEEBRUGGE": "BEL",
	"FELIXSTOWE": "GBR", "SOUTHAMPTON": "GBR", "LONDON GATEWAY": "GBR",
	"LE HAVRE":   "FRA", "FOS": "FRA", "MARSEILLE": "FRA",
	"GENOA":      "ITA", "LIVORNO": "ITA", "LA SPEZIA": "ITA", "GIOIA TAURO": "ITA",
	"VALENCIA":   "ESP", "BARCELONA": "ESP", "ALGECIRAS": "ESP",
	"PIRAEUS":    "GRC", "THESSALONIKI": "GRC",
	"MERSIN":     "TUR", "ISTANBUL": "TUR", "AMBARLI": "TUR", "IZMIR": "TUR",
	// North America
	"LOS ANGELES": "USA", "LONG BEACH": "USA", "OAKLAND": "USA",
	"TACOMA":      "USA", "SEATTLE": "USA",
	"NEW YORK":    "USA", "SAVANNAH": "USA", "CHARLESTON": "USA",
	"HOUSTON":     "USA", "MIAMI": "USA", "NORFOLK": "USA",
	"VANCOUVER":   "CAN", "MONTREAL": "CAN", "HALIFAX": "CAN",
	// Latin America
	"SANTOS":        "BRA", "ITAJAI": "BRA", "PARANAGUA": "BRA",
	"BUENOS AIRES":  "ARG",
	"MANZANILLO":    "MEX", "VERACRUZ": "MEX", "LAZARO CARDENAS": "MEX",
	"CALLAO":        "PER",
	"VALPARAISO":    "CHL", "SAN ANTONIO": "CHL",
	"CARTAGENA":     "COL",
	// Oceania
	"SYDNEY":   "AUS", "MELBOURNE": "AUS", "BRISBANE": "AUS", "FREMANTLE": "AUS",
	"AUCKLAND": "NZL", "TAURANGA": "NZL",
}

// PortToISO3 returns the ISO3 country code for a port name, plus a flag
// indicating whether the port was found in the lookup table. Matching is
// case-insensitive and tolerant of leading/trailing whitespace.
func PortToISO3(port string) (string, bool) {
	norm := strings.ToUpper(strings.TrimSpace(port))
	if norm == "" {
		return "", false
	}
	iso3, ok := portToISO3[norm]
	return iso3, ok
}
