package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Client represents a database client.
type Client struct {
	db *sql.DB
}

// NewClient creates a new database client.
func NewClient(ctx context.Context, url string) (*Client, error) {
	// Open the database
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	return &Client{
		db: db,
	}, nil
}

// DB returns the underlying sql.DB.
func (c *Client) DB() *sql.DB {
	return c.db
}

// Close closes the database connection.
func (c *Client) Close(_ context.Context) error {
	return c.db.Close()
}

// GetFlaggedAndConfirmedUsers retrieves all user IDs from both tables.
func (c *Client) GetFlaggedAndConfirmedUsers(ctx context.Context) (map[uint64]uint8, error) {
	users := make(map[uint64]uint8)

	rows, err := c.db.QueryContext(ctx, `
		SELECT id, 1 as flag_type FROM flagged_users
		UNION ALL
		SELECT id, 2 as flag_type FROM confirmed_users
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uint64
		var flagType uint8
		if err := rows.Scan(&id, &flagType); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		users[id] = flagType
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return users, nil
}
