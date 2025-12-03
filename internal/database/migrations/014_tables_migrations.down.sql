--
-- Rollback: Drop migrations schema tables
--

DROP TABLE IF EXISTS migrations.execution_logs;
DROP TABLE IF EXISTS migrations.app;
DROP TABLE IF EXISTS migrations.fluxbase;
