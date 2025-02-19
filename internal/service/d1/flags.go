package d1

import (
	"context"
	"database/sql"
	"fmt"
)

// FlagResponse is the response type for flag operations.
type FlagResponse struct {
	Flag       uint8
	Confidence float32
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

	// Build query
	query := "SELECT user_id, flag_type, confidence FROM user_flags WHERE user_id IN ("
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
	}
	query += ")"

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
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
		if err := rows.Scan(&id, &flag, &confidence); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		flags[id] = FlagResponse{Flag: flag, Confidence: confidence}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return flags, nil
}
