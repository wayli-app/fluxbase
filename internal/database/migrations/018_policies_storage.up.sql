-- ============================================================================
-- STORAGE SCHEMA RLS
-- ============================================================================
-- This file contains all Row Level Security (RLS) policies for the Storage schema.
-- These policies control access to buckets, objects, and object permissions.
-- Includes custom RLS policy examples and fixes for infinite recursion prevention.
-- ============================================================================

-- Storage buckets
ALTER TABLE storage.buckets ENABLE ROW LEVEL SECURITY;
ALTER TABLE storage.buckets FORCE ROW LEVEL SECURITY;

-- Admins and service roles can do everything with buckets
CREATE POLICY storage_buckets_admin ON storage.buckets
    FOR ALL
    USING (auth.current_user_role() IN ('dashboard_admin', 'service_role'))
    WITH CHECK (auth.current_user_role() IN ('dashboard_admin', 'service_role'));

COMMENT ON POLICY storage_buckets_admin ON storage.buckets IS 'Dashboard admins and service role have full access to all buckets';

-- Anyone can view public buckets (read-only)
CREATE POLICY storage_buckets_public_view ON storage.buckets
    FOR SELECT
    USING (public = true);

COMMENT ON POLICY storage_buckets_public_view ON storage.buckets IS 'Public buckets are visible to everyone (including unauthenticated users)';

-- Storage objects
ALTER TABLE storage.objects ENABLE ROW LEVEL SECURITY;
ALTER TABLE storage.objects FORCE ROW LEVEL SECURITY;

-- Admins and service roles can do everything with objects
CREATE POLICY storage_objects_admin ON storage.objects
    FOR ALL
    USING (auth.current_user_role() IN ('dashboard_admin', 'service_role'))
    WITH CHECK (auth.current_user_role() IN ('dashboard_admin', 'service_role'));

COMMENT ON POLICY storage_objects_admin ON storage.objects IS 'Dashboard admins and service role have full access to all objects';

-- Owners can do everything with their files
CREATE POLICY storage_objects_owner ON storage.objects
    FOR ALL
    USING (auth.current_user_id() = owner_id)
    WITH CHECK (auth.current_user_id() = owner_id);

COMMENT ON POLICY storage_objects_owner ON storage.objects IS 'Users can fully manage their own files';

-- Anyone can read files in public buckets (unauthenticated access allowed)
CREATE POLICY storage_objects_public_read ON storage.objects
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM storage.buckets
            WHERE buckets.id = objects.bucket_id
            AND buckets.public = true
        )
    );

COMMENT ON POLICY storage_objects_public_read ON storage.objects IS 'Files in public buckets are readable by everyone (including unauthenticated users)';

-- Users can read files shared with them (via object_permissions)
CREATE POLICY storage_objects_shared_read ON storage.objects
    FOR SELECT
    USING (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM storage.object_permissions
            WHERE object_permissions.object_id = objects.id
            AND object_permissions.user_id = auth.current_user_id()
            AND object_permissions.permission IN ('read', 'write')
        )
    );

COMMENT ON POLICY storage_objects_shared_read ON storage.objects IS 'Users can read files that have been shared with them';

-- Users can update/delete files shared with them with write permission
CREATE POLICY storage_objects_shared_write ON storage.objects
    FOR UPDATE
    USING (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM storage.object_permissions
            WHERE object_permissions.object_id = objects.id
            AND object_permissions.user_id = auth.current_user_id()
            AND object_permissions.permission = 'write'
        )
    );

COMMENT ON POLICY storage_objects_shared_write ON storage.objects IS 'Users can update files that have been shared with them with write permission';

CREATE POLICY storage_objects_shared_delete ON storage.objects
    FOR DELETE
    USING (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM storage.object_permissions
            WHERE object_permissions.object_id = objects.id
            AND object_permissions.user_id = auth.current_user_id()
            AND object_permissions.permission = 'write'
        )
    );

COMMENT ON POLICY storage_objects_shared_delete ON storage.objects IS 'Users can delete files that have been shared with them with write permission';

-- Authenticated users can insert objects (owner_id will be set by application)
CREATE POLICY storage_objects_insert ON storage.objects
    FOR INSERT
    WITH CHECK (
        auth.current_user_role() IN ('dashboard_admin', 'service_role')
        OR (auth.current_user_id() IS NOT NULL AND auth.current_user_id() = owner_id)
    );

