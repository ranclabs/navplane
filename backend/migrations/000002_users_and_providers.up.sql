-- User identities from Auth0 (minimal local storage)
-- We only store what we need for authorization, Auth0 handles passwords/MFA
CREATE TABLE user_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth0_user_id VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    is_admin BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_identities_auth0_id ON user_identities(auth0_user_id);
CREATE INDEX idx_user_identities_email ON user_identities(email);

-- User-Org membership (many-to-many)
CREATE TABLE org_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES user_identities(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, user_id)
);

CREATE INDEX idx_org_members_org_id ON org_members(org_id);
CREATE INDEX idx_org_members_user_id ON org_members(user_id);

-- Org provider settings (enable/disable providers and models per org)
CREATE TABLE org_provider_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    allowed_models TEXT[],
    blocked_models TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, provider)
);

CREATE INDEX idx_org_provider_settings_org_id ON org_provider_settings(org_id);

-- Add envelope encryption support to provider_keys
-- encrypted_dek stores the Data Encryption Key encrypted with the master KEK
ALTER TABLE provider_keys ADD COLUMN IF NOT EXISTS encrypted_dek BYTEA;
ALTER TABLE provider_keys ADD COLUMN IF NOT EXISTS dek_nonce BYTEA;

-- Triggers for updated_at
CREATE TRIGGER trg_user_identities_updated_at
    BEFORE UPDATE ON user_identities
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_org_provider_settings_updated_at
    BEFORE UPDATE ON org_provider_settings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
