// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 gv-factory contributors

package carriers

import (
	"context"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

// oceanParams encapsulates configuration parameters for mocking or parsing ocean voyages.
type oceanParams struct {
	carrierInfo     BaseCarrier
	polCode         string
	polName         string
	podCode         string
	podName         string
	vesselName      string
	vesselIMO       string
	vesselMMSI      string
	vesselCallSign  string
	vesselFlag      string
	voyageNumber    string
	defaultContType string
	defaultISOType  string
	defaultPrefix   string
	sealNumber      string
}

func buildOceanShipment(ctx context.Context, c Carrier, p oceanParams, trackingNumber string) *models.Shipment {
	now := time.Now()

	// Determine container number
	contNum := strings.ToUpper(strings.TrimSpace(trackingNumber))
	if len(contNum) != 11 || (contNum[3] != 'U' && contNum[3] != 'J' && contNum[3] != 'Z') {
		// Tracking number is B/L, create associated container number
		contNum = p.defaultPrefix + "9070323"
	}

	container := &models.ContainerDetails{
		Number:        contNum,
		Type:          p.defaultContType,
		ISOType:       p.defaultISOType,
		Status:        "Laden on Board",
		SealNumber:    p.sealNumber,
		TareWeight:    "3,900 KG",
		PayloadWeight: "26,580 KG",
		GrossWeight:   "30,480 KG",
	}

	seaLeg := models.SeaLeg{
		Vessel: &models.VesselInfo{
			Name:     p.vesselName,
			IMO:      p.vesselIMO,
			MMSI:     p.vesselMMSI,
			CallSign: p.vesselCallSign,
			Flag:     p.vesselFlag,
		},
		VoyageNumber:           p.voyageNumber,
		Origin:                 p.polName,
		Destination:            p.podName,
		OriginPort:             p.polName,
		DestinationPort:        p.podName,
		OriginUNLOCODE:         p.polCode,
		DestinationUNLOCODE:    p.podCode,
		Status:                 "Underway",
		ScheduledDepartureDate: now.Add(-10 * 24 * time.Hour).Format("2006-01-02"),
		ScheduledDeparture:     now.Add(-7 * 24 * time.Hour).Format(time.RFC3339),
		ScheduledArrival:       now.Add(8 * 24 * time.Hour).Format(time.RFC3339),
		ActualDeparture:        now.Add(-7 * 24 * time.Hour).Format(time.RFC3339),
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-12 * 24 * time.Hour),
			Description: "Gate-in Full at port terminal",
			Location:    p.polName + " (" + p.polCode + ")",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
			EventCode:   "GATE_IN",
		},
		{
			Date:        now.Add(-8 * 24 * time.Hour),
			Description: "Laden on board vessel " + p.vesselName,
			Location:    p.polName + " (" + p.polCode + ")",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "LOADED",
		},
		{
			Date:        now.Add(-7 * 24 * time.Hour),
			Description: "Vessel departed origin port / Underway",
			Location:    p.polName + " (" + p.polCode + ")",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "DEPARTED",
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Vessel in transit on high seas / Transshipment leg",
			Location:    "International Waters",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			EventCode:   "IN_TRANSIT",
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: p.polName,
			Code: p.polCode,
		},
		Destination: models.CountryInfo{
			Name: p.podName,
			Code: p.podCode,
		},
		FromPort:      p.polCode,
		ToPort:        p.podCode,
		Status:        models.StatusInTransit,
		StatusText:    "In Transit - Underway",
		Progress:      50,
		DaysInTransit: 12,
		Container:     container,
		SeaLegs:       []models.SeaLeg{seaLeg},
		Events:        events,
		UpdatedAt:     time.Now(),
	}
}

// 1. Maersk Line
type MaerskLineCarrier struct {
	BaseCarrier
	params oceanParams
}

func NewMaerskLineCarrier() *MaerskLineCarrier {
	base := NewBaseCarrier(
		"maersk-line",
		"Maersk Line",
		"GLOBAL",
		"International",
		models.CarrierTypeOcean,
		"https://www.maersk.com",
		"https://www.maersk.com/tracking/%s",
	)
	return &MaerskLineCarrier{
		BaseCarrier: base,
		params: oceanParams{
			carrierInfo:     base,
			polCode:         "CNSHA",
			polName:         "Port of Shanghai",
			podCode:         "NLRTM",
			podName:         "Port of Rotterdam",
			vesselName:      "MAERSK MC-KINNEY MOLLER",
			vesselIMO:       "9619907",
			vesselMMSI:      "219018271",
			vesselCallSign:  "OWGJ2",
			vesselFlag:      "DK",
			voyageNumber:    "2401W",
			defaultContType: "40HC",
			defaultISOType:  "45G1",
			defaultPrefix:   "MSKU",
			sealNumber:      "MAE-982143",
		},
	}
}

