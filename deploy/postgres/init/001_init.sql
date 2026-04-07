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
