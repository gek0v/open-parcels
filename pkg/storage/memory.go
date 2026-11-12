package storage

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"open-parcels/pkg/models"
)

type cachedShipmentItem struct {
	carrier        string
	trackingNumber string
	postalCode     string
	shipment       *models.Shipment
	expiresAt      time.Time
}

type cachedOAuthItem struct {
	token     string
	expiresAt time.Time
}

// MemoryStore is an in-memory, concurrent-safe implementation of Store.
type MemoryStore struct {
	mu        sync.RWMutex
	shipments map[string]*cachedShipmentItem
	jobs      map[string]*AsyncJob
	tokens    map[string]*cachedOAuthItem
}

// NewMemoryStore creates an initialized in-memory Store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		shipments: make(map[string]*cachedShipmentItem),
		jobs:      make(map[string]*AsyncJob),
		tokens:    make(map[string]*cachedOAuthItem),
	}
}

func shipmentKey(carrier, trackingNumber, postalCode string) string {
	return fmt.Sprintf("%s|%s|%s", normalizeCarrier(carrier), normalizeTrackingNumber(trackingNumber), normalizePostalCode(postalCode))
}

// GetShipment retrieves a cached shipment if it exists and has not expired.
// If carrier is empty, it matches any active shipment for trackingNumber and postalCode.
func (m *MemoryStore) GetShipment(ctx context.Context, carrier, trackingNumber, postalCode string) (*models.Shipment, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	normCarrier := normalizeCarrier(carrier)
	normNumber := normalizeTrackingNumber(trackingNumber)
	normPostal := normalizePostalCode(postalCode)
	now := time.Now()

	if normCarrier != "" {
		key := fmt.Sprintf("%s|%s|%s", normCarrier, normNumber, normPostal)
		item, found := m.shipments[key]
		if !found {
			return nil, false, nil
		}
		if !now.Before(item.expiresAt) {
			return nil, false, nil
		}
		cp := cloneShipment(item.shipment)
		cp.FromCache = true
		return cp, true, nil
	}

	// Match any carrier for trackingNumber and postalCode
	for _, item := range m.shipments {
		if item.trackingNumber == normNumber && item.postalCode == normPostal {
			if !now.Before(item.expiresAt) {
				continue
			}
			cp := cloneShipment(item.shipment)
			cp.FromCache = true
			return cp, true, nil
		}
	}

	return nil, false, nil
}

// SetShipment stores a shipment with the specified TTL.
func (m *MemoryStore) SetShipment(ctx context.Context, carrier, trackingNumber, postalCode string, shipment *models.Shipment, ttl time.Duration) error {
	if shipment == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	normCarrier := normalizeCarrier(carrier)
	if normCarrier == "" && shipment.Carrier.Slug != "" {
		normCarrier = normalizeCarrier(shipment.Carrier.Slug)
	}
	normNumber := normalizeTrackingNumber(trackingNumber)
	if normNumber == "" && shipment.TrackingNumber != "" {
		normNumber = normalizeTrackingNumber(shipment.TrackingNumber)
	}
	normPostal := normalizePostalCode(postalCode)

	key := fmt.Sprintf("%s|%s|%s", normCarrier, normNumber, normPostal)
	m.shipments[key] = &cachedShipmentItem{
		carrier:        normCarrier,
		trackingNumber: normNumber,
		postalCode:     normPostal,
		shipment:       cloneShipment(shipment),
		expiresAt:      time.Now().Add(ttl),
	}

	return nil
}

// DeleteShipment removes a cached shipment entry.
func (m *MemoryStore) DeleteShipment(ctx context.Context, carrier, trackingNumber, postalCode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	normCarrier := normalizeCarrier(carrier)
	normNumber := normalizeTrackingNumber(trackingNumber)
	normPostal := normalizePostalCode(postalCode)

	if normCarrier != "" {
		key := fmt.Sprintf("%s|%s|%s", normCarrier, normNumber, normPostal)
		delete(m.shipments, key)
		return nil
	}

	for k, item := range m.shipments {
		if item.trackingNumber == normNumber && item.postalCode == normPostal {
			delete(m.shipments, k)
		}
	}
	return nil
}

// GetAsyncJob retrieves a ParcelsApp v4 async job by request ID.
func (m *MemoryStore) GetAsyncJob(ctx context.Context, requestID string) (*AsyncJob, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	reqID := strings.TrimSpace(requestID)
	job, found := m.jobs[reqID]
	if !found {
		return nil, false, nil
	}
	if !time.Now().Before(job.ExpiresAt) {
		return nil, false, nil
	}
	return cloneAsyncJob(job), true, nil
}

// SetAsyncJob persists or updates a ParcelsApp v4 async tracking job.
func (m *MemoryStore) SetAsyncJob(ctx context.Context, job *AsyncJob) error {
	if job == nil || strings.TrimSpace(job.RequestID) == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	reqID := strings.TrimSpace(job.RequestID)
	cp := cloneAsyncJob(job)
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	cp.UpdatedAt = time.Now()
	if cp.ExpiresAt.IsZero() {
		cp.ExpiresAt = cp.UpdatedAt.Add(1 * time.Hour)
	}

	m.jobs[reqID] = cp
	return nil
}

// DeleteAsyncJob removes an async job from the store.
func (m *MemoryStore) DeleteAsyncJob(ctx context.Context, requestID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.jobs, strings.TrimSpace(requestID))
	return nil
}

// GetOAuthToken retrieves a valid, non-expired OAuth token for a carrier.
func (m *MemoryStore) GetOAuthToken(ctx context.Context, provider string) (string, time.Time, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p := normalizeProvider(provider)
	item, found := m.tokens[p]
	if !found {
		return "", time.Time{}, false, nil
	}
	if !time.Now().Before(item.expiresAt) {
		return "", time.Time{}, false, nil
	}
	return item.token, item.expiresAt, true, nil
}

// SetOAuthToken saves or replaces an OAuth token for a carrier.
func (m *MemoryStore) SetOAuthToken(ctx context.Context, provider, token string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p := normalizeProvider(provider)
	m.tokens[p] = &cachedOAuthItem{
		token:     token,
		expiresAt: expiresAt,
	}
	return nil
}

// DeleteOAuthToken deletes an OAuth token for a carrier.
func (m *MemoryStore) DeleteOAuthToken(ctx context.Context, provider string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.tokens, normalizeProvider(provider))
	return nil
}

// PurgeExpired clears entries whose TTL has elapsed.
func (m *MemoryStore) PurgeExpired(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for k, v := range m.shipments {
		if !now.Before(v.expiresAt) {
			delete(m.shipments, k)
		}
	}
	for k, v := range m.jobs {
		if !now.Before(v.ExpiresAt) {
			delete(m.jobs, k)
		}
	}
	for k, v := range m.tokens {
		if !now.Before(v.expiresAt) {
			delete(m.tokens, k)
		}
	}
	return nil
}

// Close is a no-op for in-memory storage.
func (m *MemoryStore) Close() error {
	return nil
}