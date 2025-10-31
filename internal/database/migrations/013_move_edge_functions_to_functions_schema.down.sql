-- Rollback: Move edge functions tables back to public schema

-- Move tables back to public schema
ALTER TABLE functions.edge_functions SET SCHEMA public;
ALTER TABLE functions.edge_function_executions SET SCHEMA public;
ALTER TABLE functions.edge_function_triggers SET SCHEMA public;

-- Move functions back to public schema
ALTER FUNCTION functions.update_edge_function_updated_at() SET SCHEMA public;
ALTER FUNCTION functions.cleanup_old_edge_function_executions() SET SCHEMA public;

-- Update the trigger
DROP TRIGGER IF EXISTS edge_functions_updated_at ON public.edge_functions;
CREATE TRIGGER edge_functions_updated_at
    BEFORE UPDATE ON public.edge_functions
    FOR EACH ROW
    EXECUTE FUNCTION public.update_edge_function_updated_at();

-- Restore original grants
REVOKE ALL ON functions.edge_functions FROM authenticated;
REVOKE ALL ON functions.edge_function_executions FROM authenticated;
REVOKE ALL ON functions.edge_function_triggers FROM authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.edge_functions TO authenticated;
GRANT SELECT ON public.edge_function_executions TO authenticated;
GRANT SELECT ON public.edge_function_triggers TO authenticated;
