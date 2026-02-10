CREATE TABLE IF NOT EXISTS events (
  id UUID PRIMARY KEY,
  type TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  payload JSONB NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_type ON events (type);
CREATE UNIQUE INDEX IF NOT EXISTS ux_events_idempotency_type ON events (idempotency_key, type);
