--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 18.3
-- Dumped by pgschema version 1.7.4

SET search_path TO functions, public;


--
-- Name: edge_functions; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS edge_functions (
    id uuid DEFAULT gen_random_uuid(),
    name text NOT NULL,
    namespace text DEFAULT 'default' NOT NULL,
    description text,
    code text NOT NULL,
    original_code text,
    is_bundled boolean DEFAULT false NOT NULL,
    bundle_error text,
    enabled boolean DEFAULT true,
    timeout_seconds integer DEFAULT 30,
    memory_limit_mb integer DEFAULT 128,
    allow_net boolean DEFAULT true,
    allow_env boolean DEFAULT true,
    allow_read boolean DEFAULT false,
    allow_write boolean DEFAULT false,
    allow_unauthenticated boolean DEFAULT false,
    is_public boolean DEFAULT true,
    cron_schedule text,
    version integer DEFAULT 1,
    created_by uuid,
    source text DEFAULT 'filesystem' NOT NULL,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    needs_rebundle boolean DEFAULT false,
    cors_origins text,
    cors_methods text,
    cors_headers text,
    cors_credentials boolean,
    cors_max_age integer,
    disable_execution_logs boolean DEFAULT false NOT NULL,
    rate_limit_per_minute integer,
    rate_limit_per_hour integer,
    rate_limit_per_day integer,
    tenant_id uuid,
    CONSTRAINT edge_functions_pkey PRIMARY KEY (id)
);

ALTER TABLE edge_functions DROP CONSTRAINT IF EXISTS unique_function_name_namespace;

CREATE UNIQUE INDEX IF NOT EXISTS unique_function_name_namespace_tenant
    ON edge_functions (tenant_id, name, namespace) WHERE tenant_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS unique_function_name_namespace_default
    ON edge_functions (name, namespace) WHERE tenant_id IS NULL;


COMMENT ON COLUMN functions.edge_functions.namespace IS 'Namespace for isolating functions across different apps/deployments. Functions with same name can exist in different namespaces.';


COMMENT ON COLUMN functions.edge_functions.original_code IS 'Original source code before bundling (for editing in UI)';


COMMENT ON COLUMN functions.edge_functions.is_bundled IS 'Whether the code field contains bundled output with dependencies';


COMMENT ON COLUMN functions.edge_functions.bundle_error IS 'Error message if bundling failed (function still works with unbundled code)';


COMMENT ON COLUMN functions.edge_functions.allow_unauthenticated IS 'When true, allows this function to be invoked without authentication. Use with caution.';


COMMENT ON COLUMN functions.edge_functions.is_public IS 'Whether the function is publicly listed in the functions directory. Private functions can still be invoked if the name is known.';


COMMENT ON COLUMN functions.edge_functions.source IS 'Source of function: filesystem or api';


COMMENT ON COLUMN functions.edge_functions.needs_rebundle IS 'Flag indicating the function needs rebundling due to shared module updates';


COMMENT ON COLUMN functions.edge_functions.cors_origins IS 'Comma-separated list of allowed CORS origins (NULL means use global config)';


COMMENT ON COLUMN functions.edge_functions.cors_methods IS 'Comma-separated list of allowed CORS methods (NULL means use global config)';


COMMENT ON COLUMN functions.edge_functions.cors_headers IS 'Comma-separated list of allowed CORS headers (NULL means use global config)';


COMMENT ON COLUMN functions.edge_functions.cors_credentials IS 'Allow credentials in CORS requests (NULL means use global config)';


COMMENT ON COLUMN functions.edge_functions.cors_max_age IS 'Max age for CORS preflight cache in seconds (NULL means use global config)';


COMMENT ON COLUMN functions.edge_functions.disable_execution_logs IS 'When true, execution logs are not created for this function (from @fluxbase:disable-execution-logs annotation)';


COMMENT ON COLUMN functions.edge_functions.rate_limit_per_minute IS 'Maximum requests per minute per user/IP. NULL means unlimited.';


COMMENT ON COLUMN functions.edge_functions.rate_limit_per_hour IS 'Maximum requests per hour per user/IP. NULL means unlimited.';


