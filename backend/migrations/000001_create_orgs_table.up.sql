-- Create orgs table for organization management
CREATE TABLE orgs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Org names must be unique
    CONSTRAINT orgs_name_unique UNIQUE (name)
);

-- Index on enabled for filtering active orgs
CREATE INDEX idx_orgs_enabled ON orgs(enabled);

-- Auto-update updated_at on row changes
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER orgs_updated_at
    BEFORE UPDATE ON orgs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE orgs IS 'Organizations that use NavPlane';
COMMENT ON COLUMN orgs.enabled IS 'Kill switch: set to false to disable all API access for this org';
