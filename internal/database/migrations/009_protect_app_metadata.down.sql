-- Migration 009 Rollback: Remove app_metadata protection

-- Drop triggers
DROP TRIGGER IF EXISTS validate_app_metadata_trigger ON auth.users;
DROP TRIGGER IF EXISTS validate_app_metadata_trigger ON dashboard.users;

-- Drop function
DROP FUNCTION IF EXISTS auth.validate_app_metadata_update();
