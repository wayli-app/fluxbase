-- AI Schema Migration
-- Creates tables for AI chatbots, providers, conversations, and audit logging

-- Create AI schema
CREATE SCHEMA IF NOT EXISTS ai;

-- Grant usage on ai schema
GRANT USAGE ON SCHEMA ai TO anon, authenticated, service_role;

-- ============================================================================
-- AI PROVIDERS
-- Configuration for AI service providers (OpenAI, Azure, Ollama)
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai.providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    provider_type TEXT NOT NULL CHECK (provider_type IN ('openai', 'azure', 'ollama')),
    is_default BOOLEAN DEFAULT false,
    config JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_providers_name ON ai.providers(name);
CREATE INDEX IF NOT EXISTS idx_ai_providers_type ON ai.providers(provider_type);
CREATE INDEX IF NOT EXISTS idx_ai_providers_enabled ON ai.providers(enabled);

-- Ensure only one default provider
CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_providers_single_default
    ON ai.providers(is_default) WHERE is_default = true;

COMMENT ON TABLE ai.providers IS 'AI provider configurations (OpenAI, Azure, Ollama)';
COMMENT ON COLUMN ai.providers.config IS 'Provider-specific config (api_key, endpoint, model) - should be encrypted at application level';

-- ============================================================================
-- USER PROVIDER PREFERENCES
-- Allows users to override default provider (when enabled via settings)
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai.user_provider_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    provider_id UUID REFERENCES ai.providers(id) ON DELETE SET NULL,
    api_key_encrypted TEXT,
    model_override TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX IF NOT EXISTS idx_ai_user_prefs_user ON ai.user_provider_preferences(user_id);

COMMENT ON TABLE ai.user_provider_preferences IS 'User-level AI provider overrides (when enabled)';

-- ============================================================================
-- CHATBOTS
-- AI chatbot definitions synced from filesystem or API
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai.chatbots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    namespace TEXT NOT NULL DEFAULT 'default',
    description TEXT,
    code TEXT NOT NULL,
    original_code TEXT,
    is_bundled BOOLEAN DEFAULT false,
    bundle_error TEXT,

    -- Parsed from annotations
    allowed_tables TEXT[] DEFAULT ARRAY[]::TEXT[],
    allowed_operations TEXT[] DEFAULT ARRAY['SELECT']::TEXT[],
    allowed_schemas TEXT[] DEFAULT ARRAY['public']::TEXT[],

    -- Runtime config
    enabled BOOLEAN DEFAULT true,
    max_tokens INTEGER DEFAULT 4096,
    temperature NUMERIC(3,2) DEFAULT 0.7,
    provider_id UUID REFERENCES ai.providers(id) ON DELETE SET NULL,

    -- Conversation config
    persist_conversations BOOLEAN DEFAULT false,
    conversation_ttl_hours INTEGER DEFAULT 24,
    max_conversation_turns INTEGER DEFAULT 50,

    -- Rate limiting (per user, per chatbot)
    rate_limit_per_minute INTEGER DEFAULT 20,
    daily_request_limit INTEGER DEFAULT 500,
    daily_token_budget INTEGER DEFAULT 100000,

    -- Access control
    allow_unauthenticated BOOLEAN DEFAULT false,
    is_public BOOLEAN DEFAULT true,

    -- HTTP request tool config
    http_allowed_domains TEXT[] DEFAULT ARRAY[]::TEXT[],

    version INTEGER DEFAULT 1,
    source TEXT NOT NULL DEFAULT 'filesystem',
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_chatbot_name_namespace UNIQUE (name, namespace)
);

CREATE INDEX IF NOT EXISTS idx_ai_chatbots_name ON ai.chatbots(name);
CREATE INDEX IF NOT EXISTS idx_ai_chatbots_namespace ON ai.chatbots(namespace);
CREATE INDEX IF NOT EXISTS idx_ai_chatbots_enabled ON ai.chatbots(enabled);
CREATE INDEX IF NOT EXISTS idx_ai_chatbots_source ON ai.chatbots(source);
CREATE INDEX IF NOT EXISTS idx_ai_chatbots_http_domains ON ai.chatbots USING GIN (http_allowed_domains);

COMMENT ON TABLE ai.chatbots IS 'AI chatbot definitions with system prompts and tool configurations';
COMMENT ON COLUMN ai.chatbots.allowed_tables IS 'Tables the chatbot can query (from @fluxbase:allowed-tables annotation)';
COMMENT ON COLUMN ai.chatbots.allowed_operations IS 'SQL operations allowed (SELECT, INSERT, UPDATE, DELETE)';
COMMENT ON COLUMN ai.chatbots.rate_limit_per_minute IS 'Max requests per minute per user (from @fluxbase:rate-limit annotation)';
COMMENT ON COLUMN ai.chatbots.http_allowed_domains IS 'Allowed domains for HTTP requests (from @fluxbase:http-allowed-domains annotation)';

