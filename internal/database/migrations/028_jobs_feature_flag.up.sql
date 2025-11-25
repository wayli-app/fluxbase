-- Add jobs feature flag to app settings
INSERT INTO app.settings (key, value, value_type, is_secret, description, editable_by)
VALUES (
    'app.features.enable_jobs',
    'false',
    'boolean',
    false,
    'Enable long-running background jobs system',
    ARRAY['admin', 'dashboard_admin']
)
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    value_type = EXCLUDED.value_type,
    description = EXCLUDED.description;
