-- Fluxbase Initial Database Schema - Schemas
-- This file creates all database schemas

--
-- _FLUXBASE SCHEMA
-- Internal Fluxbase system schema for migration tracking and system tables
--

CREATE SCHEMA IF NOT EXISTS _fluxbase;
GRANT USAGE, CREATE ON SCHEMA _fluxbase TO CURRENT_USER;

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
