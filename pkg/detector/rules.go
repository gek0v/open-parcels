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
			CarrierSlug:    "4px",
			CarrierName:    "4PX Express",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       84,
			ConfidenceBase: 0.95,
			Regex:          regexp.MustCompile(`^(4PX\d{10,16}|MP\d{10,16}|FPX[0-9A-Z]+)$`),
			Description:    "4PX Express international tracking number",
		},

		// 6a. China Post UPU S10
		{
			CarrierSlug:    "china-post",
			CarrierName:    "China Post",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypePostal,
			Priority:       85,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}CN$`),
			Validator:      validateUPUS10,
			Description:    "China Post UPU S10 tracking format",
		},

		// 6b. China Post Domestic 13-digit barcode
		{
			CarrierSlug:    "china-post",
			CarrierName:    "China Post",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypePostal,
			Priority:       79,
			ConfidenceBase: 0.88,
			Regex:          regexp.MustCompile(`^(10|11|98|99)\d{11}$`),
			Description:    "China Post domestic 13-digit barcode",
		},

		// 6c. Hong Kong Post UPU S10
		{
			CarrierSlug:    "hongkong-post",
			CarrierName:    "Hong Kong Post",
			Country:        "HK",
			CountryName:    "Hong Kong",
			CarrierType:    models.CarrierTypePostal,
			Priority:       85,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}HK$`),
			Validator:      validateUPUS10,
			Description:    "Hong Kong Post UPU S10 tracking format",
		},

		// 6d. SF Express (SF prefix)
		{
			CarrierSlug:    "sf-express",
			CarrierName:    "SF Express",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       86,
			ConfidenceBase: 0.95,
			Regex:          regexp.MustCompile(`^SF\d{12,14}$`),
			Description:    "SF Express standard tracking (SF prefix)",
		},

		// 6e. SF Express Domestic (15 digits)
		{
			CarrierSlug:    "sf-express",
			CarrierName:    "SF Express",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       75,
			ConfidenceBase: 0.82,
			Regex:          regexp.MustCompile(`^\d{15}$`),
			Description:    "SF Express domestic 15-digit barcode",
		},

		// 6f. Yanwen Logistics
		{
			CarrierSlug:    "yanwen",
			CarrierName:    "Yanwen Logistics",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       86,
			ConfidenceBase: 0.95,
			Regex:          regexp.MustCompile(`^((UD|VR|LP|UB|UV|YT)\d{9}YP|YW\d{11,15})$`),
			Description:    "Yanwen Logistics cross-border tracking format",
		},

		// 6g. SunYou / Sytrack
		{
			CarrierSlug:    "sytrack",
			CarrierName:    "SunYou - Sytrack",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       85,
			ConfidenceBase: 0.95,
			Regex:          regexp.MustCompile(`^(SY\d{11,13}|SY[A-Z]{2,4}\d{8,14}|LP\d{14}SY)$`),
			Description:    "SunYou / Sytrack tracking format",
		},

		// 6h. CNE Express
		{
			CarrierSlug:    "cne-express",
			CarrierName:    "CNE Express",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       85,
			ConfidenceBase: 0.94,
			Regex:          regexp.MustCompile(`^(3A[0-9A-Z]{10,16}|CNE\d{10,14})$`),
			Description:    "CNE Express cross-border tracking format",
		},

		// 6i. YTO Express
		{
			CarrierSlug:    "yto-express",
			CarrierName:    "YTO Express",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       82,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^(YT\d{10,13}|DD\d{10,13}|8\d{11,13})$`),
			Description:    "YTO Express domestic/international barcode",
		},

		// 6j. ZTO International / Asia
		{
			CarrierSlug:    "zto-international",
			CarrierName:    "ZTO International",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       82,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^(ZTO\d{10,14}|7\d{11,13})$`),
			Description:    "ZTO Express / International tracking number",
		},

		// 6k. Deppon Logistics
		{
			CarrierSlug:    "deppon",
			CarrierName:    "Deppon",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       84,
			ConfidenceBase: 0.92,
			Regex:          regexp.MustCompile(`^DPK\d{11,14}$`),
			Description:    "Deppon Logistics DPK barcode",
		},

		// 6l. JD Logistics
		{
			CarrierSlug:    "jdl-cn",
			CarrierName:    "JD Logistics",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       83,
			ConfidenceBase: 0.91,
			Regex:          regexp.MustCompile(`^(JD\d{10,14}|JDX\d{10,14})$`),
			Description:    "JD Logistics tracking number",
		},

		// 6m. Cirro
		{
			CarrierSlug:    "cirro",
			CarrierName:    "Cirro",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       83,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^(CIR\d{10,14}|LL\d{9}CN)$`),
			Description:    "Cirro cross-border tracking format",
		},

		// 6n. SendFromChina (SFC)
		{
			CarrierSlug:    "sfc-service",
			CarrierName:    "SFC - SendFromChina",
			Country:        "CN",
			CountryName:    "China",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       82,
			ConfidenceBase: 0.89,
			Regex:          regexp.MustCompile(`^SFC[0-9A-Z]{10,16}$`),
			Description:    "SendFromChina tracking number",
		},

		// 6o. CXC Express HK
		{
			CarrierSlug:    "cxc-express-hk",
			CarrierName:    "CXC Express Hong Kong",
			Country:        "HK",
			CountryName:    "Hong Kong",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       82,
			ConfidenceBase: 0.89,
			Regex:          regexp.MustCompile(`^CXC\d{10,14}$`),
			Description:    "CXC Express Hong Kong tracking number",
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

		// 11. CTT Express (Spain & Portugal)
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

		// 12. SEUR (14 to 16 digits)
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

		// 13. Paack Delivery
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

		// 14. MRW (12 digits or 5 agency + 7 order)
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

		// 15. GLS (8, 9, 11 or 12 digits, or GLS prefix)
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

		// 16. USPS 20-22 digits (94, 92, 93, 95...)
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

		// 17. USPS UPU S10 (ending in US)
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

		// 18. FedEx 12, 15, 20 digits or 96 prefix
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

		// 19. DHL Express 10 digits
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

		// 20. DHL eCommerce / Parcel (GM..., JJ..., JJD...)
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

		// 21. Royal Mail (UK UPU S10 ending in GB)
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

		// 22. Colissimo / La Poste France
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

		// 23. InPost / Mondial Relay
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

		// 24. General UPU S10 International Postal
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
