package detector

import (
	"testing"
)

func TestDetector(t *testing.T) {
	d := New()

	tests := []struct {
		name         string
		number       string
		destCountry  string
		expectedSlug string
	}{
		{"UPS standard 1Z", "1Z9999999999999999", "US", "ups"},
		{"YunExpress YT", "YT2401234567890123", "", "yunexpress"},
		{"Cainiao LP", "LP00561234567890", "ES", "cainiao"},
		{"Cainiao CNESP", "CNESP0012345678", "ES", "cainiao"},
		{"Correos Express 16 digits", "6900123456789012", "ES", "correosexpress"},
		{"CTT Express", "0001234567890123456789", "ES", "ctt-express"},
		{"SEUR 14 digits", "36150127630406", "ES", "seur"},
		{"USPS 22 digits", "9400111206213785678901", "US", "usps"},
		{"FedEx 12 digits", "123456789012", "US", "fedex"},
		{"DHL Express 10 digits", "1234567890", "", "dhl-express"},
		{"DHL Parcel GM", "GM5551234567890123", "DE", "dhl-parcel"},
		{"DHL Parcel Spain", "JJD0000123456789012", "ES", "dhl-parcel-es"},
		{"Air Cargo AWB", "157-52471764", "", "air-cargo"},
		{"Ocean Container", "MRKU4656243", "", "ocean-container"},
		{"Paack", "PCK98765432101", "ES", "paack"},
		{"Correos Spain Domestic", "PQ4F6P07000000000000000", "ES", "correos-spain"},
		{"Royal Mail UPU", "RN123456789GB", "GB", "royal-mail"},
		{"Colissimo", "6A12345678901", "FR", "colissimo"},
		{"SpeedPAK", "EE1234567890123456", "", "speedpak"},
		{"Pitney Bowes", "UPAAB123456789012", "", "pitney-bowes"},
		{"Aramex", "AWB12345678", "", "aramex"},
		{"Landmark Global", "LTN123456789", "", "landmark-global"},
		{"WanbExpress", "WNB123456789012", "", "wanbexpress"},
		{"Winit", "ID123456789012", "", "winit"},
		{"China Post UPU S10", "EA123456785CN", "", "china-post"},
		{"China Post Domestic", "1101234567890", "", "china-post"},
		{"Hong Kong Post UPU S10", "RC123456785HK", "", "hongkong-post"},
		{"SF Express SF prefix", "SF123456789012", "", "sf-express"},
		{"Yanwen Express YP", "UD123456789YP", "", "yanwen"},
		{"SunYou SY", "SY12345678901", "", "sytrack"},
		{"CNE Express 3A", "3A5V123456789", "", "cne-express"},
		{"YTO Express YT", "YT1234567890", "", "yto-express"},
		{"Deppon DPK", "DPK12345678901", "", "deppon"},
		{"JD Logistics", "JD001234567890", "", "jdl-cn"},
		{"4PX Express", "4PX123456789012", "", "4px"},
		{"Cirro", "CIR12345678901", "", "cirro"},
		// Italy and San Marino formats
		{"Poste Italiane UPU IT", "RA123456785IT", "IT", "poste-italiane"},
		{"Poste Italiane UPU SM", "RR123456785SM", "SM", "poste-italiane"},
		{"Poste Italiane Crono", "CRONO123456789", "IT", "poste-italiane"},
		{"BRT Bartolini 12 digits", "012345678901", "IT", "brt-bartolini"},
		{"BRT Parcel ID", "05512345678901", "IT", "brt-italy-parcelid"},
		{"myBRT.it", "MYBRT123456789", "IT", "mybrt-it"},
		{"GLS Italy", "GLSIT123456789", "IT", "gls-italy"},
		{"InPost Italy", "623456789012345678901234", "IT", "inpost-it"},
		{"TNT Italy", "TNTIT123456789", "IT", "tnt-italy"},
		{"Amazon Italy", "IT1234567890", "IT", "amazon-it"},
		{"Sailpost Italy San Marino", "SP1234567890", "SM", "sailpost"},
		{"Alfred 24 Italy", "ALF123456789", "IT", "alfred-it"},
		{"Arco Spedizioni", "ARC12345678", "IT", "arco-spedizioni"},
		{"CEVA Italy", "CEVA123456789", "IT", "ceva-it"},
		{"CST Italia", "BDM12345678", "IT", "bdm-it"},
		{"Deliver It", "DIT123456789", "IT", "deliver-it"},
		{"Liccardi Trasporti", "LIC12345678", "IT", "liccardi-it"},
		{"Mail Boxes Etc Italy", "MBE123456789", "IT", "mbe-it"},
		{"Rhenus Italia", "RH123456789", "IT", "rhenus-it"},
		{"Spediamo", "SPD12345678", "IT", "spediamo-it"},
		{"Unieuro", "UNI123456789", "IT", "unieuro-it"},
		{"Nacex Spain", "NCX123456789", "ES", "nacex-spain"},
		{"TIPSA", "TIP123456789", "ES", "tipsa"},
		{"Zeleris", "ZEL123456789", "ES", "zeleris"},
		{"Sending", "SND123456789", "ES", "sending-es"},
		{"Celeritas", "CEL123456789", "ES", "celeritas"},
		{"Envialia", "ENV123456789", "ES", "envialia"},
		{"GLS Spain (ASM)", "ASM123456789", "ES", "asm-gls-spain"},
		{"Ontime", "ONT123456789", "ES", "ontime"},
		{"Packlink", "PL123456789", "ES", "packlink"},
		{"Asendia Spain", "AE123456789", "ES", "asendia-es"},
		{"Asigna", "ASG123456789", "ES", "asigna-es"},
		{"Activos24", "ACT123456789", "ES", "activos24"},
		{"Instapack", "IPK123456789", "ES", "instapack-es"},
		{"Mail Boxes Etc.", "MBE123456789", "ES", "mbe-es"},
		{"Redur Spain", "RDR123456789", "ES", "redur-spain"},
		{"Redur Spain Ref", "REF123456789", "ES", "redur-spain-ref"},
		{"Redur Tracking", "RD1234567890", "ES", "redur-tracking"},
		{"InPost Spain", "612345678901234567890123", "ES", "inpost-es"},
		{"MRW Spain 12 digits", "123456789012", "ES", "mrw-spain"},
		{"Portugal Post CTT UPU", "DA123456785PT", "PT", "portugal-post"},
		{"DPD Portugal", "09801234567890", "PT", "dpd-portugal"},
		{"Delnext", "DLN123456789PT", "PT", "delnext"},
		{"WePickUp", "WPU123456789", "PT", "wepickup-pt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := d.BestMatch(tt.number, tt.destCountry)
			if res == nil {
				t.Fatalf("expected match for %s, got nil", tt.number)
			}
			if res.Carrier.Slug != tt.expectedSlug {
				t.Errorf("expected slug %s, got %s (reason: %s)", tt.expectedSlug, res.Carrier.Slug, res.Reason)
			}
		})
	}
}

