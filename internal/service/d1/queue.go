package d1

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrUserAlreadyFlagged = errors.New("user is already flagged or confirmed")
	ErrUserRecentlyQueued = errors.New("user was queued within the past 7 days")
)

// QueueService handles user queue operations in D1.
type QueueService struct {
	db          *sql.DB
	flagService *FlagService
}

// NewQueueService creates a new queue service.
func NewQueueService(db *sql.DB, flagService *FlagService) *QueueService {
	return &QueueService{
		db:          db,
		flagService: flagService,
	}
}

// QueueUser adds a user to the processing queue.
func (s *QueueService) QueueUser(ctx context.Context, userID uint64) error {
	// Step 1: Check if the user is already flagged or confirmed
	flags, err := s.flagService.GetUserFlags(ctx, []uint64{userID})
	if err != nil {
		return fmt.Errorf("error checking user flags: %w", err)
	}

	if _, exists := flags[userID]; exists {
		return ErrUserAlreadyFlagged
	}

	// Step 2: Check if the user was queued in the past 7 days
	var queuedAt sql.NullInt64
	err = s.db.QueryRowContext(ctx,
		"SELECT queued_at FROM queued_users WHERE user_id = ?",
		userID,
	).Scan(&queuedAt)

	// If we found a record, check its timestamp
	if err == nil && queuedAt.Valid {
		cutoffTime := time.Now().AddDate(0, 0, -7).Unix()
		if queuedAt.Int64 > cutoffTime {
			return ErrUserRecentlyQueued
		}

		// Update existing record
		_, err = s.db.ExecContext(ctx,
			"UPDATE queued_users SET queued_at = ?, processed = 0 WHERE user_id = ?",
			time.Now().Unix(), userID,
		)
		return err
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("error checking queue status: %w", err)
	}

	// Step 3: Add user to queue
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO queued_users (user_id, queued_at, processed) VALUES (?, ?, 0)",
		userID, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("error adding user to queue: %w", err)
	}

	return nil
}

// InitQueueTable ensures the queue table exists.
func (s *QueueService) InitQueueTable(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS queued_users (
            user_id INTEGER PRIMARY KEY,
            queued_at INTEGER NOT NULL,
            processed INTEGER NOT NULL DEFAULT 0,
            processing INTEGER NOT NULL DEFAULT 0,
            flagged INTEGER NOT NULL DEFAULT 0
        );
        
        -- Index to efficiently find unprocessed and non-processing users ordered by queue time
        CREATE INDEX IF NOT EXISTS idx_queue_status 
        ON queued_users (processed, processing, queued_at)
        WHERE processed = 0 AND processing = 0;

        -- Index to efficiently find processed and flagged users
        CREATE INDEX IF NOT EXISTS idx_processed_flagged
        ON queued_users (processed, flagged)
        WHERE processed = 1 AND flagged = 1;
    `)
	return err
}
