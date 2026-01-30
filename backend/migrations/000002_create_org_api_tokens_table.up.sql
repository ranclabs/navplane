-- Create org_api_tokens table for API authentication
CREATE TABLE org_api_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    token_hash   TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    
    -- Each token hash must be unique (prevents duplicate tokens)
    -- Note: UNIQUE constraint automatically creates a btree index, so no separate index needed
    CONSTRAINT org_api_tokens_hash_unique UNIQUE (token_hash)
);

-- Index on org_id for listing tokens by org
CREATE INDEX idx_org_api_tokens_org_id ON org_api_tokens(org_id);

-- Comments
COMMENT ON TABLE org_api_tokens IS 'API tokens for org authentication. Tokens are stored as SHA-256 hashes.';
COMMENT ON COLUMN org_api_tokens.token_hash IS 'SHA-256 hash of the API token (hex encoded). Never store plaintext tokens.';
COMMENT ON COLUMN org_api_tokens.last_used_at IS 'Updated on each successful authentication (optional, for auditing)';
