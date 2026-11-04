package server

import (
	"encoding/json"
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
	} else if r.Method == http.MethodPost {
		var req struct {
			TrackingNumber     string `json:"tracking_number"`
			CarrierHint        string `json:"carrier_hint,omitempty"`
			Language           string `json:"language,omitempty"`
			DestinationCountry string `json:"destination_country,omitempty"`
			PostalCode         string `json:"postal_code,omitempty"`
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
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	shipment, err := s.engine.Track(r.Context(), trackingNumber, opts)
	if err != nil {
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
			Email:              shp.Email,
			Phone:              shp.Phone,
			AdditionalFields:   shp.AdditionalFields,
		}

		shipment, err := s.engine.Track(r.Context(), shp.TrackingNumber, opts)
		if err != nil {
			rejected = append(rejected, models.ParcelsAppRejected{
				Index:          idx,
				TrackingNumber: shp.TrackingNumber,
				Code:           "TRACKING_FAILED",
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
		ExpiresAt: time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339),
		Results:   resultItems,
		Rejected:  rejected,
	}

	// Cache request
	s.mu.Lock()
	s.requests[reqID] = finalResp
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, finalResp)
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

	s.mu.RLock()
	resp, ok := s.requests[reqID]
	s.mu.RUnlock()

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
