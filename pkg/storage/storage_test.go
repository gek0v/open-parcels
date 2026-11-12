package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"open-parcels/pkg/models"
)

func makeTestShipment(carrierSlug, trackingNumber string, status models.Status) *models.Shipment {
	now := time.Now().UTC().Truncate(time.Second)
	return &models.Shipment{
		TrackingNumber: trackingNumber,
		Carrier: models.CarrierInfo{
			Slug: carrierSlug,
			Name: "Test Carrier",
		},
		Status:     status,
		StatusText: string(status),
		Progress:   50,
		Events: []models.TrackingEvent{
			{
				Date:        now,
				Status:      status,
				Description: "Package scanned",
				Location:    "Madrid",
				CarrierSlug: carrierSlug,
				CarrierName: "Test Carrier",
			},
		},
		UpdatedAt: now,
	}
}

func runStoreContractTests(t *testing.T, newStore func(t *testing.T) Store) {
	t.Helper()
	ctx := context.Background()

	t.Run("ShipmentCRUDAndPostalCodeIsolation", func(t *testing.T) {
		st := newStore(t)
		defer st.Close()

		s1 := makeTestShipment("inpost", "7561674", models.StatusInTransit)
		s2 := makeTestShipment("inpost", "7561674", models.StatusDelivered)

		if err := st.SetShipment(ctx, "inpost", "7561674", "28001", s1, time.Minute); err != nil {
			t.Fatalf("SetShipment s1 failed: %v", err)
		}
		if err := st.SetShipment(ctx, "inpost", "7561674", "08001", s2, time.Minute); err != nil {
			t.Fatalf("SetShipment s2 failed: %v", err)
		}

		got1, found, err := st.GetShipment(ctx, "inpost", "7561674", "28001")
		if err != nil || !found {
			t.Fatalf("expected s1 found, err=%v found=%v", err, found)
		}
		if !got1.FromCache || got1.Status != models.StatusInTransit {
			t.Errorf("unexpected s1: fromCache=%v status=%s", got1.FromCache, got1.Status)
		}

		got2, found, err := st.GetShipment(ctx, "inpost", "7561674", "08001")
		if err != nil || !found {
			t.Fatalf("expected s2 found, err=%v found=%v", err, found)
		}
		if got2.Status != models.StatusDelivered {
			t.Errorf("expected s2 Delivered, got %s", got2.Status)
		}

		// Empty carrier lookup should match by trackingNumber + postalCode
		gotAny, found, err := st.GetShipment(ctx, "", "7561674", "28001")
		if err != nil || !found || gotAny.Status != models.StatusInTransit {
			t.Fatalf("expected empty carrier match for 28001, got err=%v found=%v", err, found)
		}

		if err := st.DeleteShipment(ctx, "inpost", "7561674", "28001"); err != nil {
			t.Fatalf("DeleteShipment failed: %v", err)
		}
		_, found, err = st.GetShipment(ctx, "inpost", "7561674", "28001")
		if err != nil || found {
			t.Fatalf("expected s1 deleted, err=%v found=%v", err, found)
		}
	})

	t.Run("TTLExpirationAndPurge", func(t *testing.T) {
		st := newStore(t)
		defer st.Close()

		short := makeTestShipment("cainiao", "LP111", models.StatusInTransit)
		long := makeTestShipment("cainiao", "LP222", models.StatusDelivered)

		if err := st.SetShipment(ctx, "cainiao", "LP111", "", short, 40*time.Millisecond); err != nil {
			t.Fatalf("SetShipment short failed: %v", err)
		}
		if err := st.SetShipment(ctx, "cainiao", "LP222", "", long, time.Hour); err != nil {
			t.Fatalf("SetShipment long failed: %v", err)
		}

		// Negative TTL expires immediately
		neg := makeTestShipment("cainiao", "LP333", models.StatusInTransit)
		if err := st.SetShipment(ctx, "cainiao", "LP333", "", neg, -5*time.Second); err != nil {
			t.Fatalf("SetShipment neg failed: %v", err)
		}
		if _, found, err := st.GetShipment(ctx, "cainiao", "LP333", ""); err != nil || found {
			t.Fatalf("expected negative TTL expired immediately, err=%v found=%v", err, found)
		}

		time.Sleep(70 * time.Millisecond)

		if _, found, err := st.GetShipment(ctx, "cainiao", "LP111", ""); err != nil || found {
			t.Fatalf("expected short TTL expired, err=%v found=%v", err, found)
		}
		if _, found, err := st.GetShipment(ctx, "cainiao", "LP222", ""); err != nil || !found {
			t.Fatalf("expected long TTL still valid, err=%v found=%v", err, found)
		}

		// Async job & OAuth token expiration
		if err := st.SetAsyncJob(ctx, &AsyncJob{
			RequestID: "req_expired",
			Status:    "completed",
			ExpiresAt: time.Now().Add(-time.Minute),
		}); err != nil {
			t.Fatalf("SetAsyncJob failed: %v", err)
		}
		if _, found, err := st.GetAsyncJob(ctx, "req_expired"); err != nil || found {
			t.Fatalf("expected expired async job not found, err=%v found=%v", err, found)
		}

		if err := st.SetOAuthToken(ctx, "ups", "old-tok", time.Now().Add(-time.Minute)); err != nil {
			t.Fatalf("SetOAuthToken failed: %v", err)
		}
		if _, _, found, err := st.GetOAuthToken(ctx, "ups"); err != nil || found {
			t.Fatalf("expected expired oauth token not found, err=%v found=%v", err, found)
		}

		if err := st.PurgeExpired(ctx); err != nil {
			t.Fatalf("PurgeExpired failed: %v", err)
		}
		if _, found, err := st.GetShipment(ctx, "cainiao", "LP222", ""); err != nil || !found {
			t.Fatalf("expected non-expired shipment preserved after purge, err=%v found=%v", err, found)
		}
	})

	t.Run("Concurrency", func(t *testing.T) {
		st := newStore(t)
		defer st.Close()

		const workers = 16
		const iterations = 15
		var wg sync.WaitGroup
		errCh := make(chan error, workers*iterations*2)

		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for i := 0; i < iterations; i++ {
					trackNum := fmt.Sprintf("LP%04d%04d", workerID, i)
					shp := makeTestShipment("cainiao", trackNum, models.StatusInTransit)
					if err := st.SetShipment(ctx, "cainiao", trackNum, "", shp, time.Minute); err != nil {
						errCh <- fmt.Errorf("SetShipment: %w", err)
						return
					}
					if _, found, err := st.GetShipment(ctx, "cainiao", trackNum, ""); err != nil || !found {
						errCh <- fmt.Errorf("GetShipment found=%v err=%v", found, err)
						return
					}

					reqID := fmt.Sprintf("req_%d_%d", workerID, i)
					job := &AsyncJob{
						RequestID: reqID,
						Status:    "completed",
						Response: &models.ParcelsAppResponse{
							RequestID: reqID,
							Status:    "completed",
							Done:      true,
						},
						ExpiresAt: time.Now().Add(time.Minute),
					}
					if err := st.SetAsyncJob(ctx, job); err != nil {
						errCh <- fmt.Errorf("SetAsyncJob: %w", err)
						return
					}
					if _, found, err := st.GetAsyncJob(ctx, reqID); err != nil || !found {
						errCh <- fmt.Errorf("GetAsyncJob found=%v err=%v", found, err)
						return
					}

					provider := fmt.Sprintf("provider_%d", workerID%3)
					if err := st.SetOAuthToken(ctx, provider, "tok-"+reqID, time.Now().Add(time.Hour)); err != nil {
						errCh <- fmt.Errorf("SetOAuthToken: %w", err)
						return
					}
					if _, _, found, err := st.GetOAuthToken(ctx, provider); err != nil || !found {
						errCh <- fmt.Errorf("GetOAuthToken found=%v err=%v", found, err)
						return
					}
				}
			}(w)
		}

		wg.Wait()
		close(errCh)
		for err := range errCh {
			t.Errorf("concurrent store error: %v", err)
		}
	})
}

