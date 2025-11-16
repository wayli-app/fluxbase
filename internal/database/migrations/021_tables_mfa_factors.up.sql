--
-- MFA FACTORS TABLE
-- Supabase-compatible MFA factors for multi-factor authentication
--

-- MFA factors table (Supabase-compatible)
CREATE TABLE IF NOT EXISTS auth.mfa_factors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE NOT NULL,
    friendly_name TEXT,
    factor_type TEXT NOT NULL CHECK (factor_type IN ('totp', 'phone')),
    status TEXT NOT NULL DEFAULT 'unverified' CHECK (status IN ('verified', 'unverified')),
    secret TEXT, -- TOTP secret
    phone TEXT, -- Phone number for SMS
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_auth_mfa_factors_user_id ON auth.mfa_factors(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_mfa_factors_user_id_status ON auth.mfa_factors(user_id, status) WHERE status = 'verified';
CREATE INDEX IF NOT EXISTS idx_auth_mfa_factors_factor_type ON auth.mfa_factors(factor_type);

-- Trigger to update updated_at timestamp
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON auth.mfa_factors
    FOR EACH ROW
    EXECUTE FUNCTION public.update_updated_at();

-- Comments
COMMENT ON TABLE auth.mfa_factors IS 'Supabase-compatible MFA factors table for multi-factor authentication';
COMMENT ON COLUMN auth.mfa_factors.factor_type IS 'Type of MFA factor: totp (authenticator app) or phone (SMS)';
COMMENT ON COLUMN auth.mfa_factors.status IS 'Factor status: verified (active) or unverified (pending verification)';
COMMENT ON COLUMN auth.mfa_factors.secret IS 'TOTP secret for authenticator apps (encrypted at application level)';
COMMENT ON COLUMN auth.mfa_factors.phone IS 'Phone number for SMS-based 2FA';
