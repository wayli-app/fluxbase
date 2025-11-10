-- Migration 009: Protect app_metadata from unauthorized updates
-- Ensures only admins and dashboard admins can modify app_metadata field

-- Create a function to validate app_metadata updates
CREATE OR REPLACE FUNCTION auth.validate_app_metadata_update()
RETURNS TRIGGER AS $$
DECLARE
    user_role TEXT;
BEGIN
    -- Get the current user's role
    user_role := auth.current_user_role();

    -- Check if app_metadata is being modified
    IF OLD.app_metadata IS DISTINCT FROM NEW.app_metadata THEN
        -- Only allow admins and dashboard admins to modify app_metadata
        IF user_role != 'admin' AND user_role != 'dashboard_admin' THEN
            -- Also check if user has admin privileges via is_admin() function
            IF NOT auth.is_admin() THEN
                RAISE EXCEPTION 'Only admins can modify app_metadata'
                    USING ERRCODE = 'insufficient_privilege';
            END IF;
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Create trigger on auth.users to validate app_metadata updates
DROP TRIGGER IF EXISTS validate_app_metadata_trigger ON auth.users;
CREATE TRIGGER validate_app_metadata_trigger
    BEFORE UPDATE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.validate_app_metadata_update();

-- Add comment explaining the protection
COMMENT ON FUNCTION auth.validate_app_metadata_update() IS
'Validates that only admins and dashboard admins can modify the app_metadata field on auth.users';

-- Do the same for dashboard.users if the table exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'dashboard'
        AND table_name = 'users'
    ) THEN
        -- Create trigger on dashboard.users to validate app_metadata updates
        DROP TRIGGER IF EXISTS validate_app_metadata_trigger ON dashboard.users;
        CREATE TRIGGER validate_app_metadata_trigger
            BEFORE UPDATE ON dashboard.users
            FOR EACH ROW
            EXECUTE FUNCTION auth.validate_app_metadata_update();
    END IF;
END $$;
