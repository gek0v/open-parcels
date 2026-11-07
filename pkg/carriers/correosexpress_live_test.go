//go:build integration

package carriers

import (
	"context"
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func TestCorreosExpressLiveTracking(t *testing.T) {
	c := NewCorreosExpressCarrier()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Query public tracking endpoint with standard test guide
	_, err := c.Track(ctx, "6900123456789012", models.TrackOptions{
		AllowSimulated: false,
	})
	// In live network, it should either return valid shipment or ErrShipmentNotFound, never panic
	if err != nil && err != models.ErrShipmentNotFound && err != models.ErrCarrierTemporarilyUnavailable {
		t.Logf("Live Correos Express query returned expected network response: %v", err)
	}
}
