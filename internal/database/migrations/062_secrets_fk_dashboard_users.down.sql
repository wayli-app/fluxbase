-- Revert: Change secrets foreign keys back to auth.users

ALTER TABLE functions.secrets DROP CONSTRAINT IF EXISTS secrets_created_by_fkey;
ALTER TABLE functions.secrets DROP CONSTRAINT IF EXISTS secrets_updated_by_fkey;

ALTER TABLE functions.secrets
    ADD CONSTRAINT secrets_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES auth.users(id) ON DELETE SET NULL;

ALTER TABLE functions.secrets
    ADD CONSTRAINT secrets_updated_by_fkey
    FOREIGN KEY (updated_by) REFERENCES auth.users(id) ON DELETE SET NULL;

ALTER TABLE functions.secret_versions DROP CONSTRAINT IF EXISTS secret_versions_created_by_fkey;

ALTER TABLE functions.secret_versions
    ADD CONSTRAINT secret_versions_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES auth.users(id) ON DELETE SET NULL;
