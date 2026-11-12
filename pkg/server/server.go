package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"open-parcels/pkg/engine"
	"open-parcels/pkg/mcp"
	"open-parcels/pkg/models"
	"open-parcels/pkg/storage"
)

type cachedRequest struct {
	response  *models.ParcelsAppResponse
	expiresAt time.Time
}

// Server is the HTTP REST and MCP server daemon.
type Server struct {
	engine            *engine.Engine
	store             storage.Store
	mcpServer         *mcp.Server
	port              int
	enableMCP         bool
	httpServer        *http.Server
	webhookDispatcher *WebhookDispatcher
	apiKeys           []string
	rateLimiter       *RateLimiter

	mu       sync.RWMutex
	requests map[string]*cachedRequest
	wg       sync.WaitGroup
}

// Config configures the Server.
type Config struct {
	Port                 int
	EnableMCP            bool
	DBPath               string
	Store                storage.Store
	Engine               *engine.Engine
	WebhookDispatcher    *WebhookDispatcher
	APIKey               string
	APIKeys              []string
	RateLimit            float64 // Requests per second (<= 0 disables rate limiting)
	RateBurst            int     // Maximum burst size (<= 0 defaults to ceil(RateLimit))
	RateLimitInactiveTTL time.Duration
}

// New creates a new Server instance.
func New(cfg Config) *Server {
	if cfg.Port <= 0 {
		cfg.Port = 8080
	}

	st := cfg.Store
	if st == nil {
		dbPath := cfg.DBPath
		if dbPath == "" {
			dbPath = os.Getenv(storage.EnvDBPath)
		}
		if dbPath != "" {
			var err error
			st, err = storage.NewStore(dbPath)
			if err != nil {
				log.Printf("[open-parcels] warning: failed to open store from %s: %v, falling back to memory store", dbPath, err)
				st = storage.NewMemoryStore()
			}
		} else {
			st = storage.NewMemoryStore()
		}
	}

	eng := cfg.Engine
	if eng == nil {
		eng = engine.NewWithStore(st)
	} else if eng.Store() == nil {
		eng.SetStore(st)
	}

	dispatcher := cfg.WebhookDispatcher
	if dispatcher == nil {
		dispatcher = NewWebhookDispatcher()
	}

	apiKeys := mcp.ParseAPIKeys(cfg.APIKeys, cfg.APIKey)
	if len(apiKeys) == 0 {
		apiKeys = mcp.LoadAPIKeysFromEnv()
	}

	rateLimit := cfg.RateLimit
	if rateLimit <= 0 {
		if envRate := os.Getenv("OPEN_PARCELS_RATE_LIMIT"); envRate != "" {
			parsed, err := ParseRateLimit(envRate)
			if err != nil {
				log.Printf("[open-parcels] warning: ignoring invalid OPEN_PARCELS_RATE_LIMIT (%q): %v", envRate, err)
			} else {
				rateLimit = parsed
			}
		}
	}

	rateBurst := cfg.RateBurst
	if rateBurst <= 0 {
		if envBurst := os.Getenv("OPEN_PARCELS_RATE_BURST"); envBurst != "" {
			if parsed, err := strconv.Atoi(envBurst); err == nil && parsed > 0 {
				rateBurst = parsed
			}
		}
	}

	var rl *RateLimiter
	if rateLimit > 0 {
		rl = NewRateLimiter(rateLimit, rateBurst, cfg.RateLimitInactiveTTL)
	}

	mcpSrv := mcp.NewServer(eng)
	mcpSrv.SetAPIKeys(apiKeys)

	s := &Server{
		engine:            eng,
		store:             st,
		mcpServer:         mcpSrv,
		port:              cfg.Port,
		enableMCP:         cfg.EnableMCP,
		webhookDispatcher: dispatcher,
		apiKeys:           apiKeys,
		rateLimiter:       rl,
		requests:          make(map[string]*cachedRequest),
	}

	return s
}

// Store returns the underlying persistent store.
func (s *Server) Store() storage.Store {
	return s.store
}

