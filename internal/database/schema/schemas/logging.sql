--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 18.3
-- Dumped by pgschema version 1.7.4

SET search_path TO logging, public;


--
-- Name: entries; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS entries (
    id uuid DEFAULT gen_random_uuid(),
    timestamp timestamptz DEFAULT now() NOT NULL,
    category text,
    level text NOT NULL,
    message text NOT NULL,
    request_id text,
    trace_id text,
    component text,
    user_id uuid,
    ip_address inet,
    fields jsonb,
    execution_id uuid,
    line_number integer,
    custom_category text,
    tenant_id uuid,
    CONSTRAINT entries_pkey PRIMARY KEY (category, id),
    CONSTRAINT valid_category CHECK (category IN ('system'::text, 'http'::text, 'security'::text, 'execution'::text, 'ai'::text, 'custom'::text)),
    CONSTRAINT valid_level CHECK (level IN ('trace'::text, 'debug'::text, 'info'::text, 'warn'::text, 'error'::text, 'fatal'::text, 'panic'::text))
) PARTITION BY LIST (category);


COMMENT ON TABLE entries IS 'Unified log entries table, partitioned by category';


COMMENT ON COLUMN logging.entries.custom_category IS 'User-defined category name when category=custom';


COMMENT ON COLUMN logging.entries.tenant_id IS 'Tenant this log entry belongs to';

--
-- Name: idx_logging_entries_category_timestamp; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_category_timestamp ON entries (category, "timestamp" DESC);

--
-- Name: idx_logging_entries_component; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_component ON entries (component) WHERE (component IS NOT NULL);

--
-- Name: idx_logging_entries_custom_category; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_custom_category ON entries (custom_category) WHERE (custom_category IS NOT NULL);

--
-- Name: idx_logging_entries_execution_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_execution_id ON entries (execution_id) WHERE (execution_id IS NOT NULL);

--
-- Name: idx_logging_entries_execution_line; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_execution_line ON entries (execution_id, line_number) WHERE (execution_id IS NOT NULL);

--
-- Name: idx_logging_entries_level; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_level ON entries (level);

--
-- Name: idx_logging_entries_message_search; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_message_search ON entries USING gin (to_tsvector('english'::regconfig, message));

--
-- Name: idx_logging_entries_request_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_request_id ON entries (request_id) WHERE (request_id IS NOT NULL);

--
-- Name: idx_logging_entries_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_tenant_id ON entries (tenant_id);

--
-- Name: idx_logging_entries_timestamp; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_timestamp ON entries ("timestamp" DESC);

--
-- Name: idx_logging_entries_trace_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_trace_id ON entries (trace_id) WHERE (trace_id IS NOT NULL);

--
-- Name: idx_logging_entries_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_user_id ON entries (user_id) WHERE (user_id IS NOT NULL);

--
-- Name: entries_ai; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS entries_ai (
    id uuid DEFAULT gen_random_uuid(),
    timestamp timestamptz DEFAULT now() NOT NULL,
    category text,
    level text NOT NULL,
    message text NOT NULL,
    request_id text,
    trace_id text,
    component text,
    user_id uuid,
    ip_address inet,
    fields jsonb,
    execution_id uuid,
    line_number integer,
    custom_category text,
    tenant_id uuid,
    CONSTRAINT entries_ai_pkey PRIMARY KEY (category, id),
    CONSTRAINT valid_category CHECK (category IN ('system'::text, 'http'::text, 'security'::text, 'execution'::text, 'ai'::text, 'custom'::text)),
    CONSTRAINT valid_level CHECK (level IN ('trace'::text, 'debug'::text, 'info'::text, 'warn'::text, 'error'::text, 'fatal'::text, 'panic'::text))
);


COMMENT ON COLUMN logging.entries_ai.tenant_id IS 'Tenant this AI log entry belongs to';

--
-- Name: entries_ai_category_timestamp_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_ai_category_timestamp_idx ON entries_ai (category, "timestamp" DESC);

--
-- Name: entries_ai_component_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_ai_component_idx ON entries_ai (component) WHERE (component IS NOT NULL);

--
-- Name: entries_ai_custom_category_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_ai_custom_category_idx ON entries_ai (custom_category) WHERE (custom_category IS NOT NULL);

--
-- Name: entries_ai_execution_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_ai_execution_id_idx ON entries_ai (execution_id) WHERE (execution_id IS NOT NULL);

