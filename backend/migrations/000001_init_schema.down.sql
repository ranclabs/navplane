DROP TRIGGER IF EXISTS trg_provider_keys_updated_at ON provider_keys;
DROP TRIGGER IF EXISTS trg_organizations_updated_at ON organizations;
DROP FUNCTION IF EXISTS set_updated_at();
DROP TABLE IF EXISTS request_logs;
DROP TABLE IF EXISTS provider_keys;
DROP TABLE IF EXISTS organizations;
