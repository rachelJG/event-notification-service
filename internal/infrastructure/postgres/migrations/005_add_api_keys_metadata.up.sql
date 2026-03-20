-- Add metadata column to api_keys table for storing client information
ALTER TABLE api_keys ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}';

-- Create index on client_id within metadata for efficient lookups
CREATE INDEX idx_api_keys_metadata_client_id ON api_keys ((metadata->>'client_id'));

-- Add comment explaining the metadata structure
COMMENT ON COLUMN api_keys.metadata IS 'Client metadata (client_id, organization, contact_email, etc.)';
