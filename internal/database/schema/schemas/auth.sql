--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 18.3
-- Dumped by pgschema version 1.7.4

SET search_path TO auth, public;


--
-- Name: captcha_challenges; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS captcha_challenges (
    id uuid DEFAULT gen_random_uuid(),
    challenge_id text NOT NULL,
    endpoint text NOT NULL,
    email text,
    ip_address inet NOT NULL,
    device_fingerprint text,
    user_agent text,
    trust_score integer NOT NULL,
    captcha_required boolean NOT NULL,
    reason text NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    captcha_verified boolean DEFAULT false,
    CONSTRAINT captcha_challenges_pkey PRIMARY KEY (id),
    CONSTRAINT captcha_challenges_challenge_id_key UNIQUE (challenge_id),
    CONSTRAINT captcha_challenges_valid_expiry CHECK (expires_at > created_at)
);


COMMENT ON TABLE captcha_challenges IS 'Pre-flight CAPTCHA challenges linking check requests to auth submissions';

--
-- Name: idx_captcha_challenges_challenge_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_captcha_challenges_challenge_id ON captcha_challenges (challenge_id);

--
-- Name: idx_captcha_challenges_expires; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_captcha_challenges_expires ON captcha_challenges (expires_at);

--
-- Name: idx_captcha_challenges_ip_created; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_captcha_challenges_ip_created ON captcha_challenges (ip_address, created_at);

--
-- Name: captcha_challenges; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE captcha_challenges ENABLE ROW LEVEL SECURITY;

--
-- Name: captcha_challenges; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE captcha_challenges FORCE ROW LEVEL SECURITY;

--
-- Name: captcha_trust_tokens; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS captcha_trust_tokens (
    id uuid DEFAULT gen_random_uuid(),
    token_hash text NOT NULL,
    ip_address inet NOT NULL,
    device_fingerprint text,
    user_agent text,
    created_at timestamptz DEFAULT now() NOT NULL,
    expires_at timestamptz NOT NULL,
    used_count integer DEFAULT 0,
    last_used_at timestamptz,
    CONSTRAINT captcha_trust_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT captcha_trust_tokens_token_hash_key UNIQUE (token_hash),
    CONSTRAINT captcha_trust_tokens_valid_expiry CHECK (expires_at > created_at)
);


COMMENT ON TABLE captcha_trust_tokens IS 'Short-lived tokens that allow skipping CAPTCHA after successful verification';

--
-- Name: idx_captcha_trust_tokens_expires; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_captcha_trust_tokens_expires ON captcha_trust_tokens (expires_at);

--
-- Name: idx_captcha_trust_tokens_hash; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_captcha_trust_tokens_hash ON captcha_trust_tokens (token_hash);

--
-- Name: captcha_trust_tokens; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE captcha_trust_tokens ENABLE ROW LEVEL SECURITY;

--
-- Name: captcha_trust_tokens; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE captcha_trust_tokens FORCE ROW LEVEL SECURITY;

--
-- Name: emergency_revocation; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS emergency_revocation (
    id BIGSERIAL,
    revoked_at timestamptz DEFAULT now() NOT NULL,
    revoked_by text NOT NULL,
    reason text,
    revokes_all boolean DEFAULT false NOT NULL,
    revoked_jti text,
    expires_at timestamptz DEFAULT (now() + '7 days'::interval) NOT NULL,
    CONSTRAINT emergency_revocation_pkey PRIMARY KEY (id),
    CONSTRAINT emergency_revocation_revoked_jti_key UNIQUE (revoked_jti)
);


COMMENT ON TABLE emergency_revocation IS 'Emergency revocation table for service_role tokens. Allows immediate revocation of compromised service keys without waiting for expiry.';


COMMENT ON COLUMN auth.emergency_revocation.revokes_all IS 'When true, all service_role tokens are considered revoked. Used for security incidents requiring immediate global revocation.';


COMMENT ON COLUMN auth.emergency_revocation.revoked_jti IS 'Specific JWT ID to revoke. Only used when revokes_all is false.';


COMMENT ON COLUMN auth.emergency_revocation.expires_at IS 'Records auto-expire after 7 days for cleanup. Active revocations have expires_at > NOW().';

--
-- Name: idx_emergency_revocation_active; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_emergency_revocation_active ON emergency_revocation (expires_at);

--
-- Name: idx_emergency_revocation_all; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_emergency_revocation_all ON emergency_revocation (revokes_all, expires_at);

--
-- Name: idx_emergency_revocation_jti; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_emergency_revocation_jti ON emergency_revocation (revoked_jti) WHERE (revoked_jti IS NOT NULL);

--
-- Name: emergency_revocation; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE emergency_revocation ENABLE ROW LEVEL SECURITY;

--
-- Name: emergency_revocation; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE emergency_revocation FORCE ROW LEVEL SECURITY;

--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS users (
    id uuid DEFAULT gen_random_uuid(),
    email text NOT NULL,
    password_hash text,
    email_verified boolean DEFAULT false,
    role text DEFAULT 'authenticated',
    user_metadata jsonb DEFAULT '{}',
    app_metadata jsonb DEFAULT '{}',
    totp_secret varchar(255),
    totp_enabled boolean DEFAULT false,
    backup_codes text[],
    failed_login_attempts integer DEFAULT 0,
    is_locked boolean DEFAULT false,
    locked_until timestamptz,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);


COMMENT ON COLUMN auth.users.user_metadata IS 'User-editable metadata. Users can update this field themselves. Included in JWT claims.';


COMMENT ON COLUMN auth.users.app_metadata IS 'Application/admin-only metadata. Can only be updated by admins or service role. Included in JWT claims.';


COMMENT ON COLUMN auth.users.failed_login_attempts IS 'Number of consecutive failed login attempts';


COMMENT ON COLUMN auth.users.is_locked IS 'Whether the account is locked due to too many failed attempts';


COMMENT ON COLUMN auth.users.locked_until IS 'When the account lock expires (null = permanent until admin unlocks)';

--
-- Name: auth_users_email_tenant_null_unique; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS auth_users_email_tenant_null_unique ON users (email) WHERE (tenant_id IS NULL);

--
-- Name: auth_users_email_tenant_unique; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS auth_users_email_tenant_unique ON users (tenant_id, email) WHERE (tenant_id IS NOT NULL);

--
-- Name: idx_auth_users_app_metadata; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_users_app_metadata ON users USING gin (app_metadata);

--
-- Name: idx_auth_users_email; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_users_email ON users (email);

--
-- Name: idx_auth_users_is_locked; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_users_is_locked ON users (is_locked) WHERE (is_locked = true);

--
-- Name: idx_auth_users_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_users_tenant_id ON users (tenant_id);

--
-- Name: idx_auth_users_totp_enabled; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_users_totp_enabled ON users (totp_enabled) WHERE (totp_enabled = true);

--
-- Name: idx_auth_users_user_metadata; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_users_user_metadata ON users USING gin (user_metadata);

--
-- Name: users; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

--
-- Name: users; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE users FORCE ROW LEVEL SECURITY;

--
-- Name: magic_links; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS magic_links (
    id uuid DEFAULT gen_random_uuid(),
    email text NOT NULL,
    token_hash text NOT NULL,
    expires_at timestamptz NOT NULL,
    used boolean DEFAULT false,
    used_at timestamptz,
    ip_address text,
    user_agent text,
    created_at timestamptz DEFAULT now(),
    user_id uuid,
    tenant_id uuid,
    CONSTRAINT magic_links_pkey PRIMARY KEY (id),
    CONSTRAINT magic_links_token_key UNIQUE (token_hash),
    CONSTRAINT magic_links_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);


COMMENT ON COLUMN auth.magic_links.token_hash IS 'SHA-256 hash of the magic link token (base64 encoded). Plaintext token is never stored.';

--
-- Name: idx_auth_magic_links_email; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_magic_links_email ON magic_links (email);

--
-- Name: idx_auth_magic_links_token_hash; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_magic_links_token_hash ON magic_links (token_hash);

--
-- Name: idx_magic_links_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_magic_links_user_id ON magic_links (user_id) WHERE (user_id IS NOT NULL);

--
-- Name: idx_magic_links_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_magic_links_tenant_id ON magic_links (tenant_id);

--
-- Name: magic_links; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE magic_links ENABLE ROW LEVEL SECURITY;

--
-- Name: magic_links; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE magic_links FORCE ROW LEVEL SECURITY;

--
-- Name: oauth_states; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS oauth_states (
    state text,
    provider text NOT NULL,
    redirect_uri text,
    code_verifier text,
    nonce text,
    created_at timestamptz DEFAULT now() NOT NULL,
    expires_at timestamptz DEFAULT (now() + '00:10:00'::interval) NOT NULL,
    CONSTRAINT oauth_states_pkey PRIMARY KEY (state)
);


COMMENT ON TABLE oauth_states IS 'OAuth state tokens for CSRF protection in multi-instance deployments';


COMMENT ON COLUMN auth.oauth_states.state IS 'Random state token for CSRF protection';


COMMENT ON COLUMN auth.oauth_states.provider IS 'OAuth provider name';


COMMENT ON COLUMN auth.oauth_states.redirect_uri IS 'Custom redirect URI for this flow';


COMMENT ON COLUMN auth.oauth_states.code_verifier IS 'PKCE code verifier for enhanced security';

--
-- Name: idx_oauth_states_expires_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_oauth_states_expires_at ON oauth_states (expires_at);

--
-- Name: idx_oauth_states_provider; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_oauth_states_provider ON oauth_states (provider);

--
-- Name: oauth_states; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE oauth_states ENABLE ROW LEVEL SECURITY;

--
-- Name: oauth_states; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE oauth_states FORCE ROW LEVEL SECURITY;

--
-- Name: otp_codes; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS otp_codes (
    id uuid DEFAULT gen_random_uuid(),
    email text,
    phone text,
    code varchar(10) NOT NULL,
    code_hash text,
    type text NOT NULL,
    purpose text NOT NULL,
    expires_at timestamptz NOT NULL,
    used boolean DEFAULT false,
    used_at timestamptz,
    attempts integer DEFAULT 0,
    max_attempts integer DEFAULT 3,
    ip_address text,
    user_agent text,
    created_at timestamptz DEFAULT now(),
    user_id uuid,
    tenant_id uuid,
    CONSTRAINT otp_codes_pkey PRIMARY KEY (id),
    CONSTRAINT otp_codes_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);


COMMENT ON TABLE otp_codes IS 'One-time password codes for email/SMS passwordless authentication. Entries expire after configured period and should be cleaned up periodically.';


COMMENT ON COLUMN auth.otp_codes.type IS 'Type of OTP: email, sms';


COMMENT ON COLUMN auth.otp_codes.purpose IS 'Purpose: signin, signup, recovery, email_change, phone_change';


COMMENT ON COLUMN auth.otp_codes.attempts IS 'Number of failed verification attempts. Locked after max_attempts.';

--
-- Name: idx_auth_otp_codes_code; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_otp_codes_code_hash ON otp_codes (code_hash) WHERE code_hash IS NOT NULL;

--
-- Name: idx_auth_otp_codes_email; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_otp_codes_email ON otp_codes (email) WHERE (email IS NOT NULL);

--
-- Name: idx_auth_otp_codes_expires_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_otp_codes_expires_at ON otp_codes (expires_at);

--
-- Name: idx_auth_otp_codes_phone; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_otp_codes_phone ON otp_codes (phone) WHERE (phone IS NOT NULL);

--
-- Name: idx_auth_otp_codes_type; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_otp_codes_type ON otp_codes (type);

--
-- Name: idx_otp_codes_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_otp_codes_user_id ON otp_codes (user_id) WHERE (user_id IS NOT NULL);

--
-- Name: idx_otp_codes_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_otp_codes_tenant_id ON otp_codes (tenant_id);

--
-- Name: otp_codes; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE otp_codes ENABLE ROW LEVEL SECURITY;

--
-- Name: otp_codes; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE otp_codes FORCE ROW LEVEL SECURITY;

--
-- Name: rls_audit_log; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS rls_audit_log (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid,
    role text NOT NULL,
    operation text NOT NULL,
    table_schema text NOT NULL,
    table_name text NOT NULL,
    allowed boolean DEFAULT false NOT NULL,
    row_count integer DEFAULT 0,
    ip_address inet,
    user_agent text,
    request_id text,
    execution_time_ms integer,
    details jsonb DEFAULT '{}',
    created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT rls_audit_log_pkey PRIMARY KEY (id)
);


COMMENT ON TABLE rls_audit_log IS 'Audit log for Row Level Security policy evaluations, primarily tracking access denials and violations for security monitoring and compliance';


COMMENT ON COLUMN auth.rls_audit_log.allowed IS 'false indicates RLS policy blocked the operation (violation), true indicates policy allowed it';


COMMENT ON COLUMN auth.rls_audit_log.details IS 'Flexible JSONB field for storing additional context like error messages, query hints, or policy names';

--
-- Name: idx_rls_audit_allowed; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_rls_audit_allowed ON rls_audit_log (allowed) WHERE (allowed = false);

--
-- Name: idx_rls_audit_created_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_rls_audit_created_at ON rls_audit_log (created_at DESC);

--
-- Name: idx_rls_audit_operation; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_rls_audit_operation ON rls_audit_log (operation);

--
-- Name: idx_rls_audit_request_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_rls_audit_request_id ON rls_audit_log (request_id) WHERE (request_id IS NOT NULL);

--
-- Name: idx_rls_audit_role; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_rls_audit_role ON rls_audit_log (role);

--
-- Name: idx_rls_audit_table; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_rls_audit_table ON rls_audit_log (table_schema, table_name);

--
-- Name: idx_rls_audit_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_rls_audit_user_id ON rls_audit_log (user_id) WHERE (user_id IS NOT NULL);

