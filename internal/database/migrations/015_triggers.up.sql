-- ============================================================================
-- TRIGGERS
-- ============================================================================
-- This file contains all trigger definitions for the Fluxbase database.
-- Triggers are used to automatically update timestamps and validate data.
-- ============================================================================

--

-- Auth schema triggers (DROP IF EXISTS for idempotency since auth schema is preserved in db-reset)
DROP TRIGGER IF EXISTS update_auth_users_updated_at ON auth.users;
CREATE TRIGGER update_auth_users_updated_at BEFORE UPDATE ON auth.users
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

DROP TRIGGER IF EXISTS update_auth_sessions_updated_at ON auth.sessions;
CREATE TRIGGER update_auth_sessions_updated_at BEFORE UPDATE ON auth.sessions
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

DROP TRIGGER IF EXISTS update_auth_api_keys_updated_at ON auth.api_keys;
CREATE TRIGGER update_auth_api_keys_updated_at BEFORE UPDATE ON auth.api_keys
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

DROP TRIGGER IF EXISTS update_oauth_links_updated_at ON auth.oauth_links;
CREATE TRIGGER update_oauth_links_updated_at BEFORE UPDATE ON auth.oauth_links
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

DROP TRIGGER IF EXISTS update_oauth_tokens_updated_at ON auth.oauth_tokens;
CREATE TRIGGER update_oauth_tokens_updated_at BEFORE UPDATE ON auth.oauth_tokens
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

DROP TRIGGER IF EXISTS update_auth_webhooks_updated_at ON auth.webhooks;
CREATE TRIGGER update_auth_webhooks_updated_at BEFORE UPDATE ON auth.webhooks
    FOR EACH ROW EXECUTE FUNCTION auth.update_webhook_updated_at();

-- App metadata protection trigger
DROP TRIGGER IF EXISTS validate_app_metadata_trigger ON auth.users;
CREATE TRIGGER validate_app_metadata_trigger BEFORE UPDATE ON auth.users
    FOR EACH ROW EXECUTE FUNCTION auth.validate_app_metadata_update();

-- Dashboard schema triggers (DROP IF EXISTS for idempotency)
DROP TRIGGER IF EXISTS update_dashboard_users_updated_at ON dashboard.users;
CREATE TRIGGER update_dashboard_users_updated_at BEFORE UPDATE ON dashboard.users
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

DROP TRIGGER IF EXISTS update_dashboard_sessions_updated_at ON dashboard.sessions;
CREATE TRIGGER update_dashboard_sessions_updated_at BEFORE UPDATE ON dashboard.sessions
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

DROP TRIGGER IF EXISTS update_dashboard_oauth_providers_updated_at ON dashboard.oauth_providers;
CREATE TRIGGER update_dashboard_oauth_providers_updated_at BEFORE UPDATE ON dashboard.oauth_providers
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

-- Note: dashboard.auth_settings, dashboard.system_settings, and dashboard.custom_settings
-- have been migrated to app.settings. Triggers for app.settings are in migration 015_policies_app

DROP TRIGGER IF EXISTS update_dashboard_email_templates_updated_at ON dashboard.email_templates;
CREATE TRIGGER update_dashboard_email_templates_updated_at BEFORE UPDATE ON dashboard.email_templates
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

-- App metadata protection trigger for dashboard users
DROP TRIGGER IF EXISTS validate_app_metadata_trigger ON dashboard.users;
CREATE TRIGGER validate_app_metadata_trigger BEFORE UPDATE ON dashboard.users
    FOR EACH ROW EXECUTE FUNCTION auth.validate_app_metadata_update();

-- Functions schema triggers (DROP IF EXISTS for idempotency)
DROP TRIGGER IF EXISTS update_functions_edge_functions_updated_at ON functions.edge_functions;
CREATE TRIGGER update_functions_edge_functions_updated_at BEFORE UPDATE ON functions.edge_functions
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

DROP TRIGGER IF EXISTS update_functions_edge_triggers_updated_at ON functions.edge_triggers;
CREATE TRIGGER update_functions_edge_triggers_updated_at BEFORE UPDATE ON functions.edge_triggers
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

DROP TRIGGER IF EXISTS update_function_dependencies_updated_at ON functions.function_dependencies;
CREATE TRIGGER update_function_dependencies_updated_at BEFORE UPDATE ON functions.function_dependencies
    FOR EACH ROW EXECUTE FUNCTION functions.update_function_dependencies_updated_at();

DROP TRIGGER IF EXISTS trigger_mark_functions_on_shared_module_update ON functions.shared_modules;
CREATE TRIGGER trigger_mark_functions_on_shared_module_update AFTER UPDATE ON functions.shared_modules
    FOR EACH ROW
    WHEN (OLD.content IS DISTINCT FROM NEW.content OR OLD.version IS DISTINCT FROM NEW.version)
    EXECUTE FUNCTION functions.mark_dependent_functions_for_rebundle();

-- Storage schema triggers (DROP IF EXISTS for idempotency)
DROP TRIGGER IF EXISTS update_storage_buckets_updated_at ON storage.buckets;
CREATE TRIGGER update_storage_buckets_updated_at BEFORE UPDATE ON storage.buckets
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

DROP TRIGGER IF EXISTS update_storage_objects_updated_at ON storage.objects;
CREATE TRIGGER update_storage_objects_updated_at BEFORE UPDATE ON storage.objects
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

-- Realtime schema triggers (DROP IF EXISTS for idempotency)
DROP TRIGGER IF EXISTS update_realtime_schema_registry_updated_at ON realtime.schema_registry;
CREATE TRIGGER update_realtime_schema_registry_updated_at BEFORE UPDATE ON realtime.schema_registry
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
