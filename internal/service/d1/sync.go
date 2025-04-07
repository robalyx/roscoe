package d1

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

const (
	batchSize     = 25
	maxConcurrent = 5
)

// Record represents a user flag record.
type Record struct {
	userID     uint64
	flagType   uint8
	confidence float32
	reasons    string
}

// SyncService handles syncing flags from Postgres to D1.
type SyncService struct {
	sourceDB    *sql.DB
	cfAPI       *CloudflareAPI
	totalFlags  atomic.Int64
	syncedFlags atomic.Int64
}

// NewSyncService creates a new sync service.
func NewSyncService(sourceDB *sql.DB, accountID, d1ID, token string) *SyncService {
	return &SyncService{
		sourceDB: sourceDB,
		cfAPI:    NewCloudflareAPI(accountID, d1ID, token),
	}
}

// UpdateFlags syncs the latest flag data from Postgres to D1.
func (s *SyncService) UpdateFlags(ctx context.Context) error {
	if err := s.initializeTables(ctx); err != nil {
		return fmt.Errorf("failed to initialize tables: %w", err)
	}

	records, err := s.fetchRecords(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch records: %w", err)
	}

	if len(records) == 0 {
		log.Printf("No flags to sync")
		return nil
	}

	if err := s.processBatches(ctx, records); err != nil {
		return fmt.Errorf("failed to process batches: %w", err)
	}

	if err := s.swapTables(ctx); err != nil {
		return fmt.Errorf("failed to swap tables: %w", err)
	}

	log.Printf("✅ Successfully synced %d flags", len(records))
	return nil
}

// initializeTables creates the necessary tables in D1.
func (s *SyncService) initializeTables(ctx context.Context) error {
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS user_flags (
			user_id INTEGER PRIMARY KEY,
			flag_type INTEGER NOT NULL,
			confidence REAL NOT NULL,
			reasons TEXT
		);
		CREATE TABLE IF NOT EXISTS api_keys (
			key TEXT PRIMARY KEY,
			description TEXT,
			created_at INTEGER NOT NULL
		);
		DROP TABLE IF EXISTS new_flags;
		CREATE TABLE new_flags (
			user_id INTEGER PRIMARY KEY,
			flag_type INTEGER NOT NULL,
			confidence REAL NOT NULL,
			reasons TEXT
		);
	`
	if _, err := s.cfAPI.ExecuteSQL(ctx, createTableSQL, nil); err != nil {
		return fmt.Errorf("error creating tables: %w", err)
	}
	return nil
}

// fetchRecords retrieves all records from the source database.
func (s *SyncService) fetchRecords(ctx context.Context) ([]Record, error) {
	log.Printf("📊 Fetching users from database...")
	rows, err := s.sourceDB.QueryContext(ctx, `
		SELECT id, 1 as flag_type, confidence, reasons FROM flagged_users
		UNION ALL
		SELECT id, 2 as flag_type, confidence, reasons FROM confirmed_users
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying users: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var userID uint64
		var flagType uint8
		var confidence float32
		var reasons string
		if err := rows.Scan(&userID, &flagType, &confidence, &reasons); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		records = append(records, Record{userID: userID, flagType: flagType, confidence: confidence, reasons: reasons})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return records, nil
}

// processBatches processes records in concurrent batches.
func (s *SyncService) processBatches(ctx context.Context, records []Record) error {
	s.totalFlags.Store(int64(len(records)))
	s.syncedFlags.Store(0)

	// Process records in concurrent batches
	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(maxConcurrent)
	errCh := make(chan error, len(records)/batchSize+1)

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]
		wg.Add(1)

		// Acquire semaphore
		if err := sem.Acquire(ctx, 1); err != nil {
			return fmt.Errorf("failed to acquire semaphore: %w", err)
		}

		go func(batch []Record) {
			defer wg.Done()
			defer sem.Release(1)

			if err := s.processBatch(ctx, batch); err != nil {
				errCh <- fmt.Errorf("error processing batch: %w", err)
			}
		}(batch)
	}

	// Wait for all batches to complete
	wg.Wait()
	close(errCh)

	// Check for any errors
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

// swapTables swaps the new_flags table with the user_flags table.
func (s *SyncService) swapTables(ctx context.Context) error {
	if _, err := s.cfAPI.ExecuteSQL(ctx, `
		-- Rename the current table to old_flags
		ALTER TABLE user_flags RENAME TO old_flags;
		
		-- Rename new_flags to be the main table
		ALTER TABLE new_flags RENAME TO user_flags;
		
		-- Clean up the old table
		DROP TABLE old_flags;
	`, nil); err != nil {
		return fmt.Errorf("error swapping tables: %w", err)
	}
	return nil
}

func (s *SyncService) processBatch(ctx context.Context, batch []Record) error {
	if len(batch) == 0 {
		return nil
	}

	// Process records in smaller chunks to stay within SQLite limits
	for i := 0; i < len(batch); i += batchSize {
		end := i + batchSize
		if end > len(batch) {
			end = len(batch)
		}
		chunk := batch[i:end]

		// Build batch insert statement
		sqlStmt := `
			INSERT INTO new_flags (user_id, flag_type, confidence, reasons)
			VALUES
		`

		// Build values and params
		params := make([]any, 0, len(chunk)*4)
		for j, rec := range chunk {
			if j > 0 {
				sqlStmt += ","
			}
			sqlStmt += "(?, ?, ?, ?)"
			params = append(params,
				rec.userID,
				rec.flagType,
				rec.confidence,
				rec.reasons,
			)
		}

		if _, err := s.cfAPI.ExecuteSQL(ctx, sqlStmt, params); err != nil {
			return fmt.Errorf("error executing D1 statement: %w", err)
		}
	}

	// Update progress after successful batch
	synced := s.syncedFlags.Add(int64(len(batch)))
	total := s.totalFlags.Load()
	percentage := float64(synced) / float64(total) * 100
	log.Printf("☁️  Progress: %.1f%% (%d/%d flags)", percentage, synced, total)

	return nil
}
