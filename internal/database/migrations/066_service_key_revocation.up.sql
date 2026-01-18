-- Add revocation and rotation support to service keys
-- Enables emergency revocation and graceful key rotation

-- Add revocation fields
ALTER TABLE auth.service_keys
ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS revoked_by UUID REFERENCES dashboard.users(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS revocation_reason TEXT;

-- Add rotation/deprecation fields for graceful key rotation
ALTER TABLE auth.service_keys
ADD COLUMN IF NOT EXISTS deprecated_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS grace_period_ends_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS replaced_by UUID REFERENCES auth.service_keys(id) ON DELETE SET NULL;

-- Index for finding revoked keys
CREATE INDEX IF NOT EXISTS idx_service_keys_revoked_at
    ON auth.service_keys(revoked_at)
    WHERE revoked_at IS NOT NULL;

-- Index for finding deprecated keys in grace period
CREATE INDEX IF NOT EXISTS idx_service_keys_grace_period
    ON auth.service_keys(grace_period_ends_at)
    WHERE deprecated_at IS NOT NULL AND grace_period_ends_at IS NOT NULL;

-- Add comments
COMMENT ON COLUMN auth.service_keys.revoked_at IS 'When the key was emergency revoked (NULL if not revoked)';
COMMENT ON COLUMN auth.service_keys.revoked_by IS 'Admin who revoked the key';
COMMENT ON COLUMN auth.service_keys.revocation_reason IS 'Reason for emergency revocation';
COMMENT ON COLUMN auth.service_keys.deprecated_at IS 'When the key was marked for rotation';
COMMENT ON COLUMN auth.service_keys.grace_period_ends_at IS 'When the grace period for rotation ends';
COMMENT ON COLUMN auth.service_keys.replaced_by IS 'Reference to the replacement key (for rotation)';

-- Table to audit service key revocations
CREATE TABLE IF NOT EXISTS auth.service_key_revocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id UUID NOT NULL REFERENCES auth.service_keys(id) ON DELETE CASCADE,
    key_prefix TEXT NOT NULL,
    revoked_by UUID REFERENCES dashboard.users(id) ON DELETE SET NULL,
    reason TEXT NOT NULL,
    revocation_type TEXT NOT NULL CHECK (revocation_type IN ('emergency', 'rotation', 'expiration')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_service_key_revocations_key_id
    ON auth.service_key_revocations(key_id);

CREATE INDEX IF NOT EXISTS idx_service_key_revocations_created_at
    ON auth.service_key_revocations(created_at);

COMMENT ON TABLE auth.service_key_revocations IS 'Audit log of service key revocations';
