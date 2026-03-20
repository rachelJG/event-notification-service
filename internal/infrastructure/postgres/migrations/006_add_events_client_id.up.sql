-- Add client_id column to events table for tracking which client submitted each event
ALTER TABLE events ADD COLUMN client_id TEXT;

-- Create index on client_id for efficient filtering and analytics
CREATE INDEX idx_events_client_id ON events (client_id);

-- Add comment explaining the column purpose
COMMENT ON COLUMN events.client_id IS 'Client identifier from API key metadata, used for auditing and analytics';
