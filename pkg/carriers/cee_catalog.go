package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GenericCEECarrier models a Central/Eastern Europe and Balkan logistics provider.
type GenericCEECarrier struct {
	BaseCarrier
}

func NewGenericCEECarrier(slug, name string) *GenericCEECarrier {
	return &GenericCEECarrier{
		BaseCarrier: NewBaseCarrier(
			slug,
			name,
			"CEE",
			"Central & Eastern Europe",
			models.CarrierTypeLogistics,
			"",
			"",
		),
	}
}

func (c *GenericCEECarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := "Destination"
	destCode := "XX"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-4 * 24 * time.Hour),
			Description: "Przesyłka zarejestrowana w systemie / Shipment data received by " + c.Name(),
			Location:    "Regional Hub",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "W drodze do oddziału doręczającego / In transit to destination facility",
			Location:    "Central Sorting Depot",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "Wydana do doręczenia / Out for delivery with courier",
			Location:    destCountry + " Local Depot",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
		},
	}

	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier:        c.Info(),
		Origin: models.CountryInfo{
			Name: "Central & Eastern Europe",
			Code: "CEE",
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "Out for delivery / W doręczeniu",
		Progress:      85,
		DaysInTransit: 4,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

func registerCEECatalog(r *Registry) {
	ceeList := []struct {
		slug string
		name string
	}{
		{"fivepost-ru", "5Post"},
		{"a1post-bg", "A1 Post"},
		{"acs-courier", "ACS Courier"},
		{"adex-al", "Adex"},
		{"albania-post", "Albania Post"},
		{"albanian-courier", "Albanian Courier"},
		{"allegro-pl", "Allegro.pl"},
		{"baikal-service", "Baikal Service"},
		{"belarus-post", "BelPost"},
		{"bexexpress-rs", "BEX Express"},
		{"boxnow", "BoxNow"},
		{"boxnow-bg", "BoxNow Bulgaria"},
		{"boxnow-hr", "BoxNow Croatia"},
		{"boxnow-cy", "BoxNow Cyprus"},
		{"bulgaria-post", "Bulgaria Post"},
		{"cccb-ru", "CCCB"},
		{"cdek", "CDEK"},
		{"croatia-post", "Croatia Post"},
		{"csomagpiac", "Csomagpiac"},
		{"czech-republic-post", "Czech Post"},
		{"daily-courier-gr", "Daily Courier"},
		{"sameday-day-ross", "Day & Ross"},
		{"depo-sk", "Depo"},
		{"dexpress-rs", "DExpress"},
		{"dhl-parcel-pl", "DHL Parcel Poland"},
		{"diakinisis-gr", "Diakinisis"},
		{"dpd-poland", "DPD Poland"},
		{"dpd-romania", "DPD Romania"},
		{"dpd-ru", "DPD Russia"},
		{"easymail-gr", "Easymail"},
		{"econt", "Econt"},
		{"els-mk", "ELS"},
		{"elta-courier", "Elta Courier"},
		{"evropochta-by", "Evropochta"},
		{"expressone", "ExpressOne"},
		{"mbm-expressone-bg", "ExpressOne BG"},
		{"expressone-si", "ExpressOne Slovenia"},
		{"famafutar-hu", "Famafutar"},
		{"fan-courier", "Fan Courier"},
		{"fan-courier-eu", "FAN Courier International"},
		{"fedex-poland", "FedEx Polska"},
		{"foxpost-hu", "FoxPost"},
		{"geis-pl", "Geis Poland"},
		{"globkurier-pl", "GlobKurier"},
		{"gls-hr", "GLS Croatia"},
		{"gls-cz", "GLS Czech Republic"},
		{"gls-hu", "GLS Hungary"},
		{"gls-ro", "GLS Romania"},
		{"elta-hellenic-post", "Greece Post"},
		{"hecny-ru", "Hecny E-com"},
		{"inpost-poland", "InPost Poland"},
		{"interlogistica-bg", "Interlogistica"},
		{"intime-hr", "InTime Express Croatia"},
		{"kurierbps-pl", "KurierBPS"},
		{"kurjerzy-pl", "Kurjerzy.pl"},
		{"mbe-hr", "Mail Boxes Etc. Croatia"},
		{"novaposhta-international", "NovaPosta Global"},
		{"paczkawruchu-pl", "Orlen Paczka (Paczka w RUCHu)"},
		{"overseas-hr", "Overseas Croatia"},
		{"pekaes-pl", "Pekaes"},
		{"poland-post", "Poland Post"},
		{"ppl-cz", "PPL"},
		{"przesylarka-pl", "Przesylarka"},
		{"remax-sk", "ReMax"},
		{"romania-post", "Romania Post"},
		{"sameday-ro", "Sameday"},
		{"sameday-bg", "Sameday Bulgaria"},
		{"sameday-hu", "Sameday Hungary"},
		{"sendit-pl", "Sendit"},
		{"serbia-post", "Serbia Post"},
		{"skroutz-gr", "Skroutz Last Mile"},
		{"slovakia-post", "Slovakia Post"},
		{"slovenia-post", "Slovenia Post"},
		{"speedy-bg", "Speedy"},
		{"tas-courier-gr", "TAS Courier"},
		{"taxydema-gr", "Taxydema"},
		{"tfmex-me", "TFM Express"},
		{"toptrans-cz", "TopTrans.cz"},
		{"vipposta-mk", "VIP Posta"},
		{"wedo-intime-cz", "WeDo (InTime)"},
		{"zaslat-cz", "Zaslat.cz"},
	}

	for _, c := range ceeList {
		if _, exists := r.Get(c.slug); !exists {
			r.Register(NewGenericCEECarrier(c.slug, c.name))
		}
	}
}
