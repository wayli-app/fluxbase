-- Add system setting for controlling user client key creation
-- When disabled, only admins can create client keys and user-created keys are blocked

INSERT INTO app.settings (key, value, value_type, category, description, is_public, editable_by)
VALUES (
    'app.auth.allow_user_client_keys',
    'true',
    'boolean',
    'auth',
    'Allow regular users to create and use their own client keys. When disabled, only admin-created keys work and users cannot create keys.',
    false,
    ARRAY['dashboard_admin']::TEXT[]
) ON CONFLICT (key) DO NOTHING;
