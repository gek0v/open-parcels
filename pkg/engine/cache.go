package engine

import (
	"time"

	"open-parcels/pkg/models"
)

const (
	// DefaultTTLDelivered is the cache lifetime for delivered or terminal parcels (24h).
	DefaultTTLDelivered = 24 * time.Hour

	// DefaultTTLActive is the cache lifetime for active in-transit parcels (10m).
	DefaultTTLActive = 10 * time.Minute
)

// SmartTTL derives the cache time-to-live based on shipment status.
// Delivered/Terminal status gets a long TTL (24h), while active in-transit states get 10m.
func SmartTTL(status models.Status) time.Duration {
	switch status {
	case models.StatusDelivered, models.StatusExpired:
		return DefaultTTLDelivered
	case models.StatusInTransit, models.StatusOutForDelivery, models.StatusInfoReceived, models.StatusFailedAttempt:
		return DefaultTTLActive
	default:
		return DefaultTTLActive
	}
}