--
-- Name: entries_ai_execution_id_line_number_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_ai_execution_id_line_number_idx ON entries_ai (execution_id, line_number) WHERE (execution_id IS NOT NULL);

--
-- Name: entries_ai_level_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_ai_level_idx ON entries_ai (level);

--
-- Name: entries_ai_request_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_ai_request_id_idx ON entries_ai (request_id) WHERE (request_id IS NOT NULL);

--
-- Name: entries_ai_tenant_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_ai_tenant_id_idx ON entries_ai (tenant_id);

--
-- Name: entries_ai_timestamp_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_ai_timestamp_idx ON entries_ai ("timestamp" DESC);

--
-- Name: entries_ai_to_tsvector_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_ai_to_tsvector_idx ON entries_ai USING gin (to_tsvector('english'::regconfig, message));

--
-- Name: entries_ai_trace_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_ai_trace_id_idx ON entries_ai (trace_id) WHERE (trace_id IS NOT NULL);

--
-- Name: entries_ai_user_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_ai_user_id_idx ON entries_ai (user_id) WHERE (user_id IS NOT NULL);

--
-- Name: idx_logging_entries_ai_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_ai_tenant_id ON entries_ai (tenant_id);

--
-- Name: entries_custom; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS entries_custom (
    id uuid DEFAULT gen_random_uuid(),
    timestamp timestamptz DEFAULT now() NOT NULL,
    category text,
    level text NOT NULL,
    message text NOT NULL,
    request_id text,
    trace_id text,
    component text,
    user_id uuid,
    ip_address inet,
    fields jsonb,
    execution_id uuid,
    line_number integer,
    custom_category text,
    tenant_id uuid,
    CONSTRAINT entries_custom_pkey PRIMARY KEY (category, id),
    CONSTRAINT valid_category CHECK (category IN ('system'::text, 'http'::text, 'security'::text, 'execution'::text, 'ai'::text, 'custom'::text)),
    CONSTRAINT valid_level CHECK (level IN ('trace'::text, 'debug'::text, 'info'::text, 'warn'::text, 'error'::text, 'fatal'::text, 'panic'::text))
);


COMMENT ON COLUMN logging.entries_custom.tenant_id IS 'Tenant this custom log entry belongs to';

--
-- Name: entries_custom_category_timestamp_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_custom_category_timestamp_idx ON entries_custom (category, "timestamp" DESC);

--
-- Name: entries_custom_component_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_custom_component_idx ON entries_custom (component) WHERE (component IS NOT NULL);

--
-- Name: entries_custom_custom_category_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_custom_custom_category_idx ON entries_custom (custom_category) WHERE (custom_category IS NOT NULL);

--
-- Name: entries_custom_execution_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_custom_execution_id_idx ON entries_custom (execution_id) WHERE (execution_id IS NOT NULL);

--
-- Name: entries_custom_execution_id_line_number_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_custom_execution_id_line_number_idx ON entries_custom (execution_id, line_number) WHERE (execution_id IS NOT NULL);

--
-- Name: entries_custom_level_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_custom_level_idx ON entries_custom (level);

--
-- Name: entries_custom_request_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_custom_request_id_idx ON entries_custom (request_id) WHERE (request_id IS NOT NULL);

--
-- Name: entries_custom_tenant_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_custom_tenant_id_idx ON entries_custom (tenant_id);

--
-- Name: entries_custom_timestamp_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_custom_timestamp_idx ON entries_custom ("timestamp" DESC);

--
-- Name: entries_custom_to_tsvector_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_custom_to_tsvector_idx ON entries_custom USING gin (to_tsvector('english'::regconfig, message));

--
-- Name: entries_custom_trace_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_custom_trace_id_idx ON entries_custom (trace_id) WHERE (trace_id IS NOT NULL);

--
-- Name: entries_custom_user_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_custom_user_id_idx ON entries_custom (user_id) WHERE (user_id IS NOT NULL);

--
-- Name: idx_logging_entries_custom_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_custom_tenant_id ON entries_custom (tenant_id);

--
-- Name: entries_execution; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS entries_execution (
    id uuid DEFAULT gen_random_uuid(),
    timestamp timestamptz DEFAULT now() NOT NULL,
    category text,
    level text NOT NULL,
    message text NOT NULL,
    request_id text,
    trace_id text,
    component text,
    user_id uuid,
    ip_address inet,
    fields jsonb,
    execution_id uuid,
    line_number integer,
    custom_category text,
    tenant_id uuid,
    CONSTRAINT entries_execution_pkey PRIMARY KEY (category, id),
    CONSTRAINT valid_category CHECK (category IN ('system'::text, 'http'::text, 'security'::text, 'execution'::text, 'ai'::text, 'custom'::text)),
    CONSTRAINT valid_level CHECK (level IN ('trace'::text, 'debug'::text, 'info'::text, 'warn'::text, 'error'::text, 'fatal'::text, 'panic'::text))
);


