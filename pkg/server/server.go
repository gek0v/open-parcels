package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
	engine     *engine.Engine
	store      storage.Store
	mcpServer  *mcp.Server
	port       int
	enableMCP  bool
	httpServer *http.Server

	mu       sync.RWMutex
	requests map[string]*cachedRequest
	wg       sync.WaitGroup
}

// Config configures the Server.
type Config struct {
	Port      int
	EnableMCP bool
	Store     storage.Store
	Engine    *engine.Engine
}

// New creates a new Server instance.
func New(cfg Config) *Server {
	if cfg.Port <= 0 {
		cfg.Port = 8080
	}

	st := cfg.Store
	if st == nil {
		if dbPath := os.Getenv(storage.EnvDBPath); dbPath != "" {
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

	s := &Server{
		engine:    eng,
		store:     st,
		mcpServer: mcp.NewServer(eng),
		port:      cfg.Port,
		enableMCP: cfg.EnableMCP,
		requests:  make(map[string]*cachedRequest),
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