//go:build integration

package carriers

import (
	"context"
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func TestCorreosSpainLiveTracking(t *testing.T) {
	c := NewCorreosSpainCarrier()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.Track(ctx, "PQ4F6P07000000000000000", models.TrackOptions{
		AllowSimulated: false,
	})
	if err != nil && err != models.ErrShipmentNotFound && err != models.ErrCarrierTemporarilyUnavailable {
		t.Logf("Live Correos Spain tracking response: %v", err)
	}
}
