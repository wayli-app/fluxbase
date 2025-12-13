-- Remove feature flags and settings
DELETE FROM app.settings WHERE key IN (
    'app.features.enable_jobs',
    'app.features.enable_ai',
    'app.ai.allow_user_provider_override',
    'app.ai.default_rate_limit_per_minute',
    'app.ai.default_daily_token_budget',
    'app.features.enable_rpc',
    'app.rpc.default_max_execution_time_seconds',
    'app.rpc.max_max_execution_time_seconds',
    'app.rpc.default_max_rows'
);
