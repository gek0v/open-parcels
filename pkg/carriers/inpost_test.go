package carriers

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestInPostLiveTracking(t *testing.T) {
	c := NewInPostCarrier()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	shipment, err := c.Track(ctx, "75616714", models.TrackOptions{
		PostalCode:     "47140",
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("live tracking failed for 75616714: %v", err)
	}

	if shipment.Status != models.StatusInfoReceived {
		t.Errorf("expected status %s, got %s", models.StatusInfoReceived, shipment.Status)
	}

	if shipment.StatusText != "En preparación" {
		t.Errorf("expected status text 'En preparación', got %q", shipment.StatusText)
	}

	if len(shipment.Events) == 0 {
		t.Fatalf("expected at least 1 tracking event, got 0")
	}

	foundEvent := false
	for _, ev := range shipment.Events {
		if ev.Date.Format("2006-01-02") == "2026-11-06" &&
			ev.Description == "El envío ha sido registrado informáticamente y estamos a la espera de recibir el paquete." {
			foundEvent = true
			break
		}
	}

	if !foundEvent {
		t.Errorf("did not find expected 2026-11-06 event in events: %+v", shipment.Events)
	}
}