func TestDetectorCandidates(t *testing.T) {
	d := New()
	// A 12-digit number could be FedEx, MRW, GLS
	candidates := d.Detect("123456789012", "ES")
	if len(candidates) < 2 {
		t.Fatalf("expected multiple candidates for ambiguous 12-digit number, got %d", len(candidates))
	}
	// For ES hint, MRW or Spanish carriers should be prioritized or present
	foundMRW := false
	for _, c := range candidates {
		if c.Carrier.Slug == "mrw-spain" || c.Carrier.Slug == "mrw" {
			foundMRW = true
			break
		}
	}
	if !foundMRW {
		t.Errorf("expected mrw-spain in candidates for 12 digits with ES hint")
	}

	// For IT hint, brt-bartolini should be prioritized
	itCandidates := d.Detect("012345678901", "IT")
	if len(itCandidates) == 0 {
		t.Fatalf("expected candidates for 12 digits with IT hint")
	}
	if itCandidates[0].Carrier.Slug != "brt-bartolini" {
		t.Errorf("expected top candidate to be brt-bartolini with IT hint, got %s", itCandidates[0].Carrier.Slug)
	}

	// For SM hint, Poste Italiane / San Marino should be boosted
	smCandidates := d.Detect("RR123456785SM", "SM")
	if len(smCandidates) == 0 {
		t.Fatalf("expected candidates for UPU SM with SM hint")
	}
	if smCandidates[0].Carrier.Slug != "poste-italiane" {
		t.Errorf("expected top candidate to be poste-italiane for SM UPU, got %s", smCandidates[0].Carrier.Slug)
	}
}
