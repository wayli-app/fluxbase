--
-- AUTH SCHEMA TABLES
-- Application user authentication, API keys, sessions, and webhooks
--

-- Users table (with 2FA and split metadata support)
CREATE TABLE IF NOT EXISTS auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT,
    email_verified BOOLEAN DEFAULT false,
    role TEXT DEFAULT 'authenticated',
    user_metadata JSONB DEFAULT '{}'::JSONB,
    app_metadata JSONB DEFAULT '{}'::JSONB,
    totp_secret VARCHAR(32),
    totp_enabled BOOLEAN DEFAULT FALSE,
    backup_codes TEXT[],
    failed_login_attempts INTEGER DEFAULT 0,
    is_locked BOOLEAN DEFAULT false,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_users_email ON auth.users(email);
CREATE INDEX IF NOT EXISTS idx_auth_users_totp_enabled ON auth.users(totp_enabled) WHERE totp_enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_auth_users_user_metadata ON auth.users USING GIN (user_metadata);
CREATE INDEX IF NOT EXISTS idx_auth_users_app_metadata ON auth.users USING GIN (app_metadata);
CREATE INDEX IF NOT EXISTS idx_auth_users_is_locked ON auth.users(is_locked) WHERE is_locked = true;

COMMENT ON COLUMN auth.users.user_metadata IS 'User-editable metadata. Users can update this field themselves. Included in JWT claims.';
COMMENT ON COLUMN auth.users.app_metadata IS 'Application/admin-only metadata. Can only be updated by admins or service role. Included in JWT claims.';
COMMENT ON COLUMN auth.users.failed_login_attempts IS 'Number of consecutive failed login attempts';
COMMENT ON COLUMN auth.users.is_locked IS 'Whether the account is locked due to too many failed attempts';
COMMENT ON COLUMN auth.users.locked_until IS 'When the account lock expires (null = permanent until admin unlocks)';

-- Sessions table
CREATE TABLE IF NOT EXISTS auth.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE NOT NULL,
    access_token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth.sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_access_token ON auth.sessions(access_token);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_refresh_token ON auth.sessions(refresh_token);

-- Magic links table
CREATE TABLE IF NOT EXISTS auth.magic_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT false,
    used_at TIMESTAMPTZ,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_magic_links_token ON auth.magic_links(token);
CREATE INDEX IF NOT EXISTS idx_auth_magic_links_email ON auth.magic_links(email);

-- OTP codes table (for email/SMS one-time password authentication)
CREATE TABLE IF NOT EXISTS auth.otp_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT,
    phone TEXT,
    code VARCHAR(10) NOT NULL,
    type TEXT NOT NULL,
    purpose TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT false,
    used_at TIMESTAMPTZ,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT otp_email_or_phone CHECK (email IS NOT NULL OR phone IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_auth_otp_codes_email ON auth.otp_codes(email) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_auth_otp_codes_phone ON auth.otp_codes(phone) WHERE phone IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_auth_otp_codes_code ON auth.otp_codes(code);
CREATE INDEX IF NOT EXISTS idx_auth_otp_codes_expires_at ON auth.otp_codes(expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_otp_codes_type ON auth.otp_codes(type);

COMMENT ON TABLE auth.otp_codes IS 'One-time password codes for email/SMS passwordless authentication. Entries expire after configured period and should be cleaned up periodically.';
COMMENT ON COLUMN auth.otp_codes.type IS 'Type of OTP: email, sms';
COMMENT ON COLUMN auth.otp_codes.purpose IS 'Purpose: signin, signup, recovery, email_change, phone_change';
COMMENT ON COLUMN auth.otp_codes.attempts IS 'Number of failed verification attempts. Locked after max_attempts.';

-- Password reset tokens table
CREATE TABLE IF NOT EXISTS auth.password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT false,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_password_reset_tokens_token ON auth.password_reset_tokens(token);
CREATE INDEX IF NOT EXISTS idx_auth_password_reset_tokens_user_id ON auth.password_reset_tokens(user_id);

-- Token blacklist table
CREATE TABLE IF NOT EXISTS auth.token_blacklist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_jti TEXT UNIQUE NOT NULL,
    token_type TEXT NOT NULL DEFAULT 'access',
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_token_blacklist_token_jti ON auth.token_blacklist(token_jti);
CREATE INDEX IF NOT EXISTS idx_auth_token_blacklist_expires_at ON auth.token_blacklist(expires_at);

-- API keys table
CREATE TABLE IF NOT EXISTS auth.api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    key_hash TEXT UNIQUE NOT NULL,
    key_prefix TEXT NOT NULL,
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    scopes TEXT[] DEFAULT ARRAY[]::TEXT[],
    rate_limit_per_minute INTEGER DEFAULT 60,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked BOOLEAN DEFAULT false,
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_api_keys_key_hash ON auth.api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_auth_api_keys_user_id ON auth.api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_api_keys_key_prefix ON auth.api_keys(key_prefix);

-- API key usage tracking
CREATE TABLE IF NOT EXISTS auth.api_key_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID REFERENCES auth.api_keys(id) ON DELETE CASCADE NOT NULL,
    endpoint TEXT NOT NULL,
    method TEXT NOT NULL,
    status_code INTEGER,
    response_time_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_api_key_usage_api_key_id ON auth.api_key_usage(api_key_id);
CREATE INDEX IF NOT EXISTS idx_auth_api_key_usage_created_at ON auth.api_key_usage(created_at DESC);

-- Service keys table (for service role authentication)
CREATE TABLE IF NOT EXISTS auth.service_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    key_hash TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    scopes TEXT[] DEFAULT ARRAY['*']::TEXT[],
    enabled BOOLEAN DEFAULT true,
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    UNIQUE(key_prefix)
);

CREATE INDEX IF NOT EXISTS idx_service_keys_prefix ON auth.service_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_service_keys_enabled ON auth.service_keys(enabled);

COMMENT ON TABLE auth.service_keys IS 'Service role keys with elevated privileges that bypass RLS. Use for backend services only.';
COMMENT ON COLUMN auth.service_keys.key_hash IS 'Bcrypt hash of the full service key. Never store keys in plaintext.';
COMMENT ON COLUMN auth.service_keys.key_prefix IS 'First 16 characters of the key for identification in logs (e.g., "sk_test_Ab3xY...").';
COMMENT ON COLUMN auth.service_keys.scopes IS 'Optional array of scope restrictions. Defaults to [''*''] for full service role access.';

-- OAuth user linking table
CREATE TABLE IF NOT EXISTS auth.oauth_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, provider_user_id),
    CONSTRAINT fk_oauth_links_user FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_oauth_links_user ON auth.oauth_links(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_links_provider ON auth.oauth_links(provider, provider_user_id);

-- OAuth tokens storage
CREATE TABLE IF NOT EXISTS auth.oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    token_expiry TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, provider),
    CONSTRAINT fk_oauth_tokens_user FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_oauth_tokens_user ON auth.oauth_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_tokens_provider ON auth.oauth_tokens(user_id, provider);

