//go:build integration

package carriers

import (
	"context"
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func TestSEURLiveTracking(t *testing.T) {
	c := NewSEURCarrier()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.Track(ctx, "36150127630406", models.TrackOptions{
		PostalCode:     "08001",
		AllowSimulated: false,
	})
	if err != nil && err != models.ErrShipmentNotFound && err != models.ErrCarrierTemporarilyUnavailable {
		t.Logf("Live SEUR tracking returned: %v", err)
	}
}
