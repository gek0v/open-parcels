package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"open-parcels/pkg/engine"
	"open-parcels/pkg/mcp"
	"open-parcels/pkg/models"
)

type cachedRequest struct {
	response  *models.ParcelsAppResponse
	expiresAt time.Time
}

// Server is the HTTP REST and MCP server daemon.
type Server struct {
	engine            *engine.Engine
	mcpServer         *mcp.Server
	port              int
	enableMCP         bool
	httpServer        *http.Server
	webhookDispatcher *WebhookDispatcher

	mu       sync.RWMutex
	requests map[string]*cachedRequest
}

// Config configures the Server.
type Config struct {
	Port              int
	EnableMCP         bool
	WebhookDispatcher *WebhookDispatcher
}

// New creates a new Server instance.
func New(cfg Config) *Server {
	if cfg.Port <= 0 {
		cfg.Port = 8080
	}

	dispatcher := cfg.WebhookDispatcher
	if dispatcher == nil {
		dispatcher = NewWebhookDispatcher()
	}

	eng := engine.New()
	s := &Server{
		engine:            eng,
		mcpServer:         mcp.NewServer(eng),
		port:              cfg.Port,
		enableMCP:         cfg.EnableMCP,
		webhookDispatcher: dispatcher,
		requests:          make(map[string]*cachedRequest),
	}

	return s
}

// WebhookDispatcher returns the server's webhook dispatcher instance.
func (s *Server) WebhookDispatcher() *WebhookDispatcher {
	return s.webhookDispatcher
}

// SetWebhookDispatcher replaces the server's webhook dispatcher.
func (s *Server) SetWebhookDispatcher(d *WebhookDispatcher) {
	if d != nil {
		s.webhookDispatcher = d
	}
}

func (s *Server) setRequest(reqID string, resp *models.ParcelsAppResponse, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests[reqID] = &cachedRequest{
		response:  resp,
		expiresAt: time.Now().Add(ttl),
	}
}

func (s *Server) getRequest(reqID string) (*models.ParcelsAppResponse, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cached, found := s.requests[reqID]
	if !found {
		return nil, false
	}
	if time.Now().After(cached.expiresAt) {
		delete(s.requests, reqID)
		return nil, false
	}
	return cached.response, true
}

// Start launches the HTTP server and blocks until ctx is canceled or an error occurs.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// Carriers catalog
	mux.HandleFunc("/api/v1/carriers", s.handleCarriers)
	mux.HandleFunc("/tracking-carriers", s.handleCarriers)

	// Carrier detection
	mux.HandleFunc("/api/v1/detect", s.handleDetect)

	// REST Tracking endpoints
	mux.HandleFunc("/api/v1/track", s.handleTrackV1)
	mux.HandleFunc("/trackings", s.handleParcelsAppTrackings)
	mux.HandleFunc("/trackings/", s.handleParcelsAppGetTracking)

	// MCP over HTTP endpoint
	if s.enableMCP {
		mux.HandleFunc("/mcp", s.handleMCPHTTP)
	}

	// Logging & CORS middleware
	handler := s.corsMiddleware(mux)

	addr := fmt.Sprintf(":%d", s.port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 25 * time.Second,
	}

	log.Printf("[open-parcels] REST API server listening on http://localhost:%d", s.port)
	if s.enableMCP {
		log.Printf("[open-parcels] MCP streamable HTTP endpoint enabled on http://localhost:%d/mcp", s.port)
	}

	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Parcels-Signature")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