-- 2FA setup tracking table
CREATE TABLE IF NOT EXISTS auth.two_factor_setups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    factor_id UUID NOT NULL DEFAULT gen_random_uuid(),
    secret VARCHAR(32) NOT NULL,
    qr_code_url TEXT, -- Legacy field, kept for backward compatibility
    qr_code_data_uri TEXT, -- QR code as base64 data URI (data:image/png;base64,...)
    otpauth_uri TEXT, -- TOTP otpauth:// URI for manual entry or app deeplinks
    verified BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP + INTERVAL '10 minutes',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_2fa_setup_user FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE,
    CONSTRAINT two_factor_setups_user_id_key UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_2fa_setup_user ON auth.two_factor_setups(user_id);
CREATE INDEX IF NOT EXISTS idx_2fa_setup_expires ON auth.two_factor_setups(expires_at);

COMMENT ON TABLE auth.two_factor_setups IS 'Temporary storage for 2FA setup process. Entries expire after 10 minutes and should be cleaned up periodically.';
COMMENT ON COLUMN auth.two_factor_setups.factor_id IS 'Unique identifier for this 2FA factor';
COMMENT ON COLUMN auth.two_factor_setups.qr_code_data_uri IS 'QR code image as base64 data URI (data:image/png;base64,...)';
COMMENT ON COLUMN auth.two_factor_setups.otpauth_uri IS 'TOTP otpauth:// URI for manual entry or app deeplinks';

-- 2FA recovery/backup code usage tracking table
CREATE TABLE IF NOT EXISTS auth.two_factor_recovery_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    code_used VARCHAR(255),
    success BOOLEAN NOT NULL,
    ip_address INET,
    user_agent TEXT,
    attempted_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_2fa_recovery_user FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_2fa_recovery_user ON auth.two_factor_recovery_attempts(user_id);
CREATE INDEX IF NOT EXISTS idx_2fa_recovery_time ON auth.two_factor_recovery_attempts(attempted_at);

COMMENT ON TABLE auth.two_factor_recovery_attempts IS 'Audit log for 2FA recovery/backup code usage attempts for security monitoring.';

-- Webhooks table
CREATE TABLE IF NOT EXISTS auth.webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    url TEXT NOT NULL,
    events JSONB DEFAULT '[]'::JSONB,
    secret TEXT,
    enabled BOOLEAN DEFAULT true,
    headers JSONB DEFAULT '{}'::JSONB,
    timeout_seconds INTEGER DEFAULT 30,
    max_retries INTEGER DEFAULT 3,
    retry_backoff_seconds INTEGER DEFAULT 5,
    scope TEXT DEFAULT 'user' CHECK (scope IN ('user', 'global')),
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_webhooks_enabled ON auth.webhooks(enabled);

COMMENT ON COLUMN auth.webhooks.scope IS 'Scope determines which events trigger the webhook: user = only events on records owned by created_by, global = all events (admin only)';

