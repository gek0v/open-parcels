package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"open-parcels/pkg/models"
)

// SQLiteStore is an embedded persistent storage backend powered by pure-Go SQLite.
type SQLiteStore struct {
	db   *sql.DB
	path string
}

// NewSQLiteStore opens or creates a SQLiteStore at dbPath and runs automatic schema migrations.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	cleanPath := strings.TrimSpace(dbPath)
	if cleanPath == "" {
		cleanPath = ":memory:"
	}

	isMem := cleanPath == ":memory:" || strings.HasPrefix(cleanPath, "file::memory:")
	dsn := cleanPath
	if !isMem && !strings.HasPrefix(cleanPath, "file:") {
		dir := filepath.Dir(cleanPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("failed to create sqlite database directory %s: %w", dir, err)
			}
		}
		dsn = cleanPath
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Single open connection avoids SQLITE_BUSY write-lock contention across goroutines
	// and preserves in-memory databases across queries.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	store := &SQLiteStore{
		db:   db,
		path: cleanPath,
	}

	if err := store.configureAndMigrate(isMem); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite schema migration failed: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) configureAndMigrate(isMem bool) error {
	if _, err := s.db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		return err
	}
	if !isMem {
		if _, err := s.db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
			return err
		}
		if _, err := s.db.Exec(`PRAGMA synchronous = NORMAL;`); err != nil {
			return err
		}
	}
	if _, err := s.db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS shipments (
		carrier TEXT NOT NULL,
		tracking_number TEXT NOT NULL,
		postal_code TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		payload TEXT NOT NULL,
		expires_at INTEGER NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		PRIMARY KEY (carrier, tracking_number, postal_code)
	);

	CREATE INDEX IF NOT EXISTS idx_shipments_lookup ON shipments(tracking_number, postal_code);
	CREATE INDEX IF NOT EXISTS idx_shipments_expires ON shipments(expires_at);

	CREATE TABLE IF NOT EXISTS async_jobs (
		request_id TEXT PRIMARY KEY,
		status TEXT NOT NULL,
		error TEXT NOT NULL DEFAULT '',
		payload TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		expires_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_async_jobs_expires ON async_jobs(expires_at);

	CREATE TABLE IF NOT EXISTS oauth_tokens (
		provider TEXT PRIMARY KEY,
		token TEXT NOT NULL,
		expires_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_oauth_tokens_expires ON oauth_tokens(expires_at);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	_, err := s.db.Exec(`INSERT OR IGNORE INTO schema_migrations (version, applied_at) VALUES (1, ?);`, time.Now().UTC().UnixNano())
	return err
}

// Path returns the configured database path.
func (s *SQLiteStore) Path() string {
	return s.path
}

// GetShipment retrieves a cached shipment if it exists and has not expired.
// If carrier is empty, it finds the most recent active record matching trackingNumber and postalCode.
func (s *SQLiteStore) GetShipment(ctx context.Context, carrier, trackingNumber, postalCode string) (*models.Shipment, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	normCarrier := normalizeCarrier(carrier)
	normNumber := normalizeTrackingNumber(trackingNumber)
	normPostal := normalizePostalCode(postalCode)
	nowNano := time.Now().UTC().UnixNano()

	var payload string
	var err error
	if normCarrier != "" {
		query := `SELECT payload FROM shipments
		          WHERE carrier = ? AND tracking_number = ? AND postal_code = ? AND expires_at > ? LIMIT 1;`
		err = s.db.QueryRowContext(ctx, query, normCarrier, normNumber, normPostal, nowNano).Scan(&payload)
	} else {
		query := `SELECT payload FROM shipments
		          WHERE tracking_number = ? AND postal_code = ? AND expires_at > ?
		          ORDER BY updated_at DESC LIMIT 1;`
		err = s.db.QueryRowContext(ctx, query, normNumber, normPostal, nowNano).Scan(&payload)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	var shp models.Shipment
	if err := json.Unmarshal([]byte(payload), &shp); err != nil {
		return nil, false, fmt.Errorf("failed to deserialize cached shipment: %w", err)
	}

	shp.FromCache = true
	return &shp, true, nil
}

// SetShipment stores or updates a shipment cache record.
func (s *SQLiteStore) SetShipment(ctx context.Context, carrier, trackingNumber, postalCode string, shipment *models.Shipment, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if shipment == nil {
		return nil
	}

	normCarrier := normalizeCarrier(carrier)
	if normCarrier == "" && shipment.Carrier.Slug != "" {
		normCarrier = normalizeCarrier(shipment.Carrier.Slug)
	}
	normNumber := normalizeTrackingNumber(trackingNumber)
	if normNumber == "" && shipment.TrackingNumber != "" {
		normNumber = normalizeTrackingNumber(shipment.TrackingNumber)
	}
	normPostal := normalizePostalCode(postalCode)

	payloadBytes, err := json.Marshal(shipment)
	if err != nil {
		return fmt.Errorf("failed to serialize shipment: %w", err)
	}

	now := time.Now().UTC()
	nowNano := now.UnixNano()
	expiresNano := now.Add(ttl).UnixNano()

	query := `
	INSERT INTO shipments (carrier, tracking_number, postal_code, status, payload, expires_at, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(carrier, tracking_number, postal_code) DO UPDATE SET
		status = excluded.status,
		payload = excluded.payload,
		expires_at = excluded.expires_at,
		updated_at = excluded.updated_at;
	`
	_, err = s.db.ExecContext(ctx, query,
		normCarrier,
		normNumber,
		normPostal,
		string(shipment.Status),
		string(payloadBytes),
		expiresNano,
		nowNano,
		nowNano,
	)
	return err
}

// DeleteShipment deletes a cached shipment record.
func (s *SQLiteStore) DeleteShipment(ctx context.Context, carrier, trackingNumber, postalCode string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	normCarrier := normalizeCarrier(carrier)
	normNumber := normalizeTrackingNumber(trackingNumber)
	normPostal := normalizePostalCode(postalCode)

	if normCarrier != "" {
		query := `DELETE FROM shipments WHERE carrier = ? AND tracking_number = ? AND postal_code = ?;`
		_, err := s.db.ExecContext(ctx, query, normCarrier, normNumber, normPostal)
		return err
	}

	query := `DELETE FROM shipments WHERE tracking_number = ? AND postal_code = ?;`
	_, err := s.db.ExecContext(ctx, query, normNumber, normPostal)
	return err
}

// GetAsyncJob retrieves an async ParcelsApp tracking job by request_id.
func (s *SQLiteStore) GetAsyncJob(ctx context.Context, requestID string) (*AsyncJob, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	reqID := strings.TrimSpace(requestID)
	nowNano := time.Now().UTC().UnixNano()

	query := `SELECT status, error, payload, created_at, updated_at, expires_at
	          FROM async_jobs WHERE request_id = ? AND expires_at > ? LIMIT 1;`

	var status, errMsg, payload string
	var createdNano, updatedNano, expiresNano int64

	err := s.db.QueryRowContext(ctx, query, reqID, nowNano).Scan(&status, &errMsg, &payload, &createdNano, &updatedNano, &expiresNano)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	var resp *models.ParcelsAppResponse
	if strings.TrimSpace(payload) != "" {
		var r models.ParcelsAppResponse
		if err := json.Unmarshal([]byte(payload), &r); err != nil {
			return nil, false, fmt.Errorf("failed to deserialize async job payload: %w", err)
		}
		resp = &r
	}

	job := &AsyncJob{
		RequestID: reqID,
		Status:    status,
		Response:  resp,
		Error:     errMsg,
		CreatedAt: time.Unix(0, createdNano).UTC(),
		UpdatedAt: time.Unix(0, updatedNano).UTC(),
		ExpiresAt: time.Unix(0, expiresNano).UTC(),
	}

	return job, true, nil
}

// SetAsyncJob persists or updates an async job.
func (s *SQLiteStore) SetAsyncJob(ctx context.Context, job *AsyncJob) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if job == nil || strings.TrimSpace(job.RequestID) == "" {
		return nil
	}

	reqID := strings.TrimSpace(job.RequestID)
	now := time.Now().UTC()

	createdAt := job.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = now
	}
	expiresAt := job.ExpiresAt.UTC()
	if expiresAt.IsZero() {
		expiresAt = now.Add(1 * time.Hour)
	}

	var payload string
	if job.Response != nil {
		b, err := json.Marshal(job.Response)
		if err != nil {
			return fmt.Errorf("failed to serialize async job response: %w", err)
		}
		payload = string(b)
	}

	query := `
	INSERT INTO async_jobs (request_id, status, error, payload, created_at, updated_at, expires_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(request_id) DO UPDATE SET
		status = excluded.status,
		error = excluded.error,
		payload = excluded.payload,
		updated_at = excluded.updated_at,
		expires_at = excluded.expires_at;
	`
	_, err := s.db.ExecContext(ctx, query,
		reqID,
		job.Status,
		job.Error,
		payload,
		createdAt.UnixNano(),
		now.UnixNano(),
		expiresAt.UnixNano(),
	)
	return err
}

