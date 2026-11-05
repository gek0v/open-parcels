package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GenericOceanCarrier models a container shipping line or maritime ocean freight operator.
type GenericOceanCarrier struct {
	BaseCarrier
}

func NewGenericOceanCarrier(slug, name string) *GenericOceanCarrier {
	return &GenericOceanCarrier{
		BaseCarrier: NewBaseCarrier(
			slug,
			name,
			"GLOBAL",
			"International",
			models.CarrierTypeFreight,
			"",
			"",
		),
	}
}

func (c *GenericOceanCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-12 * 24 * time.Hour),
			Description: "Gate in full at origin port terminal / Container loaded by " + c.Name(),
			Location:    "Port of Shanghai, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StagePickedUp,
		},
		{
			Date:        now.Add(-10 * 24 * time.Hour),
			Description: "Loaded on board vessel at departure port",
			Location:    "Port of Shanghai, China",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Vessel in transit on high seas / Ocean leg underway",
			Location:    "Mediterranean Sea / Atlantic Ocean",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: "Shanghai Port, China",
			Code: "CNSHA",
		},
		Destination: models.CountryInfo{
			Name: "Valencia Port, Spain",
			Code: "ESVLC",
		},
		Status:        models.StatusInTransit,
		StatusText:    "In Transit - Ocean Voyage",
		Progress:      55,
		DaysInTransit: 12,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

func registerOceanFreightCatalog(r *Registry) {
	shippingLines := []struct {
		slug string
		name string
	}{
		{"aquiline", "Aquiline"},
		{"dellin", "Business Line"},
		{"cma-cgm", "CMA CGM"},
		{"cosco-bol", "COSCO Bill of Lading"},
		{"cosco", "COSCO Shipping"},
		{"tourline", "CTT Express"},
		{"culines", "CULines"},
		{"ecoscooting", "EcoScooting"},
		{"emirates-shipping-line", "Emirates Shipping Line"},
		{"evergreen", "Evergreen"},
		{"fancourier-md", "FAN Courier Moldova"},
		{"firstline-cn", "First Line"},
		{"hapag-lloyd", "Hapag Lloyd"},
		{"hmm", "HMM"},
		{"hst-online", "HST Online"},
		{"ilinexpress", "iLine"},
		{"kmtc-line", "KMTC"},
		{"lineclear", "Line Clear Express & Logistics Sdn Bhd"},
		{"linear-logistics", "Linear Logistics"},
		{"lineer-post", "Lineer Post"},
		{"maersk-ecommerce", "Maersk eCommerce"},
		{"maersk-line", "Maersk Line"},
		{"matson", "Matson"},
		{"messina-line", "Messina Line"},
		{"moldova-post", "Moldavia Post"},
		{"msc", "MSC"},
		{"norlines-no", "Norlines"},
		{"northline-au", "Northline"},
		{"oak-harbor-freight-lines", "Oak Harbor Freight Lines"},
		{"old-dominion-freight", "Old Dominion Freight Line"},
		{"one-line", "ONE Ocean Network Express"},
		{"oocl", "OOCL Orient Overseas International"},
		{"pil", "Pacific International Lines"},
		{"peninsula-truck-lines", "Peninsula Truck Lines"},
		{"rpx-online", "RPX Online"},
		{"rpx-online-ref", "RPX Online Reference"},
		{"safmarine", "Safmarine"},
		{"servidis-online", "Servidis Online"},
		{"shopline", "ShopLine"},
		{"sitc", "SITC Shipping Lines"},
		{"sm-line", "SM Line"},
		{"southeastern-freight-lines", "Southeastern Freight Lines"},
		{"spedisci-online", "Spedisci Online"},
		{"turkon-lines", "Turkon Lines"},
		{"wan-hai", "Wan Hai"},
		{"wec-lines", "WEC Lines"},
		{"yang-ming", "Yang Ming"},
		{"zim", "ZIM"},
		{"zim-cn", "ZIM Logistics"},
	}

	for _, c := range shippingLines {
		if _, exists := r.Get(c.slug); !exists {
			r.Register(NewGenericOceanCarrier(c.slug, c.name))
		}
	}
}
