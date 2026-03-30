-- Drop trigger
DROP TRIGGER IF EXISTS trigger_update_addresses_updated_at ON addresses;
DROP FUNCTION IF EXISTS update_addresses_updated_at();

-- Drop addresses table
DROP TABLE IF EXISTS addresses;

-- Drop indexes from users
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_is_active;

-- Remove new columns from users
ALTER TABLE users
DROP COLUMN IF EXISTS role,
DROP COLUMN IF EXISTS avatar_url,
DROP COLUMN IF EXISTS is_active;