-- ============================================================================
-- CONVERSATIONS
-- Stores conversation state for persistent chatbots
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai.conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chatbot_id UUID NOT NULL REFERENCES ai.chatbots(id) ON DELETE CASCADE,
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    session_id TEXT,

    title TEXT,
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),
    turn_count INTEGER DEFAULT 0,

    -- Token usage tracking
    total_prompt_tokens INTEGER DEFAULT 0,
    total_completion_tokens INTEGER DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_message_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ai_conversations_chatbot ON ai.conversations(chatbot_id);
CREATE INDEX IF NOT EXISTS idx_ai_conversations_user ON ai.conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_conversations_session ON ai.conversations(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ai_conversations_status ON ai.conversations(status);
CREATE INDEX IF NOT EXISTS idx_ai_conversations_expires ON ai.conversations(expires_at) WHERE expires_at IS NOT NULL;

COMMENT ON TABLE ai.conversations IS 'AI conversation sessions with token tracking';

-- ============================================================================
-- MESSAGES
-- Individual messages within conversations
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai.messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES ai.conversations(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system', 'tool')),
    content TEXT NOT NULL,

    -- For tool calls/results
    tool_call_id TEXT,
    tool_name TEXT,
    tool_input JSONB,
    tool_output JSONB,

    -- SQL execution info
    executed_sql TEXT,
    sql_result_summary TEXT,
    sql_row_count INTEGER,
    sql_error TEXT,
    sql_duration_ms INTEGER,

    -- Token counts
    prompt_tokens INTEGER,
    completion_tokens INTEGER,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    sequence_number INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_messages_conversation ON ai.messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_ai_messages_sequence ON ai.messages(conversation_id, sequence_number);
CREATE INDEX IF NOT EXISTS idx_ai_messages_role ON ai.messages(role);

COMMENT ON TABLE ai.messages IS 'Individual messages within AI conversations';

-- ============================================================================
-- QUERY AUDIT LOG
-- Audit trail for all SQL queries generated by chatbots
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai.query_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chatbot_id UUID REFERENCES ai.chatbots(id) ON DELETE SET NULL,
    conversation_id UUID REFERENCES ai.conversations(id) ON DELETE SET NULL,
    message_id UUID REFERENCES ai.messages(id) ON DELETE SET NULL,
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,

    -- Query details
    generated_sql TEXT NOT NULL,
    sanitized_sql TEXT,
    executed BOOLEAN DEFAULT false,

    -- Validation result
    validation_passed BOOLEAN,
    validation_errors TEXT[],

    -- Execution result
    success BOOLEAN,
    error_message TEXT,
    rows_returned INTEGER,
    execution_duration_ms INTEGER,

    -- Context
    tables_accessed TEXT[],
    operations_used TEXT[],
    rls_user_id TEXT,
    rls_role TEXT,

    -- Request context
    ip_address INET,
    user_agent TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_query_audit_chatbot ON ai.query_audit_log(chatbot_id);
CREATE INDEX IF NOT EXISTS idx_ai_query_audit_user ON ai.query_audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_query_audit_created ON ai.query_audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_query_audit_success ON ai.query_audit_log(success);
CREATE INDEX IF NOT EXISTS idx_ai_query_audit_executed ON ai.query_audit_log(executed);

COMMENT ON TABLE ai.query_audit_log IS 'Audit log for all SQL queries generated and executed by AI chatbots';

-- ============================================================================
-- USER CHATBOT USAGE
-- Tracks daily usage per user per chatbot for rate limiting
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai.user_chatbot_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    chatbot_id UUID NOT NULL REFERENCES ai.chatbots(id) ON DELETE CASCADE,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    request_count INTEGER DEFAULT 0,
    tokens_used INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, chatbot_id, date)
);

CREATE INDEX IF NOT EXISTS idx_ai_usage_lookup ON ai.user_chatbot_usage(user_id, chatbot_id, date);
CREATE INDEX IF NOT EXISTS idx_ai_usage_date ON ai.user_chatbot_usage(date);

COMMENT ON TABLE ai.user_chatbot_usage IS 'Daily usage tracking per user per chatbot for rate limiting';

-- ============================================================================
-- ROW LEVEL SECURITY
-- ============================================================================

ALTER TABLE ai.providers ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai.user_provider_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai.chatbots ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai.conversations ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai.messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai.query_audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai.user_chatbot_usage ENABLE ROW LEVEL SECURITY;