--
-- Name: rls_audit_log; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE rls_audit_log ENABLE ROW LEVEL SECURITY;

--
-- Name: rls_audit_log; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE rls_audit_log FORCE ROW LEVEL SECURITY;

--
-- Name: saml_assertion_ids; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS saml_assertion_ids (
    assertion_id text,
    expires_at timestamptz NOT NULL,
    created_at timestamptz DEFAULT now(),
    CONSTRAINT saml_assertion_ids_pkey PRIMARY KEY (assertion_id)
);


COMMENT ON TABLE saml_assertion_ids IS 'SAML assertion IDs for replay attack prevention';

--
-- Name: idx_saml_assertion_ids_expires; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_saml_assertion_ids_expires ON saml_assertion_ids (expires_at);

--
-- Name: saml_assertion_ids; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE saml_assertion_ids ENABLE ROW LEVEL SECURITY;

--
-- Name: saml_assertion_ids; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE saml_assertion_ids FORCE ROW LEVEL SECURITY;

--
-- Name: saml_providers; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS saml_providers (
    id uuid DEFAULT gen_random_uuid(),
    name text NOT NULL,
    enabled boolean DEFAULT true,
    idp_metadata_url text,
    idp_metadata_xml text,
    idp_metadata_cached text,
    idp_metadata_cached_at timestamptz,
    entity_id text NOT NULL,
    acs_url text NOT NULL,
    certificate text,
    private_key text,
    attribute_mapping jsonb DEFAULT '{"name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name", "email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"}',
    auto_create_users boolean DEFAULT true,
    default_role text DEFAULT 'authenticated',
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    allow_dashboard_login boolean DEFAULT false,
    allow_app_login boolean DEFAULT true,
    allow_idp_initiated boolean DEFAULT false,
    allowed_redirect_hosts text[] DEFAULT ARRAY[]::text[],
    source text DEFAULT 'database',
    display_name text,
    sp_certificate text,
    sp_private_key_encrypted bytea,
    idp_slo_url text,
    required_groups text[],
    required_groups_all text[],
    denied_groups text[],
    group_attribute text DEFAULT 'groups',
    tenant_id uuid,
    CONSTRAINT saml_providers_pkey PRIMARY KEY (id),
    CONSTRAINT saml_providers_name_key UNIQUE (name)
);


COMMENT ON TABLE saml_providers IS 'SAML 2.0 Identity Provider configurations for enterprise SSO';


COMMENT ON COLUMN auth.saml_providers.allow_dashboard_login IS 'Allow this provider for dashboard admin SSO login';


COMMENT ON COLUMN auth.saml_providers.allow_app_login IS 'Allow this provider for application user authentication';


COMMENT ON COLUMN auth.saml_providers.allow_idp_initiated IS 'Allow IdP-initiated SSO (less secure)';


COMMENT ON COLUMN auth.saml_providers.allowed_redirect_hosts IS 'Whitelist of allowed hosts for RelayState redirects';


COMMENT ON COLUMN auth.saml_providers.source IS 'Provider source: database (UI-managed) or config (YAML file)';


COMMENT ON COLUMN auth.saml_providers.display_name IS 'Human-friendly display name for the provider';


COMMENT ON COLUMN auth.saml_providers.sp_certificate IS 'PEM-encoded X.509 certificate for signing SAML messages (SLO)';


COMMENT ON COLUMN auth.saml_providers.sp_private_key_encrypted IS 'Encrypted PEM-encoded private key for signing SAML messages (SLO)';


COMMENT ON COLUMN auth.saml_providers.idp_slo_url IS 'IdP Single Logout URL extracted from metadata';


COMMENT ON COLUMN auth.saml_providers.required_groups IS 'User must be member of at least ONE of these groups (OR logic)';


COMMENT ON COLUMN auth.saml_providers.required_groups_all IS 'User must be member of ALL of these groups (AND logic)';


COMMENT ON COLUMN auth.saml_providers.denied_groups IS 'Reject users who are members of any of these groups';


COMMENT ON COLUMN auth.saml_providers.group_attribute IS 'SAML attribute name containing group memberships (default: groups)';

--
-- Name: idx_saml_providers_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_saml_providers_tenant_id ON saml_providers (tenant_id);

--
-- Name: saml_providers; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE saml_providers ENABLE ROW LEVEL SECURITY;

--
-- Name: saml_providers; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE saml_providers FORCE ROW LEVEL SECURITY;

