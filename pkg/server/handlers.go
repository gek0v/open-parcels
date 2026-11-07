package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"open-parcels/pkg/models"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "open-parcels",
		"version": "1.0.0",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleCarriers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	carriers := s.engine.Registry().List(query)
	writeJSON(w, http.StatusOK, map[string]any{
		"total":    len(carriers),
		"carriers": carriers,
	})
}

func (s *Server) handleDetect(w http.ResponseWriter, r *http.Request) {
	num := strings.TrimSpace(r.URL.Query().Get("number"))
	if num == "" {
		writeError(w, http.StatusBadRequest, "MISSING_NUMBER", "Query parameter 'number' is required.")
		return
	}
	country := r.URL.Query().Get("country")
	candidates := s.engine.Detector().Detect(num, country)

	writeJSON(w, http.StatusOK, map[string]any{
		"tracking_number": num,
		"candidates":      candidates,
	})
}

func (s *Server) handleTrackV1(w http.ResponseWriter, r *http.Request) {
	var opts models.TrackOptions
	var trackingNumber string

	if r.Method == http.MethodGet {
		trackingNumber = strings.TrimSpace(r.URL.Query().Get("number"))
		if trackingNumber == "" {
			writeError(w, http.StatusBadRequest, "MISSING_NUMBER", "Query parameter 'number' is required.")
			return
		}
		opts.CarrierHint = r.URL.Query().Get("carrier")
		opts.Language = r.URL.Query().Get("lang")
		opts.DestinationCountry = r.URL.Query().Get("country")
		opts.PostalCode = r.URL.Query().Get("postal_code")
		if opts.PostalCode == "" {
			opts.PostalCode = r.URL.Query().Get("postalCode")
		}
		mockVal := r.URL.Query().Get("mock")
		if mockVal == "" {
			mockVal = r.URL.Query().Get("allow_simulated")
		}
		if strings.EqualFold(mockVal, "true") || mockVal == "1" {
			opts.AllowSimulated = true
		}
	} else if r.Method == http.MethodPost {
		var req struct {
			TrackingNumber     string `json:"tracking_number"`
			CarrierHint        string `json:"carrier_hint,omitempty"`
			Language           string `json:"language,omitempty"`
			DestinationCountry string `json:"destination_country,omitempty"`
			PostalCode         string `json:"postal_code,omitempty"`
			AllowSimulated     bool   `json:"allow_simulated,omitempty"`
			Mock               bool   `json:"mock,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "Failed to parse JSON body.")
			return
		}
		trackingNumber = req.TrackingNumber
		opts.CarrierHint = req.CarrierHint
		opts.Language = req.Language
		opts.DestinationCountry = req.DestinationCountry
		opts.PostalCode = req.PostalCode
		opts.AllowSimulated = req.AllowSimulated || req.Mock
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	shipment, err := s.engine.Track(r.Context(), trackingNumber, opts)
	if err != nil {
		if errors.Is(err, models.ErrPostalCodeRequired) || errors.Is(err, models.ErrMissingPostalCode) {
			writeError(w, http.StatusUnprocessableEntity, "POSTAL_CODE_REQUIRED", err.Error())
			return
		}
		if errors.Is(err, models.ErrShipmentNotFound) {
			writeError(w, http.StatusNotFound, "SHIPMENT_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, models.ErrCarrierNotImplemented) {
			writeError(w, http.StatusNotImplemented, "CARRIER_NOT_IMPLEMENTED", err.Error())
			return
		}
		if errors.Is(err, models.ErrCarrierRateLimited) {
			writeError(w, http.StatusTooManyRequests, "CARRIER_RATE_LIMITED", err.Error())
			return
		}
		if errors.Is(err, models.ErrCarrierTemporarilyUnavailable) {
			writeError(w, http.StatusBadGateway, "CARRIER_UNAVAILABLE", err.Error())
			return
		}
		writeError(w, http.StatusNotFound, "TRACKING_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, shipment)
}

// handleParcelsAppTrackings implements the official ParcelsApp v4 POST /trackings endpoint.
func (s *Server) handleParcelsAppTrackings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req models.ParcelsAppTrackingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Malformed JSON body.")
		return
	}

	if len(req.Shipments) == 0 {
		writeError(w, http.StatusBadRequest, "NO_SHIPMENTS", "No valid shipments remain after validation.")
		return
	}

	reqID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	isAsync := strings.EqualFold(r.URL.Query().Get("async"), "true") || strings.TrimSpace(req.WebhookURL) != ""

	if isAsync {
		initialResp := &models.ParcelsAppResponse{
			RequestID: reqID,
			Status:    "processing",
			Done:      false,
			ExpiresAt: time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
			Results:   make([]models.ParcelsAppResultItem, 0),
			Rejected:  make([]models.ParcelsAppRejected, 0),
		}

		s.setRequest(reqID, initialResp, 1*time.Hour)
		go s.processAsyncTrackings(reqID, req)

		writeJSON(w, http.StatusOK, initialResp)
		return
	}

	var resultItems []models.ParcelsAppResultItem
	var rejected []models.ParcelsAppRejected

	for idx, shp := range req.Shipments {
		if strings.TrimSpace(shp.TrackingNumber) == "" {
			rejected = append(rejected, models.ParcelsAppRejected{
				Index:   idx,
				Code:    "INVALID_TRACKING_NUMBER",
				Message: "tracking_number is not valid for tracking.",
			})
			continue
		}

		opts := models.TrackOptions{
			CarrierHint:        shp.CarrierHint,
			Language:           req.Language,
			DestinationCountry: shp.DestinationCountry,
			PostalCode:         shp.PostalCode,
			AllowSimulated:     shp.AllowSimulated,
			Email:              shp.Email,
			Phone:              shp.Phone,
			AdditionalFields:   shp.AdditionalFields,
		}

		shipment, err := s.engine.Track(r.Context(), shp.TrackingNumber, opts)
		if err != nil {
			errCode := "TRACKING_FAILED"
			if errors.Is(err, models.ErrPostalCodeRequired) || errors.Is(err, models.ErrMissingPostalCode) {
				errCode = "POSTAL_CODE_REQUIRED"
			} else if errors.Is(err, models.ErrShipmentNotFound) {
				errCode = "SHIPMENT_NOT_FOUND"
			} else if errors.Is(err, models.ErrCarrierNotImplemented) {
				errCode = "CARRIER_NOT_IMPLEMENTED"
			} else if errors.Is(err, models.ErrCarrierRateLimited) {
				errCode = "CARRIER_RATE_LIMITED"
			}
			rejected = append(rejected, models.ParcelsAppRejected{
				Index:          idx,
				TrackingNumber: shp.TrackingNumber,
				Code:           errCode,
				Message:        err.Error(),
			})
			continue
		}

		pResp := s.engine.ToParcelsAppResponse(shipment, reqID)
		if len(pResp.Results) > 0 {
			resultItems = append(resultItems, pResp.Results[0])
		}
	}

	if len(resultItems) == 0 && len(rejected) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": map[string]any{
				"code":     "NO_SHIPMENTS",
				"message":  "No valid shipments remain after validation.",
				"rejected": rejected,
			},
		})
		return
	}

	finalResp := &models.ParcelsAppResponse{
		RequestID: reqID,
		Status:    "completed",
		Done:      true,
		ExpiresAt: time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
		Results:   resultItems,
		Rejected:  rejected,
	}

	// Cache request with 1 hour TTL
	s.setRequest(reqID, finalResp, 1*time.Hour)

	writeJSON(w, http.StatusOK, finalResp)
}

func (s *Server) processAsyncTrackings(reqID string, req models.ParcelsAppTrackingRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var resultItems []models.ParcelsAppResultItem
	var rejected []models.ParcelsAppRejected

	for idx, shp := range req.Shipments {
		if strings.TrimSpace(shp.TrackingNumber) == "" {
			rejected = append(rejected, models.ParcelsAppRejected{
				Index:   idx,
				Code:    "INVALID_TRACKING_NUMBER",
				Message: "tracking_number is not valid for tracking.",
			})
			continue
		}

		opts := models.TrackOptions{
			CarrierHint:        shp.CarrierHint,
			Language:           req.Language,
			DestinationCountry: shp.DestinationCountry,
			PostalCode:         shp.PostalCode,
			AllowSimulated:     shp.AllowSimulated,
			Email:              shp.Email,
			Phone:              shp.Phone,
			AdditionalFields:   shp.AdditionalFields,
		}

		shipment, err := s.engine.Track(ctx, shp.TrackingNumber, opts)
		if err != nil {
			errCode := "TRACKING_FAILED"
			if errors.Is(err, models.ErrPostalCodeRequired) || errors.Is(err, models.ErrMissingPostalCode) {
				errCode = "POSTAL_CODE_REQUIRED"
			} else if errors.Is(err, models.ErrShipmentNotFound) {
				errCode = "SHIPMENT_NOT_FOUND"
			} else if errors.Is(err, models.ErrCarrierNotImplemented) {
				errCode = "CARRIER_NOT_IMPLEMENTED"
			} else if errors.Is(err, models.ErrCarrierRateLimited) {
				errCode = "CARRIER_RATE_LIMITED"
			}
			rejected = append(rejected, models.ParcelsAppRejected{
				Index:          idx,
				TrackingNumber: shp.TrackingNumber,
				Code:           errCode,
				Message:        err.Error(),
			})
			continue
		}

		pResp := s.engine.ToParcelsAppResponse(shipment, reqID)
		if len(pResp.Results) > 0 {
			resultItems = append(resultItems, pResp.Results[0])
		}
	}

	finalResp := &models.ParcelsAppResponse{
		RequestID: reqID,
		Status:    "completed",
		Done:      true,
		ExpiresAt: time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
		Results:   resultItems,
		Rejected:  rejected,
	}

	s.setRequest(reqID, finalResp, 1*time.Hour)

	if req.WebhookURL != "" {
		sendWebhookCallback(req.WebhookURL, finalResp)
	}
}

func sendWebhookCallback(webhookURL string, payload *models.ParcelsAppResponse) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "open-parcels-server/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
}

// handleParcelsAppGetTracking implements GET /trackings/{request_id}.
func (s *Server) handleParcelsAppGetTracking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		writeError(w, http.StatusBadRequest, "MISSING_REQUEST_ID", "Request ID is required.")
		return
	}
	reqID := parts[1]

	resp, ok := s.getRequest(reqID)
	if !ok {
		writeError(w, http.StatusNotFound, "REQUEST_NOT_FOUND", "The requested tracking record was not found or has expired.")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleMCPHTTP handles JSON-RPC 2.0 requests over HTTP.
func (s *Server) handleMCPHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	s.mcpServer.HandleHTTP(w, r)
}
