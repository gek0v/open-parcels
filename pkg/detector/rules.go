// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package detector

import (
	"regexp"

	"open-parcels/pkg/models"
)

func (d *Detector) initRules() {
	d.rules = []PatternRule{
		// 1. Air Waybill (hyphenated NNN-NNNNNNNN)
		{
			CarrierSlug:    "air-cargo",
			CarrierName:    "IATA Air Cargo",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeFreight,
			Priority:       100,
			ConfidenceBase: 0.99,
			Regex:          regexp.MustCompile(`^\d{3}-\d{8}$`),
			Validator:      validateAWB,
			Description:    "IATA Air Waybill standard with Mod 7 checksum",
		},

		// 2. ISO 6346 Shipping Container
		{
			CarrierSlug:    "ocean-container",
			CarrierName:    "ISO Ocean Container",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeFreight,
			Priority:       95,
			ConfidenceBase: 0.98,
			Regex:          regexp.MustCompile(`^[A-Z]{3}[UJZ]\d{7}$`),
			Validator:      validateContainer,
			Description:    "ISO 6346 standard container identification code",
		},

		// 3. UPS standard 1Z
		{
			CarrierSlug:    "ups",
			CarrierName:    "UPS",
			Country:        "US",
			CountryName:    "United States",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       90,
			ConfidenceBase: 0.98,
			Regex:          regexp.MustCompile(`^1Z[0-9A-Z]{16}$`),
			Validator:      validateUPS,
			Description:    "UPS 1Z Tracking number with Mod 10 check digit",
		},

		// 4. YunExpress (YT + 16 or 18 digits)
		{
			CarrierSlug:    "yunexpress",
			CarrierName:    "YunExpress",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       85,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^YT\d{16,18}$`),
			Description:    "YunExpress cross-border parcel format (YT prefix)",
		},

		// 5. Cainiao / AliExpress
		{
			CarrierSlug:    "cainiao",
			CarrierName:    "Cainiao Global",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       85,
			ConfidenceBase: 0.95,
			Regex:          regexp.MustCompile(`^(LP\d{14,16}|CAINIAO\d+|CNESP\d+|UN\d{9}CN|SYB\d+|SYA\d+)$`),
			Description:    "Cainiao / AliExpress standard tracking (LP, CAINIAO, CNESP, UN...CN)",
		},

		// 6. 4PX
		{
			CarrierSlug:    "four-px",
			CarrierName:    "4PX Express",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       84,
			ConfidenceBase: 0.95,
			Regex:          regexp.MustCompile(`^(4PX\d{10,16}|MP\d{10,16}|FPX[0-9A-Z]+)$`),
			Description:    "4PX Express international tracking number",
		},

		// 7. Correos España (UPU S10 ending in ES)
		{
			CarrierSlug:    "correos-spain",
			CarrierName:    "Correos España",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypePostal,
			Priority:       85,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}ES$`),
			Validator:      validateUPUS10,
			Description:    "Correos España UPU S10 postal code (XX123456789ES)",
		},

		// 8. Correos España domestic barcode (PK, PQ, PA, 72 + 23 alphanumeric/digits)
		{
			CarrierSlug:    "correos-spain",
			CarrierName:    "Correos España",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypePostal,
			Priority:       82,
			ConfidenceBase: 0.94,
			Regex:          regexp.MustCompile(`^(PK|PQ|PA|72|00|ES)\w{21}$`),
			Description:    "Correos España 23-digit domestic parcel barcode",
		},

		// 9. Correos Express (16 digits starting with 69, 32, 00, 08)
		{
			CarrierSlug:    "correos-express",
			CarrierName:    "Correos Express",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       82,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^(69|32|00|08)\d{14}$`),
			Description:    "Correos Express 16-digit parcel tracking number",
		},

		// 10. Correos Express 23-digit format
		{
			CarrierSlug:    "correos-express",
			CarrierName:    "Correos Express",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       81,
			ConfidenceBase: 0.92,
			Regex:          regexp.MustCompile(`^(32|69)\d{21}$`),
			Description:    "Correos Express 23-digit shipment identifier",
		},

		// 11. NACEX specific prefix NX and 4/7 albarán (XXXX/XXXXXXX)
		{
			CarrierSlug:    "nacex",
			CarrierName:    "NACEX",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       83,
			ConfidenceBase: 0.94,
			Regex:          regexp.MustCompile(`^(NX[0-9A-Z]{6,14}|\d{4}/\d{7})$`),
			Description:    "NACEX tracking number (NX prefix or XXXX/XXXXXXX albarán)",
		},

		// 12. Envialia specific prefix ENV and 4/6 albarán (XXXX/XXXXXX)
		{
			CarrierSlug:    "envialia",
			CarrierName:    "Envialia",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       83,
			ConfidenceBase: 0.94,
			Regex:          regexp.MustCompile(`^(ENV[0-9A-Z]{6,14}|\d{4}/\d{6})$`),
			Description:    "Envialia tracking number (ENV prefix or XXXX/XXXXXX albarán)",
		},

		// 13. TIPSA specific prefix TIPSA / TIP
		{
			CarrierSlug:    "tipsa",
			CarrierName:    "TIPSA",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       82,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^(TIPSA[0-9A-Z]{5,14}|TIP[0-9A-Z]{6,14})$`),
			Description:    "TIPSA tracking code (TIPSA prefix)",
		},

		// 14. Zeleris specific prefixes ZEL and EXP
		{
			CarrierSlug:    "zeleris",
			CarrierName:    "Zeleris",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       82,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^(ZEL[0-9A-Z]{6,14}|EXP[0-9A-Z]{6,14})$`),
			Description:    "Zeleris tracking number (ZEL or EXP prefix)",
		},

		// 15. Ontime Transporte specific prefix ONT
		{
			CarrierSlug:    "ontime",
			CarrierName:    "Ontime Transporte",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       82,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^ONT[0-9A-Z]{6,14}$`),
			Description:    "Ontime Transporte tracking code (ONT prefix)",
		},

		// 16. Sending Transporte Urgente specific prefix SND
		{
			CarrierSlug:    "sending",
			CarrierName:    "Sending Transporte Urgente",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       82,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^SND[0-9A-Z]{6,14}$`),
			Description:    "Sending Transporte tracking code (SND prefix)",
		},

		// 17. Redur specific prefix R followed by digits / RDR
		{
			CarrierSlug:    "redur",
			CarrierName:    "Redur",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       82,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^(R\d{8,12}|RDR\d{7,12})$`),
			Description:    "Redur tracking reference (R prefix followed by digits)",
		},

		// 18. GLS Spain (ASM) prefix
		{
			CarrierSlug:    "gls-spain",
			CarrierName:    "GLS Spain",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       82,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^(ASM[0-9A-Z]{6,14}|GLSES[0-9A-Z]{6,14})$`),
			Description:    "GLS Spain / ASM parcel barcode",
		},

		// 19. CTT Express (Spain & Portugal)
		{
			CarrierSlug:    "ctt-express",
			CarrierName:    "CTT Express",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.92,
			Regex:          regexp.MustCompile(`^(000\d{19,21}|EXP\d{10,14}|CTT\d{10,14}|6\d{12,13})$`),
			Description:    "CTT Express tracking number (000..., EXP..., 6...)",
		},

		// 20. SEUR (14 to 16 digits)
		{
			CarrierSlug:    "seur",
			CarrierName:    "SEUR",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       78,
			ConfidenceBase: 0.88,
			Regex:          regexp.MustCompile(`^(SEUR\d{10,14}|36\d{12}|\d{14,16})$`),
			Description:    "SEUR Spain parcel number (14-16 digits)",
		},

		// 21. Paack Delivery
		{
			CarrierSlug:    "paack",
			CarrierName:    "Paack Logistics",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       78,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^(PCK|PK|PAACK)[0-9A-Z]{9,15}$`),
			Description:    "Paack Logistics parcel code (PCK/PAACK prefix)",
		},

		// 22. MRW (12 digits or 5 agency + 7 order)
		{
			CarrierSlug:    "mrw",
			CarrierName:    "MRW",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       77,
			ConfidenceBase: 0.87,
			Regex:          regexp.MustCompile(`^\d{5}/\d{7}$|^\d{12}$`),
			Description:    "MRW Spain 12-digit franchise parcel code",
		},

		// 23. NACEX 10 digits
		{
			CarrierSlug:    "nacex",
			CarrierName:    "NACEX",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       72,
			ConfidenceBase: 0.84,
			Regex:          regexp.MustCompile(`^\d{10}$`),
			Description:    "NACEX 10-digit domestic tracking code",
		},

		// 24. TIPSA 10-14 digits
		{
			CarrierSlug:    "tipsa",
			CarrierName:    "TIPSA",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       70,
			ConfidenceBase: 0.83,
			Regex:          regexp.MustCompile(`^\d{10,14}$`),
			Description:    "TIPSA 10-14 digit domestic barcode",
		},

		// 25. Zeleris 10-12 digits
		{
			CarrierSlug:    "zeleris",
			CarrierName:    "Zeleris",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       70,
			ConfidenceBase: 0.83,
			Regex:          regexp.MustCompile(`^\d{10,12}$`),
			Description:    "Zeleris 10-12 digit reference number",
		},

		// 26. Envialia 10-12 digits
		{
			CarrierSlug:    "envialia",
			CarrierName:    "Envialia",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       70,
			ConfidenceBase: 0.83,
			Regex:          regexp.MustCompile(`^\d{10,12}$`),
			Description:    "Envialia 10-12 digit service code",
		},

		// 27. Ontime 10-14 digits
		{
			CarrierSlug:    "ontime",
			CarrierName:    "Ontime Transporte",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       70,
			ConfidenceBase: 0.83,
			Regex:          regexp.MustCompile(`^\d{10,14}$`),
			Description:    "Ontime Transporte 10-14 digit waybill",
		},

		// 28. Sending 10-14 digits
		{
			CarrierSlug:    "sending",
			CarrierName:    "Sending Transporte Urgente",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       70,
			ConfidenceBase: 0.83,
			Regex:          regexp.MustCompile(`^\d{10,14}$`),
			Description:    "Sending Transporte 10-14 digit consignment",
		},

		// 29. Redur 10-12 digits
		{
			CarrierSlug:    "redur",
			CarrierName:    "Redur",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       70,
			ConfidenceBase: 0.83,
			Regex:          regexp.MustCompile(`^\d{10,12}$`),
			Description:    "Redur 10-12 digit parcel number",
		},

		// 30. GLS (8, 9, 11 or 12 digits, or GLS prefix)
		{
			CarrierSlug:    "gls",
			CarrierName:    "GLS",
			Country:        "GLOBAL",
			CountryName:    "Global",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       75,
			ConfidenceBase: 0.86,
			Regex:          regexp.MustCompile(`^(GLS[0-9A-Z]{8,14}|\d{8}|\d{11}|\d{12}|\d{20})$`),
			Description:    "GLS parcel tracking number (8, 11, 12 digits or GLS prefix)",
		},

		// 31. USPS 20-22 digits (94, 92, 93, 95...)
		{
			CarrierSlug:    "usps",
			CarrierName:    "USPS",
			Country:        "US",
			CountryName:    "United States",
			CarrierType:    models.CarrierTypePostal,
			Priority:       82,
			ConfidenceBase: 0.94,
			Regex:          regexp.MustCompile(`^(94|92|93|95|91)\d{18,20}$`),
			Description:    "USPS 20-22 digit domestic package barcode",
		},

		// 32. USPS UPU S10 (ending in US)
		{
			CarrierSlug:    "usps",
			CarrierName:    "USPS",
			Country:        "US",
			CountryName:    "United States",
			CarrierType:    models.CarrierTypePostal,
			Priority:       85,
			ConfidenceBase: 0.95,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}US$`),
			Validator:      validateUPUS10,
			Description:    "USPS International UPU S10 (e.g. EA123456789US)",
		},

		// 33. FedEx 12, 15, 20 digits or 96 prefix
		{
			CarrierSlug:    "fedex",
			CarrierName:    "FedEx",
			Country:        "US",
			CountryName:    "United States",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       76,
			ConfidenceBase: 0.88,
			Regex:          regexp.MustCompile(`^(\d{12}|\d{15}|\d{20}|96\d{20})$`),
			Description:    "FedEx Ground or Express tracking number (12, 15, or 20 digits)",
		},

		// 34. DHL Express 10 digits
		{
			CarrierSlug:    "dhl-express",
			CarrierName:    "DHL Express",
			Country:        "GLOBAL",
			CountryName:    "Global",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       76,
			ConfidenceBase: 0.88,
			Regex:          regexp.MustCompile(`^\d{10}$`),
			Description:    "DHL Express 10-digit Air Waybill / Tracking Number",
		},

		// 35. DHL eCommerce / Parcel (GM..., JJ..., JJD...)
		{
			CarrierSlug:    "dhl-parcel",
			CarrierName:    "DHL Parcel",
			Country:        "DE",
			CountryName:    "Germany",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       83,
			ConfidenceBase: 0.94,
			Regex:          regexp.MustCompile(`^(GM\d{16,18}|JJ\d{16}|JJD\d{16,18}|0034043\d{11})$`),
			Description:    "DHL eCommerce & Parcel international barcode",
		},

		// 36. Royal Mail (UK UPU S10 ending in GB)
		{
			CarrierSlug:    "royal-mail",
			CarrierName:    "Royal Mail",
			Country:        "GB",
			CountryName:    "United Kingdom",
			CarrierType:    models.CarrierTypePostal,
			Priority:       85,
			ConfidenceBase: 0.95,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}GB$`),
			Validator:      validateUPUS10,
			Description:    "Royal Mail International UPU S10 postal format",
		},

		// 37. Colissimo / La Poste France
		{
			CarrierSlug:    "colissimo",
			CarrierName:    "Colissimo / La Poste",
			Country:        "FR",
			CountryName:    "France",
			CarrierType:    models.CarrierTypePostal,
			Priority:       83,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^(6A|6M|6C|6H|9V)\d{11}$|^[A-Z]{2}\d{9}FR$`),
			Description:    "Colissimo / La Poste French postal tracking",
		},

		// 38. InPost / Mondial Relay
		{
			CarrierSlug:    "inpost",
			CarrierName:    "InPost / Mondial Relay",
			Country:        "GLOBAL",
			CountryName:    "Global",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       74,
			ConfidenceBase: 0.85,
			Regex:          regexp.MustCompile(`^(6\d{23}|\d{8}|\d{10})$`),
			Description:    "InPost Locker / Mondial Relay parcel number",
		},

		// 39. General UPU S10 International Postal
		{
			CarrierSlug:    "upu-s10",
			CarrierName:    "Universal Postal Union (UPU S10)",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypePostal,
			Priority:       60,
			ConfidenceBase: 0.80,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}[A-Z]{2}$`),
			Validator:      validateUPUS10,
			Description:    "UPU S10 Standard International Postal barcode",
		},
	}
}