-- Service role can do everything (bypasses RLS)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'providers' AND policyname = 'ai_providers_service_all') THEN
        CREATE POLICY "ai_providers_service_all" ON ai.providers FOR ALL TO service_role USING (true);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'user_provider_preferences' AND policyname = 'ai_user_prefs_service_all') THEN
        CREATE POLICY "ai_user_prefs_service_all" ON ai.user_provider_preferences FOR ALL TO service_role USING (true);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'chatbots' AND policyname = 'ai_chatbots_service_all') THEN
        CREATE POLICY "ai_chatbots_service_all" ON ai.chatbots FOR ALL TO service_role USING (true);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'conversations' AND policyname = 'ai_conversations_service_all') THEN
        CREATE POLICY "ai_conversations_service_all" ON ai.conversations FOR ALL TO service_role USING (true);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'messages' AND policyname = 'ai_messages_service_all') THEN
        CREATE POLICY "ai_messages_service_all" ON ai.messages FOR ALL TO service_role USING (true);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'query_audit_log' AND policyname = 'ai_query_audit_service_all') THEN
        CREATE POLICY "ai_query_audit_service_all" ON ai.query_audit_log FOR ALL TO service_role USING (true);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'user_chatbot_usage' AND policyname = 'ai_usage_service_all') THEN
        CREATE POLICY "ai_usage_service_all" ON ai.user_chatbot_usage FOR ALL TO service_role USING (true);
    END IF;
END $$;

-- Authenticated users can read enabled providers
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'providers' AND policyname = 'ai_providers_read') THEN
        CREATE POLICY "ai_providers_read" ON ai.providers
            FOR SELECT TO authenticated
            USING (enabled = true);
    END IF;
END $$;

-- Users can manage their own provider preferences
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'user_provider_preferences' AND policyname = 'ai_user_prefs_own') THEN
        CREATE POLICY "ai_user_prefs_own" ON ai.user_provider_preferences
            FOR ALL TO authenticated
            USING (user_id = auth.current_user_id());
    END IF;
END $$;

-- Users can read public, enabled chatbots
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'chatbots' AND policyname = 'ai_chatbots_read') THEN
        CREATE POLICY "ai_chatbots_read" ON ai.chatbots
            FOR SELECT TO authenticated
            USING (enabled = true AND is_public = true);
    END IF;
END $$;

-- Users can manage their own conversations
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'conversations' AND policyname = 'ai_conversations_own') THEN
        CREATE POLICY "ai_conversations_own" ON ai.conversations
            FOR ALL TO authenticated
            USING (user_id = auth.current_user_id());
    END IF;
END $$;

-- Users can access messages in their own conversations
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'messages' AND policyname = 'ai_messages_own') THEN
        CREATE POLICY "ai_messages_own" ON ai.messages
            FOR ALL TO authenticated
            USING (conversation_id IN (
                SELECT id FROM ai.conversations WHERE user_id = auth.current_user_id()
            ));
    END IF;
END $$;

-- Users can read their own usage stats
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'user_chatbot_usage' AND policyname = 'ai_usage_own_read') THEN
        CREATE POLICY "ai_usage_own_read" ON ai.user_chatbot_usage
            FOR SELECT TO authenticated
            USING (user_id = auth.current_user_id());
    END IF;
END $$;

-- ============================================================================
-- PERMISSIONS
-- ============================================================================

-- Grant permissions on ai schema tables
GRANT SELECT ON ai.providers TO authenticated;
GRANT ALL ON ai.providers TO service_role;

GRANT ALL ON ai.user_provider_preferences TO authenticated, service_role;

GRANT SELECT ON ai.chatbots TO authenticated;
GRANT ALL ON ai.chatbots TO service_role;

GRANT ALL ON ai.conversations TO authenticated, service_role;
GRANT ALL ON ai.messages TO authenticated, service_role;

GRANT ALL ON ai.query_audit_log TO service_role;

GRANT SELECT ON ai.user_chatbot_usage TO authenticated;
GRANT ALL ON ai.user_chatbot_usage TO service_role;

-- ============================================================================
-- DEFAULT SETTINGS
-- Add AI feature flags to app.settings
-- ============================================================================

INSERT INTO app.settings (key, value, value_type, category, description, is_public, is_secret, editable_by)
VALUES
    (
        'app.features.enable_ai',
        '{"value": true}'::JSONB,
        'boolean',
        'system',
        'Enable or disable AI chatbot functionality',
        false,
        false,
        ARRAY['dashboard_admin']::TEXT[]
    ),
    (
        'app.ai.allow_user_provider_override',
        '{"value": false}'::JSONB,
        'boolean',
        'system',
        'Allow users to configure their own AI provider credentials',
        false,
        false,
        ARRAY['dashboard_admin']::TEXT[]
    ),
    (
        'app.ai.default_rate_limit_per_minute',
        '{"value": 20}'::JSONB,
        'number',
        'system',
        'Default rate limit for AI chatbot requests per minute per user',
        false,
        false,
        ARRAY['dashboard_admin']::TEXT[]
    ),
    (
        'app.ai.default_daily_token_budget',
        '{"value": 100000}'::JSONB,
        'number',
        'system',
        'Default daily token budget per user for AI chatbots',
        false,
        false,
        ARRAY['dashboard_admin']::TEXT[]
    )
ON CONFLICT (key) DO NOTHING;
