package engine

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"open-parcels/pkg/carriers"
	"open-parcels/pkg/models"
	"open-parcels/pkg/storage"
	"path/filepath"
	"time"
)

func TestEngineTrackChaining(t *testing.T) {
	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	eng := New()

	// Track Cainiao number with handover to Correos Express
	shipment, err := eng.Track(context.Background(), "LP00561234567890", models.TrackOptions{
		DestinationCountry: "ES",
		Language:           "es",
		AllowSimulated:     true,
	})
	if err != nil {
		t.Fatalf("expected successful tracking, got err: %v", err)
	}

	if shipment.Carrier.Slug != "cainiao" {
		t.Errorf("expected primary carrier cainiao, got %s", shipment.Carrier.Slug)
	}

	if len(shipment.AdditionalCarriers) == 0 {
		t.Errorf("expected additional carrier (Correos Express) chained for last mile")
	}

	if len(shipment.Events) < 5 {
		t.Errorf("expected merged timeline with events, got %d", len(shipment.Events))
	}

	// Verify chronological ordering: newest event first
	for i := 0; i < len(shipment.Events)-1; i++ {
		if shipment.Events[i].Date.Before(shipment.Events[i+1].Date) {
			t.Errorf("events not sorted in descending chronological order at index %d", i)
		}
	}

	// Check status normalization and progress
	if shipment.Progress <= 0 || shipment.Progress > 100 {
		t.Errorf("progress should be between 1 and 100, got %d", shipment.Progress)
	}

	if shipment.StatusText == "" {
		t.Errorf("expected localized status text, got empty")
	}

	// Test ParcelsApp v4 response conversion
	pResp := eng.ToParcelsAppResponse(shipment, "test-req-123")
	if pResp == nil || len(pResp.Results) == 0 {
		t.Fatalf("expected ParcelsApp response with results")
	}
	if pResp.Results[0].Shipment.TrackingID != "LP00561234567890" {
		t.Errorf("expected tracking ID in ParcelsApp shipment, got %s", pResp.Results[0].Shipment.TrackingID)
	}
	if len(pResp.Results[0].Shipment.Services) < 2 {
		t.Errorf("expected multiple services listed in ParcelsApp response for chained journey")
	}
}

func TestEngineAirCargo(t *testing.T) {
	eng := New()
	shipment, err := eng.Track(context.Background(), "157-52471764", models.TrackOptions{
		AllowSimulated: true,
	})
	if err != nil {
		t.Fatalf("unexpected error tracking AWB: %v", err)
	}

	if shipment.Carrier.Slug != "air-cargo" {
		t.Errorf("expected air-cargo carrier, got %s", shipment.Carrier.Slug)
	}

	if len(shipment.FlightLegs) == 0 {
		t.Errorf("expected flight legs for air cargo tracking")
	}
}

