package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

type ModerationLog struct {
	RequestID        string
	OriginalText     string
	PreprocessedText string
	LabelsJSON       string
	RiskScore        float64
	Action           string
	LLMError         string
}

func NewPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	db := &Postgres{pool: pool}
	if err := db.ensureSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return db, nil
}

func (p *Postgres) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

func (p *Postgres) ensureSchema(ctx context.Context) error {
	const query = `
CREATE TABLE IF NOT EXISTS moderation_logs (
  id BIGSERIAL PRIMARY KEY,
  request_id TEXT NOT NULL,
  original_text TEXT NOT NULL,
  preprocessed_text TEXT NOT NULL,
  labels JSONB NOT NULL,
  risk_score DOUBLE PRECISION NOT NULL,
  action TEXT NOT NULL,
  llm_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_moderation_logs_created_at ON moderation_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_moderation_logs_action ON moderation_logs (action);
`
	_, err := p.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}
	return nil
}

func (p *Postgres) InsertLog(ctx context.Context, log ModerationLog) error {
	const query = `
INSERT INTO moderation_logs (
  request_id,
  original_text,
  preprocessed_text,
  labels,
  risk_score,
  action,
  llm_error
) VALUES ($1, $2, $3, $4::jsonb, $5, $6, NULLIF($7, ''))`

	_, err := p.pool.Exec(
		ctx,
		query,
		log.RequestID,
		log.OriginalText,
		log.PreprocessedText,
		log.LabelsJSON,
		log.RiskScore,
		log.Action,
		log.LLMError,
	)
	if err != nil {
		return fmt.Errorf("insert log: %w", err)
	}
	return nil
}
