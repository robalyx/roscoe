package d1

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// FlagResponse is the response type for flag operations.
type FlagResponse struct {
	Flag       uint8          `json:"flagType"`
	Confidence *float32       `json:"confidence,omitempty"`
	Reasons    sql.NullString `json:"reasons,omitempty"`
}

// FlagService handles flag operations in D1.
type FlagService struct {
	db *sql.DB
}

// NewFlagService creates a new flag service.
func NewFlagService(db *sql.DB) *FlagService {
	return &FlagService{
		db: db,
	}
}

// GetUserFlags retrieves flags for the given user IDs.
func (s *FlagService) GetUserFlags(ctx context.Context, ids []uint64) (map[uint64]FlagResponse, error) {
	if len(ids) == 0 {
		return make(map[uint64]FlagResponse), nil
	}

	// Build query for user_flags table
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT user_id, flag_type, confidence, reasons FROM user_flags WHERE user_id IN (")

	args := make([]any, len(ids))
	for i, id := range ids {
		if i > 0 {
			queryBuilder.WriteString(",")
		}
		queryBuilder.WriteString("?")
		args[i] = id
	}
	queryBuilder.WriteString(")")

	// Execute query
	rows, err := s.db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("error querying flags: %w", err)
	}
	defer rows.Close()

	// Read results
	flags := make(map[uint64]FlagResponse)
	for rows.Next() {
		var id uint64
		var flag uint8
		var confidence float32
		var reasons sql.NullString
		if err := rows.Scan(&id, &flag, &confidence, &reasons); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		flags[id] = FlagResponse{
			Flag:       flag,
			Confidence: &confidence,
			Reasons:    reasons,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Build query for queued_users table
	queryBuilder.Reset()
	queryBuilder.WriteString("SELECT user_id FROM queued_users WHERE processed = 1 AND flagged = 1 AND user_id IN (")
	for i := range ids {
		if i > 0 {
			queryBuilder.WriteString(",")
		}
		queryBuilder.WriteString("?")
	}
	queryBuilder.WriteString(")")

	// Execute queue query
	queueRows, err := s.db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("error querying queued users: %w", err)
	}
	defer queueRows.Close()

	// Add queued users that aren't already in flags map
	for queueRows.Next() {
		var id uint64
		if err := queueRows.Scan(&id); err != nil {
			return nil, fmt.Errorf("error scanning queue row: %w", err)
		}
		// Only add if not already in flags map
		if _, exists := flags[id]; !exists {
			flags[id] = FlagResponse{Flag: 3}
		}
	}
	if err := queueRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating queue rows: %w", err)
	}

	return flags, nil
}