--
-- Name: client_keys; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS client_keys (
    id uuid DEFAULT gen_random_uuid(),
    name text NOT NULL,
    description text,
    key_hash text NOT NULL,
    key_prefix text NOT NULL,
    user_id uuid,
    scopes text[] DEFAULT ARRAY[]::text[],
    rate_limit_per_minute integer DEFAULT 60,
    last_used_at timestamptz,
    expires_at timestamptz,
    revoked boolean DEFAULT false,
    revoked_at timestamptz,
    revoked_by uuid,
    tenant_id uuid,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    allowed_namespaces text[],
    CONSTRAINT api_keys_pkey PRIMARY KEY (id),
    CONSTRAINT api_keys_key_hash_key UNIQUE (key_hash),
    CONSTRAINT api_keys_revoked_by_fkey FOREIGN KEY (revoked_by) REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT api_keys_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

COMMENT ON COLUMN auth.client_keys.allowed_namespaces IS 'Allowed namespaces for this key. NULL = all namespaces (no restrictions), empty array = default namespace only, populated array = specific namespaces allowed.';

--
-- Name: idx_api_keys_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON client_keys (user_id);

--
-- Name: idx_auth_api_keys_key_hash; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_api_keys_key_hash ON client_keys (key_hash);

--
-- Name: idx_auth_api_keys_key_prefix; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_api_keys_key_prefix ON client_keys (key_prefix);

--
-- Name: idx_auth_api_keys_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_api_keys_user_id ON client_keys (user_id);

--
-- Name: idx_auth_client_keys_key_hash; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_client_keys_key_hash ON client_keys (key_hash);

--
-- Name: idx_auth_client_keys_key_prefix; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_client_keys_key_prefix ON client_keys (key_prefix);

--
-- Name: idx_auth_client_keys_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_client_keys_user_id ON client_keys (user_id);

--
-- Name: client_keys; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE client_keys ENABLE ROW LEVEL SECURITY;

--
-- Name: client_keys; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE client_keys FORCE ROW LEVEL SECURITY;

--
-- Name: client_key_usage; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS client_key_usage (
    id uuid DEFAULT gen_random_uuid(),
    client_key_id uuid NOT NULL,
    endpoint text NOT NULL,
    method text NOT NULL,
    status_code integer,
    response_time_ms integer,
    created_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT api_key_usage_pkey PRIMARY KEY (id),
    CONSTRAINT api_key_usage_api_key_id_fkey FOREIGN KEY (client_key_id) REFERENCES client_keys (id) ON DELETE CASCADE
);

--
-- Name: idx_api_key_usage_api_key_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_api_key_usage_api_key_id ON client_key_usage (client_key_id);

--
-- Name: idx_auth_api_key_usage_api_key_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_api_key_usage_api_key_id ON client_key_usage (client_key_id);

--
-- Name: idx_auth_api_key_usage_created_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_api_key_usage_created_at ON client_key_usage (created_at DESC);

--
-- Name: idx_auth_client_key_usage_client_key_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_client_key_usage_client_key_id ON client_key_usage (client_key_id);

--
-- Name: idx_auth_client_key_usage_created_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_client_key_usage_created_at ON client_key_usage (created_at DESC);

--
-- Name: idx_client_key_usage_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_client_key_usage_tenant_id ON client_key_usage (tenant_id);

--
-- Name: client_key_usage; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE client_key_usage ENABLE ROW LEVEL SECURITY;

--
-- Name: client_key_usage; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE client_key_usage FORCE ROW LEVEL SECURITY;

--
-- Name: email_verification_tokens; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    token_hash text NOT NULL,
    expires_at timestamptz NOT NULL,
    used boolean DEFAULT false,
    used_at timestamptz,
    created_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT email_verification_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT email_verification_tokens_token_hash_key UNIQUE (token_hash),
    CONSTRAINT email_verification_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

--
-- Name: idx_auth_email_verification_tokens_hash; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_email_verification_tokens_hash ON email_verification_tokens (token_hash);

--
-- Name: idx_auth_email_verification_tokens_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_email_verification_tokens_user_id ON email_verification_tokens (user_id);

--
-- Name: idx_email_verification_tokens_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_tenant_id ON email_verification_tokens (tenant_id);

--
-- Name: email_verification_tokens; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE email_verification_tokens ENABLE ROW LEVEL SECURITY;

--
-- Name: email_verification_tokens; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE email_verification_tokens FORCE ROW LEVEL SECURITY;

--
-- Name: impersonation_sessions; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS impersonation_sessions (
    id uuid DEFAULT gen_random_uuid(),
    admin_user_id uuid NOT NULL,
    target_user_id uuid,
    target_role text,
    impersonation_type text DEFAULT 'full' NOT NULL,
    reason text,
    ip_address text,
    user_agent text,
    started_at timestamptz DEFAULT now(),
    ended_at timestamptz,
    is_active boolean DEFAULT true,
    access_token_jti text,
    refresh_token_jti text,
    tenant_id uuid,
    CONSTRAINT impersonation_sessions_pkey PRIMARY KEY (id),
    CONSTRAINT impersonation_sessions_target_user_id_fkey FOREIGN KEY (target_user_id) REFERENCES users (id) ON DELETE CASCADE
);

--
-- Name: idx_auth_impersonation_admin_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_impersonation_admin_user_id ON impersonation_sessions (admin_user_id);

--
-- Name: idx_auth_impersonation_is_active; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_impersonation_is_active ON impersonation_sessions (is_active);

--
-- Name: idx_impersonation_sessions_admin_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_admin_user_id ON impersonation_sessions (admin_user_id);

--
-- Name: idx_impersonation_sessions_target_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_target_user_id ON impersonation_sessions (target_user_id);

CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_tenant_id ON impersonation_sessions (tenant_id);

--
-- Name: impersonation_sessions; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE impersonation_sessions ENABLE ROW LEVEL SECURITY;

--
-- Name: impersonation_sessions; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE impersonation_sessions FORCE ROW LEVEL SECURITY;

--
-- Name: mcp_oauth_clients; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS mcp_oauth_clients (
    client_id text,
    client_name text NOT NULL,
    client_type text DEFAULT 'public' NOT NULL,
    client_secret_hash text,
    redirect_uris text[] NOT NULL,
    scopes text[] DEFAULT ARRAY['read:tables', 'read:schema'] NOT NULL,
    registered_by uuid,
    metadata jsonb DEFAULT '{}',
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    tenant_id uuid,
    CONSTRAINT mcp_oauth_clients_pkey PRIMARY KEY (client_id),
    CONSTRAINT mcp_oauth_clients_registered_by_fkey FOREIGN KEY (registered_by) REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT mcp_oauth_clients_client_type_check CHECK (client_type IN ('public'::text, 'confidential'::text))
);


COMMENT ON TABLE mcp_oauth_clients IS 'OAuth 2.1 clients for MCP authentication (Dynamic Client Registration)';

--
-- Name: idx_mcp_oauth_clients_registered_by; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_mcp_oauth_clients_registered_by ON mcp_oauth_clients (registered_by) WHERE (registered_by IS NOT NULL);

--
-- Name: idx_mcp_oauth_clients_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_mcp_oauth_clients_tenant_id ON mcp_oauth_clients (tenant_id);

--
-- Name: mcp_oauth_clients; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE mcp_oauth_clients ENABLE ROW LEVEL SECURITY;

--
-- Name: mcp_oauth_clients; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE mcp_oauth_clients FORCE ROW LEVEL SECURITY;

--
-- Name: mcp_oauth_codes; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS mcp_oauth_codes (
    code text,
    client_id text NOT NULL,
    user_id uuid,
    redirect_uri text NOT NULL,
    scopes text[] NOT NULL,
    code_challenge text,
    code_challenge_method text,
    state text,
    expires_at timestamptz DEFAULT (now() + '00:10:00'::interval) NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    tenant_id uuid,
    CONSTRAINT mcp_oauth_codes_pkey PRIMARY KEY (code),
    CONSTRAINT mcp_oauth_codes_client_id_fkey FOREIGN KEY (client_id) REFERENCES mcp_oauth_clients (client_id) ON DELETE CASCADE
);


COMMENT ON TABLE mcp_oauth_codes IS 'Short-lived authorization codes for OAuth 2.1 flows';


COMMENT ON COLUMN auth.mcp_oauth_codes.user_id IS 'Platform user who authorized this code (references platform.users)';

--
-- Name: idx_mcp_oauth_codes_client_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_mcp_oauth_codes_client_id ON mcp_oauth_codes (client_id);

--
-- Name: idx_mcp_oauth_codes_expires_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_mcp_oauth_codes_expires_at ON mcp_oauth_codes (expires_at);

--
-- Name: idx_mcp_oauth_codes_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_mcp_oauth_codes_tenant_id ON mcp_oauth_codes (tenant_id);

--
-- Name: mcp_oauth_codes; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE mcp_oauth_codes ENABLE ROW LEVEL SECURITY;

--
-- Name: mcp_oauth_codes; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE mcp_oauth_codes FORCE ROW LEVEL SECURITY;

--
-- Name: mcp_oauth_tokens; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS mcp_oauth_tokens (
    id uuid DEFAULT gen_random_uuid(),
    token_type text NOT NULL,
    token_hash text NOT NULL,
    client_id text NOT NULL,
    user_id uuid,
    scopes text[] NOT NULL,
    parent_token_id uuid,
    expires_at timestamptz NOT NULL,
    is_revoked boolean DEFAULT false NOT NULL,
    revoked_reason text,
    created_at timestamptz DEFAULT now() NOT NULL,
    revoked_at timestamptz,
    tenant_id uuid,
    CONSTRAINT mcp_oauth_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT mcp_oauth_tokens_token_hash_key UNIQUE (token_hash),
    CONSTRAINT mcp_oauth_tokens_client_id_fkey FOREIGN KEY (client_id) REFERENCES mcp_oauth_clients (client_id) ON DELETE CASCADE,
    CONSTRAINT mcp_oauth_tokens_parent_token_id_fkey FOREIGN KEY (parent_token_id) REFERENCES mcp_oauth_tokens (id) ON DELETE SET NULL,
    CONSTRAINT mcp_oauth_tokens_token_type_check CHECK (token_type IN ('access'::text, 'refresh'::text))
);


COMMENT ON TABLE mcp_oauth_tokens IS 'OAuth 2.1 access and refresh tokens for MCP clients';


COMMENT ON COLUMN auth.mcp_oauth_tokens.user_id IS 'Platform user this token represents (references platform.users)';

--
-- Name: idx_mcp_oauth_tokens_client_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_mcp_oauth_tokens_client_id ON mcp_oauth_tokens (client_id);

--
-- Name: idx_mcp_oauth_tokens_expires_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_mcp_oauth_tokens_expires_at ON mcp_oauth_tokens (expires_at) WHERE (NOT is_revoked);

--
-- Name: idx_mcp_oauth_tokens_token_hash; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_mcp_oauth_tokens_token_hash ON mcp_oauth_tokens (token_hash);

--
-- Name: idx_mcp_oauth_tokens_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_mcp_oauth_tokens_user_id ON mcp_oauth_tokens (user_id) WHERE (user_id IS NOT NULL);

--
-- Name: idx_mcp_oauth_tokens_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_mcp_oauth_tokens_tenant_id ON mcp_oauth_tokens (tenant_id);

--
-- Name: mcp_oauth_tokens; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE mcp_oauth_tokens ENABLE ROW LEVEL SECURITY;

--
-- Name: mcp_oauth_tokens; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE mcp_oauth_tokens FORCE ROW LEVEL SECURITY;

--
-- Name: mfa_factors; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS mfa_factors (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    friendly_name text,
    factor_type text NOT NULL,
    status text DEFAULT 'unverified' NOT NULL,
    secret text,
    phone text,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    tenant_id uuid,
    CONSTRAINT mfa_factors_pkey PRIMARY KEY (id),
    CONSTRAINT mfa_factors_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT mfa_factors_factor_type_check CHECK (factor_type IN ('totp'::text, 'phone'::text)),
    CONSTRAINT mfa_factors_status_check CHECK (status IN ('verified'::text, 'unverified'::text))
);


COMMENT ON TABLE mfa_factors IS 'Supabase-compatible MFA factors table for multi-factor authentication';


COMMENT ON COLUMN auth.mfa_factors.factor_type IS 'Type of MFA factor: totp (authenticator app) or phone (SMS)';


COMMENT ON COLUMN auth.mfa_factors.status IS 'Factor status: verified (active) or unverified (pending verification)';


COMMENT ON COLUMN auth.mfa_factors.secret IS 'TOTP secret for authenticator apps (encrypted at application level)';


COMMENT ON COLUMN auth.mfa_factors.phone IS 'Phone number for SMS-based 2FA';

--
-- Name: idx_auth_mfa_factors_factor_type; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_mfa_factors_factor_type ON mfa_factors (factor_type);

--
-- Name: idx_auth_mfa_factors_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_mfa_factors_user_id ON mfa_factors (user_id);

--
-- Name: idx_auth_mfa_factors_user_id_status; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_mfa_factors_user_id_status ON mfa_factors (user_id, status) WHERE (status = 'verified'::text);

--
-- Name: idx_mfa_factors_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_mfa_factors_tenant_id ON mfa_factors (tenant_id);

--
-- Name: mfa_factors; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE mfa_factors ENABLE ROW LEVEL SECURITY;

--
-- Name: mfa_factors; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE mfa_factors FORCE ROW LEVEL SECURITY;

--
-- Name: nonces; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS nonces (
    nonce text,
    user_id uuid NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT nonces_pkey PRIMARY KEY (nonce),
    CONSTRAINT nonces_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);


COMMENT ON TABLE nonces IS 'Single-use nonces for reauthentication flows. Enables stateless multi-instance deployments.';

--
-- Name: idx_auth_nonces_expires_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_nonces_expires_at ON nonces (expires_at);

--
-- Name: idx_auth_nonces_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_nonces_user_id ON nonces (user_id);

--
-- Name: nonces; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE nonces ENABLE ROW LEVEL SECURITY;

--
-- Name: nonces; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE nonces FORCE ROW LEVEL SECURITY;

--
-- Name: oauth_links; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS oauth_links (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    provider varchar(50) NOT NULL,
    provider_user_id varchar(255) NOT NULL,
    email varchar(255),
    metadata jsonb,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid,
    CONSTRAINT oauth_links_pkey PRIMARY KEY (id),
    CONSTRAINT oauth_links_provider_provider_user_id_key UNIQUE (provider, provider_user_id),
    CONSTRAINT fk_oauth_links_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT oauth_links_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

--
-- Name: idx_oauth_links_provider; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_oauth_links_provider ON oauth_links (provider, provider_user_id);

--
-- Name: idx_oauth_links_user; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_oauth_links_user ON oauth_links (user_id);

--
-- Name: idx_oauth_links_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_oauth_links_tenant_id ON oauth_links (tenant_id);

--
-- Name: oauth_links; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE oauth_links ENABLE ROW LEVEL SECURITY;

--
-- Name: oauth_links; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE oauth_links FORCE ROW LEVEL SECURITY;

--
-- Name: oauth_logout_states; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS oauth_logout_states (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    provider text NOT NULL,
    state text NOT NULL,
    post_logout_redirect_uri text,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    expires_at timestamptz DEFAULT (CURRENT_TIMESTAMP + '00:10:00'::interval) NOT NULL,
    tenant_id uuid,
    CONSTRAINT oauth_logout_states_pkey PRIMARY KEY (id),
    CONSTRAINT oauth_logout_states_state_key UNIQUE (state),
    CONSTRAINT oauth_logout_states_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);


COMMENT ON TABLE oauth_logout_states IS 'Temporary storage for OAuth logout states to track SP-initiated logout flow and provide CSRF protection';

--
-- Name: idx_oauth_logout_states_expires_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_oauth_logout_states_expires_at ON oauth_logout_states (expires_at);

--
-- Name: idx_oauth_logout_states_state; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_oauth_logout_states_state ON oauth_logout_states (state);

--
-- Name: idx_oauth_logout_states_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_oauth_logout_states_tenant_id ON oauth_logout_states (tenant_id);

--
-- Name: oauth_logout_states; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE oauth_logout_states ENABLE ROW LEVEL SECURITY;

--
-- Name: oauth_logout_states; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE oauth_logout_states FORCE ROW LEVEL SECURITY;

--
-- Name: oauth_tokens; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS oauth_tokens (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    provider varchar(50) NOT NULL,
    access_token text NOT NULL,
    refresh_token text,
    token_expiry timestamptz,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    id_token text,
    tenant_id uuid,
    CONSTRAINT oauth_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT oauth_tokens_user_id_provider_key UNIQUE (user_id, provider),
    CONSTRAINT fk_oauth_tokens_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT oauth_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);


COMMENT ON COLUMN auth.oauth_tokens.id_token IS 'OIDC ID token stored for use with end_session_endpoint id_token_hint parameter';

--
-- Name: idx_oauth_tokens_provider; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_oauth_tokens_provider ON oauth_tokens (user_id, provider);

--
-- Name: idx_oauth_tokens_user; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_oauth_tokens_user ON oauth_tokens (user_id);

--
-- Name: idx_oauth_tokens_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_oauth_tokens_tenant_id ON oauth_tokens (tenant_id);

--
-- Name: oauth_tokens; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE oauth_tokens ENABLE ROW LEVEL SECURITY;

--
-- Name: oauth_tokens; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE oauth_tokens FORCE ROW LEVEL SECURITY;

--
-- Name: password_reset_tokens; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    token_hash text NOT NULL,
    expires_at timestamptz NOT NULL,
    used boolean DEFAULT false,
    used_at timestamptz,
    created_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT password_reset_tokens_token_key UNIQUE (token_hash),
    CONSTRAINT password_reset_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);


COMMENT ON COLUMN auth.password_reset_tokens.token_hash IS 'SHA-256 hash of the password reset token (base64 encoded). Plaintext token is never stored.';

--
-- Name: idx_auth_password_reset_tokens_token_hash; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_password_reset_tokens_token_hash ON password_reset_tokens (token_hash);

--
-- Name: idx_auth_password_reset_tokens_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_password_reset_tokens_user_id ON password_reset_tokens (user_id);

--
-- Name: idx_password_reset_tokens_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_tenant_id ON password_reset_tokens (tenant_id);

--
-- Name: password_reset_tokens; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE password_reset_tokens ENABLE ROW LEVEL SECURITY;

--
-- Name: password_reset_tokens; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE password_reset_tokens FORCE ROW LEVEL SECURITY;

--
-- Name: saml_sessions; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS saml_sessions (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    provider_id uuid,
    provider_name text NOT NULL,
    name_id text NOT NULL,
    name_id_format text,
    session_index text,
    attributes jsonb,
    expires_at timestamptz,
    created_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT saml_sessions_pkey PRIMARY KEY (id),
    CONSTRAINT saml_sessions_provider_id_fkey FOREIGN KEY (provider_id) REFERENCES saml_providers (id) ON DELETE SET NULL,
    CONSTRAINT saml_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);


COMMENT ON TABLE saml_sessions IS 'Active SAML authentication sessions for Single Logout support';

--
-- Name: idx_saml_sessions_name_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_saml_sessions_name_id ON saml_sessions (name_id);

--
-- Name: idx_saml_sessions_provider_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_saml_sessions_provider_name ON saml_sessions (provider_name);

--
-- Name: idx_saml_sessions_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_saml_sessions_user_id ON saml_sessions (user_id);

--
-- Name: idx_saml_sessions_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_saml_sessions_tenant_id ON saml_sessions (tenant_id);

--
-- Name: saml_sessions; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE saml_sessions ENABLE ROW LEVEL SECURITY;

--
-- Name: saml_sessions; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE saml_sessions FORCE ROW LEVEL SECURITY;

--
-- Name: service_keys; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS service_keys (
    id uuid DEFAULT gen_random_uuid(),
    name text NOT NULL,
    description text,
    key_hash text NOT NULL,
    key_prefix text NOT NULL,
    scopes text[] DEFAULT ARRAY['*'],
    enabled boolean DEFAULT true,
    created_by uuid,
    created_at timestamptz DEFAULT now(),
    last_used_at timestamptz,
    expires_at timestamptz,
    rate_limit_per_minute integer,
    rate_limit_per_hour integer,
    allowed_namespaces text[],
    revoked_at timestamptz,
    revoked_by uuid,
    revocation_reason text,
    deprecated_at timestamptz,
    grace_period_ends_at timestamptz,
    replaced_by uuid,
    key_type text DEFAULT 'service' NOT NULL,
    tenant_id uuid,
    CONSTRAINT service_keys_pkey PRIMARY KEY (id),
    CONSTRAINT service_keys_key_prefix_key UNIQUE (key_prefix),
    CONSTRAINT service_keys_replaced_by_fkey FOREIGN KEY (replaced_by) REFERENCES service_keys (id) ON DELETE SET NULL,
    CONSTRAINT auth_service_keys_key_type_check CHECK (key_type IN ('anon'::text, 'service'::text, 'publishable'::text, 'tenant_service'::text, 'global_service'::text))
);


COMMENT ON TABLE service_keys IS 'Service role keys with elevated privileges that bypass RLS. Use for backend services only.';


COMMENT ON COLUMN auth.service_keys.key_hash IS 'Bcrypt hash of the full service key. Never store keys in plaintext.';


COMMENT ON COLUMN auth.service_keys.key_prefix IS 'First 16 characters of the key for identification in logs (e.g., "sk_test_Ab3xY...").';


COMMENT ON COLUMN auth.service_keys.scopes IS 'Optional array of scope restrictions. Defaults to [''*''] for full service role access.';


COMMENT ON COLUMN auth.service_keys.rate_limit_per_minute IS 'Maximum requests per minute. NULL means no limit (unlimited).';


COMMENT ON COLUMN auth.service_keys.rate_limit_per_hour IS 'Maximum requests per hour. NULL means no limit (unlimited).';


COMMENT ON COLUMN auth.service_keys.allowed_namespaces IS 'Allowed namespaces for this key. NULL = all namespaces (no restrictions), empty array = default namespace only, populated array = specific namespaces allowed.';


COMMENT ON COLUMN auth.service_keys.revoked_at IS 'When the key was emergency revoked (NULL if not revoked)';


COMMENT ON COLUMN auth.service_keys.revoked_by IS 'Admin who revoked the key';


COMMENT ON COLUMN auth.service_keys.revocation_reason IS 'Reason for emergency revocation';


COMMENT ON COLUMN auth.service_keys.deprecated_at IS 'When the key was marked for rotation';


COMMENT ON COLUMN auth.service_keys.grace_period_ends_at IS 'When the grace period for rotation ends';


COMMENT ON COLUMN auth.service_keys.replaced_by IS 'Reference to the replacement key (for rotation)';


COMMENT ON COLUMN auth.service_keys.key_type IS 'Type of key: anon (anonymous access) or service (elevated privileges bypassing RLS)';

--
-- Name: auth_service_keys_name_tenant_null_unique; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS auth_service_keys_name_tenant_null_unique ON service_keys (name) WHERE (tenant_id IS NULL);

--
-- Name: auth_service_keys_name_tenant_unique; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS auth_service_keys_name_tenant_unique ON service_keys (tenant_id, name) WHERE (tenant_id IS NOT NULL);

--
-- Name: idx_auth_service_keys_key_type; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_service_keys_key_type ON service_keys (key_type);

--
-- Name: idx_auth_service_keys_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_service_keys_tenant_id ON service_keys (tenant_id);

--
-- Name: idx_service_keys_enabled; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_service_keys_enabled ON service_keys (enabled);

--
-- Name: idx_service_keys_grace_period; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_service_keys_grace_period ON service_keys (grace_period_ends_at) WHERE (deprecated_at IS NOT NULL) AND (grace_period_ends_at IS NOT NULL);

--
-- Name: idx_service_keys_prefix; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_service_keys_prefix ON service_keys (key_prefix);

--
-- Name: idx_service_keys_rate_limits; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_service_keys_rate_limits ON service_keys (id) WHERE (rate_limit_per_minute IS NOT NULL) OR (rate_limit_per_hour IS NOT NULL);

--
-- Name: idx_service_keys_revoked_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_service_keys_revoked_at ON service_keys (revoked_at) WHERE (revoked_at IS NOT NULL);

--
-- Name: service_keys; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE service_keys ENABLE ROW LEVEL SECURITY;

--
-- Name: service_keys; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE service_keys FORCE ROW LEVEL SECURITY;

--
-- Name: service_key_revocations; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS service_key_revocations (
    id uuid DEFAULT gen_random_uuid(),
    key_id uuid NOT NULL,
    key_prefix text NOT NULL,
    revoked_by uuid,
    reason text NOT NULL,
    revocation_type text NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    tenant_id uuid,
    CONSTRAINT service_key_revocations_pkey PRIMARY KEY (id),
    CONSTRAINT service_key_revocations_key_id_fkey FOREIGN KEY (key_id) REFERENCES service_keys (id) ON DELETE CASCADE,
    CONSTRAINT service_key_revocations_revocation_type_check CHECK (revocation_type IN ('emergency'::text, 'rotation'::text, 'expiration'::text))
);


COMMENT ON TABLE service_key_revocations IS 'Audit log of service key revocations';

--
-- Name: idx_service_key_revocations_created_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_service_key_revocations_created_at ON service_key_revocations (created_at);

--
-- Name: idx_service_key_revocations_key_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_service_key_revocations_key_id ON service_key_revocations (key_id);

--
-- Name: idx_service_key_revocations_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_service_key_revocations_tenant_id ON service_key_revocations (tenant_id);

--
-- Name: sessions; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS sessions (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    access_token_hash text NOT NULL,
    refresh_token_hash text,
    tenant_id uuid,
    CONSTRAINT sessions_pkey PRIMARY KEY (id),
    CONSTRAINT auth_sessions_access_token_hash_unique UNIQUE (access_token_hash),
    CONSTRAINT sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);


COMMENT ON COLUMN auth.sessions.access_token_hash IS 'SHA-256 hash of access token (base64 encoded). Plaintext token is never stored.';


COMMENT ON COLUMN auth.sessions.refresh_token_hash IS 'SHA-256 hash of refresh token (base64 encoded). Plaintext token is never stored.';

--
-- Name: idx_auth_sessions_access_token_hash; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_sessions_access_token_hash ON sessions (access_token_hash);

--
-- Name: idx_auth_sessions_refresh_token_hash; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_sessions_refresh_token_hash ON sessions (refresh_token_hash);

--
-- Name: idx_auth_sessions_refresh_token_hash_unique; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_sessions_refresh_token_hash_unique ON sessions (refresh_token_hash) WHERE (refresh_token_hash IS NOT NULL);

--
-- Name: idx_auth_sessions_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON sessions (user_id);

--
-- Name: idx_sessions_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_sessions_tenant_id ON sessions (tenant_id);

--
-- Name: sessions; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;

--
-- Name: sessions; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE sessions FORCE ROW LEVEL SECURITY;

--
-- Name: token_blacklist; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS token_blacklist (
    id uuid DEFAULT gen_random_uuid(),
    token_jti text NOT NULL,
    token_type text DEFAULT 'access' NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_by uuid,
    reason text,
    created_at timestamptz DEFAULT now(),
    CONSTRAINT token_blacklist_pkey PRIMARY KEY (id),
    CONSTRAINT token_blacklist_token_jti_key UNIQUE (token_jti),
    CONSTRAINT token_blacklist_revoked_by_fkey FOREIGN KEY (revoked_by) REFERENCES users (id) ON DELETE SET NULL
);

--
-- Name: idx_auth_token_blacklist_expires_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_token_blacklist_expires_at ON token_blacklist (expires_at);

--
-- Name: idx_auth_token_blacklist_token_jti; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_token_blacklist_token_jti ON token_blacklist (token_jti);

--
-- Name: token_blacklist; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE token_blacklist ENABLE ROW LEVEL SECURITY;

--
-- Name: token_blacklist; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE token_blacklist FORCE ROW LEVEL SECURITY;

--
-- Name: two_factor_recovery_attempts; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS two_factor_recovery_attempts (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    code_used varchar(255),
    success boolean NOT NULL,
    ip_address inet,
    user_agent text,
    attempted_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid,
    CONSTRAINT two_factor_recovery_attempts_pkey PRIMARY KEY (id),
    CONSTRAINT fk_2fa_recovery_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT two_factor_recovery_attempts_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);


COMMENT ON TABLE two_factor_recovery_attempts IS 'Audit log for 2FA recovery/backup code usage attempts for security monitoring.';

--
-- Name: idx_2fa_recovery_time; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_2fa_recovery_time ON two_factor_recovery_attempts (attempted_at);

--
-- Name: idx_2fa_recovery_user; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_2fa_recovery_user ON two_factor_recovery_attempts (user_id);

--
-- Name: idx_two_factor_recovery_attempts_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_two_factor_recovery_attempts_tenant_id ON two_factor_recovery_attempts (tenant_id);

--
-- Name: two_factor_recovery_attempts; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE two_factor_recovery_attempts ENABLE ROW LEVEL SECURITY;

--
-- Name: two_factor_recovery_attempts; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE two_factor_recovery_attempts FORCE ROW LEVEL SECURITY;

--
-- Name: two_factor_setups; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS two_factor_setups (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    factor_id uuid DEFAULT gen_random_uuid() NOT NULL,
    secret varchar(32) NOT NULL,
    qr_code_url text,
    qr_code_data_uri text,
    otpauth_uri text,
    verified boolean DEFAULT false,
    expires_at timestamptz DEFAULT (CURRENT_TIMESTAMP + '00:10:00'::interval) NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid,
    CONSTRAINT two_factor_setups_pkey PRIMARY KEY (id),
    CONSTRAINT two_factor_setups_user_id_key UNIQUE (user_id),
    CONSTRAINT fk_2fa_setup_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT two_factor_setups_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);


COMMENT ON TABLE two_factor_setups IS 'Temporary storage for 2FA setup process. Entries expire after 10 minutes and should be cleaned up periodically.';


COMMENT ON COLUMN auth.two_factor_setups.factor_id IS 'Unique identifier for this 2FA factor';


COMMENT ON COLUMN auth.two_factor_setups.qr_code_data_uri IS 'QR code image as base64 data URI (data:image/png;base64,...)';


COMMENT ON COLUMN auth.two_factor_setups.otpauth_uri IS 'TOTP otpauth:// URI for manual entry or app deeplinks';

--
-- Name: idx_2fa_setup_expires; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_2fa_setup_expires ON two_factor_setups (expires_at);

--
-- Name: idx_2fa_setup_user; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_2fa_setup_user ON two_factor_setups (user_id);

--
-- Name: idx_two_factor_setups_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_two_factor_setups_tenant_id ON two_factor_setups (tenant_id);

--
-- Name: two_factor_setups; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE two_factor_setups ENABLE ROW LEVEL SECURITY;

--
-- Name: two_factor_setups; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE two_factor_setups FORCE ROW LEVEL SECURITY;

--
-- Name: user_trust_signals; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS user_trust_signals (
    id uuid DEFAULT gen_random_uuid(),
    user_id uuid,
    ip_address inet NOT NULL,
    device_fingerprint text,
    user_agent text,
    first_seen_at timestamptz DEFAULT now() NOT NULL,
    last_seen_at timestamptz DEFAULT now() NOT NULL,
    successful_logins integer DEFAULT 0,
    failed_attempts integer DEFAULT 0,
    last_captcha_at timestamptz,
    is_trusted boolean DEFAULT false,
    is_blocked boolean DEFAULT false,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT user_trust_signals_pkey PRIMARY KEY (id),
    CONSTRAINT user_trust_signals_user_id_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);


COMMENT ON TABLE user_trust_signals IS 'Tracks known devices and IPs for adaptive CAPTCHA trust scoring';

--
-- Name: idx_user_trust_signals_ip; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_user_trust_signals_ip ON user_trust_signals (ip_address);

--
-- Name: idx_user_trust_signals_last_seen; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_user_trust_signals_last_seen ON user_trust_signals (last_seen_at);

--
-- Name: idx_user_trust_signals_unique; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_trust_signals_unique ON user_trust_signals (user_id, ip_address, COALESCE(device_fingerprint, ''::text));

--
-- Name: idx_user_trust_signals_user_ip; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_user_trust_signals_user_ip ON user_trust_signals (user_id, ip_address);

--
-- Name: user_trust_signals; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE user_trust_signals ENABLE ROW LEVEL SECURITY;

--
-- Name: user_trust_signals; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE user_trust_signals FORCE ROW LEVEL SECURITY;

--
-- Name: webhook_monitored_tables; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS webhook_monitored_tables (
    schema_name text,
    table_name text,
    webhook_count integer DEFAULT 0,
    created_at timestamptz DEFAULT now(),
    CONSTRAINT webhook_monitored_tables_pkey PRIMARY KEY (schema_name, table_name)
);


COMMENT ON TABLE webhook_monitored_tables IS 'Tracks which tables have webhook triggers installed and how many webhooks monitor each table';

--
-- Name: webhook_monitored_tables; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE webhook_monitored_tables ENABLE ROW LEVEL SECURITY;

--
-- Name: webhook_monitored_tables; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE webhook_monitored_tables FORCE ROW LEVEL SECURITY;

--
-- Name: webhooks; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS webhooks (
    id uuid DEFAULT gen_random_uuid(),
    name text NOT NULL,
    description text,
    url text NOT NULL,
    events jsonb DEFAULT '[]',
    secret text,
    enabled boolean DEFAULT true,
    headers jsonb DEFAULT '{}',
    timeout_seconds integer DEFAULT 30,
    max_retries integer DEFAULT 3,
    retry_backoff_seconds integer DEFAULT 5,
    scope text DEFAULT 'user',
    created_by uuid,
    tenant_id uuid,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    CONSTRAINT webhooks_pkey PRIMARY KEY (id),
    CONSTRAINT webhooks_scope_check CHECK (scope IN ('user'::text, 'global'::text))
);

COMMENT ON COLUMN auth.webhooks.scope IS 'Scope determines which events trigger the webhook: user = only events on records owned by created_by, global = all events (admin only)';

--
-- Name: idx_auth_webhooks_enabled; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_webhooks_enabled ON webhooks (enabled);

--
-- Name: webhooks; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE webhooks ENABLE ROW LEVEL SECURITY;

--
-- Name: webhooks; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE webhooks FORCE ROW LEVEL SECURITY;

--
-- Name: webhook_deliveries; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id uuid DEFAULT gen_random_uuid(),
    webhook_id uuid NOT NULL,
    event text NOT NULL,
    payload jsonb NOT NULL,
    status text NOT NULL,
    status_code integer,
    response_body text,
    error text,
    attempt integer DEFAULT 1,
    delivered_at timestamptz,
    created_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT webhook_deliveries_pkey PRIMARY KEY (id),
    CONSTRAINT webhook_deliveries_webhook_id_fkey FOREIGN KEY (webhook_id) REFERENCES webhooks (id) ON DELETE CASCADE
);