// DeleteAsyncJob deletes an async job by request ID.
func (s *SQLiteStore) DeleteAsyncJob(ctx context.Context, requestID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	query := `DELETE FROM async_jobs WHERE request_id = ?;`
	_, err := s.db.ExecContext(ctx, query, strings.TrimSpace(requestID))
	return err
}

// GetOAuthToken retrieves an OAuth token for a provider.
func (s *SQLiteStore) GetOAuthToken(ctx context.Context, provider string) (string, time.Time, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", time.Time{}, false, err
	}
	p := normalizeProvider(provider)
	nowNano := time.Now().UTC().UnixNano()

	query := `SELECT token, expires_at FROM oauth_tokens WHERE provider = ? AND expires_at > ? LIMIT 1;`
	var token string
	var expiresNano int64

	err := s.db.QueryRowContext(ctx, query, p, nowNano).Scan(&token, &expiresNano)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", time.Time{}, false, nil
		}
		return "", time.Time{}, false, err
	}

	return token, time.Unix(0, expiresNano).UTC(), true, nil
}

// SetOAuthToken persists an OAuth token for a provider.
func (s *SQLiteStore) SetOAuthToken(ctx context.Context, provider, token string, expiresAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p := normalizeProvider(provider)
	nowNano := time.Now().UTC().UnixNano()

	query := `
	INSERT INTO oauth_tokens (provider, token, expires_at, updated_at)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(provider) DO UPDATE SET
		token = excluded.token,
		expires_at = excluded.expires_at,
		updated_at = excluded.updated_at;
	`
	_, err := s.db.ExecContext(ctx, query, p, token, expiresAt.UTC().UnixNano(), nowNano)
	return err
}

// DeleteOAuthToken removes an OAuth token for a provider.
func (s *SQLiteStore) DeleteOAuthToken(ctx context.Context, provider string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	query := `DELETE FROM oauth_tokens WHERE provider = ?;`
	_, err := s.db.ExecContext(ctx, query, normalizeProvider(provider))
	return err
}

// PurgeExpired deletes expired shipments, async jobs, and tokens.
func (s *SQLiteStore) PurgeExpired(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	nowNano := time.Now().UTC().UnixNano()

	if _, err := s.db.ExecContext(ctx, `DELETE FROM shipments WHERE expires_at <= ?;`, nowNano); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM async_jobs WHERE expires_at <= ?;`, nowNano); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM oauth_tokens WHERE expires_at <= ?;`, nowNano); err != nil {
		return err
	}
	return nil
}

// Close closes the underlying SQLite database connection.
func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}