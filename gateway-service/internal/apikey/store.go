package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type APIKey struct {
	ID                string
	Name              string
	KeyHash           string
	RequestsPerMinute int
	IsActive          bool
	CreatedAt         string
}

type Store struct {
	db    *pgxpool.Pool
	cache *redis.Client
	ttl   int64
}

func NewStore(pool *pgxpool.Pool, cache *redis.Client, ttlSeconds int64) (*Store, error) {
	s := &Store{db: pool, cache: cache, ttl: ttlSeconds}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS api_keys (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		key_hash VARCHAR(64) UNIQUE NOT NULL,
		requests_per_minute INT DEFAULT 100,
		is_active BOOLEAN DEFAULT true,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_api_keys_is_active ON api_keys(is_active);
	`
	if _, err := pool.Exec(context.Background(), createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create api_keys table: %w", err)
	}

	return s, nil
}

func (s *Store) Hash(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}

func (s *Store) GenerateKey() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Store) Validate(ctx context.Context, plaintextKey string) (*APIKey, error) {
	keyHash := s.Hash(plaintextKey)

	cacheKey := fmt.Sprintf("apikey:%s", keyHash)
	_, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		return &APIKey{KeyHash: keyHash}, nil
	}

	var id string
	var name string
	var rpm int
	var isActive bool
	err = s.db.QueryRow(ctx,
		`SELECT id, name, requests_per_minute, is_active FROM api_keys WHERE key_hash = $1`,
		keyHash).
		Scan(&id, &name, &rpm, &isActive)

	if err != nil {
		return nil, fmt.Errorf("key not found: %w", err)
	}

	if !isActive {
		return nil, fmt.Errorf("key is inactive")
	}

	key := &APIKey{ID: id, Name: name, KeyHash: keyHash, RequestsPerMinute: rpm, IsActive: isActive}

	_ = s.cache.Set(ctx, cacheKey, "1", 0)
	_ = s.cache.Expire(ctx, cacheKey, time.Duration(s.ttl)*time.Second)

	return key, nil
}

func (s *Store) Create(ctx context.Context, name string, requestsPerMinute int) (*APIKey, string, error) {
	plaintextKey, err := s.GenerateKey()
	if err != nil {
		return nil, "", err
	}

	keyHash := s.Hash(plaintextKey)
	var id string
	err = s.db.QueryRow(ctx,
		`INSERT INTO api_keys (name, key_hash, requests_per_minute, is_active) 
		 VALUES ($1, $2, $3, true) RETURNING id`,
		name, keyHash, requestsPerMinute).Scan(&id)

	if err != nil {
		return nil, "", fmt.Errorf("failed to create key: %w", err)
	}

	return &APIKey{
		ID:                id,
		Name:              name,
		RequestsPerMinute: requestsPerMinute,
		IsActive:          true,
	}, plaintextKey, nil
}

func (s *Store) List(ctx context.Context) ([]APIKey, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, requests_per_minute, is_active, created_at FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.RequestsPerMinute, &k.IsActive, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}

	return keys, nil
}

func (s *Store) Deactivate(ctx context.Context, id string) error {
	tag, err := s.db.Exec(ctx, `UPDATE api_keys SET is_active = false WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("key not found")
	}

	_ = s.cache.Del(ctx, fmt.Sprintf("apikey:%s", id))

	return nil
}
