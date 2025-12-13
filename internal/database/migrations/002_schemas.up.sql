-- Fluxbase Initial Database Schema - Schemas
-- This file creates all database schemas

--
-- AUTH SCHEMA
-- Handles application user authentication, API keys, and sessions
--

CREATE SCHEMA IF NOT EXISTS auth;
GRANT USAGE, CREATE ON SCHEMA auth TO CURRENT_USER;

--
-- APP SCHEMA
-- Handles application-level configuration and settings
--

CREATE SCHEMA IF NOT EXISTS app;
GRANT USAGE, CREATE ON SCHEMA app TO CURRENT_USER;

COMMENT ON SCHEMA app IS 'Schema for application-level configuration, settings, and metadata';

--
-- DASHBOARD SCHEMA
-- Handles Fluxbase platform administrator authentication and management
--

CREATE SCHEMA IF NOT EXISTS dashboard;
GRANT USAGE, CREATE ON SCHEMA dashboard TO CURRENT_USER;

COMMENT ON SCHEMA dashboard IS 'Schema for dashboard/platform administrator authentication and management';

--
-- FUNCTIONS SCHEMA
-- Handles edge functions and their executions
--

CREATE SCHEMA IF NOT EXISTS functions;
GRANT USAGE, CREATE ON SCHEMA functions TO CURRENT_USER;

--
-- STORAGE SCHEMA
-- Handles file storage buckets and objects
--

CREATE SCHEMA IF NOT EXISTS storage;
GRANT USAGE, CREATE ON SCHEMA storage TO CURRENT_USER;

--
-- REALTIME SCHEMA
-- Handles realtime subscriptions and change tracking
--

CREATE SCHEMA IF NOT EXISTS realtime;
GRANT USAGE, CREATE ON SCHEMA realtime TO CURRENT_USER;

--
-- MIGRATIONS SCHEMA
-- Handles all migration tracking:
-- - System migrations (Fluxbase internal)
-- - User migrations (filesystem-based)
-- - API-managed migrations (multi-tenant)
--

CREATE SCHEMA IF NOT EXISTS migrations;
GRANT USAGE, CREATE ON SCHEMA migrations TO CURRENT_USER;

COMMENT ON SCHEMA migrations IS 'Schema for all migration tracking including system, user, and API-managed migrations';

--
-- JOBS SCHEMA
-- Handles long-running background jobs
--

CREATE SCHEMA IF NOT EXISTS jobs;
GRANT USAGE, CREATE ON SCHEMA jobs TO CURRENT_USER;

COMMENT ON SCHEMA jobs IS 'Long-running background jobs system';

--
-- AI SCHEMA
-- Handles AI chatbots, conversations, and query auditing
--

CREATE SCHEMA IF NOT EXISTS ai;
GRANT USAGE, CREATE ON SCHEMA ai TO CURRENT_USER;

COMMENT ON SCHEMA ai IS 'AI chatbots, conversations, and query auditing';

--
-- RPC SCHEMA
-- Handles stored procedure definitions and executions
--

CREATE SCHEMA IF NOT EXISTS rpc;
GRANT USAGE, CREATE ON SCHEMA rpc TO CURRENT_USER;

COMMENT ON SCHEMA rpc IS 'Stored procedure definitions and executions';