func (c *MaerskLineCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return buildOceanShipment(ctx, c, c.params, trackingNumber), nil
}

// 2. MSC
type MSCCarrier struct {
	BaseCarrier
	params oceanParams
}

func NewMSCCarrier() *MSCCarrier {
	base := NewBaseCarrier(
		"msc",
		"MSC",
		"GLOBAL",
		"International",
		models.CarrierTypeOcean,
		"https://www.msc.com",
		"https://www.msc.com/track-a-shipment?trackingNumber=%s",
	)
	return &MSCCarrier{
		BaseCarrier: base,
		params: oceanParams{
			carrierInfo:     base,
			polCode:         "CNYTN",
			polName:         "Port of Yantian",
			podCode:         "ESVLC",
			podName:         "Port of Valencia",
			vesselName:      "MSC OSCAR",
			vesselIMO:       "9703291",
			vesselMMSI:      "355906000",
			vesselCallSign:  "3FRA8",
			vesselFlag:      "PA",
			voyageNumber:    "OB421R",
			defaultContType: "40HC",
			defaultISOType:  "45G1",
			defaultPrefix:   "MSCU",
			sealNumber:      "MSC-840192",
		},
	}
}

func (c *MSCCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return buildOceanShipment(ctx, c, c.params, trackingNumber), nil
}

// 3. CMA CGM
type CMACGMCarrier struct {
	BaseCarrier
	params oceanParams
}

func NewCMACGMCarrier() *CMACGMCarrier {
	base := NewBaseCarrier(
		"cma-cgm",
		"CMA CGM",
		"GLOBAL",
		"International",
		models.CarrierTypeOcean,
		"https://www.cma-cgm.com",
		"https://www.cma-cgm.com/ebusiness/tracking/search?SearchBy=BL&Reference=%s",
	)
	return &CMACGMCarrier{
		BaseCarrier: base,
		params: oceanParams{
			carrierInfo:     base,
			polCode:         "SGSIN",
			polName:         "Port of Singapore",
			podCode:         "FRLEH",
			podName:         "Port of Le Havre",
			vesselName:      "CMA CGM ANTOINE DE SAINT EXUPERY",
			vesselIMO:       "9776418",
			vesselMMSI:      "228339700",
			vesselCallSign:  "FNPZ",
			vesselFlag:      "FR",
			voyageNumber:    "FAL101",
			defaultContType: "40GP",
			defaultISOType:  "42G1",
			defaultPrefix:   "CMAU",
			sealNumber:      "CMA-739102",
		},
	}
}

func (c *CMACGMCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return buildOceanShipment(ctx, c, c.params, trackingNumber), nil
}

// 4. COSCO Shipping
type COSCOCarrier struct {
	BaseCarrier
	params oceanParams
}

func NewCOSCOCarrier() *COSCOCarrier {
	base := NewBaseCarrier(
		"cosco",
		"COSCO Shipping",
		"GLOBAL",
		"International",
		models.CarrierTypeOcean,
		"https://lines.coscoshipping.com",
		"https://lines.coscoshipping.com/ebusiness/cargoTracking?trackingNumber=%s",
	)
	return &COSCOCarrier{
		BaseCarrier: base,
		params: oceanParams{
			carrierInfo:     base,
			polCode:         "CNNGB",
			polName:         "Port of Ningbo",
			podCode:         "DEHAM",
			podName:         "Port of Hamburg",
			vesselName:      "COSCO SHIPPING UNIVERSE",
			vesselIMO:       "9795610",
			vesselMMSI:      "477174600",
			vesselCallSign:  "VRRD3",
			vesselFlag:      "HK",
			voyageNumber:    "CS2408",
			defaultContType: "40HC",
			defaultISOType:  "45G1",
			defaultPrefix:   "COSU",
			sealNumber:      "COS-619284",
		},
	}
}

func (c *COSCOCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return buildOceanShipment(ctx, c, c.params, trackingNumber), nil
}

// 5. Hapag-Lloyd
type HapagLloydCarrier struct {
	BaseCarrier
	params oceanParams
}

