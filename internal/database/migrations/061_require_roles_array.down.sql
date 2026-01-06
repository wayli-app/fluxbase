-- Rollback: Convert require_roles TEXT[] back to require_role TEXT

-- Jobs: Convert require_roles back to require_role
ALTER TABLE jobs.functions ADD COLUMN IF NOT EXISTS require_role TEXT;

UPDATE jobs.functions
SET require_role = CASE
    WHEN require_roles IS NOT NULL AND array_length(require_roles, 1) > 0 THEN require_roles[1]
    ELSE NULL
END;

ALTER TABLE jobs.functions DROP COLUMN IF EXISTS require_roles;

COMMENT ON COLUMN jobs.functions.require_role IS 'Required role to submit this job (admin, authenticated, anon, or null for any)';

-- RPC: Convert require_roles back to require_role
ALTER TABLE rpc.procedures ADD COLUMN IF NOT EXISTS require_role TEXT;

UPDATE rpc.procedures
SET require_role = CASE
    WHEN require_roles IS NOT NULL AND array_length(require_roles, 1) > 0 THEN require_roles[1]
    ELSE NULL
END;

ALTER TABLE rpc.procedures DROP COLUMN IF EXISTS require_roles;

COMMENT ON COLUMN rpc.procedures.require_role IS 'Role required to invoke (authenticated, admin, anon, or null for any)';

-- Chatbots: Remove require_roles column
ALTER TABLE ai.chatbots DROP COLUMN IF EXISTS require_roles;
