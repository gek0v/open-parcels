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
	fixtureData, err := LoadFixture("inpost_response.html")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
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

func TestInPostInTransitMultiMilestoneFixture(t *testing.T) {
	fixtureData, err := LoadFixture("inpost_intransit.html")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetInPostHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "text/html"))
	defer teardown()

	c := NewInPostCarrier()
	shipment, err := c.Track(context.Background(), "12345678", models.TrackOptions{
		PostalCode:     "28001",
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("expected tracking to succeed with in-transit fixture, got error: %v", err)
	}

	if shipment.Status != models.StatusInTransit {
		t.Errorf("expected StatusInTransit, got %s", shipment.Status)
	}
	if shipment.StatusText != "En camino" {
		t.Errorf("expected StatusText 'En camino', got %q", shipment.StatusText)
	}
	if shipment.DeliveredAt != nil {
		t.Errorf("expected DeliveredAt nil for in-transit shipment, got %v", shipment.DeliveredAt)
	}
	if len(shipment.Events) != 4 {
		t.Fatalf("expected 4 tracking events, got %d", len(shipment.Events))
	}

	expectedEvents := []struct {
		code   string
		status models.Status
		stage  models.Stage
	}{
		{"CRE.1001", models.StatusInfoReceived, models.StageCreated},
		{"FMD.1001", models.StatusInfoReceived, models.StageCreated},
		{"FMD.1002", models.StatusInTransit, models.StagePickedUp},
		{"MMD.1001", models.StatusInTransit, models.StageTransit},
	}

	for i, exp := range expectedEvents {
		ev := shipment.Events[i]
		if ev.EventCode != exp.code {
			t.Errorf("event[%d]: expected code %s, got %s", i, exp.code, ev.EventCode)
		}
		if ev.Status != exp.status {
			t.Errorf("event[%d] (%s): expected status %s, got %s", i, ev.EventCode, exp.status, ev.Status)
		}
		if ev.Stage != exp.stage {
			t.Errorf("event[%d] (%s): expected stage %s, got %s", i, ev.EventCode, exp.stage, ev.Stage)
		}
	}
}

func TestInPostStatusConflictRequiresPostalCode(t *testing.T) {
	conflictHTML := `<!DOCTYPE html><html><body><astro-island props="{&quot;trackingDetailsStatus&quot;:[0,409],&quot;trackingDetails&quot;:[0,{}]}"></astro-island></body></html>`
	teardown := SetInPostHTTPClientForTest(NewFixtureClient(http.StatusOK, []byte(conflictHTML), "text/html"))
	defer teardown()

	c := NewInPostCarrier()
	_, err := c.Track(context.Background(), "12345678", models.TrackOptions{
		PostalCode:     "28001",
		AllowSimulated: false,
	})
	if !errors.Is(err, models.ErrPostalCodeRequired) {
		t.Fatalf("expected ErrPostalCodeRequired for status 409, got: %v", err)
	}
}

