-- Rollback Global Functions

DROP FUNCTION IF EXISTS functions.mark_dependent_functions_for_rebundle() CASCADE;
DROP FUNCTION IF EXISTS functions.update_function_dependencies_updated_at() CASCADE;
DROP FUNCTION IF EXISTS storage.foldername(TEXT) CASCADE;
DROP FUNCTION IF EXISTS storage.has_object_permission(UUID, UUID, TEXT) CASCADE;
DROP FUNCTION IF EXISTS storage.is_bucket_public(TEXT) CASCADE;
DROP FUNCTION IF EXISTS storage.user_can_access_object(UUID, TEXT) CASCADE;
DROP FUNCTION IF EXISTS auth.update_webhook_updated_at() CASCADE;
DROP FUNCTION IF EXISTS auth.validate_app_metadata_update() CASCADE;
DROP FUNCTION IF EXISTS auth.remove_webhook_trigger(TEXT, TEXT) CASCADE;
DROP FUNCTION IF EXISTS auth.create_webhook_trigger(TEXT, TEXT) CASCADE;
DROP FUNCTION IF EXISTS auth.queue_webhook_event() CASCADE;
DROP FUNCTION IF EXISTS public.update_updated_at() CASCADE;
