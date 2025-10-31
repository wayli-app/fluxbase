-- This migration is kept for compatibility but is essentially a no-op
-- Tables were created directly in the default schema by migration 012
-- No schema moves needed as tables are already where they should be

-- Just ensure the functions schema exists for consistency
CREATE SCHEMA IF NOT EXISTS functions;
