--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 18.3
-- Dumped by pgschema version 1.7.4

SET search_path TO app, public;


--
-- Name: settings; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS settings (
    id uuid DEFAULT gen_random_uuid(),
    key text NOT NULL,
    value jsonb NOT NULL,
    value_type text DEFAULT 'string' NOT NULL,
    category text DEFAULT 'custom' NOT NULL,
    description text,
    is_public boolean DEFAULT false,
    is_secret boolean DEFAULT false,
    editable_by text[] DEFAULT ARRAY['instance_admin'] NOT NULL,
    metadata jsonb DEFAULT '{}',
    created_by uuid,
    updated_by uuid,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    encrypted_value text,
    user_id uuid,
    tenant_id uuid,
    CONSTRAINT settings_pkey PRIMARY KEY (id),
    CONSTRAINT settings_category_check CHECK (category IN ('auth'::text, 'system'::text, 'storage'::text, 'functions'::text, 'realtime'::text, 'custom'::text)),
    CONSTRAINT settings_value_type_check CHECK (value_type IN ('string'::text, 'number'::text, 'boolean'::text, 'json'::text, 'array'::text))
);


COMMENT ON TABLE settings IS 'Application-level configuration and settings with flexible key-value storage';


COMMENT ON COLUMN app.settings.key IS 'Unique setting key (e.g., "jwt_secret", "max_upload_size")';


COMMENT ON COLUMN app.settings.value IS 'Setting value stored as JSONB for flexibility';


COMMENT ON COLUMN app.settings.value_type IS 'Type hint for the value: string, number, boolean, json, or array';


COMMENT ON COLUMN app.settings.category IS 'Category of setting: auth, system, storage, functions, realtime, or custom';


COMMENT ON COLUMN app.settings.is_public IS 'Whether this setting can be read by public/anon users';


COMMENT ON COLUMN app.settings.is_secret IS 'Whether this setting contains sensitive data (e.g., client keys, secrets)';


COMMENT ON COLUMN app.settings.editable_by IS 'Array of roles that can edit this setting';


COMMENT ON COLUMN app.settings.metadata IS 'Additional metadata about the setting (validation rules, UI hints, etc.)';


COMMENT ON COLUMN app.settings.encrypted_value IS 'AES-256-GCM encrypted value (base64). Used when is_secret=true. The value column contains a placeholder.';


COMMENT ON COLUMN app.settings.user_id IS 'Owner user ID for user-specific settings. NULL means system-level setting.';

--
-- Name: idx_app_settings_category; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_app_settings_category ON settings (category);

--
-- Name: idx_app_settings_editable_by; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_app_settings_editable_by ON settings USING gin (editable_by);

--
-- Name: idx_app_settings_encrypted; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_app_settings_encrypted ON settings (is_secret) WHERE (is_secret = true) AND (encrypted_value IS NOT NULL);

--
-- Name: idx_app_settings_is_public; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_app_settings_is_public ON settings (is_public);

--
-- Name: idx_app_settings_key; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_app_settings_key ON settings (key);

--
-- Name: idx_app_settings_key_user; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_app_settings_key_user ON settings (key, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::uuid));

--
-- Name: idx_app_settings_system_key; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_app_settings_system_key ON settings (key) WHERE (user_id IS NULL);

--
-- Name: idx_app_settings_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_app_settings_user_id ON settings (user_id) WHERE (user_id IS NOT NULL);

--
-- Name: idx_app_settings_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_app_settings_tenant_id ON settings (tenant_id);

--
-- Name: settings; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE settings ENABLE ROW LEVEL SECURITY;

--
-- Name: settings; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE settings FORCE ROW LEVEL SECURITY;

--
-- Name: Authenticated users can read non-secret settings; Type: POLICY; Schema: -; Owner: -
--

-- ponytail: these two SELECT policies include auth.has_tenant_access(tenant_id)
-- so they cannot OR-override the tenant-aware settings_tenant policy. Without
-- the predicate, Postgres ORs permissive policies and an authenticated/anon
-- caller could read non-secret/public rows from ANY tenant. Combined with the
-- TenantContext middleware on the public settings route (which sets
-- app.current_tenant_id), reads are now scoped to the caller's tenant, with
-- tenant_id IS NULL treated as the shared instance default.
CREATE POLICY "Authenticated users can read non-secret settings" ON settings FOR SELECT TO authenticated USING ((is_secret = false) AND auth.has_tenant_access(tenant_id));

--
-- Name: Public settings are readable by anyone; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY "Public settings are readable by anyone" ON settings FOR SELECT TO anon, authenticated USING ((is_public = true) AND (is_secret = false) AND auth.has_tenant_access(tenant_id));

--
-- Name: Settings can be created by authorized roles; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY "Settings can be created by authorized roles" ON settings FOR INSERT TO authenticated WITH CHECK ((auth.current_user_role() = 'service_role') OR (auth.current_user_role() = ANY (editable_by)));

--
-- Name: Settings can be deleted by authorized roles; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY "Settings can be deleted by authorized roles" ON settings FOR DELETE TO authenticated USING ((auth.current_user_role() = 'service_role') OR (auth.current_user_role() = ANY (editable_by)));

--
-- Name: Settings can be updated by authorized roles; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY "Settings can be updated by authorized roles" ON settings FOR UPDATE TO authenticated USING ((auth.current_user_role() = 'service_role') OR (auth.current_user_role() = ANY (editable_by))) WITH CHECK ((auth.current_user_role() = 'service_role') OR (auth.current_user_role() = ANY (editable_by)));

--
-- Name: Users can create their own settings; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY "Users can create their own settings" ON settings FOR INSERT TO authenticated WITH CHECK (user_id = auth.current_user_id());

--
-- Name: Users can delete their own settings; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY "Users can delete their own settings" ON settings FOR DELETE TO authenticated USING (user_id = auth.current_user_id());

--
-- Name: Users can read their own secret settings; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY "Users can read their own secret settings" ON settings FOR SELECT TO authenticated USING (user_id = auth.current_user_id());

--
-- Name: Users can update their own settings; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY "Users can update their own settings" ON settings FOR UPDATE TO authenticated USING (user_id = auth.current_user_id()) WITH CHECK (user_id = auth.current_user_id());

--
-- Name: settings_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY settings_tenant ON settings FOR SELECT TO PUBLIC USING (auth.has_tenant_access(tenant_id) AND (NOT is_secret OR user_id = auth.current_user_id()));

--
-- Name: settings_tenant_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY settings_tenant_service ON settings TO tenant_service USING (auth.has_tenant_access(tenant_id)) WITH CHECK (auth.has_tenant_access(tenant_id));

--
-- Cross-schema FKs moved to post-schema-fks.sql
-- settings_user_id_fkey
--

--
-- Name: settings; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT SELECT ON TABLE settings TO anon;

--
-- Name: settings; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE settings TO authenticated;

--
-- Name: settings; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE settings TO {{APP_USER}};

--
-- Name: settings; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: settings; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE settings TO service_role;

--
-- Name: app; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT USAGE ON SCHEMA app TO tenant_service;

--
-- Name: settings; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE app.settings TO tenant_service;