func TestInPostNormalizeEventCodesAndTypes(t *testing.T) {
	cases := []struct {
		code   string
		name   string
		mType  string
		desc   string
		status models.Status
		stage  models.Stage
	}{
		{"CRE.1001", "En preparación", "AT_THE_ORIGIN", "El envío ha sido registrado", models.StatusInfoReceived, models.StageCreated},
		{"FMD.1001", "Listo para ser recogido por el mensajero", "AT_THE_ORIGIN", "Tu paquete está preparado y será recogido por nuestros mensajeros pronto.", models.StatusInfoReceived, models.StageCreated},
		{"FMD.1002", "Recogido por el mensajero", "IN_TRANSIT", "Tu paquete ha iniciado el tránsito y se dirige a uno de nuestros centros logísticos.", models.StatusInTransit, models.StagePickedUp},
		{"MMD.1001", "Recibido en el centro logístico", "IN_TRANSIT", "Tu paquete ha sido recibido en uno de nuestros centros logísticos.", models.StatusInTransit, models.StageTransit},
		{"LMD.1001", "En reparto", "IN_TRANSIT_LAST_MILE", "El mensajero lleva tu paquete", models.StatusOutForDelivery, models.StageLastMile},
		{"LMD.1004", "Disponible en Punto Pack", "AWAITING_COLLECTION", "Disponible para recoger", models.StatusOutForDelivery, models.StageLastMile},
		{"EOL.1001", "Entregado", "DELIVERED", "Recogido por el destinatario", models.StatusDelivered, models.StageDelivery},
		{"RTS.1001", "Devuelto al remitente", "RETURN", "Paquete devuelto", models.StatusException, models.StageException},
		{"", "Recogido por el mensajero", "", "", models.StatusInTransit, models.StagePickedUp},
		{"", "Recogido por el destinatario", "", "", models.StatusDelivered, models.StageDelivery},
	}

	for _, tc := range cases {
		st, sg := normalizeInPostEventStatus(tc.code, tc.name, tc.mType, tc.desc)
		if st != tc.status || sg != tc.stage {
			t.Errorf("case (%s, %s, %s): expected (%s, %s), got (%s, %s)", tc.code, tc.name, tc.mType, tc.status, tc.stage, st, sg)
		}
	}
}

func TestInPostItalyDeterministicShipXFixture(t *testing.T) {
	fixtureData, err := LoadFixture("inpost_italy_shipx.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := SetInPostItalyHTTPClientForTest(NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	c := NewInPostItalyCarrier()
	shipment, err := c.Track(context.Background(), "12345678", models.TrackOptions{
		Language:       "it",
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("expected InPost Italy tracking to succeed with fixture, got: %v", err)
	}

	if shipment.Status != models.StatusInTransit {
		t.Errorf("expected StatusInTransit, got %s", shipment.Status)
	}
	if shipment.Origin.Name != "Roma, Italy" || shipment.Origin.Code != "IT" {
		t.Errorf("expected Origin Roma, Italy (IT), got %+v", shipment.Origin)
	}
	if shipment.Destination.Name != "Madrid, Spain" || shipment.Destination.Code != "ES" {
		t.Errorf("expected Destination Madrid, Spain (ES), got %+v", shipment.Destination)
	}
	if len(shipment.Events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(shipment.Events))
	}
	if shipment.Events[3].Location != "Roma (RM), Italy" {
		t.Errorf("expected latest event location 'Roma (RM), Italy', got %q", shipment.Events[3].Location)
	}
	if shipment.DeliveredAt != nil {
		t.Errorf("expected nil DeliveredAt, got %v", shipment.DeliveredAt)
	}
}

func TestInPostEnrichesOriginFromItalyFixture(t *testing.T) {
	esFixture, err := LoadFixture("inpost_intransit.html")
	if err != nil {
		t.Fatalf("failed to load es fixture: %v", err)
	}
	itFixture, err := LoadFixture("inpost_italy_shipx.json")
	if err != nil {
		t.Fatalf("failed to load it fixture: %v", err)
	}

	teardownES := SetInPostHTTPClientForTest(NewFixtureClient(http.StatusOK, esFixture, "text/html"))
	defer teardownES()
	teardownIT := SetInPostItalyHTTPClientForTest(NewFixtureClient(http.StatusOK, itFixture, "application/json"))
	defer teardownIT()

	c := NewInPostCarrier()
	shipment, err := c.Track(context.Background(), "12345678", models.TrackOptions{
		PostalCode:     "28001",
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("expected tracking to succeed: %v", err)
	}
	if shipment.Origin.Name != "Roma, Italy" || shipment.Origin.Code != "IT" {
		t.Errorf("expected enriched Origin Roma, Italy (IT), got %+v", shipment.Origin)
	}
	if shipment.Destination.Name != "Madrid, Spain" {
		t.Errorf("expected enriched Destination Madrid, Spain, got %+v", shipment.Destination)
	}
	if shipment.Events[3].Location != "Roma (RM), Italy" {
		t.Errorf("expected enriched event location 'Roma (RM), Italy', got %q", shipment.Events[3].Location)
	}
}
