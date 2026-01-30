-- Drop orgs table and related objects
DROP TRIGGER IF EXISTS orgs_updated_at ON orgs;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS orgs;
