CREATE TYPE notification_status AS ENUM ('pending', 'processing', 'delivered', 'failed');
CREATE TYPE notification_channel AS ENUM ('email');

CREATE TABLE IF NOT EXISTS notifications (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id      UUID NOT NULL REFERENCES events(id),
  channel       notification_channel NOT NULL,
  recipient     TEXT NOT NULL,
  subject       TEXT NOT NULL,
  body          TEXT NOT NULL,
  status        notification_status NOT NULL DEFAULT 'pending',
  attempts      INT NOT NULL DEFAULT 0,
  max_attempts  INT NOT NULL DEFAULT 5,
  last_error    TEXT,
  next_retry_at TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_pending ON notifications (next_retry_at)
  WHERE status IN ('pending', 'failed');
CREATE INDEX idx_notifications_event_id ON notifications (event_id);
