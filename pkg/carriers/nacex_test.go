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

func TestNacexLiveTrackingFlag(t *testing.T) {
	c := NewNacexCarrier()
	if !c.HasLiveTracking() {
		t.Fatalf("expected Nacex HasLiveTracking() to be true")
	}
	if !HasLiveTracking(c) {
		t.Fatalf("expected carriers.HasLiveTracking(nacex) to be true")
	}
}

func TestNacexDeliveredWithHTTPTestServer(t *testing.T) {
	fixture, err := LoadFixture("nacex_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("albaran_rec") != "NCX123456789" {
			t.Errorf("unexpected albaran_rec param: %s", r.URL.Query().Get("albaran_rec"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	teardownURL := SetNacexBaseURLForTest(srv.URL)
	defer teardownURL()
	teardownClient := SetNacexHTTPClientForTest(srv.Client())
	defer teardownClient()

	c := NewNacexCarrier()
	shipment, err := c.Track(context.Background(), "NCX123456789", models.TrackOptions{
		Language:       "es",
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("unexpected error tracking Nacex delivered: %v", err)
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
	expectedDelivered := time.Date(2024, 11, 15, 12, 45, 0, 0, time.UTC)
	if !shipment.DeliveredAt.Equal(expectedDelivered) {
		t.Errorf("expected DeliveredAt %v, got %v", expectedDelivered, shipment.DeliveredAt)
	}
	if shipment.Origin.Name != "Madrid" {
		t.Errorf("expected Origin 'Madrid', got %q", shipment.Origin.Name)
	}
	if shipment.Destination.Name != "Barcelona" {
		t.Errorf("expected Destination 'Barcelona', got %q", shipment.Destination.Name)
	}
	if len(shipment.Events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(shipment.Events))
	}
	if shipment.Events[0].Status != models.StatusDelivered {
		t.Errorf("expected newest event StatusDelivered, got %s", shipment.Events[0].Status)
	}
}

func TestNacexInTransitHTMLWithHTTPTestServer(t *testing.T) {
	fixture, err := LoadFixture("nacex_intransit.html")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	teardownURL := SetNacexBaseURLForTest(srv.URL)
	defer teardownURL()
	teardownClient := SetNacexHTTPClientForTest(srv.Client())
	defer teardownClient()

	c := NewNacexCarrier()
	shipment, err := c.Track(context.Background(), "NCX444555666", models.TrackOptions{
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("unexpected error tracking Nacex HTML: %v", err)
	}

	if shipment.Status != models.StatusInTransit {
		t.Errorf("expected StatusInTransit, got %s", shipment.Status)
	}
	if shipment.Origin.Name != "4101 - Sevilla" {
		t.Errorf("expected Origin '4101 - Sevilla', got %q", shipment.Origin.Name)
	}
	if shipment.Destination.Name != "2801 - Madrid" {
		t.Errorf("expected Destination '2801 - Madrid', got %q", shipment.Destination.Name)
	}
	if len(shipment.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(shipment.Events))
	}
}

func TestNacexExceptionWithHTTPTestServer(t *testing.T) {
	fixture, err := LoadFixture("nacex_exception.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	teardownURL := SetNacexBaseURLForTest(srv.URL)
	defer teardownURL()
	teardownClient := SetNacexHTTPClientForTest(srv.Client())
	defer teardownClient()

	c := NewNacexCarrier()
	shipment, err := c.Track(context.Background(), "NCX999888777", models.TrackOptions{
		AllowSimulated: false,
	})
	if err != nil {
		t.Fatalf("unexpected error tracking Nacex exception: %v", err)
	}

	if shipment.Status != models.StatusFailedAttempt {
		t.Errorf("expected StatusFailedAttempt, got %s", shipment.Status)
	}
}

func TestNacexErrorsAndNoSilentMock(t *testing.T) {
	c := NewNacexCarrier()

	t.Run("not_found_no_mock", func(t *testing.T) {
		fixture, _ := LoadFixture("nacex_not_found.json")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(fixture)
		}))
		defer srv.Close()

		teardownURL := SetNacexBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetNacexHTTPClientForTest(srv.Client())
		defer teardownClient()

		_, err := c.Track(context.Background(), "NCX000000000", models.TrackOptions{AllowSimulated: false})
		if !errors.Is(err, models.ErrShipmentNotFound) {
			t.Errorf("expected ErrShipmentNotFound, got %v", err)
		}

		sim, err := c.Track(context.Background(), "NCX000000000", models.TrackOptions{AllowSimulated: true})
		if err != nil || sim == nil || len(sim.Events) == 0 {
			t.Errorf("expected simulated fallback when AllowSimulated=true, got err=%v", err)
		}
	})

	t.Run("blocked_captcha", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><body>Please complete Captcha challenge</body></html>"))
		}))
		defer srv.Close()

		teardownURL := SetNacexBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetNacexHTTPClientForTest(srv.Client())
		defer teardownClient()

		_, err := c.Track(context.Background(), "NCX123456789", models.TrackOptions{AllowSimulated: false})
		if !errors.Is(err, models.ErrCarrierBlocked) {
			t.Errorf("expected ErrCarrierBlocked, got %v", err)
		}
	})

	t.Run("schema_changed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><body><p>New unknown HTML layout</p></body></html>"))
		}))
		defer srv.Close()

		teardownURL := SetNacexBaseURLForTest(srv.URL)
		defer teardownURL()
		teardownClient := SetNacexHTTPClientForTest(srv.Client())
		defer teardownClient()

		_, err := c.Track(context.Background(), "NCX123456789", models.TrackOptions{AllowSimulated: false})
		if !errors.Is(err, models.ErrScraperSchemaChanged) {
			t.Errorf("expected ErrScraperSchemaChanged, got %v", err)
		}
	})
}
