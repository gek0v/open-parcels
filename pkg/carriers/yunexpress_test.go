package carriers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func TestYunExpressLiveTrackingFlag(t *testing.T) {
	c := NewYunExpressCarrier()
	if !c.HasLiveTracking() {
		t.Fatalf("expected YunExpress HasLiveTracking() to be true")
	}
	if !HasLiveTracking(c) {
		t.Fatalf("expected carriers.HasLiveTracking(yunexpress) to be true")
	}
}

func TestYunExpressDeliveredWithLastMileHTTPTestServer(t *testing.T) {
	fixture, err := LoadFixture("yunexpress_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("nums") != "YT2401234567890123" {
			t.Errorf("unexpected nums param: %s", r.URL.Query().Get("nums"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	teardownURL := SetYunExpressBaseURLForTest(srv.URL)
	defer teardownURL()
	teardownClient := SetYunExpressHTTPClientForTest(srv.Client())
	defer teardownClient()

	c := NewYunExpressCarrier()
	shipment, err := c.Track(context.Background(), "YT2401234567890123", models.TrackOptions{
		DestinationCountry: "ES",
		AllowSimulated:     false,
	})
	if err != nil {
		t.Fatalf("unexpected error tracking YunExpress delivered: %v", err)
	}

	if shipment.Status != models.StatusDelivered {
		t.Errorf("expected StatusDelivered, got %s", shipment.Status)
	}
	if shipment.Progress != 100 {
		t.Errorf("expected Progress 100, got %d", shipment.Progress)
	}
	if shipment.DeliveredAt == nil {
		t.Fatalf("expected DeliveredAt populated")
	}
	expectedDelivered := time.Date(2024, 11, 15, 14, 20, 0, 0, time.UTC)
	if !shipment.DeliveredAt.Equal(expectedDelivered) {
		t.Errorf("expected DeliveredAt %v, got %v", expectedDelivered, shipment.DeliveredAt)
	}
	if shipment.LastMileTrackingNumber != "PQ4F6P07000000000000000" {
		t.Errorf("expected LastMileTrackingNumber PQ4F6P07000000000000000, got %q", shipment.LastMileTrackingNumber)
	}
	if shipment.LastMileCarrier == nil || shipment.LastMileCarrier.Slug != "correos-spain" {
		t.Errorf("expected LastMileCarrier correos-spain, got %v", shipment.LastMileCarrier)
	}
	if len(shipment.AdditionalCarriers) == 0 || shipment.AdditionalCarriers[0].Slug != "correos-spain" {
		t.Errorf("expected Correos Spain in AdditionalCarriers, got %v", shipment.AdditionalCarriers)
	}
	if shipment.Origin.Code != "CN" {
		t.Errorf("expected Origin code CN, got %s", shipment.Origin.Code)
	}
	if shipment.Destination.Code != "ES" {
		t.Errorf("expected Destination code ES, got %s", shipment.Destination.Code)
	}
	if len(shipment.Events) != 7 {
		t.Fatalf("expected 7 events, got %d", len(shipment.Events))
	}
	if shipment.Events[0].CarrierSlug != "correos-spain" || shipment.Events[0].CarrierIndex != 1 {
		t.Errorf("expected newest event attributed to correos-spain (index 1), got %s (index %d)", shipment.Events[0].CarrierSlug, shipment.Events[0].CarrierIndex)
	}
	if shipment.Events[6].CarrierSlug != "yunexpress" || shipment.Events[6].CarrierIndex != 0 {
		t.Errorf("expected oldest event attributed to yunexpress (index 0), got %s (index %d)", shipment.Events[6].CarrierSlug, shipment.Events[6].CarrierIndex)
	}
}

func TestYunExpressInTransitAndExceptionWithHTTPTestServer(t *testing.T) {
	c := NewYunExpressCarrier()

	t.Run("in_transit", func(t *testing.T) {
		fixture, _ := LoadFixture("yunexpress_intransit.json")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fixture)
		}))
		defer srv.Close()

		teardownURL := SetYunExpressBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetYunExpressHTTPClientForTest(srv.Client())
		defer teardownClient()

		shipment, err := c.Track(context.Background(), "YT2409876543210987", models.TrackOptions{AllowSimulated: false})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if shipment.Status != models.StatusInTransit {
			t.Errorf("expected StatusInTransit, got %s", shipment.Status)
		}
		if shipment.DeliveredAt != nil {
			t.Errorf("expected DeliveredAt nil")
		}
	})

	t.Run("exception", func(t *testing.T) {
		fixture, _ := LoadFixture("yunexpress_exception.json")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fixture)
		}))
		defer srv.Close()

		teardownURL := SetYunExpressBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetYunExpressHTTPClientForTest(srv.Client())
		defer teardownClient()

		shipment, err := c.Track(context.Background(), "YT2405556667778889", models.TrackOptions{AllowSimulated: false})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if shipment.Status != models.StatusException {
			t.Errorf("expected StatusException, got %s", shipment.Status)
		}
	})
}

func TestYunExpressErrorsAndNoSilentMock(t *testing.T) {
	c := NewYunExpressCarrier()

	t.Run("not_found", func(t *testing.T) {
		fixture, _ := LoadFixture("yunexpress_not_found.json")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fixture)
		}))
		defer srv.Close()

		teardownURL := SetYunExpressBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetYunExpressHTTPClientForTest(srv.Client())
		defer teardownClient()

		_, err := c.Track(context.Background(), "YT0000000000000000", models.TrackOptions{AllowSimulated: false})
		if !errors.Is(err, models.ErrShipmentNotFound) {
			t.Errorf("expected ErrShipmentNotFound, got %v", err)
		}

		sim, err := c.Track(context.Background(), "YT0000000000000000", models.TrackOptions{AllowSimulated: true})
		if err != nil || sim == nil || len(sim.Events) == 0 {
			t.Errorf("expected simulated fallback when AllowSimulated=true, got err=%v", err)
		}
	})

	t.Run("blocked_antibot", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success":false,"code":"SEC_CHALLENGE","message":"ANTI_BOT check required"}`))
		}))
		defer srv.Close()

		teardownURL := SetYunExpressBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetYunExpressHTTPClientForTest(srv.Client())
		defer teardownClient()

		_, err := c.Track(context.Background(), "YT2401234567890123", models.TrackOptions{AllowSimulated: false})
		if !errors.Is(err, models.ErrCarrierBlocked) {
			t.Errorf("expected ErrCarrierBlocked, got %v", err)
		}
	})

	t.Run("schema_changed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success":true,"code":"200","data":{"unknown":123}}`))
		}))
		defer srv.Close()

		teardownURL := SetYunExpressBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetYunExpressHTTPClientForTest(srv.Client())
		defer teardownClient()

		_, err := c.Track(context.Background(), "YT2401234567890123", models.TrackOptions{AllowSimulated: false})
		if !errors.Is(err, models.ErrScraperSchemaChanged) {
			t.Errorf("expected ErrScraperSchemaChanged, got %v", err)
		}
	})
}
