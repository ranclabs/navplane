-- Organizations table
-- Each organization has a NavPlane API key for authentication
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    api_key_hash VARCHAR(64) NOT NULL UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_organizations_api_key_hash ON organizations(api_key_hash);
CREATE INDEX idx_organizations_enabled ON organizations(enabled);

-- Provider keys table
-- Stores encrypted provider API keys (BYOK - Bring Your Own Key)
CREATE TABLE provider_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    key_alias VARCHAR(100) NOT NULL,
    encrypted_key BYTEA NOT NULL,
    key_nonce BYTEA NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, provider, key_alias)
);

CREATE INDEX idx_provider_keys_org_id ON provider_keys(org_id);
CREATE INDEX idx_provider_keys_provider ON provider_keys(org_id, provider, is_active);

-- Request logs table
-- Captures metadata for each proxied request (async write)
CREATE TABLE request_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    request_id VARCHAR(100),
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(100),
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER NOT NULL,
    latency_ms INTEGER NOT NULL,
    prompt_tokens INTEGER,
    completion_tokens INTEGER,
    total_tokens INTEGER,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_request_logs_org_id ON request_logs(org_id);
CREATE INDEX idx_request_logs_created_at ON request_logs(created_at DESC);
CREATE INDEX idx_request_logs_org_created ON request_logs(org_id, created_at DESC);
