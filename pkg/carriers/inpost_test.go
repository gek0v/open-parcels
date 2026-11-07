package carriers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"open-parcels/pkg/models"
)

func TestInPostMissingPostalCode(t *testing.T) {
	c := NewInPostCarrier()
	_, err := c.Track(context.Background(), "75616714", models.TrackOptions{
		AllowSimulated: false,
		PostalCode:     "",
	})
	if !errors.Is(err, models.ErrMissingPostalCode) {
		t.Fatalf("expected ErrMissingPostalCode, got: %v", err)
	}

	cSpain := NewInPostSpainCarrier()
	_, errSpain := cSpain.Track(context.Background(), "75616714", models.TrackOptions{
		AllowSimulated: false,
		PostalCode:     "",
	})
	if !errors.Is(errSpain, models.ErrMissingPostalCode) {
		t.Fatalf("expected ErrMissingPostalCode for inpost-es, got: %v", errSpain)
	}
}

func TestInPostExtractPropsAndNormalize(t *testing.T) {
	sampleHTML := `
<!DOCTYPE html><html><body>
<astro-island uid="Ztest" props="{&quot;trackingDetails&quot;:[0,{&quot;packageId&quot;:[0,&quot;75616714&quot;],&quot;milestones&quot;:[1,[[0,{&quot;type&quot;:[0,&quot;AT_THE_ORIGIN&quot;],&quot;name&quot;:[0,&quot;En preparación&quot;],&quot;date&quot;:[0,&quot;2026-11-06T18:52:46Z&quot;],&quot;events&quot;:[1,[[0,{&quot;code&quot;:[0,&quot;CRE.1001&quot;],&quot;name&quot;:[0,&quot;En preparación&quot;],&quot;date&quot;:[0,&quot;2026-11-06T18:52:46Z&quot;],&quot;description&quot;:[0,&quot;El envío ha sido registrado informáticamente y estamos a la espera de recibir el paquete.&quot;]}]]]}]]]}],&quot;trackingDetailsStatus&quot;:[0,200]}" client="load"></astro-island>
</body></html>`

	props, err := extractInPostProps(sampleHTML)
	if err != nil {
		t.Fatalf("unexpected extract error: %v", err)
	}

	if !errors.Is(err, nil) && props == "" {
		t.Fatalf("empty props extracted")
	}

	status, stage := normalizeInPostStatus("En preparación", "AT_THE_ORIGIN", "El envío ha sido registrado")
	if status != models.StatusInfoReceived {
		t.Errorf("expected StatusInfoReceived, got %s", status)
	}
	if stage != models.StageCreated {
		t.Errorf("expected StageCreated, got %s", stage)
	}
}

func TestInPostDeterministicFixture(t *testing.T) {
	fixtureData := LoadFixture(t, "inpost_response.html")
	origClient := inpostHTTPClient
	inpostHTTPClient = NewFixtureClient(http.StatusOK, fixtureData, "text/html")
	defer func() { inpostHTTPClient = origClient }()

	c := NewInPostCarrier()
	shipment, err := c.Track(context.Background(), "75616714", models.TrackOptions{
		PostalCode:     "47140",
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("expected tracking to succeed with fixture, got error: %v", err)
	}

	if shipment.Status != models.StatusInfoReceived {
		t.Errorf("expected StatusInfoReceived, got %s", shipment.Status)
	}
	if len(shipment.Events) == 0 {
		t.Fatalf("expected at least 1 tracking event, got 0")
	}
	if shipment.Events[0].Description != "El envío ha sido registrado informáticamente y estamos a la espera de recibir el paquete." {
		t.Errorf("unexpected event description: %s", shipment.Events[0].Description)
	}
}

func TestInPostOriginAndDestinationDetermination(t *testing.T) {
	// 1. Unknown when no explicit location
	evsNoLoc := []models.TrackingEvent{
		{
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
			Description: "El envío ha sido registrado informáticamente",
		},
	}
	orig1 := extractInPostOrigin(evsNoLoc)
	if orig1.Name != "Unknown" || orig1.Code != "" {
		t.Errorf("expected Unknown origin, got %+v", orig1)
	}

	// 2. Cross-border origin: France
	evsFR := []models.TrackingEvent{
		{
			Status:      models.StatusInfoReceived,
			Stage:       models.StageCreated,
			Description: "Colis enregistré",
			Location:    "Hub Lille, France",
		},
		{
			Status:      models.StatusOutForDelivery,
			Stage:       models.StageLastMile,
			Description: "En reparto",
			Location:    "Madrid Locker, Spain",
		},
	}
	origFR := extractInPostOrigin(evsFR)
	if origFR.Name != "France" || origFR.Code != "FR" {
		t.Errorf("expected France / FR, got %+v", origFR)
	}

	// 3. Cross-border origin: Poland
	evsPL := []models.TrackingEvent{
		{
			Status:      models.StatusInTransit,
			Stage:       models.StageTransit,
			Description: "Przesyłka w oddziale",
			Location:    "Paczkomat Warszawa (PL)",
		},
	}
	origPL := extractInPostOrigin(evsPL)
	if origPL.Name != "Poland" || origPL.Code != "PL" {
		t.Errorf("expected Poland / PL, got %+v", origPL)
	}

	// 4. Destination resolution: Spanish 5-digit postal code
	cGlobal := NewInPostCarrier()
	destES := resolveInPostDestination(cGlobal, "47140", models.TrackOptions{})
	if destES.Name != "Spain" || destES.Code != "ES" {
		t.Errorf("expected Spain / ES for 47140, got %+v", destES)
	}

	// 5. Destination resolution: carrier with country ES
	cSpain := NewInPostSpainCarrier()
	destSpainCarrier := resolveInPostDestination(cSpain, "", models.TrackOptions{})
	if destSpainCarrier.Name != "Spain" || destSpainCarrier.Code != "ES" {
		t.Errorf("expected Spain / ES for inpost-es, got %+v", destSpainCarrier)
	}
}
