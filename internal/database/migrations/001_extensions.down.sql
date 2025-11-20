-- Rollback Extensions
-- Note: Extensions are typically not dropped as they may be used by other databases

DROP EXTENSION IF EXISTS "postgis";
DROP EXTENSION IF EXISTS "btree_gin";
DROP EXTENSION IF EXISTS "pg_trgm";
DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";
