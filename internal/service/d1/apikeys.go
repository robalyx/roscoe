package d1

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrKeyNotFound = errors.New("key not found")

// APIKey represents an API key record.
type APIKey struct {
	Key         string
	Description string
	CreatedAt   int64
}

// APIKeyService handles API key operations in D1.
type APIKeyService struct {
	db *sql.DB
}

// NewAPIKeyService creates a new API key service.
func NewAPIKeyService(db *sql.DB) *APIKeyService {
	return &APIKeyService{
		db: db,
	}
}

// AddKey adds a new API key.
func (s *APIKeyService) AddKey(ctx context.Context, key, description string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO api_keys (key, description, created_at) VALUES (?, ?, ?)",
		key, description, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("error adding API key: %w", err)
	}
	return nil
}

// RemoveKey removes an API key.
func (s *APIKeyService) RemoveKey(ctx context.Context, key string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM api_keys WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("error removing API key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}
	if rows == 0 {
		return ErrKeyNotFound
	}

	return nil
}

// ValidateKey checks if an API key is valid.
func (s *APIKeyService) ValidateKey(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM api_keys WHERE key = ?)",
		key,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error validating API key: %w", err)
	}
	return exists, nil
}

// ListKeys returns all API keys.
func (s *APIKeyService) ListKeys(ctx context.Context) ([]APIKey, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT key, description, created_at FROM api_keys ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("error querying API keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var key APIKey
		if err := rows.Scan(&key.Key, &key.Description, &key.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return keys, nil
}

// GenerateKey generates a secure random API key.
func GenerateKey() (string, error) {
	// Generate 32 bytes of random data
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("error generating random bytes: %w", err)
	}

	// Encode as URL-safe base64 and remove padding
	key := base64.URLEncoding.EncodeToString(bytes)
	key = strings.TrimRight(key, "=")

	return key, nil
}