COMMENT ON POLICY storage_objects_insert ON storage.objects IS 'Users can upload files (owner_id must match their user ID). Public buckets allow READ but not WRITE for unauthenticated users.';

-- Storage object permissions
ALTER TABLE storage.object_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE storage.object_permissions FORCE ROW LEVEL SECURITY;

-- Admins can manage all permissions
CREATE POLICY storage_object_permissions_admin ON storage.object_permissions
    FOR ALL
    USING (auth.current_user_role() IN ('dashboard_admin', 'service_role'))
    WITH CHECK (auth.current_user_role() IN ('dashboard_admin', 'service_role'));

COMMENT ON POLICY storage_object_permissions_admin ON storage.object_permissions IS 'Dashboard admins and service role can manage all file sharing permissions';

-- Owners can share their own files
CREATE POLICY storage_object_permissions_owner_manage ON storage.object_permissions
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM storage.objects
            WHERE objects.id = object_permissions.object_id
            AND objects.owner_id = auth.current_user_id()
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM storage.objects
            WHERE objects.id = object_permissions.object_id
            AND objects.owner_id = auth.current_user_id()
        )
    );

COMMENT ON POLICY storage_object_permissions_owner_manage ON storage.object_permissions IS 'File owners can manage sharing permissions for their files';

-- Users can view permissions for files shared with them
CREATE POLICY storage_object_permissions_view_shared ON storage.object_permissions
    FOR SELECT
    USING (
        user_id = auth.current_user_id()
    );

COMMENT ON POLICY storage_object_permissions_view_shared ON storage.object_permissions IS 'Users can view sharing permissions for files shared with them';

-- ============================================================================
-- CUSTOM RLS POLICY EXAMPLES
-- ============================================================================
--
-- Below are examples of custom RLS policies you can add to storage.objects
-- to implement specific access patterns for your application.
--
-- EXAMPLE 1: User folder restriction
-- Restrict users to only upload files to paths matching their user ID
-- Useful for user-uploads bucket where each user has their own folder
--
-- CREATE POLICY user_uploads_path_restriction ON storage.objects
--     FOR INSERT
--     WITH CHECK (
--         bucket_id = 'user-uploads'
--         AND path LIKE (auth.current_user_id()::TEXT || '/%')
--         AND owner_id = auth.current_user_id()
--     );
--
-- EXAMPLE 2: Role-based bucket access
-- Restrict certain buckets to users with specific roles
--
-- CREATE POLICY premium_content_access ON storage.objects
--     FOR SELECT
--     USING (
--         bucket_id = 'premium-content'
--         AND (
--             auth.current_user_role() IN ('dashboard_admin', 'service_role')
--             OR EXISTS (
--                 SELECT 1 FROM auth.users
--                 WHERE id = auth.current_user_id()
--                 AND user_metadata->>'subscription' = 'premium'
--             )
--         )
--     );
--
-- EXAMPLE 3: Read-only public bucket with admin-only writes
-- Allow everyone to read but only admins to write
--
-- CREATE POLICY public_assets_read_only ON storage.objects
--     FOR SELECT
--     USING (bucket_id = 'public-assets');
--
-- CREATE POLICY public_assets_admin_write ON storage.objects
--     FOR INSERT
--     WITH CHECK (
--         bucket_id = 'public-assets'
--         AND auth.current_user_role() = 'dashboard_admin'
--     );
--
-- EXAMPLE 4: Team/organization-based access
-- Allow users in the same organization to access each other's files
--
-- CREATE POLICY organization_files_access ON storage.objects
--     FOR SELECT
--     USING (
--         bucket_id = 'team-files'
--         AND EXISTS (
--             SELECT 1 FROM auth.users owner
--             JOIN auth.users viewer ON owner.user_metadata->>'org_id' = viewer.user_metadata->>'org_id'
--             WHERE owner.id = objects.owner_id
--             AND viewer.id = auth.current_user_id()
--         )
--     );
--
-- EXAMPLE 5: Time-based access restrictions
-- Only allow access to files during certain hours or after certain dates
--
-- CREATE POLICY scheduled_content_access ON storage.objects
--     FOR SELECT
--     USING (
--         bucket_id = 'scheduled-releases'
--         AND (
--             auth.current_user_role() = 'dashboard_admin'
--             OR (objects.metadata->>'release_date')::TIMESTAMPTZ <= NOW()
--         )
--     );
--

-- ============================================================================
-- STORAGE RLS FIXES - Prevent Infinite Recursion
-- ============================================================================