func TestMemoryStoreContract(t *testing.T) {
	runStoreContractTests(t, func(t *testing.T) Store {
		return NewMemoryStore()
	})
}

func TestSQLiteStoreMemoryContract(t *testing.T) {
	runStoreContractTests(t, func(t *testing.T) Store {
		st, err := NewSQLiteStore(":memory:")
		if err != nil {
			t.Fatalf("NewSQLiteStore(:memory:) failed: %v", err)
		}
		return st
	})
}

func TestSQLiteStoreFileContract(t *testing.T) {
	runStoreContractTests(t, func(t *testing.T) Store {
		dbPath := filepath.Join(t.TempDir(), "nested", "open-parcels.db")
		st, err := NewSQLiteStore(dbPath)
		if err != nil {
			t.Fatalf("NewSQLiteStore(%s) failed: %v", dbPath, err)
		}
		return st
	})
}

func TestSQLitePersistenceAcrossReopenAndWAL(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "persist.db")

	st1, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore open 1 failed: %v", err)
	}

	// Verify WAL journal mode is active
	var journalMode string
	if err := st1.db.QueryRow(`PRAGMA journal_mode;`).Scan(&journalMode); err != nil {
		t.Fatalf("failed checking journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("expected journal_mode=wal, got %s", journalMode)
	}

	// Persist shipment, async job, and OAuth token
	shp := makeTestShipment("ups", "1Z999AA10123456784", models.StatusDelivered)
	if err := st1.SetShipment(ctx, "ups", "1Z999AA10123456784", "", shp, 24*time.Hour); err != nil {
		t.Fatalf("SetShipment failed: %v", err)
	}

	job := &AsyncJob{
		RequestID: "req_persist_1",
		Status:    "completed",
		Response: &models.ParcelsAppResponse{
			RequestID: "req_persist_1",
			Status:    "completed",
			Done:      true,
			Results: []models.ParcelsAppResultItem{
				{TrackingNumber: "1Z999AA10123456784", State: "completed"},
			},
		},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := st1.SetAsyncJob(ctx, job); err != nil {
		t.Fatalf("SetAsyncJob failed: %v", err)
	}

	tokExpiry := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	if err := st1.SetOAuthToken(ctx, "ups", "persisted-ups-token", tokExpiry); err != nil {
		t.Fatalf("SetOAuthToken failed: %v", err)
	}

	if err := st1.Close(); err != nil {
		t.Fatalf("st1.Close failed: %v", err)
	}

	// Reopen SQLite file from disk
	st2, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore reopen failed: %v", err)
	}
	defer st2.Close()

	gotShp, found, err := st2.GetShipment(ctx, "ups", "1Z999AA10123456784", "")
	if err != nil || !found {
		t.Fatalf("expected persisted shipment after reopen, err=%v found=%v", err, found)
	}
	if !gotShp.FromCache || gotShp.Status != models.StatusDelivered {
		t.Errorf("unexpected reopened shipment: fromCache=%v status=%s", gotShp.FromCache, gotShp.Status)
	}

	gotJob, found, err := st2.GetAsyncJob(ctx, "req_persist_1")
	if err != nil || !found {
		t.Fatalf("expected persisted async job after reopen, err=%v found=%v", err, found)
	}
	if gotJob.Status != "completed" || gotJob.Response == nil || len(gotJob.Response.Results) != 1 {
		t.Errorf("unexpected reopened async job: %+v", gotJob)
	}

	tok, exp, found, err := st2.GetOAuthToken(ctx, "ups")
	if err != nil || !found {
		t.Fatalf("expected persisted oauth token after reopen, err=%v found=%v", err, found)
	}
	if tok != "persisted-ups-token" || !exp.Equal(tokExpiry) {
		t.Errorf("unexpected reopened token: tok=%s exp=%v (want %v)", tok, exp, tokExpiry)
	}
}

