//go:build integration

package carriers

import (
	"context"
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func TestGLSSpainLiveTracking(t *testing.T) {
	c := NewGLSSpainCarrier()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.Track(ctx, "ASM123456789", models.TrackOptions{
		PostalCode:     "28001",
		AllowSimulated: false,
	})
	if err != nil && err != models.ErrShipmentNotFound && err != models.ErrCarrierTemporarilyUnavailable {
		t.Logf("Live GLS Spain tracking returned: %v", err)
	}
}
