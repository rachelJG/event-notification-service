-- Remove client_id index
DROP INDEX IF EXISTS idx_events_client_id;

-- Remove client_id column
ALTER TABLE events DROP COLUMN IF EXISTS client_id;