COMMENT ON COLUMN functions.edge_functions.rate_limit_per_day IS 'Maximum requests per day per user/IP. NULL means unlimited.';

--
-- Name: idx_functions_edge_functions_cron_schedule; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_cron_schedule ON edge_functions (cron_schedule) WHERE (cron_schedule IS NOT NULL);

--
-- Name: idx_functions_edge_functions_enabled; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_enabled ON edge_functions (enabled);

--
-- Name: idx_functions_edge_functions_is_public; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_is_public ON edge_functions (is_public);

--
-- Name: idx_functions_edge_functions_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_name ON edge_functions (name);

--
-- Name: idx_functions_edge_functions_namespace; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_namespace ON edge_functions (namespace);

--
-- Name: idx_functions_namespace; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_functions_namespace ON edge_functions (namespace);

--
-- Name: edge_functions; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE edge_functions ENABLE ROW LEVEL SECURITY;

--
-- Name: edge_functions; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE edge_functions FORCE ROW LEVEL SECURITY;

--
-- Name: functions_edge_functions_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_edge_functions_admin ON edge_functions TO PUBLIC USING (auth.current_user_role() = 'instance_admin') WITH CHECK (auth.current_user_role() = 'instance_admin');

--
-- Name: functions_edge_functions_owner; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_edge_functions_owner ON edge_functions TO PUBLIC USING ((auth.current_user_id() IS NOT NULL) AND (created_by = auth.current_user_id())) WITH CHECK ((auth.current_user_id() IS NOT NULL) AND (created_by = auth.current_user_id()));

--
-- Name: functions_edge_functions_public_read; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_edge_functions_public_read ON edge_functions FOR SELECT TO PUBLIC USING ((is_public = true) AND (enabled = true));

--
-- Name: functions_edge_functions_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_edge_functions_service ON edge_functions TO PUBLIC USING (auth.current_user_role() = 'service_role') WITH CHECK (auth.current_user_role() = 'service_role');

--
-- Name: edge_executions; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS edge_executions (
    id uuid DEFAULT gen_random_uuid(),
    function_id uuid NOT NULL,
    trigger_type text NOT NULL,
    status text NOT NULL,
    status_code integer,
    error_message text,
    logs text,
    result text,
    duration_ms integer,
    started_at timestamptz DEFAULT now(),
    completed_at timestamptz,
    tenant_id uuid,
    CONSTRAINT edge_executions_pkey PRIMARY KEY (id),
    CONSTRAINT edge_executions_function_id_fkey FOREIGN KEY (function_id) REFERENCES edge_functions (id) ON DELETE CASCADE
);

--
-- Name: idx_functions_edge_executions_function_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_functions_edge_executions_function_id ON edge_executions (function_id);

--
-- Name: idx_functions_edge_executions_started_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_functions_edge_executions_started_at ON edge_executions (started_at DESC);

--
-- Name: idx_functions_edge_executions_status; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_functions_edge_executions_status ON edge_executions (status);

--
-- Name: edge_executions; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE edge_executions ENABLE ROW LEVEL SECURITY;

--
-- Name: edge_executions; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE edge_executions FORCE ROW LEVEL SECURITY;

--
-- Name: functions_edge_executions_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_edge_executions_admin ON edge_executions TO PUBLIC USING (auth.current_user_role() = 'instance_admin') WITH CHECK (auth.current_user_role() = 'instance_admin');

--
-- Name: functions_edge_executions_owner; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_edge_executions_owner ON edge_executions FOR SELECT TO PUBLIC USING ((auth.current_user_id() IS NOT NULL) AND (EXISTS ( SELECT 1 FROM edge_functions ef WHERE ((ef.id = edge_executions.function_id) AND (ef.created_by = auth.current_user_id())))));

--
-- Name: functions_edge_executions_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_edge_executions_service ON edge_executions TO PUBLIC USING (auth.current_user_role() = 'service_role') WITH CHECK (auth.current_user_role() = 'service_role');

