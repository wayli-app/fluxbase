-- ============================================================================
-- DEFAULT DATA
-- ============================================================================
-- This file contains default data insertions for feature flags and settings.
-- ============================================================================

-- Jobs feature flag
INSERT INTO app.settings (key, value, value_type, is_secret, description, editable_by)
VALUES (
    'app.features.enable_jobs',
    '{"value": false}'::JSONB,
    'boolean',
    false,
    'Enable long-running background jobs system',
    ARRAY['admin', 'dashboard_admin']
)
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    value_type = EXCLUDED.value_type,
    description = EXCLUDED.description;

-- AI feature flags
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

-- RPC feature flags
INSERT INTO app.settings (key, value, value_type, category, description, is_public, is_secret, editable_by)
VALUES
    (
        'app.features.enable_rpc',
        '{"value": true}'::JSONB,
        'boolean',
        'system',
        'Enable or disable RPC procedure functionality',
        false,
        false,
        ARRAY['dashboard_admin']::TEXT[]
    ),
    (
        'app.rpc.default_max_execution_time_seconds',
        '{"value": 30}'::JSONB,
        'number',
        'system',
        'Default maximum execution time for RPC procedures in seconds',
        false,
        false,
        ARRAY['dashboard_admin']::TEXT[]
    ),
    (
        'app.rpc.max_max_execution_time_seconds',
        '{"value": 300}'::JSONB,
        'number',
        'system',
        'Maximum allowed execution time for RPC procedures in seconds',
        false,
        false,
        ARRAY['dashboard_admin']::TEXT[]
    ),
    (
        'app.rpc.default_max_rows',
        '{"value": 1000}'::JSONB,
        'number',
        'system',
        'Default maximum rows returned by RPC procedures',
        false,
        false,
        ARRAY['dashboard_admin']::TEXT[]
    )
ON CONFLICT (key) DO NOTHING;
