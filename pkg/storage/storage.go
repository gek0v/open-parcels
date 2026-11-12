package storage

import (
	"context"
	"os"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

// EnvDBPath is the environment variable configuring the SQLite database file path.
const EnvDBPath = "OPEN_PARCELS_DB_PATH"

// AsyncJob represents a persisted ParcelsApp v4 asynchronous tracking job.
type AsyncJob struct {
	RequestID string                     `json:"request_id"`
	Status    string                     `json:"status"` // "processing" | "completed" | "failed"
	Response  *models.ParcelsAppResponse `json:"response,omitempty"`
	Error     string                     `json:"error,omitempty"`
	CreatedAt time.Time                  `json:"created_at"`
	UpdatedAt time.Time                  `json:"updated_at"`
	ExpiresAt time.Time                  `json:"expires_at"`
}

// Store defines a pluggable persistence and caching backend for shipments,
// asynchronous tracking jobs, and OAuth2 carrier tokens.
type Store interface {
	// Shipment cache keyed by (carrier, trackingNumber, postalCode)
	GetShipment(ctx context.Context, carrier, trackingNumber, postalCode string) (*models.Shipment, bool, error)
	SetShipment(ctx context.Context, carrier, trackingNumber, postalCode string, shipment *models.Shipment, ttl time.Duration) error
	DeleteShipment(ctx context.Context, carrier, trackingNumber, postalCode string) error

	// Async v4 tracking jobs
	GetAsyncJob(ctx context.Context, requestID string) (*AsyncJob, bool, error)
	SetAsyncJob(ctx context.Context, job *AsyncJob) error
	DeleteAsyncJob(ctx context.Context, requestID string) error

	// OAuth2 tokens
	GetOAuthToken(ctx context.Context, provider string) (token string, expiresAt time.Time, found bool, err error)
	SetOAuthToken(ctx context.Context, provider, token string, expiresAt time.Time) error
	DeleteOAuthToken(ctx context.Context, provider string) error

	// Maintenance & lifecycle
	PurgeExpired(ctx context.Context) error
	Close() error
}

// NewStore opens a Store for the given path.
// If dbPath is empty, it returns an in-memory Store.
// If dbPath is ":memory:", it returns an in-memory SQLiteStore.
// Otherwise it opens a persistent SQLiteStore at dbPath.
func NewStore(dbPath string) (Store, error) {
	trimmed := strings.TrimSpace(dbPath)
	if trimmed == "" {
		return NewMemoryStore(), nil
	}
	return NewSQLiteStore(trimmed)
}

// NewStoreFromEnv initializes a Store based on OPEN_PARCELS_DB_PATH.
func NewStoreFromEnv() (Store, error) {
	return NewStore(os.Getenv(EnvDBPath))
}

func normalizeCarrier(carrier string) string {
	return strings.ToLower(strings.TrimSpace(carrier))
}

func normalizeTrackingNumber(trackingNumber string) string {
	return strings.ToUpper(strings.TrimSpace(trackingNumber))
}

func normalizePostalCode(postalCode string) string {
	return strings.ToUpper(strings.TrimSpace(postalCode))
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func cloneShipment(s *models.Shipment) *models.Shipment {
	if s == nil {
		return nil
	}
	cp := *s
	if s.AdditionalCarriers != nil {
		cp.AdditionalCarriers = make([]models.CarrierInfo, len(s.AdditionalCarriers))
		copy(cp.AdditionalCarriers, s.AdditionalCarriers)
	}
	if s.LastMileCarrier != nil {
		lm := *s.LastMileCarrier
		cp.LastMileCarrier = &lm
	}
	if s.EstimatedDelivery != nil {
		t := *s.EstimatedDelivery
		cp.EstimatedDelivery = &t
	}
	if s.DeliveredAt != nil {
		t := *s.DeliveredAt
		cp.DeliveredAt = &t
	}
	if s.Events != nil {
		cp.Events = make([]models.TrackingEvent, len(s.Events))
		copy(cp.Events, s.Events)
	}
	if s.FlightLegs != nil {
		cp.FlightLegs = make([]models.FlightLeg, len(s.FlightLegs))
		copy(cp.FlightLegs, s.FlightLegs)
	}
	return &cp
}

func cloneAsyncJob(j *AsyncJob) *AsyncJob {
	if j == nil {
		return nil
	}
	cp := *j
	if j.Response != nil {
		cp.Response = cloneParcelsAppResponse(j.Response)
	}
	return &cp
}

func cloneParcelsAppResponse(r *models.ParcelsAppResponse) *models.ParcelsAppResponse {
	if r == nil {
		return nil
	}
	cp := *r
	if r.Results != nil {
		cp.Results = make([]models.ParcelsAppResultItem, len(r.Results))
		for i, item := range r.Results {
			cpItem := item
			if item.Shipment != nil {
				sh := *item.Shipment
				if item.Shipment.States != nil {
					sh.States = make([]models.ParcelsAppStateItem, len(item.Shipment.States))
					copy(sh.States, item.Shipment.States)
				}
				if item.Shipment.Services != nil {
					sh.Services = make([]models.ParcelsAppServiceItem, len(item.Shipment.Services))
					copy(sh.Services, item.Shipment.Services)
				}
				if item.Shipment.Detected != nil {
					sh.Detected = make([]int, len(item.Shipment.Detected))
					copy(sh.Detected, item.Shipment.Detected)
				}
				if item.Shipment.Attributes != nil {
					sh.Attributes = make([]models.ParcelsAppAttribute, len(item.Shipment.Attributes))
					copy(sh.Attributes, item.Shipment.Attributes)
				}
				if item.Shipment.ExternalTracking != nil {
					sh.ExternalTracking = make([]models.ParcelsAppExternalTrack, len(item.Shipment.ExternalTracking))
					copy(sh.ExternalTracking, item.Shipment.ExternalTracking)
				}
				if item.Shipment.FlightLegs != nil {
					sh.FlightLegs = make([]models.FlightLeg, len(item.Shipment.FlightLegs))
					copy(sh.FlightLegs, item.Shipment.FlightLegs)
				}
				cpItem.Shipment = &sh
			}
			cp.Results[i] = cpItem
		}
	}
	if r.Rejected != nil {
		cp.Rejected = make([]models.ParcelsAppRejected, len(r.Rejected))
		copy(cp.Rejected, r.Rejected)
	}
	return &cp
}