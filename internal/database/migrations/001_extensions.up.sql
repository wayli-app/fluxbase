-- Fluxbase Initial Database Schema - Extensions
-- This file enables required PostgreSQL extensions

-- UUID generation functions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";    -- Provides uuid_generate_v4()
CREATE EXTENSION IF NOT EXISTS "pgcrypto";     -- Provides gen_random_uuid() and crypto functions

-- Text search and indexing
CREATE EXTENSION IF NOT EXISTS "pg_trgm";      -- Trigram text search
CREATE EXTENSION IF NOT EXISTS "btree_gin";    -- GIN indexes for btree-indexable data types

-- Geospatial data support
CREATE EXTENSION IF NOT EXISTS "postgis";      -- PostGIS for geographic objects