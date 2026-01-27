--
-- ROLLBACK: Drop partial unique index for system settings
--

DROP INDEX IF EXISTS app.idx_app_settings_system_key;