// Engine returns the underlying tracking engine.
func (s *Server) Engine() *engine.Engine {
	return s.engine
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

// Close releases background resources owned by the server (e.g. rate limiter cleaner).
func (s *Server) Close() {
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}
}

// APIKeys returns a copy of the configured API keys.
func (s *Server) APIKeys() []string {
	if len(s.apiKeys) == 0 {
		return nil
	}
	out := make([]string, len(s.apiKeys))
	copy(out, s.apiKeys)
	return out
}

// RateLimiter returns the configured RateLimiter, or nil if disabled.
func (s *Server) RateLimiter() *RateLimiter {
	return s.rateLimiter
}
func (s *Server) setRequest(reqID string, resp *models.ParcelsAppResponse, ttl time.Duration) {
	s.mu.Lock()
	expiresAt := time.Now().Add(ttl)
	s.requests[reqID] = &cachedRequest{
		response:  resp,
		expiresAt: expiresAt,
	}
	s.mu.Unlock()

	if s.store != nil {
		status := "processing"
		if resp != nil && (resp.Done || resp.Status == "completed") {
			status = "completed"
		}
		job := &storage.AsyncJob{
			RequestID: reqID,
			Status:    status,
			Response:  resp,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			ExpiresAt: expiresAt,
		}
		_ = s.store.SetAsyncJob(context.Background(), job)
	}
}

func (s *Server) getRequest(reqID string) (*models.ParcelsAppResponse, bool) {
	s.mu.Lock()
	cached, found := s.requests[reqID]
	if found {
		if !time.Now().Before(cached.expiresAt) {
			delete(s.requests, reqID)
			s.mu.Unlock()
			return nil, false
		}
		resp := cached.response
		s.mu.Unlock()
		return resp, true
	}
	s.mu.Unlock()

	// Check persistent store if not found in memory (e.g. after server restart)
	if s.store != nil {
		job, ok, err := s.store.GetAsyncJob(context.Background(), reqID)
		if err == nil && ok && job != nil && job.Response != nil {
			s.mu.Lock()
			s.requests[reqID] = &cachedRequest{
				response:  job.Response,
				expiresAt: job.ExpiresAt,
			}
			s.mu.Unlock()
			return job.Response, true
		}
	}

	return nil, false
}

// Handler builds and returns the root HTTP handler with CORS, rate limiting, and auth middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health check (always public)
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

	return s.corsMiddleware(s.rateLimitMiddleware(s.authMiddleware(mux)))
}

// Start launches the HTTP server and blocks until ctx is canceled or an error occurs.
func (s *Server) Start(ctx context.Context) error {
	defer s.Close()

	addr := fmt.Sprintf(":%d", s.port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 25 * time.Second,
	}

	log.Printf("[open-parcels] REST API server listening on http://localhost:%d", s.port)
	if s.enableMCP {
		log.Printf("[open-parcels] MCP streamable HTTP endpoint enabled on http://localhost:%d/mcp", s.port)
	}
	if len(s.apiKeys) > 0 {
		log.Printf("[open-parcels] Bearer / API Key authentication enabled (%d key(s) configured)", len(s.apiKeys))
	}
	if s.rateLimiter != nil {
		log.Printf("[open-parcels] Rate limiting enabled (%.2f req/s, burst %d)", s.rateLimiter.rate, int(s.rateLimiter.burst))
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
		err := s.httpServer.Shutdown(shutdownCtx)
		if s.store != nil {
			_ = s.store.Close()
		}
		return err
	case err := <-errChan:
		if s.store != nil {
			_ = s.store.Close()
		}
		return err
	}
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-API-Key, Parcels-Signature")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	if len(s.apiKeys) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		token := mcp.ExtractToken(r)
		if !mcp.ValidateAPIKey(token, s.apiKeys) {
			mcp.WriteUnauthorized(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	if s.rateLimiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		clientKey := ResolveClientKey(r)
		allowed, retryAfter := s.rateLimiter.Allow(clientKey)
		if !allowed {
			WriteRateLimitExceeded(w, retryAfter)
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
