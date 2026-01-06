-- Migration: Convert require_role TEXT to require_roles TEXT[] for jobs, RPCs, and chatbots
-- This allows specifying multiple roles with OR semantics (user needs ANY of the specified roles)

-- Jobs: Convert require_role to require_roles
ALTER TABLE jobs.functions ADD COLUMN IF NOT EXISTS require_roles TEXT[] DEFAULT ARRAY[]::TEXT[];

UPDATE jobs.functions
SET require_roles = CASE
    WHEN require_role IS NOT NULL AND require_role != '' THEN ARRAY[require_role]
    ELSE ARRAY[]::TEXT[]
END
WHERE require_roles = ARRAY[]::TEXT[] OR require_roles IS NULL;

ALTER TABLE jobs.functions DROP COLUMN IF EXISTS require_role;

COMMENT ON COLUMN jobs.functions.require_roles IS 'Required roles to submit this job (admin, authenticated, anon, or custom roles). User needs ANY of the specified roles.';

-- RPC: Convert require_role to require_roles
ALTER TABLE rpc.procedures ADD COLUMN IF NOT EXISTS require_roles TEXT[] DEFAULT ARRAY[]::TEXT[];

UPDATE rpc.procedures
SET require_roles = CASE
    WHEN require_role IS NOT NULL AND require_role != '' THEN ARRAY[require_role]
    ELSE ARRAY[]::TEXT[]
END
WHERE require_roles = ARRAY[]::TEXT[] OR require_roles IS NULL;

ALTER TABLE rpc.procedures DROP COLUMN IF EXISTS require_role;

COMMENT ON COLUMN rpc.procedures.require_roles IS 'Roles required to invoke (authenticated, admin, anon, or custom roles). User needs ANY of the specified roles.';

-- Chatbots: Add require_roles column (new feature)
ALTER TABLE ai.chatbots ADD COLUMN IF NOT EXISTS require_roles TEXT[] DEFAULT ARRAY[]::TEXT[];

COMMENT ON COLUMN ai.chatbots.require_roles IS 'Required roles to access this chatbot. User needs ANY of the specified roles.';
