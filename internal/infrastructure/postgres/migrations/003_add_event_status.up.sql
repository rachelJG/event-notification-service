ALTER TABLE events
  ADD COLUMN status TEXT NOT NULL DEFAULT 'accepted'
  CHECK (status IN ('accepted', 'processing', 'notified', 'failed'));

CREATE INDEX idx_events_status ON events (status)
  WHERE status = 'accepted';
