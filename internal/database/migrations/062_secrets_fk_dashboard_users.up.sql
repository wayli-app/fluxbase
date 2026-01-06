-- Change secrets foreign keys from auth.users to dashboard.users
-- Secrets are managed by dashboard admins, not app users

-- Drop existing foreign key constraints
ALTER TABLE functions.secrets DROP CONSTRAINT IF EXISTS secrets_created_by_fkey;
ALTER TABLE functions.secrets DROP CONSTRAINT IF EXISTS secrets_updated_by_fkey;

-- Add new foreign key constraints referencing dashboard.users
ALTER TABLE functions.secrets
    ADD CONSTRAINT secrets_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES dashboard.users(id) ON DELETE SET NULL;

ALTER TABLE functions.secrets
    ADD CONSTRAINT secrets_updated_by_fkey
    FOREIGN KEY (updated_by) REFERENCES dashboard.users(id) ON DELETE SET NULL;

-- Same for secret_versions table
ALTER TABLE functions.secret_versions DROP CONSTRAINT IF EXISTS secret_versions_created_by_fkey;

ALTER TABLE functions.secret_versions
    ADD CONSTRAINT secret_versions_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES dashboard.users(id) ON DELETE SET NULL;