--
-- Name: idx_auth_webhook_deliveries_created_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_webhook_deliveries_created_at ON webhook_deliveries (created_at DESC);

--
-- Name: idx_auth_webhook_deliveries_status; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_webhook_deliveries_status ON webhook_deliveries (status);

--
-- Name: idx_auth_webhook_deliveries_webhook_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_auth_webhook_deliveries_webhook_id ON webhook_deliveries (webhook_id);

--
-- Name: idx_webhook_deliveries_webhook_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON webhook_deliveries (webhook_id);

--
-- Name: idx_webhook_deliveries_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_tenant_id ON webhook_deliveries (tenant_id);

--
-- Name: webhook_deliveries; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE webhook_deliveries ENABLE ROW LEVEL SECURITY;

--
-- Name: webhook_deliveries; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE webhook_deliveries FORCE ROW LEVEL SECURITY;

--
-- Name: webhook_events; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS webhook_events (
    id uuid DEFAULT gen_random_uuid(),
    webhook_id uuid,
    event_type varchar(50) NOT NULL,
    table_schema varchar(255) NOT NULL,
    table_name varchar(255) NOT NULL,
    record_id text,
    old_data jsonb,
    new_data jsonb,
    processed boolean DEFAULT false,
    attempts integer DEFAULT 0,
    last_attempt_at timestamptz,
    next_retry_at timestamptz,
    error_message text,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid,
    CONSTRAINT webhook_events_pkey PRIMARY KEY (id),
    CONSTRAINT fk_webhook_event_webhook FOREIGN KEY (webhook_id) REFERENCES webhooks (id) ON DELETE CASCADE,
    CONSTRAINT webhook_events_webhook_id_fkey FOREIGN KEY (webhook_id) REFERENCES webhooks (id) ON DELETE CASCADE
);


