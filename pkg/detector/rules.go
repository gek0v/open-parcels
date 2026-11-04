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

		// 2. ISO 6346 Shipping Container (Preserved for backward compatibility)
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

		// 3. Evergreen Marine Bill of Lading (EGLV + 10-12 digits)
		{
			CarrierSlug:    "evergreen",
			CarrierName:    "Evergreen",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeOcean,
			Priority:       92,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^(EGLV|EISU|EGSU)\d{9,12}$`),
			Description:    "Evergreen Marine Bill of Lading (B/L)",
		},

		// 4. Maersk Line Bill of Lading (MAEU / MSK / MSKU + 9-12 digits)
		{
			CarrierSlug:    "maersk-line",
			CarrierName:    "Maersk Line",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeOcean,
			Priority:       92,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^(MAEU|MSK|MSKU)\d{9,12}$`),
			Description:    "Maersk Line Bill of Lading (B/L)",
		},

		// 5. MSC Bill of Lading (MEDU / MSCU + 8-12 chars, or MSC + 7-12 chars)
		{
			CarrierSlug:    "msc",
			CarrierName:    "MSC",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeOcean,
			Priority:       92,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^(MEDU|MSCU)[A-Z0-9]{8,12}$|^MSC[A-Z0-9]{7,12}$`),
			Description:    "MSC Mediterranean Shipping Company Bill of Lading (B/L)",
		},

		// 6. Hapag-Lloyd Bill of Lading (HLCU / HLXU + 8-12 chars, or HLD + 7-10 digits)
		{
			CarrierSlug:    "hapag-lloyd",
			CarrierName:    "Hapag Lloyd",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeOcean,
			Priority:       92,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^(HLCU|HLXU)[A-Z0-9]{8,12}$|^HLD\d{7,10}$`),
			Description:    "Hapag-Lloyd Bill of Lading (B/L)",
		},

		// 7. COSCO Shipping Bill of Lading (COSU / COAU / CCLU + 9-12 digits or 8-12 chars)
		{
			CarrierSlug:    "cosco",
			CarrierName:    "COSCO Shipping",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeOcean,
			Priority:       92,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^(COSU|COAU|CCLU)\d{9,12}$|^COSU[A-Z0-9]{8,12}$`),
			Description:    "COSCO Shipping Bill of Lading (B/L)",
		},

		// 8. CMA CGM Bill of Lading
		{
			CarrierSlug:    "cma-cgm",
			CarrierName:    "CMA CGM",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeOcean,
			Priority:       92,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^(CMAU|CGMU|ANLU)[A-Z0-9]{8,12}$|^NAM\d{7,10}$`),
			Description:    "CMA CGM Bill of Lading (B/L)",
		},

		// 9. ONE (Ocean Network Express) Bill of Lading
		{
			CarrierSlug:    "one-line",
			CarrierName:    "ONE Ocean Network Express",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeOcean,
			Priority:       92,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^(ONEU|ONEY)[A-Z0-9]{8,12}$`),
			Description:    "ONE Ocean Network Express Bill of Lading (B/L)",
		},

		// 10. Yang Ming Bill of Lading
		{
			CarrierSlug:    "yang-ming",
			CarrierName:    "Yang Ming",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeOcean,
			Priority:       92,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^YMLU[A-Z0-9]{8,12}$`),
			Description:    "Yang Ming Bill of Lading (B/L)",
		},

		// 11. HMM Bill of Lading
		{
			CarrierSlug:    "hmm",
			CarrierName:    "HMM",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeOcean,
			Priority:       92,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^HDMU[A-Z0-9]{8,12}$`),
			Description:    "HMM Bill of Lading (B/L)",
		},

		// 12. ZIM Bill of Lading
		{
			CarrierSlug:    "zim",
			CarrierName:    "ZIM",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeOcean,
			Priority:       92,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^ZIMU[A-Z0-9]{8,12}$`),
			Description:    "ZIM Bill of Lading (B/L)",
		},

		// 13. Dole Ocean Cargo Bill of Lading
		{
			CarrierSlug:    "dole-ocean-cargo",
			CarrierName:    "Dole Ocean Cargo",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeOcean,
			Priority:       92,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^DOLU[A-Z0-9]{8,12}$`),
			Description:    "Dole Ocean Cargo Bill of Lading (B/L)",
		},

		// 14. UPS standard 1Z
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

		// 15. YunExpress (YT + 16 or 18 digits)
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

		// 16. Cainiao / AliExpress
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

		// 17. 4PX
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

		// 18. Correos España (UPU S10 ending in ES)
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

		// 19. Correos España domestic barcode (PK, PQ, PA, 72 + 23 alphanumeric/digits)
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

		// 20. Correos Express (16 digits starting with 69, 32, 00, 08)
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

		// 21. Correos Express 23-digit format
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

		// 22. CTT Express (Spain & Portugal)
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

		// 23. SEUR (14 to 16 digits)
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

		// 24. Paack Delivery
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

		// 25. MRW (12 digits or 5 agency + 7 order)
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

		// 26. GLS (8, 9, 11 or 12 digits, or GLS prefix)
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

		// 27. USPS 20-22 digits (94, 92, 93, 95...)
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

		// 28. USPS UPU S10 (ending in US)
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

		// 29. FedEx 12, 15, 20 digits or 96 prefix
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

		// 30. DHL Express 10 digits
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

		// 31. DHL eCommerce / Parcel (GM..., JJ..., JJD...)
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

		// 32. Royal Mail (UK UPU S10 ending in GB)
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

		// 33. Colissimo / La Poste France
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

		// 34. InPost / Mondial Relay
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

		// 35. General UPU S10 International Postal
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
