//go:build integration

package carriers

import (
	"context"
	"os"
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func TestDHLLiveTracking(t *testing.T) {
	apiKey := os.Getenv("DHL_API_KEY")
	if apiKey == "" {
		t.Skip("skipping live DHL test: DHL_API_KEY not set")
	}

	c := NewDHLExpressCarrier()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.Track(ctx, "1234567890", models.TrackOptions{
		AllowSimulated: false,
	})
	if err != nil && err != models.ErrShipmentNotFound && err != models.ErrCarrierTemporarilyUnavailable {
		t.Logf("Live DHL response: %v", err)
	}
}
