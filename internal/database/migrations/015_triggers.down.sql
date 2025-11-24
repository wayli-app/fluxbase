-- Drop all triggers

-- Realtime schema triggers
DROP TRIGGER IF EXISTS update_realtime_schema_registry_updated_at ON realtime.schema_registry;

-- Storage schema triggers
DROP TRIGGER IF EXISTS update_storage_objects_updated_at ON storage.objects;
DROP TRIGGER IF EXISTS update_storage_buckets_updated_at ON storage.buckets;

-- Functions schema triggers
DROP TRIGGER IF EXISTS trigger_mark_functions_on_shared_module_update ON functions.shared_modules;
DROP TRIGGER IF EXISTS update_function_dependencies_updated_at ON functions.function_dependencies;
DROP TRIGGER IF EXISTS update_functions_edge_function_triggers_updated_at ON functions.edge_function_triggers;
DROP TRIGGER IF EXISTS update_functions_edge_functions_updated_at ON functions.edge_functions;

-- Dashboard schema triggers
DROP TRIGGER IF EXISTS validate_app_metadata_trigger ON dashboard.users;
DROP TRIGGER IF EXISTS update_dashboard_email_templates_updated_at ON dashboard.email_templates;
-- Note: dashboard.auth_settings, dashboard.system_settings, and dashboard.custom_settings
-- have been migrated to app.settings
DROP TRIGGER IF EXISTS update_dashboard_oauth_providers_updated_at ON dashboard.oauth_providers;
DROP TRIGGER IF EXISTS update_dashboard_sessions_updated_at ON dashboard.sessions;
DROP TRIGGER IF EXISTS update_dashboard_users_updated_at ON dashboard.users;

-- Auth schema triggers
DROP TRIGGER IF EXISTS validate_app_metadata_trigger ON auth.users;
DROP TRIGGER IF EXISTS update_auth_webhooks_updated_at ON auth.webhooks;
DROP TRIGGER IF EXISTS update_oauth_tokens_updated_at ON auth.oauth_tokens;
DROP TRIGGER IF EXISTS update_oauth_links_updated_at ON auth.oauth_links;
DROP TRIGGER IF EXISTS update_auth_api_keys_updated_at ON auth.api_keys;
DROP TRIGGER IF EXISTS update_auth_sessions_updated_at ON auth.sessions;
DROP TRIGGER IF EXISTS update_auth_users_updated_at ON auth.users;