func TestEngineItalianTrackingAndLocalization(t *testing.T) {
	eng := New()

	shipment, err := eng.Track(context.Background(), "RA123456785IT", models.TrackOptions{
		DestinationCountry: "IT",
		Language:           "it",
		AllowSimulated:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error tracking Italian shipment: %v", err)
	}

	if shipment.Carrier.Slug != "poste-italiane" {
		t.Errorf("expected carrier poste-italiane, got %s", shipment.Carrier.Slug)
	}

	if shipment.StatusText != "In consegna" {
		t.Errorf("expected Italian localized status 'In consegna', got %s", shipment.StatusText)
	}

	// Verify normalizer for Italian events
	status, stage := NormalizeEventStatus("Pacco consegnato al destinatario")
	if status != models.StatusDelivered || stage != models.StageDelivery {
		t.Errorf("expected Delivered/Delivery, got %s/%s", status, stage)
	}

	status, stage = NormalizeEventStatus("Spedizione in giacenza")
	if status != models.StatusException || stage != models.StageException {
		t.Errorf("expected Exception/Exception, got %s/%s", status, stage)
	}
}

func TestEngineTracking323CainiaoAndCorreosExpress(t *testing.T) {
	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	eng := New()

	// 1. Auto-detection tracking: 32300035210642701471403 with ES language
	shipment, err := eng.Track(context.Background(), "32300035210642701471403", models.TrackOptions{
		Language: "es",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if shipment.Status != models.StatusDelivered {
		t.Errorf("expected status Delivered, got %s", shipment.Status)
	}
	if shipment.Progress != 100 {
		t.Errorf("expected progress 100, got %d", shipment.Progress)
	}
	if shipment.Carrier.Slug != "cainiao" {
		t.Errorf("expected primary carrier cainiao, got %s", shipment.Carrier.Slug)
	}

	// Verify Correos Express is in AdditionalCarriers
	hasCE := false
	for _, ac := range shipment.AdditionalCarriers {
		if ac.Slug == "correosexpress" {
			hasCE = true
			break
		}
	}
	if !hasCE {
		t.Errorf("expected Correos Express in AdditionalCarriers")
	}

	// Verify events show Correos Express for local delivery
	hasCorreosEvent := false
	for _, ev := range shipment.Events {
		if ev.CarrierSlug == "correosexpress" {
			hasCorreosEvent = true
			break
		}
	}
	if !hasCorreosEvent {
		t.Errorf("expected at least one event attributed to Correos Express")
	}

	// 2. Direct query with correosexpress hint
	ceShipment, err := eng.Track(context.Background(), "32300035210642701471403", models.TrackOptions{
		CarrierHint:    "correosexpress",
		Language:       "es",
		AllowSimulated: true,
	})
	if err != nil {
		t.Fatalf("unexpected error with correosexpress hint: %v", err)
	}
	if ceShipment.Status != models.StatusDelivered {
		t.Errorf("expected status Delivered with correosexpress hint, got %s", ceShipment.Status)
	}
	if ceShipment.Progress != 100 {
		t.Errorf("expected progress 100 with correosexpress hint, got %d", ceShipment.Progress)
	}
	// Verify Cainiao is present
	hasCainiao := false
	if ceShipment.Carrier.Slug == "cainiao" {
		hasCainiao = true
	}
	for _, ac := range ceShipment.AdditionalCarriers {
		if ac.Slug == "cainiao" {
			hasCainiao = true
			break
		}
	}
	if !hasCainiao {
		t.Errorf("expected Cainiao Global represented when tracked with correosexpress hint")
	}
}

func TestNormalizerInPostBeingDelivered(t *testing.T) {
	// 1. Normalizer should map "Parcel is being delivered to destination locker / Punto Pack" to OutForDelivery
	desc := "Parcel is being delivered to destination locker / Punto Pack"
	status, stage := NormalizeEventStatus(desc)
	if status != models.StatusOutForDelivery {
		t.Errorf("expected StatusOutForDelivery for '%s', got %s", desc, status)
	}
	if stage != models.StageLastMile {
		t.Errorf("expected StageLastMile, got %s", stage)
	}

	// 2. IsDeliveredText should return false for "being delivered", "to be delivered", "undelivered"
	if IsDeliveredText(desc) {
		t.Errorf("IsDeliveredText returned true for '%s'", desc)
	}
	if IsDeliveredText("Package is to be delivered today") {
		t.Errorf("IsDeliveredText returned true for 'Package is to be delivered today'")
	}
	if IsDeliveredText("Package undelivered: address incomplete") {
		t.Errorf("IsDeliveredText returned true for 'Package undelivered: address incomplete'")
	}

	// 3. True delivery text must still return true
	if !IsDeliveredText("Parcel delivered to locker") {
		t.Errorf("IsDeliveredText returned false for 'Parcel delivered to locker'")
	}
	if !IsDeliveredText("Envío entregado al destinatario") {
		t.Errorf("IsDeliveredText returned false for 'Envío entregado al destinatario'")
	}

	// 4. Test InPost tracking through engine (guide 7561674)
	eng := New()
	shipment, err := eng.Track(context.Background(), "7561674", models.TrackOptions{
		CarrierHint:    "inpost",
		PostalCode:     "28001",
		AllowSimulated: true,
	})
	if err != nil {
		t.Fatalf("unexpected error tracking InPost: %v", err)
	}

	if shipment.Status != models.StatusOutForDelivery {
		t.Errorf("expected status OutForDelivery (%s), got %s", models.StatusOutForDelivery, shipment.Status)
	}
	if shipment.Progress != 85 {
		t.Errorf("expected progress 85%%, got %d%%", shipment.Progress)
	}
	if shipment.DeliveredAt != nil {
		t.Errorf("DeliveredAt should be nil for out_for_delivery parcel, got %v", shipment.DeliveredAt)
	}
}

func TestEngineNoSimulatedByDefault(t *testing.T) {
	eng := New()

	// 1. Invalid or non-existent tracking number must return ErrShipmentNotFound
	_, err := eng.Track(context.Background(), "FAKE000000000", models.TrackOptions{})
	if err == nil {
		t.Fatalf("expected error for non-existent guide, got nil")
	}
	if !errors.Is(err, models.ErrShipmentNotFound) && !errors.Is(err, models.ErrCarrierNotImplemented) {
		t.Errorf("expected ErrShipmentNotFound or ErrCarrierNotImplemented, got: %v", err)
	}

	// 2. Catalog/mock carrier without live adapter must return ErrCarrierNotImplemented when AllowSimulated is false
	_, err = eng.Track(context.Background(), "TIP123456789", models.TrackOptions{
		CarrierHint:    "tipsa",
		AllowSimulated: false,
	})
	if err == nil {
		t.Fatalf("expected error for unimplemented carrier without AllowSimulated")
	}
	if !errors.Is(err, models.ErrCarrierNotImplemented) {
		t.Errorf("expected ErrCarrierNotImplemented, got: %v", err)
	}

	// 3. When AllowSimulated is true, simulated carrier returns shipment without error
	simShipment, err := eng.Track(context.Background(), "TIP123456789", models.TrackOptions{
		CarrierHint:    "tipsa",
		AllowSimulated: true,
	})
	if err != nil {
		t.Fatalf("expected simulated shipment with AllowSimulated=true, got: %v", err)
	}
	if simShipment == nil || len(simShipment.Events) == 0 {
		t.Errorf("expected non-empty simulated events when AllowSimulated=true")
	}

	// 4. InPost without postal code must return ErrPostalCodeRequired
	_, err = eng.Track(context.Background(), "7561674", models.TrackOptions{
		CarrierHint:    "inpost",
		AllowSimulated: false,
	})
	if err == nil {
		t.Fatalf("expected error for InPost without postal code")
	}
	if !errors.Is(err, models.ErrPostalCodeRequired) && !errors.Is(err, models.ErrMissingPostalCode) {
		t.Errorf("expected ErrPostalCodeRequired, got: %v", err)
	}
}




func TestSmartTTL(t *testing.T) {
	if got := SmartTTL(models.StatusDelivered); got != 24*time.Hour {
		t.Errorf("expected 24h for delivered, got %v", got)
	}
	if got := SmartTTL(models.StatusExpired); got != 24*time.Hour {
		t.Errorf("expected 24h for expired, got %v", got)
	}
	if got := SmartTTL(models.StatusInTransit); got != 10*time.Minute {
		t.Errorf("expected 10m for in_transit, got %v", got)
	}
	if got := SmartTTL(models.StatusOutForDelivery); got != 10*time.Minute {
		t.Errorf("expected 10m for out_for_delivery, got %v", got)
	}
	if got := SmartTTL(models.StatusInfoReceived); got != 10*time.Minute {
		t.Errorf("expected 10m for info_received, got %v", got)
	}
	if got := SmartTTL(models.StatusFailedAttempt); got != 10*time.Minute {
		t.Errorf("expected 10m for failed_attempt, got %v", got)
	}
}

func TestEngineSmartCachingAndForceRefresh(t *testing.T) {
	fixtureData, err := carriers.LoadFixture("cainiao_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}
	teardown := carriers.SetCainiaoHTTPClientForTest(carriers.NewFixtureClient(http.StatusOK, fixtureData, "application/json"))
	defer teardown()

	stores := map[string]storage.Store{
		"memory": storage.NewMemoryStore(),
	}
	tempDB := filepath.Join(t.TempDir(), "engine_cache.db")
	sqlStore, err := storage.NewSQLiteStore(tempDB)
	if err != nil {
		t.Fatalf("failed to open sqlite store: %v", err)
	}
	stores["sqlite"] = sqlStore

	for name, st := range stores {
		t.Run(name, func(t *testing.T) {
			defer st.Close()
			eng := NewWithStore(st)

			// 1. First track: live tracking executed, FromCache is false
			s1, err := eng.Track(context.Background(), "LP00561234567890", models.TrackOptions{
				AllowSimulated: true,
				Language:       "es",
			})
			if err != nil {
				t.Fatalf("unexpected error tracking: %v", err)
			}
			if s1.FromCache {
				t.Errorf("first tracking must have FromCache=false")
			}
			if s1.Status != models.StatusDelivered {
				t.Errorf("expected StatusDelivered, got %s", s1.Status)
			}

			// 2. Second track: fetched from cache, FromCache is true
			s2, err := eng.Track(context.Background(), "LP00561234567890", models.TrackOptions{
				AllowSimulated: true,
				Language:       "en",
			})
			if err != nil {
				t.Fatalf("unexpected error on second tracking: %v", err)
			}
			if !s2.FromCache {
				t.Errorf("second tracking must hit cache (FromCache=true)")
			}
			if s2.StatusText != "Delivered" {
				t.Errorf("expected status text localized to English from cache, got %s", s2.StatusText)
			}

			// 3. Third track with ForceRefresh: bypasses cache and updates record
			s3, err := eng.Track(context.Background(), "LP00561234567890", models.TrackOptions{
				AllowSimulated: true,
				Language:       "es",
				ForceRefresh:   true,
			})
			if err != nil {
				t.Fatalf("unexpected error on forced refresh: %v", err)
			}
			if s3.FromCache {
				t.Errorf("forced refresh must have FromCache=false")
			}
		})
	}
}

func TestEngineNeverCachesTransientErrors(t *testing.T) {
	st := storage.NewMemoryStore()
	eng := NewWithStore(st)

	// 1. Not found tracking
	_, err := eng.Track(context.Background(), "NONEXISTENT12345", models.TrackOptions{
		CarrierHint:    "cainiao",
		AllowSimulated: true,
	})
	if err == nil {
		t.Fatalf("expected error for non-existent tracking")
	}

	cached, found, _ := st.GetShipment(context.Background(), "cainiao", "NONEXISTENT12345", "")
	if found || cached != nil {
		t.Errorf("transient error must never be saved to cache")
	}

	// 2. Missing postal code
	_, err = eng.Track(context.Background(), "7561674", models.TrackOptions{
		CarrierHint:    "inpost",
		AllowSimulated: false,
	})
	if err == nil {
		t.Fatalf("expected postal code error")
	}

	cached, found, _ = st.GetShipment(context.Background(), "inpost", "7561674", "")
	if found || cached != nil {
		t.Errorf("postal code validation error must never be saved to cache")
	}
}