COMMENT ON TABLE webhook_events IS 'Queue for webhook events to be delivered. Processed events are kept for history.';

--
-- Name: idx_webhook_events_created; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_webhook_events_created ON webhook_events (created_at);

--
-- Name: idx_webhook_events_unprocessed; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_webhook_events_unprocessed ON webhook_events (processed, next_retry_at) WHERE (processed = false);

--
-- Name: idx_webhook_events_webhook; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_webhook_events_webhook ON webhook_events (webhook_id);

--
-- Name: idx_webhook_events_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_webhook_events_tenant_id ON webhook_events (tenant_id);

--
-- Name: webhook_events; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE webhook_events ENABLE ROW LEVEL SECURITY;

--
-- Name: webhook_events; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE webhook_events FORCE ROW LEVEL SECURITY;

--
-- Name: cleanup_expired_captcha_data(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION cleanup_expired_captcha_data()
RETURNS void
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    -- Delete expired challenges older than 1 hour
    DELETE FROM captcha_challenges
    WHERE expires_at < NOW() - INTERVAL '1 hour';

    -- Delete expired trust tokens older than 1 hour
    DELETE FROM captcha_trust_tokens
    WHERE expires_at < NOW() - INTERVAL '1 hour';

    -- Delete trust signals not seen in 90 days
    DELETE FROM user_trust_signals
    WHERE last_seen_at < NOW() - INTERVAL '90 days';
END;
$$;

--
-- Name: cleanup_expired_captcha_data(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION cleanup_expired_captcha_data() IS 'Cleans up expired CAPTCHA challenges, tokens, and old trust signals';

--
-- Name: current_user_id(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION current_user_id()
RETURNS uuid
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    jwt_claims_var TEXT;
    user_id_var TEXT;
BEGIN
    -- Get user ID from request.jwt.claims (Supabase format)
    jwt_claims_var := current_setting('request.jwt.claims', true);
    IF jwt_claims_var IS NOT NULL AND jwt_claims_var <> '' THEN
        user_id_var := jwt_claims_var::json->>'sub';
        IF user_id_var IS NOT NULL AND user_id_var <> '' THEN
            RETURN user_id_var::UUID;
        END IF;
    END IF;

    RETURN NULL;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$;

--
-- Name: current_user_id(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION current_user_id() IS 'Returns the current authenticated user ID from PostgreSQL session variable request.jwt.claims (Supabase format). Returns NULL if not set or invalid.';

--
-- Name: current_user_role(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION current_user_role()
RETURNS text
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    jwt_claims_var TEXT;
    role_var TEXT;
BEGIN
    -- Get role from request.jwt.claims (Supabase format)
    jwt_claims_var := current_setting('request.jwt.claims', true);
    IF jwt_claims_var IS NOT NULL AND jwt_claims_var <> '' THEN
        role_var := jwt_claims_var::json->>'role';
        IF role_var IS NOT NULL AND role_var <> '' THEN
            RETURN role_var;
        END IF;
    END IF;

    -- Default to 'anon' if not set
    RETURN 'anon';
END;
$$;

--
-- Name: current_user_role(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION current_user_role() IS 'Returns the current user role from PostgreSQL session variable request.jwt.claims (Supabase format). Returns "anon" if not set.';

--
-- Name: disable_rls(text, text); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION disable_rls(
    table_name text,
    schema_name text DEFAULT 'public'
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    EXECUTE format('ALTER TABLE %I.%I DISABLE ROW LEVEL SECURITY', schema_name, table_name);
END;
$$;

--
-- Name: disable_rls(text, text); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION disable_rls(text, text) IS 'Disables Row Level Security on the specified table.';

--
-- Name: enable_rls(text, text); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION enable_rls(
    table_name text,
    schema_name text DEFAULT 'public'
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    EXECUTE format('ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY', schema_name, table_name);
    EXECUTE format('ALTER TABLE %I.%I FORCE ROW LEVEL SECURITY', schema_name, table_name);
END;
$$;

--
-- Name: enable_rls(text, text); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION enable_rls(text, text) IS 'Enables Row Level Security on the specified table and forces it even for table owners.';

--
-- Name: has_tenant_access(uuid); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION has_tenant_access(
    resource_tenant_id uuid
)
RETURNS boolean
LANGUAGE sql
VOLATILE
SECURITY DEFINER
SET search_path = public
AS $$
    -- Get the current tenant from session context
    SELECT CASE
        WHEN coalesce(current_setting('app.current_tenant_id', TRUE), '') = '' THEN
            -- No tenant context set, check if resource is in default tenant (NULL)
            resource_tenant_id IS NULL
        ELSE
            -- Tenant context is set, check if resource belongs to same tenant
            resource_tenant_id::text = current_setting('app.current_tenant_id', TRUE)
    END;
$$;

--
-- Name: has_tenant_access(uuid); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION has_tenant_access(uuid) IS 'Checks if the current tenant context has access to a resource. Returns TRUE if the resource_tenant_id matches the current tenant or if no tenant context is set and the resource is in the default tenant (NULL).';

--
-- Name: is_admin(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION is_admin()
RETURNS boolean
LANGUAGE plpgsql
STABLE
SET search_path = auth
AS $$
BEGIN
    RETURN current_user_role() = 'admin';
END;
$$;

--
-- Name: is_admin(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION is_admin() IS 'Returns TRUE if the current user role is "admin", FALSE otherwise.';

--
-- Name: is_authenticated(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION is_authenticated()
RETURNS boolean
LANGUAGE plpgsql
STABLE
SET search_path = auth
AS $$
BEGIN
    RETURN current_user_id() IS NOT NULL;
END;
$$;

--
-- Name: is_authenticated(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION is_authenticated() IS 'Returns TRUE if a user is authenticated (user_id is set), FALSE for anonymous users.';

--
-- Name: jwt(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION jwt()
RETURNS jsonb
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    jwt_claims_var TEXT;
BEGIN
    -- Return request.jwt.claims (Supabase format)
    jwt_claims_var := current_setting('request.jwt.claims', true);
    IF jwt_claims_var IS NOT NULL AND jwt_claims_var <> '' THEN
        BEGIN
            RETURN jwt_claims_var::JSONB;
        EXCEPTION
            WHEN OTHERS THEN
                RETURN NULL;
        END;
    END IF;

    RETURN NULL;
END;
$$;

--
-- Name: jwt(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION jwt() IS 'Supabase-compatible function that returns JWT claims as JSONB from request.jwt.claims session variable. Use ->> operator to extract text values or -> for JSONB.';

--
-- Name: queue_webhook_event(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION queue_webhook_event()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SET search_path = auth
AS $$
DECLARE
    webhook_record RECORD;
    event_type TEXT;
    old_data JSONB;
    new_data JSONB;
    record_id_value TEXT;
    record_owner_id UUID;
    should_trigger BOOLEAN;
BEGIN
    -- Determine event type and prepare data
    IF TG_OP = 'INSERT' THEN
        event_type := 'INSERT';
        old_data := NULL;
        new_data := to_jsonb(NEW);
        record_id_value := COALESCE((NEW.id)::TEXT, '');
    ELSIF TG_OP = 'UPDATE' THEN
        event_type := 'UPDATE';
        old_data := to_jsonb(OLD);
        new_data := to_jsonb(NEW);
        record_id_value := COALESCE((NEW.id)::TEXT, (OLD.id)::TEXT, '');
    ELSIF TG_OP = 'DELETE' THEN
        event_type := 'DELETE';
        old_data := to_jsonb(OLD);
        new_data := NULL;
        record_id_value := COALESCE((OLD.id)::TEXT, '');
    ELSE
        RETURN NULL;
    END IF;

    -- Extract record owner for scoping
    -- Check common ownership columns in order of precedence
    BEGIN
        record_owner_id := COALESCE(
            ((CASE WHEN TG_OP = 'DELETE' THEN old_data ELSE new_data END)->>'user_id')::UUID,
            ((CASE WHEN TG_OP = 'DELETE' THEN old_data ELSE new_data END)->>'owner_id')::UUID,
            ((CASE WHEN TG_OP = 'DELETE' THEN old_data ELSE new_data END)->>'created_by')::UUID,
            -- For users table, use the record's own id as the owner
            CASE WHEN TG_TABLE_SCHEMA = 'auth' AND TG_TABLE_NAME = 'users'
                 THEN ((CASE WHEN TG_OP = 'DELETE' THEN old_data ELSE new_data END)->>'id')::UUID
                 ELSE NULL END
        );
    EXCEPTION WHEN OTHERS THEN
        -- If UUID parsing fails, set to NULL (unowned record)
        record_owner_id := NULL;
    END;

    -- Find matching webhooks WITH SCOPING
    FOR webhook_record IN
        SELECT id, events, created_by, scope, tenant_id
        FROM auth.webhooks
        WHERE enabled = TRUE
          AND (
              scope = 'global'                    -- Global webhooks see everything
              OR created_by IS NULL              -- Legacy webhooks (no owner) see everything
              OR record_owner_id IS NULL         -- Unowned records are visible to all
              OR created_by = record_owner_id    -- User-scoped: owner matches
          )
    LOOP
        -- Check if this webhook is interested in this event
        should_trigger := FALSE;

        -- Parse the events JSONB array to check if it matches
        IF jsonb_typeof(webhook_record.events) = 'array' THEN
            should_trigger := EXISTS (
                SELECT 1
                FROM jsonb_array_elements(webhook_record.events) AS event
                WHERE
                    (event->>'table' = TG_TABLE_NAME OR event->>'table' = '*')
                    AND (
                        event->'operations' @> to_jsonb(ARRAY[event_type])
                        OR event->'operations' @> to_jsonb(ARRAY['*'])
                    )
            );
        END IF;

        -- Queue event if webhook is interested
        IF should_trigger THEN
            INSERT INTO auth.webhook_events (
                webhook_id,
                event_type,
                table_schema,
                table_name,
                record_id,
                old_data,
                new_data,
                next_retry_at,
                tenant_id
            ) VALUES (
                webhook_record.id,
                event_type,
                TG_TABLE_SCHEMA,
                TG_TABLE_NAME,
                record_id_value,
                old_data,
                new_data,
                CURRENT_TIMESTAMP,
                webhook_record.tenant_id
            );

            -- Send notification to application via pg_notify
            PERFORM pg_notify('webhook_event', webhook_record.id::TEXT);
        END IF;
    END LOOP;

    -- Return appropriate value based on operation
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$;

--
-- Name: queue_webhook_event(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION queue_webhook_event() IS 'Trigger function that queues webhook events when data changes occur, with user-based scoping support';

--
-- Name: create_webhook_trigger(text, text); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION create_webhook_trigger(
    schema_name text,
    table_name text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
AS $$
DECLARE
    trigger_name TEXT;
    full_table_name TEXT;
BEGIN
    trigger_name := format('webhook_trigger_%s_%s', schema_name, table_name);
    full_table_name := format('%I.%I', schema_name, table_name);

    -- Drop existing trigger if exists
    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %s', trigger_name, full_table_name);

    -- Create new trigger
    EXECUTE format('
        CREATE TRIGGER %I
        AFTER INSERT OR UPDATE OR DELETE ON %s
        FOR EACH ROW EXECUTE FUNCTION auth.queue_webhook_event()
    ', trigger_name, full_table_name);

    RAISE NOTICE 'Created webhook trigger % on %', trigger_name, full_table_name;
END;
$$;

--
-- Name: create_webhook_trigger(text, text); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION create_webhook_trigger(text, text) IS 'Creates a webhook trigger on a specified table';

--
-- Name: increment_webhook_table_count(text, text); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION increment_webhook_table_count(
    p_schema text,
    p_table text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = auth
AS $$
DECLARE
    v_count INTEGER;
BEGIN
    INSERT INTO webhook_monitored_tables (schema_name, table_name, webhook_count)
    VALUES (p_schema, p_table, 1)
    ON CONFLICT (schema_name, table_name)
    DO UPDATE SET webhook_count = webhook_monitored_tables.webhook_count + 1;

    -- Get the current count
    SELECT webhook_count INTO v_count
    FROM webhook_monitored_tables
    WHERE schema_name = p_schema AND table_name = p_table;

    -- Create trigger if this is the first webhook monitoring this table
    IF v_count = 1 THEN
        PERFORM create_webhook_trigger(p_schema, p_table);
    END IF;
END;
$$;

--
-- Name: increment_webhook_table_count(text, text); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION increment_webhook_table_count(text, text) IS 'Increments the webhook count for a table and creates the trigger if this is the first webhook';

--
-- Name: remove_webhook_trigger(text, text); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION remove_webhook_trigger(
    schema_name text,
    table_name text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
AS $$
DECLARE
    trigger_name TEXT;
    full_table_name TEXT;
BEGIN
    trigger_name := format('webhook_trigger_%s_%s', schema_name, table_name);
    full_table_name := format('%I.%I', schema_name, table_name);

    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %s', trigger_name, full_table_name);

    RAISE NOTICE 'Removed webhook trigger % from %', trigger_name, full_table_name;
END;
$$;

--
-- Name: remove_webhook_trigger(text, text); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION remove_webhook_trigger(text, text) IS 'Removes a webhook trigger from a specified table';

--
-- Name: decrement_webhook_table_count(text, text); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION decrement_webhook_table_count(
    p_schema text,
    p_table text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = auth
AS $$
DECLARE
    v_count INTEGER;
BEGIN
    UPDATE webhook_monitored_tables
    SET webhook_count = GREATEST(0, webhook_count - 1)
    WHERE schema_name = p_schema AND table_name = p_table
    RETURNING webhook_count INTO v_count;

    -- Remove trigger and tracking row if no webhooks left
    IF v_count IS NOT NULL AND v_count = 0 THEN
        PERFORM remove_webhook_trigger(p_schema, p_table);
        DELETE FROM webhook_monitored_tables
        WHERE schema_name = p_schema AND table_name = p_table;
    END IF;
END;
$$;

--
-- Name: decrement_webhook_table_count(text, text); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION decrement_webhook_table_count(text, text) IS 'Decrements the webhook count for a table and removes the trigger if no webhooks remain';

--
-- Name: role(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION role()
RETURNS text
LANGUAGE plpgsql
STABLE
SET search_path = auth
AS $$
BEGIN
    RETURN current_user_role();
END;
$$;

--
-- Name: role(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION role() IS 'Supabase-compatible alias for auth.current_user_role(). Returns the current user role.';

--
-- Name: set_tenant_id_from_context(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION set_tenant_id_from_context()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    -- Only set tenant_id if not already provided and context is available
    IF NEW.tenant_id IS NULL THEN
        BEGIN
            NEW.tenant_id := current_setting('app.current_tenant_id', TRUE)::UUID;
        EXCEPTION
            WHEN others THEN
                NEW.tenant_id := NULL;
        END;
    END IF;
    RETURN NEW;
END;
$$;

--
-- Name: uid(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION uid()
RETURNS uuid
LANGUAGE plpgsql
STABLE
SET search_path = auth
AS $$
BEGIN
    RETURN current_user_id();
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$;

--
-- Name: uid(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION uid() IS 'Supabase-compatible alias for auth.current_user_id(). Returns the current authenticated user ID.';

--
-- Name: update_mcp_oauth_clients_updated_at(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_mcp_oauth_clients_updated_at()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

--
-- Name: update_saml_providers_updated_at(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_saml_providers_updated_at()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

--
-- Name: update_trust_signals_updated_at(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_trust_signals_updated_at()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

--
-- Name: update_webhook_updated_at(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_webhook_updated_at()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

--
-- Name: validate_app_metadata_update(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION validate_app_metadata_update()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = auth
AS $$
DECLARE
    user_role TEXT;
BEGIN
    -- Get the current user's role
    user_role := auth.current_user_role();

    -- Check if app_metadata is being modified
    IF OLD.app_metadata IS DISTINCT FROM NEW.app_metadata THEN
        -- Only allow admins and dashboard admins to modify app_metadata
        IF user_role != 'admin' AND user_role != 'instance_admin' THEN
            -- Also check if user has admin privileges via is_admin() function
            IF NOT auth.is_admin() THEN
                RAISE EXCEPTION 'Only admins can modify app_metadata'
                    USING ERRCODE = 'insufficient_privilege';
            END IF;
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

--
-- Name: validate_app_metadata_update(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION validate_app_metadata_update() IS 'Validates that only admins and dashboard admins can modify the app_metadata field on auth.users';

--
-- Cross-schema FKs moved to post-schema-fks.sql
-- fk_auth_users_tenant, mcp_oauth_codes_user_id_fkey, mcp_oauth_tokens_user_id_fkey,
-- fk_auth_service_keys_tenant, service_keys_revoked_by_fkey, service_key_revocations_revoked_by_fkey
--

--
-- Name: Dashboard admin can manage saml_providers; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY "Dashboard admin can manage saml_providers" ON saml_providers TO authenticated USING (current_user_role() = 'instance_admin') WITH CHECK (current_user_role() = 'instance_admin');

--
-- Name: saml_providers_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY saml_providers_tenant ON saml_providers TO PUBLIC USING (has_tenant_access(tenant_id)) WITH CHECK (has_tenant_access(tenant_id));

--
-- Name: auth_client_keys_policy; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_client_keys_policy ON client_keys TO PUBLIC USING (is_admin() OR (current_user_role() = 'instance_admin') OR (CURRENT_USER = 'service_role'::name) OR (current_user_role() = 'tenant_service' AND has_tenant_access(tenant_id)) OR (current_user_id()::text = (user_id)::text)) WITH CHECK (is_admin() OR (current_user_role() = 'instance_admin') OR (CURRENT_USER = 'service_role'::name) OR (current_user_role() = 'tenant_service' AND has_tenant_access(tenant_id)) OR (current_user_id()::text = (user_id)::text));

--
-- Name: auth_service_keys_delete; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_service_keys_delete ON service_keys FOR DELETE TO PUBLIC USING ((CURRENT_USER = 'service_role'::name) OR (current_user_role() IN ('instance_admin', 'tenant_service') AND has_tenant_access(tenant_id)));

--
-- Name: auth_service_keys_insert; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_service_keys_insert ON service_keys FOR INSERT TO PUBLIC WITH CHECK ((CURRENT_USER = 'service_role'::name) OR (current_user_role() IN ('instance_admin', 'tenant_service') AND has_tenant_access(tenant_id)));

--
-- Name: auth_service_keys_select; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_service_keys_select ON service_keys FOR SELECT TO PUBLIC USING ((CURRENT_USER = 'service_role'::name) OR (current_user_role() IN ('instance_admin', 'tenant_service') AND has_tenant_access(tenant_id)));

--
-- Name: auth_service_keys_update; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_service_keys_update ON service_keys FOR UPDATE TO PUBLIC USING ((CURRENT_USER = 'service_role'::name) OR (current_user_role() IN ('instance_admin', 'tenant_service') AND has_tenant_access(tenant_id))) WITH CHECK ((CURRENT_USER = 'service_role'::name) OR (current_user_role() IN ('instance_admin', 'tenant_service') AND has_tenant_access(tenant_id)));

--
-- Name: auth_sessions_delete_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_sessions_delete_own ON sessions FOR DELETE TO authenticated USING ((user_id = current_user_id()) OR (current_user_role() = 'instance_admin'));

--
-- Name: auth_sessions_select_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_sessions_select_own ON sessions FOR SELECT TO authenticated USING ((user_id = current_user_id()) OR (current_user_role() = 'instance_admin'));

--
-- Name: auth_sessions_update_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_sessions_update_own ON sessions FOR UPDATE TO authenticated USING (user_id = current_user_id()) WITH CHECK (user_id = current_user_id());

--
-- Name: auth_users_delete; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_users_delete ON users FOR DELETE TO PUBLIC USING ((CURRENT_USER = 'service_role'::name) OR has_tenant_access(tenant_id));

--
-- Name: auth_users_delete_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_users_delete_admin ON users FOR DELETE TO authenticated USING (is_admin() OR (current_user_role() = 'instance_admin'));

--
-- Name: auth_users_insert; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_users_insert ON users FOR INSERT TO PUBLIC WITH CHECK ((CURRENT_USER = 'service_role'::name) OR has_tenant_access(tenant_id));

--
-- Name: auth_users_select; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_users_select ON users FOR SELECT TO PUBLIC USING ((CURRENT_USER = 'service_role'::name) OR (current_setting('request.jwt.claims', true) <> '' AND id = ((current_setting('request.jwt.claims', true)::jsonb ->> 'sub'))::uuid) OR has_tenant_access(tenant_id));

--
-- Name: auth_users_select_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_users_select_own ON users FOR SELECT TO authenticated USING ((id = current_user_id()) OR is_admin() OR (current_user_role() = 'instance_admin'));

--
-- Name: auth_users_update; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_users_update ON users FOR UPDATE TO PUBLIC USING ((CURRENT_USER = 'service_role'::name) OR (current_setting('request.jwt.claims', true) <> '' AND id = ((current_setting('request.jwt.claims', true)::jsonb ->> 'sub'))::uuid) OR has_tenant_access(tenant_id)) WITH CHECK ((CURRENT_USER = 'service_role'::name) OR (current_setting('request.jwt.claims', true) <> '' AND id = ((current_setting('request.jwt.claims', true)::jsonb ->> 'sub'))::uuid) OR has_tenant_access(tenant_id));

--
-- Name: auth_users_update_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_users_update_own ON users FOR UPDATE TO authenticated USING (is_admin() OR (current_user_role() = 'instance_admin') OR (current_user_id()::text = (id)::text)) WITH CHECK (is_admin() OR (current_user_role() = 'instance_admin') OR (current_user_id()::text = (id)::text));

--
-- Name: client_key_usage_service_write; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY client_key_usage_service_write ON client_key_usage FOR INSERT TO PUBLIC WITH CHECK (current_user_role() = 'service_role');

--
-- Name: client_key_usage_user_read; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY client_key_usage_user_read ON client_key_usage FOR SELECT TO PUBLIC USING ((client_key_id IN ( SELECT client_keys.id FROM client_keys WHERE (client_keys.user_id = current_user_id()))) OR is_admin() OR (current_user_role() = 'instance_admin') OR (current_user_role() = 'service_role'));

--
-- Name: email_verification_tokens_service_only; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY email_verification_tokens_service_only ON email_verification_tokens TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: impersonation_sessions_instance_admin_only; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY impersonation_sessions_instance_admin_only ON impersonation_sessions TO PUBLIC USING (
    (current_user_role() = 'service_role')
    OR (current_user_role() = 'instance_admin')
    OR (is_admin() AND has_tenant_access(tenant_id))
) WITH CHECK (
    (current_user_role() = 'service_role')
    OR (current_user_role() = 'instance_admin')
    OR (is_admin() AND has_tenant_access(tenant_id))
);

--
-- Name: magic_links_service_only; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY magic_links_service_only ON magic_links TO PUBLIC USING (current_user_role() = 'service_role');

--
-- Name: mfa_factors_admin_all; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY mfa_factors_admin_all ON mfa_factors TO PUBLIC USING (is_admin() OR (current_user_role() = 'instance_admin') OR (current_user_role() = 'service_role'));

--
-- Name: mfa_factors_delete_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY mfa_factors_delete_own ON mfa_factors FOR DELETE TO PUBLIC USING (user_id = current_user_id());

--
-- Name: mfa_factors_insert_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY mfa_factors_insert_own ON mfa_factors FOR INSERT TO PUBLIC WITH CHECK (user_id = current_user_id());

--
-- Name: mfa_factors_select_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY mfa_factors_select_own ON mfa_factors FOR SELECT TO PUBLIC USING (user_id = current_user_id());

--
-- Name: mfa_factors_update_own; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY mfa_factors_update_own ON mfa_factors FOR UPDATE TO PUBLIC USING (user_id = current_user_id()) WITH CHECK (user_id = current_user_id());

--
-- Name: oauth_links_select; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY oauth_links_select ON oauth_links FOR SELECT TO PUBLIC USING (user_id = current_user_id());

--
-- Name: oauth_links_service_all; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY oauth_links_service_all ON oauth_links TO PUBLIC USING (current_user_role() = 'service_role');

--
-- Name: oauth_logout_states_service_access; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY oauth_logout_states_service_access ON oauth_logout_states TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: oauth_tokens_select; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY oauth_tokens_select ON oauth_tokens FOR SELECT TO PUBLIC USING (user_id = current_user_id());

--
-- Name: oauth_tokens_service_all; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY oauth_tokens_service_all ON oauth_tokens TO PUBLIC USING (current_user_role() = 'service_role');

--
-- Name: password_reset_tokens_service_only; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY password_reset_tokens_service_only ON password_reset_tokens TO PUBLIC USING (current_user_role() = 'service_role');

--
-- Name: rls_audit_log_admin_select; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY rls_audit_log_admin_select ON rls_audit_log FOR SELECT TO authenticated USING (current_user_role() = ANY (ARRAY['admin', 'instance_admin', 'service_role']));

--
-- Name: rls_audit_log_service_insert; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY rls_audit_log_service_insert ON rls_audit_log FOR INSERT TO authenticated WITH CHECK (current_user_role() = 'service_role');

--
-- Name: rls_audit_log_user_select; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY rls_audit_log_user_select ON rls_audit_log FOR SELECT TO authenticated USING (current_user_id() = user_id);

--
-- Name: token_blacklist_admin_only; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY token_blacklist_admin_only ON token_blacklist TO PUBLIC USING ((current_user_role() = 'service_role') OR (current_user_role() = 'instance_admin'));

--
-- Name: two_factor_recovery_admin_select; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY two_factor_recovery_admin_select ON two_factor_recovery_attempts FOR SELECT TO PUBLIC USING (is_admin() OR (current_user_role() = 'instance_admin'));

--
-- Name: two_factor_recovery_insert; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY two_factor_recovery_insert ON two_factor_recovery_attempts FOR INSERT TO PUBLIC WITH CHECK ((current_user_role() = 'service_role') OR (user_id = current_user_id()));

--
-- Name: two_factor_recovery_select; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY two_factor_recovery_select ON two_factor_recovery_attempts FOR SELECT TO PUBLIC USING (user_id = current_user_id());

--
-- Name: two_factor_setups_admin_select; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY two_factor_setups_admin_select ON two_factor_setups FOR SELECT TO PUBLIC USING (is_admin() OR (current_user_role() = 'instance_admin'));

--
-- Name: two_factor_setups_delete; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY two_factor_setups_delete ON two_factor_setups FOR DELETE TO PUBLIC USING (user_id = current_user_id());

--
-- Name: two_factor_setups_insert; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY two_factor_setups_insert ON two_factor_setups FOR INSERT TO PUBLIC WITH CHECK (user_id = current_user_id());

--
-- Name: two_factor_setups_select; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY two_factor_setups_select ON two_factor_setups FOR SELECT TO PUBLIC USING (user_id = current_user_id());

--
-- Name: two_factor_setups_update; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY two_factor_setups_update ON two_factor_setups FOR UPDATE TO PUBLIC USING (user_id = current_user_id()) WITH CHECK (user_id = current_user_id());

--
-- Name: webhook_deliveries_admin_read; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY webhook_deliveries_admin_read ON webhook_deliveries FOR SELECT TO PUBLIC USING ((current_user_role() = 'service_role') OR (current_user_role() = 'instance_admin') OR is_admin());

--
-- Name: webhook_deliveries_service_update; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY webhook_deliveries_service_update ON webhook_deliveries FOR UPDATE TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: webhook_deliveries_service_write; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY webhook_deliveries_service_write ON webhook_deliveries FOR INSERT TO PUBLIC WITH CHECK (current_user_role() = 'service_role');

--
-- Name: webhook_deliveries_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY webhook_deliveries_tenant ON webhook_deliveries TO PUBLIC USING ((current_user_role() = 'service_role' OR current_user_role() = 'instance_admin' OR is_admin()) AND has_tenant_access(tenant_id));

--
-- Name: webhook_events_admin_select; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY webhook_events_admin_select ON webhook_events FOR SELECT TO PUBLIC USING (is_admin() OR (current_user_role() = 'instance_admin'));

--
-- Name: webhook_events_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY webhook_events_service ON webhook_events TO PUBLIC USING (current_user_role() = 'service_role');

--
-- Name: webhook_events_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY webhook_events_tenant ON webhook_events TO PUBLIC USING ((current_user_role() = 'service_role' OR current_user_role() = 'instance_admin' OR is_admin()) AND has_tenant_access(tenant_id));

--
-- Name: webhook_monitored_tables_service_only; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY webhook_monitored_tables_service_only ON webhook_monitored_tables TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: webhooks_admin_only; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY webhooks_admin_only ON webhooks TO PUBLIC USING ((current_user_role() = 'service_role') OR (current_user_role() = 'instance_admin') OR is_admin()) WITH CHECK ((current_user_role() = 'service_role') OR (current_user_role() = 'instance_admin') OR is_admin());

--
-- Name: auth_webhooks_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY auth_webhooks_tenant ON webhooks TO PUBLIC USING ((current_user_role() = 'service_role' OR current_user_role() = 'instance_admin' OR is_admin()) AND auth.has_tenant_access(tenant_id)) WITH CHECK ((current_user_role() = 'service_role' OR current_user_role() = 'instance_admin' OR is_admin()) AND auth.has_tenant_access(tenant_id));

--
-- Name: saml_assertion_ids_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY saml_assertion_ids_service ON saml_assertion_ids TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: saml_sessions_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY saml_sessions_service ON saml_sessions TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: captcha_challenges_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY captcha_challenges_service ON captcha_challenges TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: captcha_challenges_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY captcha_challenges_admin ON captcha_challenges TO authenticated USING (is_admin() OR current_user_role() = 'instance_admin') WITH CHECK (is_admin() OR current_user_role() = 'instance_admin');

--
-- Name: captcha_trust_tokens_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY captcha_trust_tokens_service ON captcha_trust_tokens TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: captcha_trust_tokens_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY captcha_trust_tokens_admin ON captcha_trust_tokens TO authenticated USING (is_admin() OR current_user_role() = 'instance_admin') WITH CHECK (is_admin() OR current_user_role() = 'instance_admin');

--
-- Name: emergency_revocation_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY emergency_revocation_service ON emergency_revocation TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: nonces_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY nonces_service ON nonces TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: user_trust_signals_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY user_trust_signals_service ON user_trust_signals TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: otp_codes_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY otp_codes_service ON otp_codes TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: oauth_states_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY oauth_states_service ON oauth_states TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: mcp_oauth_clients_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY mcp_oauth_clients_service ON mcp_oauth_clients TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: mcp_oauth_clients_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY mcp_oauth_clients_admin ON mcp_oauth_clients TO authenticated USING (is_admin() OR current_user_role() = 'instance_admin') WITH CHECK (is_admin() OR current_user_role() = 'instance_admin');

--
-- Name: mcp_oauth_codes_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY mcp_oauth_codes_service ON mcp_oauth_codes TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: mcp_oauth_tokens_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY mcp_oauth_tokens_service ON mcp_oauth_tokens TO PUBLIC USING (current_user_role() = 'service_role') WITH CHECK (current_user_role() = 'service_role');

--
-- Name: auth_service_keys_set_tenant_id; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER auth_service_keys_set_tenant_id
    BEFORE INSERT ON service_keys
    FOR EACH ROW
    EXECUTE FUNCTION set_tenant_id_from_context();

--
-- Name: auth_webhooks_set_tenant_id; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER auth_webhooks_set_tenant_id
    BEFORE INSERT ON webhooks
    FOR EACH ROW
    EXECUTE FUNCTION set_tenant_id_from_context();

--
-- Name: auth_client_keys_set_tenant_id; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER auth_client_keys_set_tenant_id
    BEFORE INSERT ON client_keys
    FOR EACH ROW
    EXECUTE FUNCTION set_tenant_id_from_context();

--
-- Name: auth_impersonation_sessions_set_tenant_id; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER auth_impersonation_sessions_set_tenant_id
    BEFORE INSERT ON impersonation_sessions
    FOR EACH ROW
    EXECUTE FUNCTION set_tenant_id_from_context();

--
-- Name: auth_users_set_tenant_id; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER auth_users_set_tenant_id
    BEFORE INSERT ON users
    FOR EACH ROW
    EXECUTE FUNCTION set_tenant_id_from_context();

--
-- Name: set_timestamp; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER set_timestamp
    BEFORE UPDATE ON mfa_factors
    FOR EACH ROW
    EXECUTE FUNCTION platform.update_updated_at();

--
-- Name: trigger_mcp_oauth_clients_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER trigger_mcp_oauth_clients_updated_at
    BEFORE UPDATE ON mcp_oauth_clients
    FOR EACH ROW
    EXECUTE FUNCTION update_mcp_oauth_clients_updated_at();

--
-- Name: trigger_update_saml_providers_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER trigger_update_saml_providers_updated_at
    BEFORE UPDATE ON saml_providers
    FOR EACH ROW
    EXECUTE FUNCTION update_saml_providers_updated_at();

--
-- Name: trigger_update_trust_signals_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER trigger_update_trust_signals_updated_at
    BEFORE UPDATE ON user_trust_signals
    FOR EACH ROW
    EXECUTE FUNCTION update_trust_signals_updated_at();

--
-- Name: update_auth_client_keys_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER update_auth_client_keys_updated_at
    BEFORE UPDATE ON client_keys
    FOR EACH ROW
    EXECUTE FUNCTION platform.update_updated_at();

--
-- Name: update_auth_sessions_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER update_auth_sessions_updated_at
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION platform.update_updated_at();

--
-- Name: update_auth_users_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER update_auth_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION platform.update_updated_at();

--
-- Name: update_auth_webhooks_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER update_auth_webhooks_updated_at
    BEFORE UPDATE ON webhooks
    FOR EACH ROW
    EXECUTE FUNCTION update_webhook_updated_at();

--
-- Name: update_oauth_links_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER update_oauth_links_updated_at
    BEFORE UPDATE ON oauth_links
    FOR EACH ROW
    EXECUTE FUNCTION platform.update_updated_at();

--
-- Name: update_oauth_tokens_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER update_oauth_tokens_updated_at
    BEFORE UPDATE ON oauth_tokens
    FOR EACH ROW
    EXECUTE FUNCTION platform.update_updated_at();

--
-- Name: validate_app_metadata_trigger; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER validate_app_metadata_trigger
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION validate_app_metadata_update();

--
-- Name: cleanup_expired_captcha_data(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION cleanup_expired_captcha_data() TO {{APP_USER}};

--
-- Name: cleanup_expired_captcha_data(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: cleanup_expired_captcha_data(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION cleanup_expired_captcha_data() TO service_role;

--
-- Name: create_webhook_trigger(schema_name text, table_name text); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION create_webhook_trigger(schema_name text, table_name text) TO {{APP_USER}};

--
-- Name: create_webhook_trigger(schema_name text, table_name text); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: current_user_id(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION current_user_id() TO {{APP_USER}};

--
-- Name: current_user_id(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: current_user_role(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION current_user_role() TO {{APP_USER}};

--
-- Name: current_user_role(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: decrement_webhook_table_count(p_schema text, p_table text); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION decrement_webhook_table_count(p_schema text, p_table text) TO {{APP_USER}};

--
-- Name: decrement_webhook_table_count(p_schema text, p_table text); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: disable_rls(table_name text, schema_name text); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION disable_rls(table_name text, schema_name text) TO {{APP_USER}};

--
-- Name: disable_rls(table_name text, schema_name text); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: enable_rls(table_name text, schema_name text); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION enable_rls(table_name text, schema_name text) TO {{APP_USER}};

--
-- Name: enable_rls(table_name text, schema_name text); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: has_tenant_access(resource_tenant_id uuid); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION has_tenant_access(resource_tenant_id uuid) TO {{APP_USER}};

--
-- Name: has_tenant_access(resource_tenant_id uuid); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: increment_webhook_table_count(p_schema text, p_table text); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION increment_webhook_table_count(p_schema text, p_table text) TO {{APP_USER}};

--
-- Name: increment_webhook_table_count(p_schema text, p_table text); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: is_admin(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION is_admin() TO {{APP_USER}};

--
-- Name: is_admin(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: is_authenticated(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION is_authenticated() TO {{APP_USER}};

--
-- Name: is_authenticated(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: jwt(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION jwt() TO {{APP_USER}};

--
-- Name: jwt(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: queue_webhook_event(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION queue_webhook_event() TO {{APP_USER}};

--
-- Name: queue_webhook_event(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: remove_webhook_trigger(schema_name text, table_name text); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION remove_webhook_trigger(schema_name text, table_name text) TO {{APP_USER}};

--
-- Name: remove_webhook_trigger(schema_name text, table_name text); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: role(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION role() TO {{APP_USER}};

--
-- Name: role(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: set_tenant_id_from_context(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION set_tenant_id_from_context() TO {{APP_USER}};

--
-- Name: set_tenant_id_from_context(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: uid(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION uid() TO {{APP_USER}};

--
-- Name: uid(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: update_mcp_oauth_clients_updated_at(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION update_mcp_oauth_clients_updated_at() TO {{APP_USER}};

--
-- Name: update_mcp_oauth_clients_updated_at(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: update_saml_providers_updated_at(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION update_saml_providers_updated_at() TO {{APP_USER}};

--
-- Name: update_saml_providers_updated_at(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: update_trust_signals_updated_at(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION update_trust_signals_updated_at() TO {{APP_USER}};

--
-- Name: update_trust_signals_updated_at(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: update_webhook_updated_at(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION update_webhook_updated_at() TO {{APP_USER}};

--
-- Name: update_webhook_updated_at(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: validate_app_metadata_update(); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION validate_app_metadata_update() TO {{APP_USER}};

--
-- Name: validate_app_metadata_update(); Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: emergency_revocation_id_seq; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT, UPDATE, USAGE ON SEQUENCE emergency_revocation_id_seq TO {{APP_USER}};

--
-- Name: emergency_revocation_id_seq; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: emergency_revocation_id_seq; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT, USAGE ON SEQUENCE emergency_revocation_id_seq TO service_role;

--
-- Name: captcha_challenges; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE captcha_challenges TO authenticated;

--
-- Name: captcha_challenges; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE captcha_challenges TO {{APP_USER}};

--
-- Name: captcha_challenges; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: captcha_challenges; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE captcha_challenges TO service_role;

--
-- Name: captcha_trust_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE captcha_trust_tokens TO authenticated;

--
-- Name: captcha_trust_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE captcha_trust_tokens TO {{APP_USER}};

--
-- Name: captcha_trust_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: captcha_trust_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE captcha_trust_tokens TO service_role;

--
-- Name: client_key_usage; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE client_key_usage TO authenticated;

--
-- Name: client_key_usage; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE client_key_usage TO {{APP_USER}};

--
-- Name: client_key_usage; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: client_key_usage; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE client_key_usage TO service_role;

--
-- Name: client_keys; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE client_keys TO authenticated;

--
-- Name: client_keys; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE client_keys TO {{APP_USER}};

--
-- Name: client_keys; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: client_keys; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE client_keys TO service_role;

--
-- Name: email_verification_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE email_verification_tokens TO authenticated;

--
-- Name: email_verification_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE email_verification_tokens TO {{APP_USER}};

--
-- Name: email_verification_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: email_verification_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE email_verification_tokens TO service_role;

--
-- Name: emergency_revocation; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE emergency_revocation TO authenticated;

--
-- Name: emergency_revocation; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE emergency_revocation TO {{APP_USER}};

--
-- Name: emergency_revocation; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: emergency_revocation; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE emergency_revocation TO service_role;

--
-- Name: impersonation_sessions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE impersonation_sessions TO authenticated;

--
-- Name: impersonation_sessions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE impersonation_sessions TO {{APP_USER}};

--
-- Name: impersonation_sessions; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: impersonation_sessions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE impersonation_sessions TO service_role;

--
-- Name: magic_links; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE magic_links TO authenticated;

--
-- Name: magic_links; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE magic_links TO {{APP_USER}};

--
-- Name: magic_links; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: magic_links; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE magic_links TO service_role;

--
-- Name: mcp_oauth_clients; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE mcp_oauth_clients TO authenticated;

--
-- Name: mcp_oauth_clients; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE mcp_oauth_clients TO {{APP_USER}};

--
-- Name: mcp_oauth_clients; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: mcp_oauth_clients; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE mcp_oauth_clients TO service_role;

--
-- Name: mcp_oauth_codes; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE mcp_oauth_codes TO authenticated;

--
-- Name: mcp_oauth_codes; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE mcp_oauth_codes TO {{APP_USER}};

--
-- Name: mcp_oauth_codes; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: mcp_oauth_codes; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE mcp_oauth_codes TO service_role;

--
-- Name: mcp_oauth_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE mcp_oauth_tokens TO authenticated;

--
-- Name: mcp_oauth_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE mcp_oauth_tokens TO {{APP_USER}};

--
-- Name: mcp_oauth_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: mcp_oauth_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE mcp_oauth_tokens TO service_role;

--
-- Name: mfa_factors; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE mfa_factors TO authenticated;

--
-- Name: mfa_factors; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE mfa_factors TO {{APP_USER}};

--
-- Name: mfa_factors; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: mfa_factors; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE mfa_factors TO service_role;

--
-- Name: nonces; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE nonces TO authenticated;

--
-- Name: nonces; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE nonces TO {{APP_USER}};

--
-- Name: nonces; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: nonces; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE nonces TO service_role;

--
-- Name: oauth_links; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE oauth_links TO authenticated;

--
-- Name: oauth_links; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE oauth_links TO {{APP_USER}};

--
-- Name: oauth_links; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: oauth_links; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE oauth_links TO service_role;

--
-- Name: oauth_logout_states; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE oauth_logout_states TO authenticated;

--
-- Name: oauth_logout_states; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE oauth_logout_states TO {{APP_USER}};

--
-- Name: oauth_logout_states; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: oauth_logout_states; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE oauth_logout_states TO service_role;

--
-- Name: oauth_states; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE oauth_states TO authenticated;

--
-- Name: oauth_states; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE oauth_states TO {{APP_USER}};

--
-- Name: oauth_states; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: oauth_states; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE oauth_states TO service_role;

--
-- Name: oauth_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE oauth_tokens TO authenticated;

--
-- Name: oauth_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE oauth_tokens TO {{APP_USER}};

--
-- Name: oauth_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: oauth_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE oauth_tokens TO service_role;

--
-- Name: otp_codes; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE otp_codes TO authenticated;

--
-- Name: otp_codes; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE otp_codes TO {{APP_USER}};

--
-- Name: otp_codes; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: otp_codes; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE otp_codes TO service_role;

--
-- Name: password_reset_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE password_reset_tokens TO authenticated;

--
-- Name: password_reset_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE password_reset_tokens TO {{APP_USER}};

--
-- Name: password_reset_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: password_reset_tokens; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE password_reset_tokens TO service_role;

--
-- Name: rls_audit_log; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE rls_audit_log TO authenticated;

--
-- Name: rls_audit_log; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE rls_audit_log TO {{APP_USER}};

--
-- Name: rls_audit_log; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: rls_audit_log; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE rls_audit_log TO service_role;

--
-- Name: saml_assertion_ids; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE saml_assertion_ids TO authenticated;

--
-- Name: saml_assertion_ids; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE saml_assertion_ids TO {{APP_USER}};

--
-- Name: saml_assertion_ids; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: saml_assertion_ids; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE saml_assertion_ids TO service_role;

--
-- Name: saml_providers; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE saml_providers TO authenticated;

--
-- Name: saml_providers; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE saml_providers TO {{APP_USER}};

--
-- Name: saml_providers; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: saml_providers; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE saml_providers TO service_role;

--
-- Name: saml_providers; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.saml_providers TO tenant_service;

--
-- Name: saml_sessions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE saml_sessions TO authenticated;

--
-- Name: saml_sessions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE saml_sessions TO {{APP_USER}};

--
-- Name: saml_sessions; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: saml_sessions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE saml_sessions TO service_role;

--
-- Name: service_key_revocations; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE service_key_revocations TO authenticated;

--
-- Name: service_key_revocations; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE service_key_revocations TO {{APP_USER}};

--
-- Name: service_key_revocations; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: service_key_revocations; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE service_key_revocations TO service_role;

--
-- Name: service_keys; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE service_keys TO authenticated;

--
-- Name: service_keys; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE service_keys TO {{APP_USER}};

--
-- Name: service_keys; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: service_keys; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE service_keys TO service_role;

--
-- Name: service_keys; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE service_keys TO tenant_service;

--
-- Name: sessions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE sessions TO authenticated;

--
-- Name: sessions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE sessions TO {{APP_USER}};

--
-- Name: sessions; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: sessions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE sessions TO service_role;

--
-- Name: token_blacklist; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE token_blacklist TO authenticated;

--
-- Name: token_blacklist; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE token_blacklist TO {{APP_USER}};

--
-- Name: token_blacklist; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: token_blacklist; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE token_blacklist TO service_role;

--
-- Name: two_factor_recovery_attempts; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE two_factor_recovery_attempts TO authenticated;

--
-- Name: two_factor_recovery_attempts; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE two_factor_recovery_attempts TO {{APP_USER}};

--
-- Name: two_factor_recovery_attempts; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: two_factor_recovery_attempts; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE two_factor_recovery_attempts TO service_role;

--
-- Name: two_factor_setups; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE two_factor_setups TO authenticated;

--
-- Name: two_factor_setups; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE two_factor_setups TO {{APP_USER}};

--
-- Name: two_factor_setups; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: two_factor_setups; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE two_factor_setups TO service_role;

--
-- Name: user_trust_signals; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE user_trust_signals TO authenticated;

--
-- Name: user_trust_signals; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE user_trust_signals TO {{APP_USER}};

--
-- Name: user_trust_signals; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: user_trust_signals; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE user_trust_signals TO service_role;

--
-- Name: users; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE users TO authenticated;

--
-- Name: users; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE users TO {{APP_USER}};

--
-- Name: users; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: users; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE users TO service_role;

--
-- Name: users; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE users TO tenant_service;

--
-- Name: webhook_deliveries; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE webhook_deliveries TO authenticated;

--
-- Name: webhook_deliveries; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE webhook_deliveries TO {{APP_USER}};

--
-- Name: webhook_deliveries; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: webhook_deliveries; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE webhook_deliveries TO service_role;

--
-- Name: webhook_events; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE webhook_events TO authenticated;

--
-- Name: webhook_events; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE webhook_events TO {{APP_USER}};

--
-- Name: webhook_events; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: webhook_events; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE webhook_events TO service_role;

--
-- Name: webhook_monitored_tables; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE webhook_monitored_tables TO authenticated;

--
-- Name: webhook_monitored_tables; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE webhook_monitored_tables TO {{APP_USER}};

--
-- Name: webhook_monitored_tables; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: webhook_monitored_tables; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE webhook_monitored_tables TO service_role;

--
-- Name: webhooks; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE webhooks TO authenticated;

--
-- Name: webhooks; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE webhooks TO {{APP_USER}};

--
-- Name: webhooks; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: webhooks; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE webhooks TO service_role;

--
-- Name: auth; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT USAGE ON SCHEMA auth TO tenant_service;

--
-- Name: webhooks; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.webhooks TO tenant_service;

--
-- Name: webhook_deliveries; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.webhook_deliveries TO tenant_service;

--
-- Name: webhook_events; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.webhook_events TO tenant_service;

--
-- Name: increment_webhook_table_count(text, text); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION auth.increment_webhook_table_count(text, text) TO tenant_service;

--
-- Name: decrement_webhook_table_count(text, text); Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT EXECUTE ON FUNCTION auth.decrement_webhook_table_count(text, text) TO tenant_service;

--
-- Name: client_keys; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.client_keys TO tenant_service;

--
-- Name: impersonation_sessions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.impersonation_sessions TO tenant_service;

-- ============================================================================
-- TRIGGER FUNCTIONS: tenant_id auto-population with user_id fallback
-- ============================================================================

CREATE OR REPLACE FUNCTION set_tenant_id_from_user_or_context()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    ctx_tenant TEXT;
BEGIN
    BEGIN
        ctx_tenant := current_setting('app.current_tenant_id', true);
        IF ctx_tenant IS NOT NULL AND ctx_tenant <> '' THEN
            NEW.tenant_id := ctx_tenant::UUID;
            RETURN NEW;
        END IF;
    EXCEPTION WHEN others THEN NULL;
    END;

    IF NEW.tenant_id IS NULL AND NEW.user_id IS NOT NULL THEN
        SELECT tenant_id INTO NEW.tenant_id FROM auth.users WHERE id = NEW.user_id;
    END IF;

    IF NEW.tenant_id IS NULL THEN
        BEGIN
            IF NEW.registered_by IS NOT NULL THEN
                SELECT tenant_id INTO NEW.tenant_id FROM auth.users WHERE id = NEW.registered_by;
            END IF;
        EXCEPTION WHEN undefined_column THEN NULL;
        END;
    END IF;

    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION set_tenant_id_from_client_key_or_context()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    ctx_tenant TEXT;
BEGIN
    BEGIN
        ctx_tenant := current_setting('app.current_tenant_id', true);
        IF ctx_tenant IS NOT NULL AND ctx_tenant <> '' THEN
            NEW.tenant_id := ctx_tenant::UUID;
            RETURN NEW;
        END IF;
    EXCEPTION WHEN others THEN NULL;
    END;

    IF NEW.tenant_id IS NULL AND NEW.client_key_id IS NOT NULL THEN
        SELECT tenant_id INTO NEW.tenant_id FROM auth.client_keys WHERE id = NEW.client_key_id;
    END IF;

    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION set_tenant_id_from_service_key_or_context()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    ctx_tenant TEXT;
BEGIN
    BEGIN
        ctx_tenant := current_setting('app.current_tenant_id', true);
        IF ctx_tenant IS NOT NULL AND ctx_tenant <> '' THEN
            NEW.tenant_id := ctx_tenant::UUID;
            RETURN NEW;
        END IF;
    EXCEPTION WHEN others THEN NULL;
    END;

    IF NEW.tenant_id IS NULL AND NEW.key_id IS NOT NULL THEN
        SELECT tenant_id INTO NEW.tenant_id FROM auth.service_keys WHERE id = NEW.key_id;
    END IF;

    RETURN NEW;
END;
$$;

-- ============================================================================
-- TENANT RLS POLICIES + GRANTS for tables with new tenant_id
-- ============================================================================

CREATE POLICY sessions_tenant ON sessions TO PUBLIC USING (has_tenant_access(tenant_id));
CREATE POLICY oauth_links_tenant ON oauth_links TO PUBLIC USING (has_tenant_access(tenant_id));
CREATE POLICY oauth_tokens_tenant ON oauth_tokens TO PUBLIC USING (has_tenant_access(tenant_id)) WITH CHECK (has_tenant_access(tenant_id));
CREATE POLICY mfa_factors_tenant ON mfa_factors TO PUBLIC USING (has_tenant_access(tenant_id)) WITH CHECK (has_tenant_access(tenant_id));
CREATE POLICY saml_sessions_tenant ON saml_sessions TO PUBLIC USING (has_tenant_access(tenant_id)) WITH CHECK (has_tenant_access(tenant_id));
CREATE POLICY magic_links_tenant ON magic_links TO PUBLIC USING (has_tenant_access(tenant_id)) WITH CHECK (has_tenant_access(tenant_id));
CREATE POLICY otp_codes_tenant ON otp_codes TO PUBLIC USING (has_tenant_access(tenant_id)) WITH CHECK (has_tenant_access(tenant_id));
CREATE POLICY email_verification_tokens_tenant ON email_verification_tokens TO PUBLIC USING (has_tenant_access(tenant_id));
CREATE POLICY password_reset_tokens_tenant ON password_reset_tokens TO PUBLIC USING (has_tenant_access(tenant_id));
CREATE POLICY two_factor_setups_tenant ON two_factor_setups TO PUBLIC USING (has_tenant_access(tenant_id)) WITH CHECK (has_tenant_access(tenant_id));
CREATE POLICY two_factor_recovery_attempts_tenant ON two_factor_recovery_attempts TO PUBLIC USING (has_tenant_access(tenant_id));
CREATE POLICY oauth_logout_states_tenant ON oauth_logout_states TO PUBLIC USING (has_tenant_access(tenant_id));
CREATE POLICY mcp_oauth_clients_tenant ON mcp_oauth_clients TO PUBLIC USING (has_tenant_access(tenant_id)) WITH CHECK (has_tenant_access(tenant_id));
CREATE POLICY mcp_oauth_codes_tenant ON mcp_oauth_codes TO PUBLIC USING (has_tenant_access(tenant_id)) WITH CHECK (has_tenant_access(tenant_id));
CREATE POLICY mcp_oauth_tokens_tenant ON mcp_oauth_tokens TO PUBLIC USING (has_tenant_access(tenant_id)) WITH CHECK (has_tenant_access(tenant_id));
CREATE POLICY client_key_usage_tenant ON client_key_usage TO PUBLIC USING (current_user_role() IN ('service_role', 'tenant_service') AND has_tenant_access(tenant_id));
CREATE POLICY service_key_revocations_tenant ON service_key_revocations TO PUBLIC USING (current_user_role() IN ('service_role', 'tenant_service', 'instance_admin') AND has_tenant_access(tenant_id));

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.sessions TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.oauth_links TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.oauth_tokens TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.mfa_factors TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.saml_sessions TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.magic_links TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.otp_codes TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.email_verification_tokens TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.password_reset_tokens TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.two_factor_setups TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.two_factor_recovery_attempts TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.oauth_logout_states TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.mcp_oauth_clients TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.mcp_oauth_codes TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.mcp_oauth_tokens TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.client_key_usage TO tenant_service;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE auth.service_key_revocations TO tenant_service;

