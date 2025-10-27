-- Rollback: Admin Impersonation

DROP TABLE IF EXISTS auth.impersonation_sessions CASCADE;