--
-- Name: edge_files; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS edge_files (
    id uuid DEFAULT gen_random_uuid(),
    function_id uuid NOT NULL,
    file_path text NOT NULL,
    content text NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    tenant_id uuid,
    CONSTRAINT edge_files_pkey PRIMARY KEY (id),
    CONSTRAINT unique_edge_file_path UNIQUE (function_id, file_path),
    CONSTRAINT edge_files_function_id_fkey FOREIGN KEY (function_id) REFERENCES edge_functions (id) ON DELETE CASCADE,
    CONSTRAINT valid_file_path CHECK (file_path ~ '^[a-zA-Z0-9_/-]+\.(ts|js|mts|mjs)$'::text AND file_path !~~ '../%'::text AND file_path !~~ '%/../%'::text)
);


COMMENT ON TABLE edge_files IS 'Supporting files for edge functions (utils, helpers, types)';

--
-- Name: idx_edge_files_function_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_edge_files_function_id ON edge_files (function_id);

--
-- Name: edge_files; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE edge_files ENABLE ROW LEVEL SECURITY;

--
-- Name: edge_files; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE edge_files FORCE ROW LEVEL SECURITY;

--
-- Name: functions_edge_files_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_edge_files_admin ON edge_files TO PUBLIC USING (auth.current_user_role() = 'instance_admin') WITH CHECK (auth.current_user_role() = 'instance_admin');

--
-- Name: functions_edge_files_owner; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_edge_files_owner ON edge_files TO PUBLIC USING ((auth.current_user_id() IS NOT NULL) AND (EXISTS ( SELECT 1 FROM edge_functions ef WHERE ((ef.id = edge_files.function_id) AND (ef.created_by = auth.current_user_id()))))) WITH CHECK ((auth.current_user_id() IS NOT NULL) AND (EXISTS ( SELECT 1 FROM edge_functions ef WHERE ((ef.id = edge_files.function_id) AND (ef.created_by = auth.current_user_id())))));

--
-- Name: functions_edge_files_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_edge_files_service ON edge_files TO PUBLIC USING (auth.current_user_role() = 'service_role') WITH CHECK (auth.current_user_role() = 'service_role');

--
-- Name: edge_triggers; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS edge_triggers (
    id uuid DEFAULT gen_random_uuid(),
    function_id uuid NOT NULL,
    trigger_type text NOT NULL,
    schema_name text,
    table_name text,
    events text[] DEFAULT ARRAY[]::text[],
    enabled boolean DEFAULT true,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT edge_triggers_pkey PRIMARY KEY (id),
    CONSTRAINT edge_triggers_function_id_fkey FOREIGN KEY (function_id) REFERENCES edge_functions (id) ON DELETE CASCADE
);

--
-- Name: idx_functions_edge_triggers_enabled; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_functions_edge_triggers_enabled ON edge_triggers (enabled);

--
-- Name: idx_functions_edge_triggers_function_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_functions_edge_triggers_function_id ON edge_triggers (function_id);

--
-- Name: idx_functions_edge_triggers_table; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_functions_edge_triggers_table ON edge_triggers (schema_name, table_name);

--
-- Name: edge_triggers; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE edge_triggers ENABLE ROW LEVEL SECURITY;

--
-- Name: edge_triggers; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE edge_triggers FORCE ROW LEVEL SECURITY;

--
-- Name: functions_edge_triggers_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_edge_triggers_admin ON edge_triggers TO PUBLIC USING (auth.current_user_role() = 'instance_admin') WITH CHECK (auth.current_user_role() = 'instance_admin');

--
-- Name: functions_edge_triggers_owner; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_edge_triggers_owner ON edge_triggers TO PUBLIC USING ((auth.current_user_id() IS NOT NULL) AND (EXISTS ( SELECT 1 FROM edge_functions ef WHERE ((ef.id = edge_triggers.function_id) AND (ef.created_by = auth.current_user_id()))))) WITH CHECK ((auth.current_user_id() IS NOT NULL) AND (EXISTS ( SELECT 1 FROM edge_functions ef WHERE ((ef.id = edge_triggers.function_id) AND (ef.created_by = auth.current_user_id())))));

--
-- Name: functions_edge_triggers_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_edge_triggers_service ON edge_triggers TO PUBLIC USING (auth.current_user_role() = 'service_role') WITH CHECK (auth.current_user_role() = 'service_role');

