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

func TestMRWLiveTrackingFlag(t *testing.T) {
	c := NewMRWCarrier()
	if !c.HasLiveTracking() {
		t.Fatalf("expected MRW HasLiveTracking() to be true")
	}
	if !HasLiveTracking(c) {
		t.Fatalf("expected carriers.HasLiveTracking(mrw) to be true")
	}
}

func TestMRWDeliveredWithHTTPTestServer(t *testing.T) {
	fixture, err := LoadFixture("mrw_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("envio") != "028001234567" {
			t.Errorf("unexpected envio param: %s", r.URL.Query().Get("envio"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	teardownURL := SetMRWBaseURLForTest(srv.URL)
	defer teardownURL()
	teardownClient := SetMRWHTTPClientForTest(srv.Client())
	defer teardownClient()

	c := NewMRWCarrier()
	shipment, err := c.Track(context.Background(), "028001234567", models.TrackOptions{
		Language:       "es",
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("unexpected error tracking MRW delivered: %v", err)
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
	expectedDelivered := time.Date(2024, 11, 15, 14, 30, 0, 0, time.UTC)
	if !shipment.DeliveredAt.Equal(expectedDelivered) {
		t.Errorf("expected DeliveredAt %v, got %v", expectedDelivered, shipment.DeliveredAt)
	}
	if shipment.Origin.Name != "Madrid Central" {
		t.Errorf("expected Origin 'Madrid Central', got %q", shipment.Origin.Name)
	}
	if shipment.Destination.Name != "Barcelona Diagonal" {
		t.Errorf("expected Destination 'Barcelona Diagonal', got %q", shipment.Destination.Name)
	}
	if len(shipment.Events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(shipment.Events))
	}
	if shipment.Events[0].Status != models.StatusDelivered {
		t.Errorf("expected newest event to be StatusDelivered, got %s", shipment.Events[0].Status)
	}
	if shipment.Events[3].Stage != models.StagePickedUp {
		t.Errorf("expected oldest event stage StagePickedUp, got %s", shipment.Events[3].Stage)
	}
}

func TestMRWInTransitHTMLWithHTTPTestServer(t *testing.T) {
	fixture, err := LoadFixture("mrw_intransit.html")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	teardownURL := SetMRWBaseURLForTest(srv.URL)
	defer teardownURL()
	teardownClient := SetMRWHTTPClientForTest(srv.Client())
	defer teardownClient()

	c := NewMRWCarrier()
	shipment, err := c.Track(context.Background(), "028009876543", models.TrackOptions{
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("unexpected error tracking MRW HTML: %v", err)
	}

	if shipment.Status != models.StatusInTransit {
		t.Errorf("expected StatusInTransit, got %s", shipment.Status)
	}
	if shipment.Origin.Name != "Valencia Centro" {
		t.Errorf("expected Origin 'Valencia Centro', got %q", shipment.Origin.Name)
	}
	if shipment.Destination.Name != "Sevilla Norte" {
		t.Errorf("expected Destination 'Sevilla Norte', got %q", shipment.Destination.Name)
	}
	if len(shipment.Events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(shipment.Events))
	}
	if shipment.Events[0].Date.Year() != 2024 || shipment.Events[0].Date.Day() != 15 {
		t.Errorf("unexpected parsed event date: %v", shipment.Events[0].Date)
	}
}

func TestMRWExceptionWithHTTPTestServer(t *testing.T) {
	fixture, err := LoadFixture("mrw_exception.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	teardownURL := SetMRWBaseURLForTest(srv.URL)
	defer teardownURL()
	teardownClient := SetMRWHTTPClientForTest(srv.Client())
	defer teardownClient()

	c := NewMRWCarrier()
	shipment, err := c.Track(context.Background(), "028005556667", models.TrackOptions{
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("unexpected error tracking MRW exception: %v", err)
	}

	if shipment.Status != models.StatusFailedAttempt {
		t.Errorf("expected StatusFailedAttempt, got %s", shipment.Status)
	}
}

func TestMRWErrorsAndNoSilentMock(t *testing.T) {
	c := NewMRWCarrier()

	t.Run("not_found_no_mock", func(t *testing.T) {
		fixture, _ := LoadFixture("mrw_not_found.html")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fixture)
		}))
		defer srv.Close()

		teardownURL := SetMRWBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetMRWHTTPClientForTest(srv.Client())
		defer teardownClient()

		_, err := c.Track(context.Background(), "000000000000", models.TrackOptions{AllowSimulated: false})
		if !errors.Is(err, models.ErrShipmentNotFound) {
			t.Errorf("expected ErrShipmentNotFound, got %v", err)
		}

		// With AllowSimulated: true it should return simulated fallback
		sim, err := c.Track(context.Background(), "000000000000", models.TrackOptions{AllowSimulated: true})
		if err != nil || sim == nil || len(sim.Events) == 0 {
			t.Errorf("expected simulated fallback when AllowSimulated=true, got err=%v", err)
		}
	})

	t.Run("blocked_waf_and_403", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("Cloudflare Access Denied"))
		}))
		defer srv.Close()

		teardownURL := SetMRWBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetMRWHTTPClientForTest(srv.Client())
		defer teardownClient()

		_, err := c.Track(context.Background(), "028001234567", models.TrackOptions{AllowSimulated: false})
		if !errors.Is(err, models.ErrCarrierBlocked) {
			t.Errorf("expected ErrCarrierBlocked, got %v", err)
		}
	})

	t.Run("schema_changed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><body><div>Unexpected layout change</div></body></html>"))
		}))
		defer srv.Close()

		teardownURL := SetMRWBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetMRWHTTPClientForTest(srv.Client())
		defer teardownClient()

		_, err := c.Track(context.Background(), "028001234567", models.TrackOptions{AllowSimulated: false})
		if !errors.Is(err, models.ErrScraperSchemaChanged) {
			t.Errorf("expected ErrScraperSchemaChanged, got %v", err)
		}
	})

	t.Run("rate_limit_and_500", func(t *testing.T) {
		srv429 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		teardownURL := SetMRWBaseURLForTest(srv429.URL)
		teardownClient := SetMRWHTTPClientForTest(srv429.Client())
		_, err := c.Track(context.Background(), "028001234567", models.TrackOptions{AllowSimulated: false})
		teardownURL()
		teardownClient()
		srv429.Close()
		if !errors.Is(err, models.ErrCarrierRateLimited) {
			t.Errorf("expected ErrCarrierRateLimited, got %v", err)
		}

		srv500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		teardownURL = SetMRWBaseURLForTest(srv500.URL)
		teardownClient = SetMRWHTTPClientForTest(srv500.Client())
		_, err = c.Track(context.Background(), "028001234567", models.TrackOptions{AllowSimulated: false})
		teardownURL()
		teardownClient()
		srv500.Close()
		if !errors.Is(err, models.ErrCarrierTemporarilyUnavailable) {
			t.Errorf("expected ErrCarrierTemporarilyUnavailable, got %v", err)
		}
	})
}