COMMENT ON COLUMN logging.entries_execution.tenant_id IS 'Tenant this execution log entry belongs to';

--
-- Name: entries_execution_category_timestamp_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_execution_category_timestamp_idx ON entries_execution (category, "timestamp" DESC);

--
-- Name: entries_execution_component_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_execution_component_idx ON entries_execution (component) WHERE (component IS NOT NULL);

--
-- Name: entries_execution_custom_category_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_execution_custom_category_idx ON entries_execution (custom_category) WHERE (custom_category IS NOT NULL);

--
-- Name: entries_execution_execution_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_execution_execution_id_idx ON entries_execution (execution_id) WHERE (execution_id IS NOT NULL);

--
-- Name: entries_execution_execution_id_line_number_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_execution_execution_id_line_number_idx ON entries_execution (execution_id, line_number) WHERE (execution_id IS NOT NULL);

--
-- Name: entries_execution_level_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_execution_level_idx ON entries_execution (level);

--
-- Name: entries_execution_request_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_execution_request_id_idx ON entries_execution (request_id) WHERE (request_id IS NOT NULL);

--
-- Name: entries_execution_tenant_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_execution_tenant_id_idx ON entries_execution (tenant_id);

--
-- Name: entries_execution_timestamp_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_execution_timestamp_idx ON entries_execution ("timestamp" DESC);

--
-- Name: entries_execution_to_tsvector_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_execution_to_tsvector_idx ON entries_execution USING gin (to_tsvector('english'::regconfig, message));

--
-- Name: entries_execution_trace_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_execution_trace_id_idx ON entries_execution (trace_id) WHERE (trace_id IS NOT NULL);

--
-- Name: entries_execution_user_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_execution_user_id_idx ON entries_execution (user_id) WHERE (user_id IS NOT NULL);

--
-- Name: idx_logging_entries_execution_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_execution_tenant_id ON entries_execution (tenant_id);

--
-- Name: entries_http; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS entries_http (
    id uuid DEFAULT gen_random_uuid(),
    timestamp timestamptz DEFAULT now() NOT NULL,
    category text,
    level text NOT NULL,
    message text NOT NULL,
    request_id text,
    trace_id text,
    component text,
    user_id uuid,
    ip_address inet,
    fields jsonb,
    execution_id uuid,
    line_number integer,
    custom_category text,
    tenant_id uuid,
    CONSTRAINT entries_http_pkey PRIMARY KEY (category, id),
    CONSTRAINT valid_category CHECK (category IN ('system'::text, 'http'::text, 'security'::text, 'execution'::text, 'ai'::text, 'custom'::text)),
    CONSTRAINT valid_level CHECK (level IN ('trace'::text, 'debug'::text, 'info'::text, 'warn'::text, 'error'::text, 'fatal'::text, 'panic'::text))
);


COMMENT ON COLUMN logging.entries_http.tenant_id IS 'Tenant this HTTP log entry belongs to';

--
-- Name: entries_http_category_timestamp_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_http_category_timestamp_idx ON entries_http (category, "timestamp" DESC);

--
-- Name: entries_http_component_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_http_component_idx ON entries_http (component) WHERE (component IS NOT NULL);

--
-- Name: entries_http_custom_category_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_http_custom_category_idx ON entries_http (custom_category) WHERE (custom_category IS NOT NULL);

--
-- Name: entries_http_execution_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_http_execution_id_idx ON entries_http (execution_id) WHERE (execution_id IS NOT NULL);

--
-- Name: entries_http_execution_id_line_number_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_http_execution_id_line_number_idx ON entries_http (execution_id, line_number) WHERE (execution_id IS NOT NULL);

--
-- Name: entries_http_level_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_http_level_idx ON entries_http (level);

--
-- Name: entries_http_request_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_http_request_id_idx ON entries_http (request_id) WHERE (request_id IS NOT NULL);

--
-- Name: entries_http_tenant_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_http_tenant_id_idx ON entries_http (tenant_id);

--
-- Name: entries_http_timestamp_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_http_timestamp_idx ON entries_http ("timestamp" DESC);

