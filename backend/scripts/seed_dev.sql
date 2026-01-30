-- Seed script for development/testing
-- Run with: psql $DATABASE_URL -f backend/scripts/seed_dev.sql

-- Create a test organization
INSERT INTO orgs (id, name, enabled)
VALUES ('00000000-0000-0000-0000-000000000001', 'dev-org', true)
ON CONFLICT (name) DO NOTHING;

-- Create an API token for the test org
-- Token: np-dev-test-token-12345
-- SHA-256 hash of 'np-dev-test-token-12345':
--   echo -n 'np-dev-test-token-12345' | shasum -a 256
--   = 7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069  (example - recalculate)

-- Let's use a known test token
-- Token: test-token-for-dev
-- You can generate the hash with: echo -n 'test-token-for-dev' | shasum -a 256

INSERT INTO org_api_tokens (id, org_id, token_hash)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    -- SHA-256 of 'test-token-for-dev'
    '4ef32ae5fa04c7fdcc0e483e9b9c6bb73b7aab64d5477b41f66e32963728211c'
)
ON CONFLICT (token_hash) DO NOTHING;

-- Verify the data
SELECT 'Orgs:' as info;
SELECT id, name, enabled FROM orgs;

SELECT 'Tokens:' as info;
SELECT t.id, o.name as org_name, substring(t.token_hash, 1, 16) || '...' as token_hash_prefix
FROM org_api_tokens t
JOIN orgs o ON t.org_id = o.id;
