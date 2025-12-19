-- Rollback: System Tables Migration
-- Remove system-level tables

DROP TABLE IF EXISTS system.rate_limits;