func TestNewStoreFromEnv(t *testing.T) {
	t.Setenv(EnvDBPath, "")
	stMem, err := NewStoreFromEnv()
	if err != nil {
		t.Fatalf("NewStoreFromEnv empty failed: %v", err)
	}
	if _, ok := stMem.(*MemoryStore); !ok {
		t.Errorf("expected MemoryStore for empty %s, got %T", EnvDBPath, stMem)
	}
	_ = stMem.Close()

	t.Setenv(EnvDBPath, ":memory:")
	stSQLMem, err := NewStoreFromEnv()
	if err != nil {
		t.Fatalf("NewStoreFromEnv :memory: failed: %v", err)
	}
	if _, ok := stSQLMem.(*SQLiteStore); !ok {
		t.Errorf("expected SQLiteStore for :memory:, got %T", stSQLMem)
	}
	_ = stSQLMem.Close()

	dbFile := filepath.Join(t.TempDir(), "env.db")
	t.Setenv(EnvDBPath, dbFile)
	stFile, err := NewStoreFromEnv()
	if err != nil {
		t.Fatalf("NewStoreFromEnv file failed: %v", err)
	}
	if sqlSt, ok := stFile.(*SQLiteStore); !ok {
		t.Errorf("expected SQLiteStore for file path, got %T", stFile)
	} else if sqlSt.Path() != dbFile {
		t.Errorf("expected path %s, got %s", dbFile, sqlSt.Path())
	}
	_ = stFile.Close()
}