--
-- Name: entries_http_to_tsvector_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_http_to_tsvector_idx ON entries_http USING gin (to_tsvector('english'::regconfig, message));

--
-- Name: entries_http_trace_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_http_trace_id_idx ON entries_http (trace_id) WHERE (trace_id IS NOT NULL);

--
-- Name: entries_http_user_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_http_user_id_idx ON entries_http (user_id) WHERE (user_id IS NOT NULL);

--
-- Name: idx_logging_entries_http_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_http_tenant_id ON entries_http (tenant_id);

--
-- Name: entries_security; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS entries_security (
    id uuid DEFAULT gen_random_uuid(),
    timestamp timestamptz DEFAULT now() NOT NULL,
    category text,
    level text NOT NULL,
    message text NOT NULL,
    request_id text,
    trace_id text,
    component text,
    user_id uuid,
    ip_address inet,
    fields jsonb,
    execution_id uuid,
    line_number integer,
    custom_category text,
    tenant_id uuid,
    CONSTRAINT entries_security_pkey PRIMARY KEY (category, id),
    CONSTRAINT valid_category CHECK (category IN ('system'::text, 'http'::text, 'security'::text, 'execution'::text, 'ai'::text, 'custom'::text)),
    CONSTRAINT valid_level CHECK (level IN ('trace'::text, 'debug'::text, 'info'::text, 'warn'::text, 'error'::text, 'fatal'::text, 'panic'::text))
);


COMMENT ON COLUMN logging.entries_security.tenant_id IS 'Tenant this security log entry belongs to';

--
-- Name: entries_security_category_timestamp_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_security_category_timestamp_idx ON entries_security (category, "timestamp" DESC);

--
-- Name: entries_security_component_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_security_component_idx ON entries_security (component) WHERE (component IS NOT NULL);

--
-- Name: entries_security_custom_category_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_security_custom_category_idx ON entries_security (custom_category) WHERE (custom_category IS NOT NULL);

--
-- Name: entries_security_execution_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_security_execution_id_idx ON entries_security (execution_id) WHERE (execution_id IS NOT NULL);

--
-- Name: entries_security_execution_id_line_number_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_security_execution_id_line_number_idx ON entries_security (execution_id, line_number) WHERE (execution_id IS NOT NULL);

--
-- Name: entries_security_level_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_security_level_idx ON entries_security (level);

--
-- Name: entries_security_request_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_security_request_id_idx ON entries_security (request_id) WHERE (request_id IS NOT NULL);

--
-- Name: entries_security_tenant_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_security_tenant_id_idx ON entries_security (tenant_id);

--
-- Name: entries_security_timestamp_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_security_timestamp_idx ON entries_security ("timestamp" DESC);

--
-- Name: entries_security_to_tsvector_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_security_to_tsvector_idx ON entries_security USING gin (to_tsvector('english'::regconfig, message));

--
-- Name: entries_security_trace_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_security_trace_id_idx ON entries_security (trace_id) WHERE (trace_id IS NOT NULL);

--
-- Name: entries_security_user_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_security_user_id_idx ON entries_security (user_id) WHERE (user_id IS NOT NULL);

--
-- Name: idx_logging_entries_security_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_security_tenant_id ON entries_security (tenant_id);

--
-- Name: entries_system; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS entries_system (
    id uuid DEFAULT gen_random_uuid(),
    timestamp timestamptz DEFAULT now() NOT NULL,
    category text,
    level text NOT NULL,
    message text NOT NULL,
    request_id text,
    trace_id text,
    component text,
    user_id uuid,
    ip_address inet,
    fields jsonb,
    execution_id uuid,
    line_number integer,
    custom_category text,
    tenant_id uuid,
    CONSTRAINT entries_system_pkey PRIMARY KEY (category, id),
    CONSTRAINT valid_category CHECK (category IN ('system'::text, 'http'::text, 'security'::text, 'execution'::text, 'ai'::text, 'custom'::text)),
    CONSTRAINT valid_level CHECK (level IN ('trace'::text, 'debug'::text, 'info'::text, 'warn'::text, 'error'::text, 'fatal'::text, 'panic'::text))
);


COMMENT ON COLUMN logging.entries_system.tenant_id IS 'Tenant this system log entry belongs to';

--
-- Name: entries_system_category_timestamp_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_system_category_timestamp_idx ON entries_system (category, "timestamp" DESC);

--
-- Name: entries_system_component_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_system_component_idx ON entries_system (component) WHERE (component IS NOT NULL);