--
-- Name: secrets; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS secrets (
    id uuid DEFAULT gen_random_uuid(),
    name text NOT NULL,
    scope text DEFAULT 'global' NOT NULL,
    namespace text,
    encrypted_value text NOT NULL,
    description text,
    version integer DEFAULT 1 NOT NULL,
    expires_at timestamptz,
    tenant_id uuid,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    created_by uuid,
    updated_by uuid,
    CONSTRAINT secrets_pkey PRIMARY KEY (id),
    CONSTRAINT secrets_scope_check CHECK (scope IN ('global'::text, 'namespace'::text))
    -- Cross-schema FK secrets_tenant_id_fkey moved to post-schema-fks.sql
);


COMMENT ON TABLE secrets IS 'Encrypted secrets injected into edge functions at runtime';


COMMENT ON COLUMN functions.secrets.scope IS 'Scope: global (all functions) or namespace (functions in specific namespace)';


COMMENT ON COLUMN functions.secrets.encrypted_value IS 'AES-256-GCM encrypted secret value (base64 with prepended nonce)';


COMMENT ON COLUMN functions.secrets.version IS 'Incremented on each update for tracking changes';

--
-- Name: idx_secrets_expires_at; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_secrets_expires_at ON secrets (expires_at) WHERE (expires_at IS NOT NULL);

--
-- Name: idx_secrets_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_secrets_name ON secrets (name);

--
-- Name: idx_secrets_namespace; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_secrets_namespace ON secrets (namespace) WHERE (namespace IS NOT NULL);

--
-- Name: idx_secrets_scope; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_secrets_scope ON secrets (scope);

--
-- Name: idx_secrets_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_secrets_tenant_id ON secrets (tenant_id) WHERE (tenant_id IS NOT NULL);

--
-- Name: unique_secrets_global_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS unique_secrets_global_name ON secrets (name) WHERE (scope = 'global'::text) AND (namespace IS NULL);

--
-- Name: unique_secrets_namespace_name; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS unique_secrets_namespace_name ON secrets (name, namespace) WHERE (scope = 'namespace'::text) AND (namespace IS NOT NULL);

--
-- Name: secrets; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE secrets ENABLE ROW LEVEL SECURITY;

--
-- Name: secrets; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE secrets FORCE ROW LEVEL SECURITY;

--
-- Name: secrets_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY secrets_tenant ON secrets TO PUBLIC USING (auth.has_tenant_access(tenant_id)) WITH CHECK (auth.has_tenant_access(tenant_id));

--
-- Name: secret_versions; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS secret_versions (
    id uuid DEFAULT gen_random_uuid(),
    secret_id uuid NOT NULL,
    version integer NOT NULL,
    encrypted_value text NOT NULL,
    tenant_id uuid,
    created_at timestamptz DEFAULT now() NOT NULL,
    created_by uuid,
    CONSTRAINT secret_versions_pkey PRIMARY KEY (id),
    CONSTRAINT unique_secret_version UNIQUE (secret_id, version),
    CONSTRAINT secret_versions_secret_id_fkey FOREIGN KEY (secret_id) REFERENCES secrets (id) ON DELETE CASCADE
);


COMMENT ON TABLE secret_versions IS 'Version history for secrets (audit trail and rollback capability)';

--
-- Name: idx_secret_versions_secret_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_secret_versions_secret_id ON secret_versions (secret_id);

--
-- Name: idx_secret_versions_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_secret_versions_tenant_id ON secret_versions (tenant_id) WHERE (tenant_id IS NOT NULL);

--
-- Name: secret_versions; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE secret_versions ENABLE ROW LEVEL SECURITY;

--
-- Name: secret_versions; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE secret_versions FORCE ROW LEVEL SECURITY;

--
-- Name: secret_versions_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY secret_versions_tenant ON secret_versions TO PUBLIC USING (auth.has_tenant_access(tenant_id)) WITH CHECK (auth.has_tenant_access(tenant_id));

--
-- Name: shared_modules; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS shared_modules (
    id uuid DEFAULT gen_random_uuid(),
    module_path text NOT NULL,
    content text NOT NULL,
    description text,
    version integer DEFAULT 1 NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    created_by uuid,
    tenant_id uuid,
    CONSTRAINT shared_modules_pkey PRIMARY KEY (id),
    CONSTRAINT shared_modules_module_path_key UNIQUE (module_path),
    CONSTRAINT valid_module_path CHECK (module_path ~ '^_shared/[a-zA-Z0-9_/-]+\.(ts|js|mts|mjs)$'::text AND module_path !~~ '%/../%'::text)
);


