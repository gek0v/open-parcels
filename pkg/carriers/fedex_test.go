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

func TestFedExDeliveredFixture(t *testing.T) {
	fixtureData, err := LoadFixture("fedex_delivered.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	mockClient := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.Path, "oauth/token") {
				tokenJSON := `{"access_token":"mock-fedex-token","expires_in":3600}`
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

	teardown := SetFedExHTTPClientForTest(mockClient)
	defer teardown()

	os.Setenv("FEDEX_CLIENT_ID", "mock-fedex-client-id")
	os.Setenv("FEDEX_CLIENT_SECRET", "mock-fedex-client-secret")
	defer func() {
		os.Unsetenv("FEDEX_CLIENT_ID")
		os.Unsetenv("FEDEX_CLIENT_SECRET")
	}()

	c := NewFedExCarrier()
	shipment, err := c.Track(context.Background(), "123456789012", models.TrackOptions{})
	if err != nil {
		t.Fatalf("unexpected error tracking FedEx fixture: %v", err)
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

func TestFedExMissingAuth(t *testing.T) {
	os.Unsetenv("FEDEX_CLIENT_ID")
	os.Unsetenv("FEDEX_CLIENT_SECRET")
	os.Unsetenv("OPEN_PARCELS_FEDEX_KEY")
	os.Unsetenv("OPEN_PARCELS_FEDEX_SECRET")

	c := NewFedExCarrier()
	_, err := c.Track(context.Background(), "123456789012", models.TrackOptions{
		AllowSimulated: false,
	})
	if err == nil {
		t.Fatalf("expected error when FedEx credentials are missing")
	}
	if !errors.Is(err, models.ErrCarrierNotImplemented) {
		t.Errorf("expected ErrCarrierNotImplemented, got: %v", err)
	}
}

func TestFedExNotFoundFixture(t *testing.T) {
	fixtureData, err := LoadFixture("fedex_not_found.json")
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	mockClient := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.Path, "oauth/token") {
				tokenJSON := `{"access_token":"mock-fedex-token","expires_in":3600}`
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

	teardown := SetFedExHTTPClientForTest(mockClient)
	defer teardown()

	os.Setenv("FEDEX_CLIENT_ID", "mock-fedex-client-id")
	os.Setenv("FEDEX_CLIENT_SECRET", "mock-fedex-client-secret")
	defer func() {
		os.Unsetenv("FEDEX_CLIENT_ID")
		os.Unsetenv("FEDEX_CLIENT_SECRET")
	}()

	c := NewFedExCarrier()
	_, err = c.Track(context.Background(), "000000000000", models.TrackOptions{})
	if err == nil {
		t.Fatalf("expected ErrShipmentNotFound, got nil")
	}
	if !errors.Is(err, models.ErrShipmentNotFound) {
		t.Errorf("expected ErrShipmentNotFound, got: %v", err)
	}
}