--
-- Name: entries_system_custom_category_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_system_custom_category_idx ON entries_system (custom_category) WHERE (custom_category IS NOT NULL);

--
-- Name: entries_system_execution_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_system_execution_id_idx ON entries_system (execution_id) WHERE (execution_id IS NOT NULL);

--
-- Name: entries_system_execution_id_line_number_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_system_execution_id_line_number_idx ON entries_system (execution_id, line_number) WHERE (execution_id IS NOT NULL);

--
-- Name: entries_system_level_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_system_level_idx ON entries_system (level);

--
-- Name: entries_system_request_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_system_request_id_idx ON entries_system (request_id) WHERE (request_id IS NOT NULL);

--
-- Name: entries_system_tenant_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_system_tenant_id_idx ON entries_system (tenant_id);

--
-- Name: entries_system_timestamp_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_system_timestamp_idx ON entries_system ("timestamp" DESC);

--
-- Name: entries_system_to_tsvector_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_system_to_tsvector_idx ON entries_system USING gin (to_tsvector('english'::regconfig, message));

--
-- Name: entries_system_trace_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_system_trace_id_idx ON entries_system (trace_id) WHERE (trace_id IS NOT NULL);

--
-- Name: entries_system_user_id_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS entries_system_user_id_idx ON entries_system (user_id) WHERE (user_id IS NOT NULL);

--
-- Name: idx_logging_entries_system_tenant_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_logging_entries_system_tenant_id ON entries_system (tenant_id);

--
-- Name: entries_tenant_id_fkey; Type: CONSTRAINT; Schema: -; Owner: -
--

-- Note: FK constraint is NOT on the parent partitioned table.
-- It is defined on each child partition instead.
-- When partitions are attached, they bring their own FK constraints.

--
-- Cross-schema FKs moved to post-schema-fks.sql
-- entries_tenant_id_fkey (for entries_ai, entries_custom, entries_execution, entries_http, entries_security, entries_system)
--

--
-- Name: execution_logs_migration_status; Type: VIEW; Schema: -; Owner: -
--

CREATE OR REPLACE VIEW execution_logs_migration_status AS
 SELECT table_schema,
    table_name,
        CASE
            WHEN table_schema::name = 'functions'::name AND table_name::name = 'execution_logs'::name THEN 'functions.edge_functions'::text
            WHEN table_schema::name = 'jobs'::name AND table_name::name = 'execution_logs'::name THEN 'jobs.functions'::text
            WHEN table_schema::name = 'rpc'::name AND table_name::name = 'execution_logs'::name THEN 'rpc.procedures'::text
            WHEN table_schema::name = 'branching'::name AND table_name::name = 'seed_execution_log'::name THEN 'branching'::text
            ELSE (table_schema::text || '.'::text) || table_name::text
        END AS source,
        CASE
            WHEN (table_schema::name = ANY (ARRAY['functions'::name, 'jobs'::name, 'rpc'::name])) AND table_name::name = 'execution_logs'::name THEN 'MIGRATE TO LOGGING'::text
            WHEN table_schema::name = 'branching'::name AND table_name::name = 'seed_execution_log'::name THEN 'MIGRATE TO LOGGING'::text
            ELSE 'NOT APPLICABLE'::text
        END AS needs_migration
   FROM information_schema.tables
  WHERE table_schema::name = 'functions'::name AND table_name::name = 'execution_logs'::name OR table_schema::name = 'jobs'::name AND table_name::name = 'execution_logs'::name OR table_schema::name = 'rpc'::name AND table_name::name = 'execution_logs'::name OR table_schema::name = 'branching'::name AND table_name::name = 'seed_execution_log'::name;

--
-- Name: entries_ai; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE entries_ai TO service_role;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE entries_ai TO {{APP_USER}};

--
-- Name: entries_custom; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE entries_custom TO service_role;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE entries_custom TO {{APP_USER}};

--
-- Name: entries_execution; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE entries_execution TO service_role;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE entries_execution TO {{APP_USER}};

--
-- Name: entries_http; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE entries_http TO service_role;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE entries_http TO {{APP_USER}};

--
-- Name: entries_security; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE entries_security TO service_role;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE entries_security TO {{APP_USER}};

--
-- Name: entries_system; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE entries_system TO service_role;
GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE entries_system TO {{APP_USER}};

--
-- Name: entries; Type: PRIVILEGE; Schema: privileges; Owner: -
--

