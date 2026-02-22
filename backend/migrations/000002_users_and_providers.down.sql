-- Remove triggers
DROP TRIGGER IF EXISTS trg_org_provider_settings_updated_at ON org_provider_settings;
DROP TRIGGER IF EXISTS trg_user_identities_updated_at ON user_identities;

-- Remove envelope encryption columns from provider_keys
ALTER TABLE provider_keys DROP COLUMN IF EXISTS dek_nonce;
ALTER TABLE provider_keys DROP COLUMN IF EXISTS encrypted_dek;

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS org_provider_settings;
DROP TABLE IF EXISTS org_members;
DROP TABLE IF EXISTS user_identities;
