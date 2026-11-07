//go:build integration

package carriers

import (
	"context"
	"os"
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func TestFedExLiveTracking(t *testing.T) {
	clientID := os.Getenv("FEDEX_CLIENT_ID")
	if clientID == "" {
		t.Skip("skipping live FedEx test: FEDEX_CLIENT_ID not set")
	}

	c := NewFedExCarrier()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.Track(ctx, "123456789012", models.TrackOptions{
		AllowSimulated: false,
	})
	if err != nil && err != models.ErrShipmentNotFound && err != models.ErrCarrierTemporarilyUnavailable {
		t.Logf("Live FedEx response: %v", err)
	}
}
