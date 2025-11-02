-- Drop all Fluxbase schemas and their contents
-- This will cascade and remove all tables, functions, and other objects

DROP SCHEMA IF EXISTS realtime CASCADE;
DROP SCHEMA IF EXISTS storage CASCADE;
DROP SCHEMA IF EXISTS functions CASCADE;
DROP SCHEMA IF EXISTS dashboard CASCADE;
DROP SCHEMA IF EXISTS auth CASCADE;
DROP SCHEMA IF EXISTS _fluxbase CASCADE;

-- Drop global helper function
DROP FUNCTION IF EXISTS public.update_updated_at() CASCADE;

-- Drop extensions (optional - comment out if other databases use these)
-- DROP EXTENSION IF EXISTS "pg_trgm";
-- DROP EXTENSION IF EXISTS "uuid-ossp";
