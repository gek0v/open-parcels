package carriers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"open-parcels/pkg/models"
)

func TestUPSDeliveredFixture(t *testing.T) {
	fixtureData, err := LoadFixture("ups_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	// Mock token + track responses
	mockClient := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.Path, "oauth/token") {
				tokenJSON := `{"access_token":"mock-ups-token","expires_in":3600}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(tokenJSON)),
					Header:     make(http.Header),
					Request:    req,
				}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(fixtureData)),
				Header:     make(http.Header),
				Request:    req,
			}
		}),
	}

	teardown := SetUPSHTTPClientForTest(mockClient)
	defer teardown()

	os.Setenv("UPS_CLIENT_ID", "mock-ups-client-id")
	os.Setenv("UPS_CLIENT_SECRET", "mock-ups-client-secret")
	defer func() {
		os.Unsetenv("UPS_CLIENT_ID")
		os.Unsetenv("UPS_CLIENT_SECRET")
	}()

	c := NewUPSCarrier()
	shipment, err := c.Track(context.Background(), "1Z12345E0205271688", models.TrackOptions{})
	if err != nil {
		t.Fatalf("unexpected error tracking UPS fixture: %v", err)
	}

	if shipment.Status != models.StatusDelivered {
		t.Errorf("expected StatusDelivered, got %s", shipment.Status)
	}
	if shipment.Progress != 100 {
		t.Errorf("expected Progress 100, got %d", shipment.Progress)
	}
	if len(shipment.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(shipment.Events))
	}
}

func TestUPSMissingAuth(t *testing.T) {
	os.Unsetenv("UPS_CLIENT_ID")
	os.Unsetenv("UPS_CLIENT_SECRET")
	os.Unsetenv("OPEN_PARCELS_UPS_KEY")
	os.Unsetenv("OPEN_PARCELS_UPS_SECRET")

	c := NewUPSCarrier()
	_, err := c.Track(context.Background(), "1Z12345E0205271688", models.TrackOptions{
		AllowSimulated: false,
	})
	if err == nil {
		t.Fatalf("expected error when UPS credentials are missing")
	}
	if !errors.Is(err, models.ErrCarrierNotImplemented) {
		t.Errorf("expected ErrCarrierNotImplemented, got: %v", err)
	}
}

func TestUPSNotFoundFixture(t *testing.T) {
	fixtureData, err := LoadFixture("ups_not_found.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	mockClient := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.Path, "oauth/token") {
				tokenJSON := `{"access_token":"mock-ups-token","expires_in":3600}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(tokenJSON)),
					Header:     make(http.Header),
					Request:    req,
				}
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader(fixtureData)),
				Header:     make(http.Header),
				Request:    req,
			}
		}),
	}

	teardown := SetUPSHTTPClientForTest(mockClient)
	defer teardown()

	os.Setenv("UPS_CLIENT_ID", "mock-ups-client-id")
	os.Setenv("UPS_CLIENT_SECRET", "mock-ups-client-secret")
	defer func() {
		os.Unsetenv("UPS_CLIENT_ID")
		os.Unsetenv("UPS_CLIENT_SECRET")
	}()

	c := NewUPSCarrier()
	_, err = c.Track(context.Background(), "1Z0000000000000000", models.TrackOptions{})
	if err == nil {
		t.Fatalf("expected ErrShipmentNotFound, got nil")
	}
	if !errors.Is(err, models.ErrShipmentNotFound) {
		t.Errorf("expected ErrShipmentNotFound, got: %v", err)
	}
}