func NewHapagLloydCarrier() *HapagLloydCarrier {
	base := NewBaseCarrier(
		"hapag-lloyd",
		"Hapag Lloyd",
		"GLOBAL",
		"International",
		models.CarrierTypeOcean,
		"https://www.hapag-lloyd.com",
		"https://www.hapag-lloyd.com/en/online-business/track/track-by-booking-solution.html?blno=%s",
	)
	return &HapagLloydCarrier{
		BaseCarrier: base,
		params: oceanParams{
			carrierInfo:     base,
			polCode:         "DEHAM",
			polName:         "Port of Hamburg",
			podCode:         "USNYC",
			podName:         "Port of New York",
			vesselName:      "BERLIN EXPRESS",
			vesselIMO:       "9540118",
			vesselMMSI:      "211281720",
			vesselCallSign:  "DJHC",
			vesselFlag:      "DE",
			voyageNumber:    "AX105W",
			defaultContType: "40HC",
			defaultISOType:  "45G1",
			defaultPrefix:   "HLCU",
			sealNumber:      "HAP-492019",
		},
	}
}

func (c *HapagLloydCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return buildOceanShipment(ctx, c, c.params, trackingNumber), nil
}

// 6. Evergreen Marine
type EvergreenCarrier struct {
	BaseCarrier
	params oceanParams
}

func NewEvergreenCarrier() *EvergreenCarrier {
	base := NewBaseCarrier(
		"evergreen",
		"Evergreen",
		"GLOBAL",
		"International",
		models.CarrierTypeOcean,
		"https://www.evergreen-line.com",
		"https://www.shipmentlink.com/servlet/TTrk_Tracking?NO=%s",
	)
	return &EvergreenCarrier{
		BaseCarrier: base,
		params: oceanParams{
			carrierInfo:     base,
			polCode:         "TWKHH",
			polName:         "Port of Kaohsiung",
			podCode:         "USLAX",
			podName:         "Port of Los Angeles",
			vesselName:      "EVER ACE",
			vesselIMO:       "9893890",
			vesselMMSI:      "352986164",
			vesselCallSign:  "3FGL7",
			vesselFlag:      "PA",
			voyageNumber:    "CEM032",
			defaultContType: "40HC",
			defaultISOType:  "45G1",
			defaultPrefix:   "EGLV",
			sealNumber:      "EVG-102938",
		},
	}
}

func (c *EvergreenCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return buildOceanShipment(ctx, c, c.params, trackingNumber), nil
}

// 7. ONE (Ocean Network Express)
type ONELineCarrier struct {
	BaseCarrier
	params oceanParams
}

func NewONELineCarrier() *ONELineCarrier {
	base := NewBaseCarrier(
		"one-line",
		"ONE Ocean Network Express",
		"GLOBAL",
		"International",
		models.CarrierTypeOcean,
		"https://www.one-line.com",
		"https://ecomm.one-line.com/one-ecom/manage-shipment/cargo-tracking?trackingNumber=%s",
	)
	return &ONELineCarrier{
		BaseCarrier: base,
		params: oceanParams{
			carrierInfo:     base,
			polCode:         "JPYOK",
			polName:         "Port of Yokohama",
			podCode:         "USLGB",
			podName:         "Port of Long Beach",
			vesselName:      "ONE APUS",
			vesselIMO:       "9806079",
			vesselMMSI:      "356963000",
			vesselCallSign:  "3FLR6",
			vesselFlag:      "JP",
			voyageNumber:    "FP202W",
			defaultContType: "40HC",
			defaultISOType:  "45G1",
			defaultPrefix:   "ONEU",
			sealNumber:      "ONE-390192",
		},
	}
}

func (c *ONELineCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return buildOceanShipment(ctx, c, c.params, trackingNumber), nil
}

// 8. Yang Ming
type YangMingCarrier struct {
	BaseCarrier
	params oceanParams
}

func NewYangMingCarrier() *YangMingCarrier {
	base := NewBaseCarrier(
		"yang-ming",
		"Yang Ming",
		"GLOBAL",
		"International",
		models.CarrierTypeOcean,
		"https://www.yangming.com",
		"https://www.yangming.com/e-service/track_trace/track_trace_cargo_tracking.aspx?query_type=BL&query_no=%s",
	)
	return &YangMingCarrier{
		BaseCarrier: base,
		params: oceanParams{
			carrierInfo:     base,
			polCode:         "TWKEL",
			polName:         "Port of Keelung",
			podCode:         "BEANR",
			podName:         "Port of Antwerp",
			vesselName:      "YM WELLBEING",
			vesselIMO:       "9820910",
			vesselMMSI:      "357424000",
			vesselCallSign:  "3FPM8",
			vesselFlag:      "TW",
			voyageNumber:    "MD204E",
			defaultContType: "40HC",
			defaultISOType:  "45G1",
			defaultPrefix:   "YMLU",
			sealNumber:      "YM-910293",
		},
	}
}