COMMENT ON TABLE shared_modules IS 'Shared modules accessible by all edge functions (_shared/*)';

--
-- Name: idx_shared_modules_module_path; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_shared_modules_module_path ON shared_modules (module_path);

--
-- Name: shared_modules; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE shared_modules ENABLE ROW LEVEL SECURITY;

--
-- Name: shared_modules; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE shared_modules FORCE ROW LEVEL SECURITY;

--
-- Name: functions_shared_modules_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_shared_modules_admin ON shared_modules TO PUBLIC USING (auth.current_user_role() = 'instance_admin') WITH CHECK (auth.current_user_role() = 'instance_admin');

--
-- Name: functions_shared_modules_owner; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_shared_modules_owner ON shared_modules TO PUBLIC USING ((auth.current_user_id() IS NOT NULL) AND (created_by = auth.current_user_id())) WITH CHECK ((auth.current_user_id() IS NOT NULL) AND (created_by = auth.current_user_id()));

--
-- Name: functions_shared_modules_read; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_shared_modules_read ON shared_modules FOR SELECT TO PUBLIC USING (auth.current_user_id() IS NOT NULL);

--
-- Name: functions_shared_modules_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_shared_modules_service ON shared_modules TO PUBLIC USING (auth.current_user_role() = 'service_role') WITH CHECK (auth.current_user_role() = 'service_role');

--
-- Name: function_dependencies; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS function_dependencies (
    id uuid DEFAULT gen_random_uuid(),
    function_id uuid NOT NULL,
    shared_module_id uuid NOT NULL,
    shared_module_version integer NOT NULL,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    tenant_id uuid,
    CONSTRAINT function_dependencies_pkey PRIMARY KEY (id),
    CONSTRAINT function_dependencies_function_id_shared_module_id_key UNIQUE (function_id, shared_module_id),
    CONSTRAINT function_dependencies_function_id_fkey FOREIGN KEY (function_id) REFERENCES edge_functions (id) ON DELETE CASCADE,
    CONSTRAINT function_dependencies_shared_module_id_fkey FOREIGN KEY (shared_module_id) REFERENCES shared_modules (id) ON DELETE CASCADE
);


COMMENT ON TABLE function_dependencies IS 'Tracks which edge functions depend on which shared modules for automatic rebundling';

--
-- Name: idx_function_dependencies_function_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_function_dependencies_function_id ON function_dependencies (function_id);

--
-- Name: idx_function_dependencies_shared_module_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_function_dependencies_shared_module_id ON function_dependencies (shared_module_id);

--
-- Name: function_dependencies; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE function_dependencies ENABLE ROW LEVEL SECURITY;

--
-- Name: function_dependencies; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE function_dependencies FORCE ROW LEVEL SECURITY;

--
-- Name: functions_dependencies_admin; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_dependencies_admin ON function_dependencies TO PUBLIC USING (auth.current_user_role() = 'instance_admin') WITH CHECK (auth.current_user_role() = 'instance_admin');

--
-- Name: functions_dependencies_owner; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_dependencies_owner ON function_dependencies TO PUBLIC USING ((auth.current_user_id() IS NOT NULL) AND (EXISTS ( SELECT 1 FROM edge_functions ef WHERE ((ef.id = function_dependencies.function_id) AND (ef.created_by = auth.current_user_id()))))) WITH CHECK ((auth.current_user_id() IS NOT NULL) AND (EXISTS ( SELECT 1 FROM edge_functions ef WHERE ((ef.id = function_dependencies.function_id) AND (ef.created_by = auth.current_user_id())))));

--
-- Name: functions_dependencies_service; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY functions_dependencies_service ON function_dependencies TO PUBLIC USING (auth.current_user_role() = 'service_role') WITH CHECK (auth.current_user_role() = 'service_role');