-- Webhook deliveries table
CREATE TABLE IF NOT EXISTS auth.webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID REFERENCES auth.webhooks(id) ON DELETE CASCADE NOT NULL,
    event TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL,
    status_code INTEGER,
    response_body TEXT,
    error TEXT,
    attempt INTEGER DEFAULT 1,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_webhook_deliveries_webhook_id ON auth.webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_auth_webhook_deliveries_created_at ON auth.webhook_deliveries(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_auth_webhook_deliveries_status ON auth.webhook_deliveries(status);

-- Webhook event queue table
CREATE TABLE IF NOT EXISTS auth.webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID REFERENCES auth.webhooks(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    table_schema VARCHAR(255) NOT NULL,
    table_name VARCHAR(255) NOT NULL,
    record_id TEXT,
    old_data JSONB,
    new_data JSONB,
    processed BOOLEAN DEFAULT FALSE,
    attempts INT DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    next_retry_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_webhook_event_webhook FOREIGN KEY (webhook_id) REFERENCES auth.webhooks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_webhook_events_unprocessed ON auth.webhook_events(processed, next_retry_at) WHERE processed = FALSE;
CREATE INDEX IF NOT EXISTS idx_webhook_events_webhook ON auth.webhook_events(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_events_created ON auth.webhook_events(created_at);

COMMENT ON TABLE auth.webhook_events IS 'Queue for webhook events to be delivered. Processed events are kept for history.';

-- Impersonation sessions table
-- Note: admin_user_id has no FK because it can reference either dashboard.users OR auth.users
-- Note: target_user_id is nullable for anon/service role impersonation (no actual target user)
CREATE TABLE IF NOT EXISTS auth.impersonation_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_user_id UUID NOT NULL,
    target_user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    target_role TEXT,
    impersonation_type TEXT NOT NULL DEFAULT 'full',
    reason TEXT,
    ip_address TEXT,
    user_agent TEXT,
    started_at TIMESTAMPTZ DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_auth_impersonation_admin_user_id ON auth.impersonation_sessions(admin_user_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_target_user_id ON auth.impersonation_sessions(target_user_id);
CREATE INDEX IF NOT EXISTS idx_auth_impersonation_is_active ON auth.impersonation_sessions(is_active);

-- RLS audit log table for tracking access control violations and security events
CREATE TABLE IF NOT EXISTS auth.rls_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- User context
    user_id UUID, -- NULL for anonymous requests
    role TEXT NOT NULL, -- anon, authenticated, admin, service_role, etc.

    -- Operation details
    operation TEXT NOT NULL, -- SELECT, INSERT, UPDATE, DELETE
    table_schema TEXT NOT NULL,
    table_name TEXT NOT NULL,

    -- RLS evaluation result
    allowed BOOLEAN NOT NULL DEFAULT false, -- true if access granted, false if denied
    row_count INTEGER DEFAULT 0, -- number of rows affected/returned

    -- Request context
    ip_address INET,
    user_agent TEXT,
    request_id TEXT, -- for correlating with HTTP request logs

    -- Performance metrics
    execution_time_ms INTEGER,

    -- Additional metadata (flexible JSONB for extensibility)
    details JSONB DEFAULT '{}'::jsonb,

    -- Timestamp
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_rls_audit_user_id ON auth.rls_audit_log(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rls_audit_created_at ON auth.rls_audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_rls_audit_table ON auth.rls_audit_log(table_schema, table_name);
CREATE INDEX IF NOT EXISTS idx_rls_audit_allowed ON auth.rls_audit_log(allowed) WHERE allowed = false; -- Focus on violations
CREATE INDEX IF NOT EXISTS idx_rls_audit_role ON auth.rls_audit_log(role);
CREATE INDEX IF NOT EXISTS idx_rls_audit_operation ON auth.rls_audit_log(operation);
CREATE INDEX IF NOT EXISTS idx_rls_audit_request_id ON auth.rls_audit_log(request_id) WHERE request_id IS NOT NULL;

-- Add comments for documentation
COMMENT ON TABLE auth.rls_audit_log IS 'Audit log for Row Level Security policy evaluations, primarily tracking access denials and violations for security monitoring and compliance';
COMMENT ON COLUMN auth.rls_audit_log.allowed IS 'false indicates RLS policy blocked the operation (violation), true indicates policy allowed it';
COMMENT ON COLUMN auth.rls_audit_log.details IS 'Flexible JSONB field for storing additional context like error messages, query hints, or policy names';

-- Webhook monitored tables (reference counting for trigger management)
CREATE TABLE IF NOT EXISTS auth.webhook_monitored_tables (
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    webhook_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (schema_name, table_name)
);

COMMENT ON TABLE auth.webhook_monitored_tables IS 'Tracks which tables have webhook triggers installed and how many webhooks monitor each table';

-- Enable RLS on webhook_monitored_tables
ALTER TABLE auth.webhook_monitored_tables ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.webhook_monitored_tables FORCE ROW LEVEL SECURITY;

-- Only service role can manage monitored tables (internal use only)
CREATE POLICY webhook_monitored_tables_service_only ON auth.webhook_monitored_tables
    FOR ALL
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');