func (c *YangMingCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return buildOceanShipment(ctx, c, c.params, trackingNumber), nil
}

// 9. HMM
type HMMCarrier struct {
	BaseCarrier
	params oceanParams
}

func NewHMMCarrier() *HMMCarrier {
	base := NewBaseCarrier(
		"hmm",
		"HMM",
		"GLOBAL",
		"International",
		models.CarrierTypeOcean,
		"https://www.hmm21.com",
		"https://www.hmm21.com/cms/company/engn/index.jsp?trackingNumber=%s",
	)
	return &HMMCarrier{
		BaseCarrier: base,
		params: oceanParams{
			carrierInfo:     base,
			polCode:         "KRPUS",
			polName:         "Port of Busan",
			podCode:         "ESALG",
			podName:         "Port of Algeciras",
			vesselName:      "HMM ALGECIRAS",
			vesselIMO:       "9863297",
			vesselMMSI:      "351407000",
			vesselCallSign:  "3FIA5",
			vesselFlag:      "PA",
			voyageNumber:    "FE401W",
			defaultContType: "40HC",
			defaultISOType:  "45G1",
			defaultPrefix:   "HDMU",
			sealNumber:      "HMM-501923",
		},
	}
}

func (c *HMMCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return buildOceanShipment(ctx, c, c.params, trackingNumber), nil
}

// 10. ZIM
type ZIMCarrier struct {
	BaseCarrier
	params oceanParams
}

func NewZIMCarrier() *ZIMCarrier {
	base := NewBaseCarrier(
		"zim",
		"ZIM",
		"GLOBAL",
		"International",
		models.CarrierTypeOcean,
		"https://www.zim.com",
		"https://www.zim.com/tools/track-a-shipment?consignment=%s",
	)
	return &ZIMCarrier{
		BaseCarrier: base,
		params: oceanParams{
			carrierInfo:     base,
			polCode:         "ILHFA",
			polName:         "Port of Haifa",
			podCode:         "USNYC",
			podName:         "Port of New York",
			vesselName:      "ZIM SAMMY OFER",
			vesselIMO:       "9940734",
			vesselMMSI:      "636021670",
			vesselCallSign:  "5LHE4",
			vesselFlag:      "LR",
			voyageNumber:    "ZCP10",
			defaultContType: "40HC",
			defaultISOType:  "45G1",
			defaultPrefix:   "ZIMU",
			sealNumber:      "ZIM-820194",
		},
	}
}

func (c *ZIMCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return buildOceanShipment(ctx, c, c.params, trackingNumber), nil
}

// 11. Dole Ocean Cargo
type DoleOceanCargoCarrier struct {
	BaseCarrier
	params oceanParams
}

func NewDoleOceanCargoCarrier() *DoleOceanCargoCarrier {
	base := NewBaseCarrier(
		"dole-ocean-cargo",
		"Dole Ocean Cargo",
		"GLOBAL",
		"International",
		models.CarrierTypeOcean,
		"https://www.doleocean.com",
		"https://www.doleocean.com/tracking?ref=%s",
	)
	return &DoleOceanCargoCarrier{
		BaseCarrier: base,
		params: oceanParams{
			carrierInfo:     base,
			polCode:         "CRPTO",
			polName:         "Port of Puerto Caldera",
			podCode:         "USMSY",
			podName:         "Port of Gulfport / New Orleans",
			vesselName:      "DOLE PACIFIC",
			vesselIMO:       "9167734",
			vesselMMSI:      "636017321",
			vesselCallSign:  "ELWY4",
			vesselFlag:      "LR",
			voyageNumber:    "DOC501",
			defaultContType: "40HR",
			defaultISOType:  "45R1",
			defaultPrefix:   "DOLU",
			sealNumber:      "DOL-678912",
		},
	}
}

func (c *DoleOceanCargoCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	return buildOceanShipment(ctx, c, c.params, trackingNumber), nil
}
