--
-- ROLLBACK: Restore global unique constraint on key
-- Note: This will fail if there are duplicate keys across users
--

ALTER TABLE app.settings ADD CONSTRAINT settings_key_key UNIQUE (key);
