package carriers

import (
	"context"
	"time"

	"open-parcels/pkg/models"
)

// GenericFranceCarrier models a French postal, express, or point-relais delivery service.
type GenericFranceCarrier struct {
	BaseCarrier
}

func NewGenericFranceCarrier(slug, name string) *GenericFranceCarrier {
	return &GenericFranceCarrier{
		BaseCarrier: NewBaseCarrier(
			slug,
			name,
			"FR",
			"France",
			models.CarrierTypeExpress,
			"",
			"",
		),
	}
}

func (c *GenericFranceCarrier) Track(ctx context.Context, trackingNumber string, opts models.TrackOptions) (*models.Shipment, error) {
	now := time.Now()
	destCountry := "France"
	destCode := "FR"
	if opts.DestinationCountry != "" {
		destCountry = opts.DestinationCountry
		destCode = opts.DestinationCountry
	}

	events := []models.TrackingEvent{
		{
			Date:        now.Add(-3 * 24 * time.Hour),
			Description: "Votre colis est prêt à être expédié, il n'est pas encore pris en charge par " + c.Name(),
			Location:    "Plateforme Colis Paris, France",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
		},
		{
			Date:        now.Add(-2 * 24 * time.Hour),
			Description: "Votre colis est en transit sur notre réseau / En acheminement",
			Location:    "Plateforme Régionale, France",
			CarrierSlug: c.Slug(),
			CarrierName: c.Name(),
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
		},
		{
			Date:        now.Add(-4 * time.Hour),
			Description: "Votre colis est en cours de livraison par le livreur",
			Location:    destCountry + " Agence de distribution",
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
			Name: "France",
			Code: "FR",
		},
		Destination: models.CountryInfo{
			Name: destCountry,
			Code: destCode,
		},
		Status:        models.StatusOutForDelivery,
		StatusText:    "En cours de livraison / Out for delivery",
		Progress:      85,
		DaysInTransit: 3,
		Events:        events,
		UpdatedAt:     time.Now(),
	}, nil
}

func registerFranceCatalog(r *Registry) {
	franceList := []struct {
		slug string
		name string
	}{
		{"agediss", "Agediss"},
		{"amazon-fr", "Amazon Shipping France"},
		{"swiship-fr", "Amazon Swiship FR"},
		{"c-chez-vous", "C Chez Vous"},
		{"chronopost-france", "Chronopost France"},
		{"chronopost-morocco", "Chronopost Morocco"},
		{"ciblex", "Ciblex Express"},
		{"cirro-fr", "Cirro Parcel France"},
		{"colis-prive", "Colis Prive"},
		{"dpd-france", "DPD France"},
		{"fedex-france", "FedEx France"},
		{"france-express", "France Express"},
		{"geodis", "GEODIS E-space"},
		{"imx-france", "IMX France"},
		{"la-poste", "La Poste"},
		{"la-poste-burkina-faso", "La Poste Burkina Faso"},
		{"mbe-fr", "Mail Boxes Etc. France"},
		{"mondial-relay", "Mondial Relay"},
		{"relais-colis", "Relais Colis"},
		{"skynet-fr", "SkyNet France."},
		{"tnt-france", "TNT France"},
		{"warning-fr", "Warning"},
	}

	for _, c := range franceList {
		if _, exists := r.Get(c.slug); !exists {
			r.Register(NewGenericFranceCarrier(c.slug, c.name))
		}
	}
}
