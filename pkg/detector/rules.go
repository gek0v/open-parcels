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
			CarrierSlug:    "correosexpress",
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
			CarrierSlug:    "correosexpress",
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
			Regex:          regexp.MustCompile(`^(SEUR\d{10,14}|36\d{12})$`),
			Description:    "SEUR Spain parcel number",
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
			CarrierSlug:    "mrw-spain",
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
			Priority:       78,
			ConfidenceBase: 0.90,
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

		// Italy and San Marino rules (Issue #9)
		{
			CarrierSlug:    "poste-italiane",
			CarrierName:    "Poste Italiane",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypePostal,
			Priority:       86,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}(IT|SM)$`),
			Validator:      validateUPUS10,
			Description:    "Poste Italiane / San Marino UPU S10 postal code (XX123456789IT/SM)",
		},
		{
			CarrierSlug:    "poste-italiane",
			CarrierName:    "Poste Italiane",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypePostal,
			Priority:       84,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^(CRONO\d{8,14}|(38|05|20)\d{10})$`),
			Description:    "Poste Italiane domestic Crono or raccomandata parcel barcode",
		},
		{
			CarrierSlug:    "brt-bartolini",
			CarrierName:    "BRT Bartolini",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.88,
			Regex:          regexp.MustCompile(`^(BRT\d{9,12}|01\d{10}|\d{12})$`),
			Description:    "BRT Bartolini 12-digit shipment code",
		},
		{
			CarrierSlug:    "brt-italy-parcelid",
			CarrierName:    "BRT Bartolini (Parcel ID)",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       83,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^(055\d{11}|055\d{15}|\d{14}|\d{18})$`),
			Description:    "BRT Bartolini 14 or 18-digit Parcel ID (ID Collo)",
		},
		{
			CarrierSlug:    "mybrt-it",
			CarrierName:    "myBRT.it",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       83,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^(MYBRT[0-9A-Z]{8,14}|074\d{11})$`),
			Description:    "myBRT.it tracking code (MYBRT... or 074...)",
		},
		{
			CarrierSlug:    "gls-italy",
			CarrierName:    "GLS Italy",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       82,
			ConfidenceBase: 0.91,
			Regex:          regexp.MustCompile(`^(GLSIT[0-9A-Z]+|[A-Z]\d{8,9}|\d{9})$`),
			Description:    "GLS Italy parcel tracking number (GLSIT prefix, branch code or 9 digits)",
		},
		{
			CarrierSlug:    "inpost-it",
			CarrierName:    "InPost.it",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       80,
			ConfidenceBase: 0.91,
			Regex:          regexp.MustCompile(`^(IT\d{22}|6\d{23})$`),
			Description:    "InPost.it Locker parcel tracking number",
		},
		{
			CarrierSlug:    "tnt-italy",
			CarrierName:    "TNT Italy",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       82,
			ConfidenceBase: 0.92,
			Regex:          regexp.MustCompile(`^(TNTIT\d+|(MY|GE|VV|BA|NA)\d{7,9})$`),
			Description:    "TNT Italy consignment tracking number",
		},
		{
			CarrierSlug:    "amazon-it",
			CarrierName:    "Amazon Italy",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       83,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^(IT\d{10,12}|TB[A-Z0-9]{10,16})$`),
			Description:    "Amazon Italy tracking barcode",
		},
		{
			CarrierSlug:    "sailpost",
			CarrierName:    "Sailpost",
			Country:        "IT",
			CountryName:    "Italy & San Marino",
			CarrierType:    models.CarrierTypePostal,
			Priority:       83,
			ConfidenceBase: 0.92,
			Regex:          regexp.MustCompile(`^(SP\d{9,12}|SLP\d{9,12})$`),
			Description:    "Sailpost Italy & San Marino postal barcode",
		},
		{
			CarrierSlug:    "alfred-it",
			CarrierName:    "Alfred 24",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       83,
			ConfidenceBase: 0.92,
			Regex:          regexp.MustCompile(`^ALF[0-9A-Z]{8,14}$`),
			Description:    "Alfred 24 smart locker parcel identifier",
		},
		{
			CarrierSlug:    "arco-spedizioni",
			CarrierName:    "Arco Spedizioni",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeFreight,
			Priority:       82,
			ConfidenceBase: 0.91,
			Regex:          regexp.MustCompile(`^(ARC|ARCO)\d{6,10}$`),
			Description:    "Arco Spedizioni freight tracking number",
		},
		{
			CarrierSlug:    "ceva-it",
			CarrierName:    "CEVA Logistics Italy",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeFreight,
			Priority:       82,
			ConfidenceBase: 0.91,
			Regex:          regexp.MustCompile(`^(CEVA|ITCEVA)\d{6,12}$`),
			Description:    "CEVA Logistics Italy consignment number",
		},
		{
			CarrierSlug:    "bdm-it",
			CarrierName:    "CST Italia",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       82,
			ConfidenceBase: 0.91,
			Regex:          regexp.MustCompile(`^(BDM|CST)\d{6,10}$`),
			Description:    "CST Italia / BDM shipment tracking code",
		},
		{
			CarrierSlug:    "deliver-it",
			CarrierName:    "Deliver It",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       82,
			ConfidenceBase: 0.91,
			Regex:          regexp.MustCompile(`^(DIT|DLV)[0-9A-Z]{7,12}$`),
			Description:    "Deliver It courier tracking code",
		},
		{
			CarrierSlug:    "liccardi-it",
			CarrierName:    "Liccardi Trasporti",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeFreight,
			Priority:       82,
			ConfidenceBase: 0.91,
			Regex:          regexp.MustCompile(`^(LIC|LCC)\d{6,10}$`),
			Description:    "Liccardi Trasporti freight tracking code",
		},
		{
			CarrierSlug:    "mbe-it",
			CarrierName:    "Mail Boxes Etc. Italy",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       82,
			ConfidenceBase: 0.91,
			Regex:          regexp.MustCompile(`^(MBE|MBEIT)\d{7,12}$`),
			Description:    "Mail Boxes Etc. Italy parcel tracking code",
		},
		{
			CarrierSlug:    "rhenus-it",
			CarrierName:    "Rhenus Italia",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeFreight,
			Priority:       82,
			ConfidenceBase: 0.91,
			Regex:          regexp.MustCompile(`^(RH|RHIT)\d{7,12}$`),
			Description:    "Rhenus Italia logistics tracking code",
		},
		{
			CarrierSlug:    "spediamo-it",
			CarrierName:    "Spediamo",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       82,
			ConfidenceBase: 0.91,
			Regex:          regexp.MustCompile(`^(SPD|SPED)\d{6,10}$`),
			Description:    "Spediamo.it online parcel tracking code",
		},
		{
			CarrierSlug:    "unieuro-it",
			CarrierName:    "Unieuro",
			Country:        "IT",
			CountryName:    "Italy",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       82,
			ConfidenceBase: 0.91,
			Regex:          regexp.MustCompile(`^(UNI|UNIEURO)\d{6,10}$`),
			Description:    "Unieuro order and delivery tracking code",
		},

		// Portugal rules (Issue #17)
		{
			CarrierSlug:    "portugal-post",
			CarrierName:    "Portugal Post - CTT",
			Country:        "PT",
			CountryName:    "Portugal",
			CarrierType:    models.CarrierTypePostal,
			Priority:       85,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}PT$`),
			Validator:      validateUPUS10,
			Description:    "Portugal Post CTT UPU S10 postal format (XX123456789PT)",
		},
		{
			CarrierSlug:    "portugal-post",
			CarrierName:    "Portugal Post - CTT",
			Country:        "PT",
			CountryName:    "Portugal",
			CarrierType:    models.CarrierTypePostal,
			Priority:       82,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^(DA|RA|LX|EM|CO|RH)\d{9}PT$`),
			Validator:      validateUPUS10,
			Description:    "Portugal Post domestic express/registered barcode",
		},
		{
			CarrierSlug:    "dpd-portugal",
			CarrierName:    "DPD Portugal",
			Country:        "PT",
			CountryName:    "Portugal",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^(098|099|058)\d{11}$|^098\d{8}$`),
			Description:    "DPD Portugal parcel tracking format (098, 099, 058)",
		},
		{
			CarrierSlug:    "delnext",
			CarrierName:    "Delnext",
			Country:        "PT",
			CountryName:    "Portugal",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^(DLN|DLX)[0-9A-Z]{7,13}$|^DLN\d{8,12}$`),
			Description:    "Delnext Portugal tracking number (DLN/DLX prefix)",
		},
		{
			CarrierSlug:    "wepickup-pt",
			CarrierName:    "WePickUp",
			Country:        "PT",
			CountryName:    "Portugal",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       82,
			ConfidenceBase: 0.92,
			Regex:          regexp.MustCompile(`^WPU[0-9A-Z]{8,14}$`),
			Description:    "WePickUp locker & PUDO network tracking number (WPU prefix)",
		},
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

		// 39. Poland Post (Poczta Polska) UPU S10
		{
			CarrierSlug:    "poland-post",
			CarrierName:    "Poland Post",
			Country:        "PL",
			CountryName:    "Poland",
			CarrierType:    models.CarrierTypePostal,
			Priority:       86,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}PL$`),
			Validator:      validateUPUS10,
			Description:    "Poland Post (Poczta Polska) UPU S10 format",
		},

		// 40. InPost Poland 24-digit barcode
		{
			CarrierSlug:    "inpost-poland",
			CarrierName:    "InPost Poland",
			Country:        "PL",
			CountryName:    "Poland",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       87,
			ConfidenceBase: 0.97,
			Regex:          regexp.MustCompile(`^6\d{23}$`),
			Description:    "InPost Poland 24-digit locker parcel number",
		},

		// 41. Czech Republic Post UPU S10
		{
			CarrierSlug:    "czech-republic-post",
			CarrierName:    "Czech Post",
			Country:        "CZ",
			CountryName:    "Czech Republic",
			CarrierType:    models.CarrierTypePostal,
			Priority:       86,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}CZ$`),
			Validator:      validateUPUS10,
			Description:    "Czech Post UPU S10 tracking format",
		},

		// 42. Romania Post UPU S10
		{
			CarrierSlug:    "romania-post",
			CarrierName:    "Romania Post",
			Country:        "RO",
			CountryName:    "Romania",
			CarrierType:    models.CarrierTypePostal,
			Priority:       86,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}RO$`),
			Validator:      validateUPUS10,
			Description:    "Romania Post UPU S10 tracking format",
		},

		// 43. Greece Post (ELTA) UPU S10
		{
			CarrierSlug:    "elta-hellenic-post",
			CarrierName:    "Greece Post (ELTA)",
			Country:        "GR",
			CountryName:    "Greece",
			CarrierType:    models.CarrierTypePostal,
			Priority:       86,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}GR$`),
			Validator:      validateUPUS10,
			Description:    "Greece Post (ELTA) UPU S10 format",
		},

		// 44. Bulgaria Post UPU S10
		{
			CarrierSlug:    "bulgaria-post",
			CarrierName:    "Bulgaria Post",
			Country:        "BG",
			CountryName:    "Bulgaria",
			CarrierType:    models.CarrierTypePostal,
			Priority:       86,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}BG$`),
			Validator:      validateUPUS10,
			Description:    "Bulgaria Post UPU S10 tracking format",
		},

		// 45. Slovakia Post UPU S10
		{
			CarrierSlug:    "slovakia-post",
			CarrierName:    "Slovakia Post",
			Country:        "SK",
			CountryName:    "Slovakia",
			CarrierType:    models.CarrierTypePostal,
			Priority:       86,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}SK$`),
			Validator:      validateUPUS10,
			Description:    "Slovakia Post UPU S10 tracking format",
		},

		// 46. Sameday Romania
		{
			CarrierSlug:    "sameday-ro",
			CarrierName:    "Sameday",
			Country:        "RO",
			CountryName:    "Romania",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       85,
			ConfidenceBase: 0.94,
			Regex:          regexp.MustCompile(`^(00|10)\d{12,16}$`),
			Description:    "Sameday courier parcel barcode",
		},

		// 47. Fan Courier
		{
			CarrierSlug:    "fan-courier",
			CarrierName:    "Fan Courier",
			Country:        "RO",
			CountryName:    "Romania",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       85,
			ConfidenceBase: 0.94,
			Regex:          regexp.MustCompile(`^(00|01|02)\d{10,14}$`),
			Description:    "Fan Courier Romania waybill number",
		},

		// 31. Australia Post UPU S10
		{
			CarrierSlug:    "australia-post",
			CarrierName:    "Australia Post",
			Country:        "AU",
			CountryName:    "Australia",
			CarrierType:    models.CarrierTypePostal,
			Priority:       86,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}AU$`),
			Validator:      validateUPUS10,
			Description:    "Australia Post UPU S10 tracking format",
		},

		// 32. Japan Post UPU S10
		{
			CarrierSlug:    "japan-post",
			CarrierName:    "Japan Post",
			Country:        "JP",
			CountryName:    "Japan",
			CarrierType:    models.CarrierTypePostal,
			Priority:       86,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}JP$`),
			Validator:      validateUPUS10,
			Description:    "Japan Post UPU S10 tracking format",
		},

		// 33. Korea Post UPU S10
		{
			CarrierSlug:    "korea-post",
			CarrierName:    "Korea Post",
			Country:        "KR",
			CountryName:    "South Korea",
			CarrierType:    models.CarrierTypePostal,
			Priority:       86,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}KR$`),
			Validator:      validateUPUS10,
			Description:    "Korea Post UPU S10 tracking format",
		},

		// 34. India Post UPU S10
		{
			CarrierSlug:    "india-post",
			CarrierName:    "India Post",
			Country:        "IN",
			CountryName:    "India",
			CarrierType:    models.CarrierTypePostal,
			Priority:       86,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}IN$`),
			Validator:      validateUPUS10,
			Description:    "India Post UPU S10 tracking format",
		},

		// 35. Singapore Post UPU S10
		{
			CarrierSlug:    "singapore-post",
			CarrierName:    "Singapore Post",
			Country:        "SG",
			CountryName:    "Singapore",
			CarrierType:    models.CarrierTypePostal,
			Priority:       86,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}SG$`),
			Validator:      validateUPUS10,
			Description:    "Singapore Post UPU S10 tracking format",
		},

		// 36. New Zealand Post UPU S10
		{
			CarrierSlug:    "new-zealand-post",
			CarrierName:    "New Zealand Post",
			Country:        "NZ",
			CountryName:    "New Zealand",
			CarrierType:    models.CarrierTypePostal,
			Priority:       86,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^[A-Z]{2}\d{9}NZ$`),
			Validator:      validateUPUS10,
			Description:    "New Zealand Post UPU S10 tracking format",
		},

		// 37. Ninja Van
		{
			CarrierSlug:    "ninja-van",
			CarrierName:    "Ninja Van",
			Country:        "SG",
			CountryName:    "Singapore",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       85,
			ConfidenceBase: 0.94,
			Regex:          regexp.MustCompile(`^(NIN|NINJA|NVAU|NVSG|NVPH|NVMY)[0-9A-Z]{8,16}$`),
			Description:    "Ninja Van southeast asia parcel format",
		},

		// 38. Delhivery
		{
			CarrierSlug:    "delhivery",
			CarrierName:    "Delhivery",
			Country:        "IN",
			CountryName:    "India",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       84,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^(DEL|DLV)\d{10,14}$`),
			Description:    "Delhivery express barcode",
		},

		// Global Cross-Border carriers (Issue #2)
		{
			CarrierSlug:    "speedpak",
			CarrierName:    "SpeedPAK",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       87,
			ConfidenceBase: 0.96,
			Regex:          regexp.MustCompile(`^(EE|ES)\d{16}$`),
			Description:    "SpeedPAK Orange Connex tracking format (EE/ES prefix)",
		},
		{
			CarrierSlug:    "pitney-bowes",
			CarrierName:    "Pitney Bowes",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       86,
			ConfidenceBase: 0.95,
			Regex:          regexp.MustCompile(`^(UPAAB|PB|PBI)\d{10,16}$`),
			Description:    "Pitney Bowes Global Shipping Program barcode",
		},
		{
			CarrierSlug:    "aramex",
			CarrierName:    "Aramex",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       74,
			ConfidenceBase: 0.86,
			Regex:          regexp.MustCompile(`^(AWB\d{8,11}|\d{11})$`),
			Description:    "Aramex international waybill barcode",
		},
		{
			CarrierSlug:    "landmark-global",
			CarrierName:    "Landmark Global",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       85,
			ConfidenceBase: 0.94,
			Regex:          regexp.MustCompile(`^(LTN|LDN)\d{8,12}$`),
			Description:    "Landmark Global tracking format",
		},
		{
			CarrierSlug:    "wanbexpress",
			CarrierName:    "WANBExpress",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       85,
			ConfidenceBase: 0.94,
			Regex:          regexp.MustCompile(`^WNB[0-9A-Z]{10,16}$`),
			Description:    "WANBExpress cross-border format",
		},
		{
			CarrierSlug:    "winit",
			CarrierName:    "Winit",
			Country:        "GLOBAL",
			CountryName:    "International",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       84,
			ConfidenceBase: 0.93,
			Regex:          regexp.MustCompile(`^(ID|WN)\d{10,16}$`),
			Description:    "Winit cross-border tracking number",
		},

		// --- Spanish Carriers (Issue #7) ---
		// 25. Nacex Spain
		{
			CarrierSlug:    "nacex-spain",
			CarrierName:    "Nacex",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^(NCX[0-9A-Z]{7,12}|\d{4}\d{7})$`),
			Description:    "Nacex Spain tracking number (NCX... or 11 digits)",
		},

		// 26. TIPSA Spain
		{
			CarrierSlug:    "tipsa",
			CarrierName:    "TIPSA",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^TIP[0-9A-Z]{7,12}$`),
			Description:    "TIPSA Spain parcel tracking code",
		},

		// 27. Zeleris
		{
			CarrierSlug:    "zeleris",
			CarrierName:    "Zeleris",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^ZEL[0-9A-Z]{7,12}$`),
			Description:    "Zeleris express tracking number",
		},

		// 28. Sending Spain
		{
			CarrierSlug:    "sending-es",
			CarrierName:    "Sending",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^SND[0-9A-Z]{7,12}$`),
			Description:    "Sending Spain parcel code",
		},

		// 29. Celeritas Transporte
		{
			CarrierSlug:    "celeritas",
			CarrierName:    "Celeritas Transporte",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^CEL[0-9A-Z]{8,14}$`),
			Description:    "Celeritas tracking code",
		},

		// 30. Envialia
		{
			CarrierSlug:    "envialia",
			CarrierName:    "Envialia",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^(ENV[0-9A-Z]{7,12}|010\d{8,10})$`),
			Description:    "Envialia parcel tracking number",
		},

		// 31. GLS Spain (ASM)
		{
			CarrierSlug:    "asm-gls-spain",
			CarrierName:    "GLS Spain (ASM)",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       81,
			ConfidenceBase: 0.91,
			Regex:          regexp.MustCompile(`^ASM[0-9A-Z]{8,14}$`),
			Description:    "GLS Spain / ASM express barcode",
		},

		// 32. Ontime
		{
			CarrierSlug:    "ontime",
			CarrierName:    "Ontime",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^ONT[0-9A-Z]{7,12}$`),
			Description:    "Ontime parcel tracking number",
		},

		// 33. Packlink
		{
			CarrierSlug:    "packlink",
			CarrierName:    "Packlink",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^(ES\d{10,12}[A-Z]{0,2}|PL[0-9A-Z]{8,14})$`),
			Description:    "Packlink tracking code",
		},

		// 34. Asendia Spain
		{
			CarrierSlug:    "asendia-es",
			CarrierName:    "Asendia Spain",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeCrossBorder,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^(AE\d{9,16}|EPA\d{9,16})$`),
			Description:    "Asendia Spain cross-border parcel number",
		},

		// 35. Asigna
		{
			CarrierSlug:    "asigna-es",
			CarrierName:    "Asigna",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^ASG[0-9A-Z]{7,12}$`),
			Description:    "Asigna parcel tracking code",
		},

		// 36. Activos24
		{
			CarrierSlug:    "activos24",
			CarrierName:    "Activos24",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^ACT[0-9A-Z]{7,12}$`),
			Description:    "Activos24 shipment tracking number",
		},

		// 37. Instapack
		{
			CarrierSlug:    "instapack-es",
			CarrierName:    "Instapack",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^IPK[0-9A-Z]{7,12}$`),
			Description:    "Instapack courier parcel number",
		},

		// 38. Mail Boxes Etc. Spain
		{
			CarrierSlug:    "mbe-es",
			CarrierName:    "Mail Boxes Etc. Spain",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeExpress,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^MBE[0-9A-Z]{7,12}$`),
			Description:    "Mail Boxes Etc. Spain parcel code",
		},

		// 39. Redur Spain
		{
			CarrierSlug:    "redur-spain",
			CarrierName:    "Redur Spain",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^RDR[0-9A-Z]{7,12}$`),
			Description:    "Redur Spain standard albaran code",
		},

		// 40. Redur Spain Reference
		{
			CarrierSlug:    "redur-spain-ref",
			CarrierName:    "Redur Spain Reference",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^REF[0-9A-Z]{7,12}$`),
			Description:    "Redur Spain customer reference format",
		},

		// 41. Redur Tracking
		{
			CarrierSlug:    "redur-tracking",
			CarrierName:    "Redur Tracking",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       80,
			ConfidenceBase: 0.90,
			Regex:          regexp.MustCompile(`^RD\d{10,12}$`),
			Description:    "Redur standard tracking format",
		},

		// 42. InPost Spain
		{
			CarrierSlug:    "inpost-es",
			CarrierName:    "InPost Spain",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       76,
			ConfidenceBase: 0.86,
			Regex:          regexp.MustCompile(`^6\d{23}$`),
			Description:    "InPost Spain 24-digit locker barcode",
		},

		// 43. DHL Parcel Spain
		{
			CarrierSlug:    "dhl-parcel-es",
			CarrierName:    "DHL Parcel Spain",
			Country:        "ES",
			CountryName:    "Spain",
			CarrierType:    models.CarrierTypeLogistics,
			Priority:       83,
			ConfidenceBase: 0.94,
			Regex:          regexp.MustCompile(`^(JJD\d{16,18}|JJ\d{16}|0034043\d{11})$`),
			Description:    "DHL Parcel Spain barcode format",
		},
	}
}