-- SECURITY DEFINER function to check if bucket is public
-- This prevents infinite recursion when RLS policies on storage.objects
-- reference storage.buckets (which also has RLS enabled)
CREATE OR REPLACE FUNCTION storage.is_bucket_public(bucket_name TEXT)
RETURNS BOOLEAN
LANGUAGE plpgsql
SECURITY DEFINER
STABLE
AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM storage.buckets
        WHERE id = bucket_name AND public = true
    );
END;
$$;

COMMENT ON FUNCTION storage.is_bucket_public IS
    'Check if a bucket is public, bypassing RLS to prevent infinite recursion';

-- SECURITY DEFINER function to check object permissions
-- This prevents infinite recursion when RLS policies on storage.objects
-- reference storage.object_permissions (which also has RLS enabled)
CREATE OR REPLACE FUNCTION storage.has_object_permission(
    p_object_id UUID,
    p_user_id UUID,
    p_permission TEXT
)
RETURNS BOOLEAN
LANGUAGE plpgsql
SECURITY DEFINER
STABLE
AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM storage.object_permissions
        WHERE object_id = p_object_id
        AND user_id = p_user_id
        AND (permission = p_permission OR (p_permission = 'read' AND permission = 'write'))
    );
END;
$$;

COMMENT ON FUNCTION storage.has_object_permission IS
    'Check if user has permission on object, bypassing RLS to prevent infinite recursion';

-- Supabase compatibility: storage.foldername() extracts folder path from object name
CREATE OR REPLACE FUNCTION storage.foldername(name TEXT)
RETURNS TEXT[] AS $$
DECLARE
    path_parts TEXT[];
    folder_parts TEXT[];
BEGIN
    IF name IS NULL OR name = '' THEN
        RETURN ARRAY[]::TEXT[];
    END IF;

    -- Split the path by '/' to get folder structure
    path_parts := string_to_array(name, '/');

    -- Remove the last element (filename) to get just folders
    IF array_length(path_parts, 1) > 1 THEN
        folder_parts := path_parts[1:array_length(path_parts, 1) - 1];
    ELSE
        -- No folders, just a filename at root
        folder_parts := ARRAY[]::TEXT[];
    END IF;

    RETURN folder_parts;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMENT ON FUNCTION storage.foldername(TEXT) IS 'Supabase-compatible function that extracts folder path components from an object name/path. Returns array of folder names. Use [1] to get first folder, [2] for second, etc.';

-- Update storage.objects public read policy to use SECURITY DEFINER function
DROP POLICY IF EXISTS storage_objects_public_read ON storage.objects;
CREATE POLICY storage_objects_public_read ON storage.objects
    FOR SELECT
    USING (storage.is_bucket_public(bucket_id));

-- Update storage.objects shared read policy to use SECURITY DEFINER function
DROP POLICY IF EXISTS storage_objects_shared_read ON storage.objects;
CREATE POLICY storage_objects_shared_read ON storage.objects
    FOR SELECT
    USING (
        auth.current_user_id() IS NOT NULL
        AND storage.has_object_permission(id, auth.current_user_id(), 'read')
    );

-- Update storage.objects shared write policy to use SECURITY DEFINER function
DROP POLICY IF EXISTS storage_objects_shared_write ON storage.objects;
CREATE POLICY storage_objects_shared_write ON storage.objects
    FOR UPDATE
    USING (
        auth.current_user_id() IS NOT NULL
        AND storage.has_object_permission(id, auth.current_user_id(), 'write')
    );

-- Update storage.objects shared delete policy to use SECURITY DEFINER function
DROP POLICY IF EXISTS storage_objects_shared_delete ON storage.objects;
CREATE POLICY storage_objects_shared_delete ON storage.objects
    FOR DELETE
    USING (
        auth.current_user_id() IS NOT NULL
        AND storage.has_object_permission(id, auth.current_user_id(), 'write')
    );

-- Fix INSERT policy to prevent unauthenticated uploads to public buckets
-- Public buckets should allow READ but not WRITE for unauthenticated users
DROP POLICY IF EXISTS storage_objects_insert ON storage.objects;
CREATE POLICY storage_objects_insert ON storage.objects
    FOR INSERT
    WITH CHECK (
        auth.current_user_role() IN ('dashboard_admin', 'service_role')
        OR (auth.current_user_id() IS NOT NULL AND auth.current_user_id() = owner_id)
    );