--
-- Name: mark_dependent_functions_for_rebundle(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION mark_dependent_functions_for_rebundle()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    -- When a shared module is updated, mark all dependent functions for rebundling
    UPDATE edge_functions
    SET needs_rebundle = TRUE
    WHERE id IN (
        SELECT function_id
        FROM function_dependencies
        WHERE shared_module_id = NEW.id
    );
    RETURN NEW;
END;
$$;

--
-- Name: mark_dependent_functions_for_rebundle(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION mark_dependent_functions_for_rebundle() IS 'Marks all functions that depend on a shared module for rebundling when the module is updated';

--
-- Name: update_function_dependencies_updated_at(); Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_function_dependencies_updated_at()
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
-- Name: update_function_dependencies_updated_at(); Type: FUNCTION; Schema: -; Owner: -
--

COMMENT ON FUNCTION update_function_dependencies_updated_at() IS 'Updates the updated_at timestamp for function_dependencies table';

--
-- Cross-schema FKs moved to post-schema-fks.sql
-- secrets_created_by_fkey, secrets_updated_by_fkey, secret_versions_created_by_fkey, shared_modules_created_by_fkey
--

--
-- Name: trigger_mark_functions_on_shared_module_update; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER trigger_mark_functions_on_shared_module_update
    AFTER UPDATE ON shared_modules
    FOR EACH ROW
    WHEN ((((OLD.content IS DISTINCT FROM NEW.content) OR (OLD.version IS DISTINCT FROM NEW.version))))
    EXECUTE FUNCTION mark_dependent_functions_for_rebundle();

--
-- Name: update_function_dependencies_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER update_function_dependencies_updated_at
    BEFORE UPDATE ON function_dependencies
    FOR EACH ROW
    EXECUTE FUNCTION update_function_dependencies_updated_at();

--
-- Name: update_functions_edge_functions_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER update_functions_edge_functions_updated_at
    BEFORE UPDATE ON edge_functions
    FOR EACH ROW
    EXECUTE FUNCTION platform.update_updated_at();

--
-- Name: update_functions_edge_triggers_updated_at; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER update_functions_edge_triggers_updated_at
    BEFORE UPDATE ON edge_triggers
    FOR EACH ROW
    EXECUTE FUNCTION platform.update_updated_at();

--
-- Name: functions_secrets_set_tenant_id; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER functions_secrets_set_tenant_id
    BEFORE INSERT ON secrets
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

--
-- Name: functions_secret_versions_set_tenant_id; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER functions_secret_versions_set_tenant_id
    BEFORE INSERT ON secret_versions
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

--
-- Multi-tenancy: Add tenant_id column, tenant policy, and auto-populate trigger
-- for all functions tables (secrets and secret_versions already handled above)
--

-- edge_functions
CREATE INDEX IF NOT EXISTS idx_functions_edge_functions_tenant_id ON edge_functions (tenant_id);

CREATE POLICY functions_edge_functions_tenant ON edge_functions TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER functions_edge_functions_set_tenant_id
    BEFORE INSERT ON edge_functions
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- edge_executions
CREATE INDEX IF NOT EXISTS idx_functions_edge_executions_tenant_id ON edge_executions (tenant_id);

CREATE POLICY functions_edge_executions_tenant ON edge_executions TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER functions_edge_executions_set_tenant_id
    BEFORE INSERT ON edge_executions
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

CREATE OR REPLACE FUNCTION notify_realtime_change()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
AS $$
DECLARE
  notification_record JSONB;
  old_notification_record JSONB;
BEGIN
  IF TG_OP != 'DELETE' THEN
    notification_record := to_jsonb(NEW) - 'logs';
  END IF;
  IF TG_OP != 'INSERT' THEN
    old_notification_record := to_jsonb(OLD) - 'logs';
  END IF;

  PERFORM pg_notify(
    'fluxbase_changes',
    json_build_object(
      'schema', TG_TABLE_SCHEMA,
      'table', TG_TABLE_NAME,
      'type', TG_OP,
      'record', notification_record,
      'old_record', old_notification_record
    )::text
  );
  RETURN COALESCE(NEW, OLD);
END;
$$;

CREATE OR REPLACE TRIGGER edge_executions_realtime_notify
    AFTER INSERT OR UPDATE OR DELETE ON edge_executions
    FOR EACH ROW
    EXECUTE FUNCTION notify_realtime_change();

-- edge_files
CREATE INDEX IF NOT EXISTS idx_functions_edge_files_tenant_id ON edge_files (tenant_id);

CREATE POLICY functions_edge_files_tenant ON edge_files TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER functions_edge_files_set_tenant_id
    BEFORE INSERT ON edge_files
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- edge_triggers
CREATE INDEX IF NOT EXISTS idx_functions_edge_triggers_tenant_id ON edge_triggers (tenant_id);

CREATE POLICY functions_edge_triggers_tenant ON edge_triggers TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER functions_edge_triggers_set_tenant_id
    BEFORE INSERT ON edge_triggers
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- shared_modules
CREATE INDEX IF NOT EXISTS idx_functions_shared_modules_tenant_id ON shared_modules (tenant_id);

CREATE POLICY functions_shared_modules_tenant ON shared_modules TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER functions_shared_modules_set_tenant_id
    BEFORE INSERT ON shared_modules
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

-- function_dependencies
CREATE INDEX IF NOT EXISTS idx_functions_function_dependencies_tenant_id ON function_dependencies (tenant_id);

CREATE POLICY functions_function_dependencies_tenant ON function_dependencies TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

CREATE OR REPLACE TRIGGER functions_function_dependencies_set_tenant_id
    BEFORE INSERT ON function_dependencies
    FOR EACH ROW
    EXECUTE FUNCTION auth.set_tenant_id_from_context();

--
-- Name: edge_executions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE edge_executions TO authenticated;

--
-- Name: edge_executions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE edge_executions TO {{APP_USER}};

--
-- Name: edge_executions; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: edge_executions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE edge_executions TO service_role;

--
-- Name: edge_files; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE edge_files TO authenticated;

--
-- Name: edge_files; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE edge_files TO {{APP_USER}};

--
-- Name: edge_files; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: edge_files; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE edge_files TO service_role;

--
-- Name: edge_functions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE edge_functions TO authenticated;

--
-- Name: edge_functions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE edge_functions TO {{APP_USER}};

--
-- Name: edge_functions; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: edge_functions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE edge_functions TO service_role;

--
-- Name: edge_triggers; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE edge_triggers TO authenticated;

--
-- Name: edge_triggers; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE edge_triggers TO {{APP_USER}};

--
-- Name: edge_triggers; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: edge_triggers; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE edge_triggers TO service_role;

--
-- Name: function_dependencies; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE function_dependencies TO authenticated;

--
-- Name: function_dependencies; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE function_dependencies TO {{APP_USER}};

--
-- Name: function_dependencies; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: function_dependencies; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE function_dependencies TO service_role;

--
-- Name: secret_versions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE secret_versions TO authenticated;

--
-- Name: secret_versions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE secret_versions TO {{APP_USER}};

--
-- Name: secret_versions; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: secret_versions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE secret_versions TO service_role;

--
-- Name: secrets; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE secrets TO authenticated;

--
-- Name: secrets; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE secrets TO {{APP_USER}};

--
-- Name: secrets; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: secrets; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE secrets TO service_role;

--
-- Name: shared_modules; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE shared_modules TO authenticated;

--
-- Name: shared_modules; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE shared_modules TO {{APP_USER}};

--
-- Name: shared_modules; Type: PRIVILEGE; Schema: privileges; Owner: -
--


--
-- Name: shared_modules; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE shared_modules TO service_role;

--
-- Name: functions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT USAGE ON SCHEMA functions TO tenant_service;

--
-- Name: edge_functions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE functions.edge_functions TO tenant_service;

--
-- Name: edge_executions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE functions.edge_executions TO tenant_service;

--
-- Name: edge_files; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE functions.edge_files TO tenant_service;

--
-- Name: edge_triggers; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE functions.edge_triggers TO tenant_service;

--
-- Name: secrets; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE functions.secrets TO tenant_service;

--
-- Name: secret_versions; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE functions.secret_versions TO tenant_service;

--
-- Name: shared_modules; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE functions.shared_modules TO tenant_service;

--
-- Name: function_dependencies; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE functions.function_dependencies TO tenant_service;

