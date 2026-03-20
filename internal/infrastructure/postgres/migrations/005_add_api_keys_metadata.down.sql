-- Remove metadata index
DROP INDEX IF EXISTS idx_api_keys_metadata_client_id;

-- Remove metadata column
ALTER TABLE api_keys DROP COLUMN IF EXISTS metadata;