-- Grant on parent partitioned table (required for INSERT routing to partitions)
GRANT DELETE, INSERT, SELECT, TRUNCATE, UPDATE ON TABLE entries TO service_role;
GRANT DELETE, INSERT, SELECT, TRUNCATE, UPDATE ON TABLE entries TO {{APP_USER}};

--
-- Name: logging; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT USAGE ON SCHEMA logging TO tenant_service;
GRANT USAGE ON SCHEMA logging TO authenticated;
GRANT SELECT ON ALL TABLES IN SCHEMA logging TO authenticated;

--
-- Name: entries; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE logging.entries TO tenant_service;

--
-- Name: entries_ai; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE logging.entries_ai TO tenant_service;

--
-- Name: entries_custom; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE logging.entries_custom TO tenant_service;

--
-- Name: entries_execution; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE logging.entries_execution TO tenant_service;

--
-- Name: entries_http; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE logging.entries_http TO tenant_service;

--
-- Name: entries_security; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE logging.entries_security TO tenant_service;

--
-- Name: entries_system; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, SELECT, UPDATE ON TABLE logging.entries_system TO tenant_service;

-- ATTACH PARTITIONS
-- The child tables are created as regular tables and then attached as partitions.
-- ATTACH PARTITION blocks are in post-schema.sql because pgschema skips DO blocks
-- during the plan path. post-schema.sql is always executed after all schema applies.

--
-- Name: execution_logs_migration_status; Type: PRIVILEGE; Schema: privileges; Owner: -
--

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE execution_logs_migration_status TO service_role;

-- ============================================================================
-- ROW LEVEL SECURITY
-- ============================================================================

--
-- Name: entries; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries ENABLE ROW LEVEL SECURITY;

--
-- Name: entries; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries FORCE ROW LEVEL SECURITY;

--
-- Name: logging_entries_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY logging_entries_tenant ON entries TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

-- logging_entries_set_tenant_id trigger is defined in post-schema.sql (as a DO $$ block)
-- because pgschema tries to DROP the auto-propagated triggers on partition children,
-- which PostgreSQL forbids. Moving it here skips pgschema's plan/apply entirely.

--
-- Name: entries_ai; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries_ai ENABLE ROW LEVEL SECURITY;

--
-- Name: entries_ai; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries_ai FORCE ROW LEVEL SECURITY;

--
-- Name: logging_entries_ai_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY logging_entries_ai_tenant ON entries_ai TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

-- Note: No separate trigger on entries_ai needed — triggers on partitioned
-- parent (entries) automatically propagate to all partitions.

--
-- Name: entries_http; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries_http ENABLE ROW LEVEL SECURITY;

--
-- Name: entries_http; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries_http FORCE ROW LEVEL SECURITY;

--
-- Name: logging_entries_http_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY logging_entries_http_tenant ON entries_http TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

-- Note: No separate trigger on entries_http needed — inherited from parent.

--
-- Name: entries_security; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries_security ENABLE ROW LEVEL SECURITY;

--
-- Name: entries_security; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries_security FORCE ROW LEVEL SECURITY;

--
-- Name: logging_entries_security_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY logging_entries_security_tenant ON entries_security TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

-- Note: No separate trigger on entries_security needed — inherited from parent.

--
-- Name: entries_execution; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries_execution ENABLE ROW LEVEL SECURITY;

--
-- Name: entries_execution; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries_execution FORCE ROW LEVEL SECURITY;

--
-- Name: logging_entries_execution_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY logging_entries_execution_tenant ON entries_execution TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

-- Note: No separate trigger on entries_execution needed — inherited from parent.

--
-- Name: entries_custom; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries_custom ENABLE ROW LEVEL SECURITY;

--
-- Name: entries_custom; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries_custom FORCE ROW LEVEL SECURITY;

--
-- Name: logging_entries_custom_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY logging_entries_custom_tenant ON entries_custom TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

-- Note: No separate trigger on entries_custom needed — inherited from parent.

--
-- Name: entries_system; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries_system ENABLE ROW LEVEL SECURITY;

--
-- Name: entries_system; Type: RLS; Schema: -; Owner: -
--

ALTER TABLE entries_system FORCE ROW LEVEL SECURITY;

--
-- Name: logging_entries_system_tenant; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY logging_entries_system_tenant ON entries_system TO PUBLIC
    USING (auth.has_tenant_access(tenant_id))
    WITH CHECK (auth.has_tenant_access(tenant_id));

-- Note: No separate trigger on entries_system needed — inherited from parent.
