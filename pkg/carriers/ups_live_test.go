//go:build integration

package carriers

import (
	"context"
	"os"
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func TestUPSLiveTracking(t *testing.T) {
	clientID := os.Getenv("UPS_CLIENT_ID")
	if clientID == "" {
		t.Skip("skipping live UPS test: UPS_CLIENT_ID not set")
	}

	c := NewUPSCarrier()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.Track(ctx, "1Z12345E0205271688", models.TrackOptions{
		AllowSimulated: false,
	})
	if err != nil && err != models.ErrShipmentNotFound && err != models.ErrCarrierTemporarilyUnavailable {
		t.Logf("Live UPS response: %v", err)
	}
}
