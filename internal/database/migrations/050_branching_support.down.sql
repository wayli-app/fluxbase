-- Revert database branching support
DROP TRIGGER IF EXISTS github_config_updated_at ON branching.github_config;
DROP TRIGGER IF EXISTS branches_updated_at ON branching.branches;
DROP FUNCTION IF EXISTS branching.update_updated_at();

DROP TABLE IF EXISTS branching.branch_access;
DROP TABLE IF EXISTS branching.github_config;
DROP TABLE IF EXISTS branching.activity_log;
DROP TABLE IF EXISTS branching.migration_history;
DROP TABLE IF EXISTS branching.branches;

DROP SCHEMA IF EXISTS branching;
