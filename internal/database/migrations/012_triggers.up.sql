-- ============================================================================
-- TRIGGERS
-- ============================================================================
-- This file contains all trigger definitions for the Fluxbase database.
-- Triggers are used to automatically update timestamps and validate data.
-- ============================================================================

--

-- Auth schema triggers
CREATE TRIGGER update_auth_users_updated_at BEFORE UPDATE ON auth.users
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_auth_sessions_updated_at BEFORE UPDATE ON auth.sessions
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_auth_api_keys_updated_at BEFORE UPDATE ON auth.api_keys
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_oauth_links_updated_at BEFORE UPDATE ON auth.oauth_links
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_oauth_tokens_updated_at BEFORE UPDATE ON auth.oauth_tokens
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_auth_webhooks_updated_at BEFORE UPDATE ON auth.webhooks
    FOR EACH ROW EXECUTE FUNCTION auth.update_webhook_updated_at();

-- App metadata protection trigger
CREATE TRIGGER validate_app_metadata_trigger BEFORE UPDATE ON auth.users
    FOR EACH ROW EXECUTE FUNCTION auth.validate_app_metadata_update();

-- Dashboard schema triggers
CREATE TRIGGER update_dashboard_users_updated_at BEFORE UPDATE ON dashboard.users
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_dashboard_sessions_updated_at BEFORE UPDATE ON dashboard.sessions
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_dashboard_oauth_providers_updated_at BEFORE UPDATE ON dashboard.oauth_providers
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_dashboard_auth_settings_updated_at BEFORE UPDATE ON dashboard.auth_settings
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_dashboard_system_settings_updated_at BEFORE UPDATE ON dashboard.system_settings
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_dashboard_custom_settings_updated_at BEFORE UPDATE ON dashboard.custom_settings
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_dashboard_email_templates_updated_at BEFORE UPDATE ON dashboard.email_templates
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

-- App metadata protection trigger for dashboard users
CREATE TRIGGER validate_app_metadata_trigger BEFORE UPDATE ON dashboard.users
    FOR EACH ROW EXECUTE FUNCTION auth.validate_app_metadata_update();

-- Functions schema triggers
CREATE TRIGGER update_functions_edge_functions_updated_at BEFORE UPDATE ON functions.edge_functions
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_functions_edge_function_triggers_updated_at BEFORE UPDATE ON functions.edge_function_triggers
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

-- Storage schema triggers
CREATE TRIGGER update_storage_buckets_updated_at BEFORE UPDATE ON storage.buckets
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

CREATE TRIGGER update_storage_objects_updated_at BEFORE UPDATE ON storage.objects
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

-- Realtime schema triggers
CREATE TRIGGER update_realtime_schema_registry_updated_at BEFORE UPDATE ON realtime.schema_registry
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();
