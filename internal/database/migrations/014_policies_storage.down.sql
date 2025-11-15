-- Drop storage helper functions
DROP FUNCTION IF EXISTS storage.foldername(TEXT);
DROP FUNCTION IF EXISTS storage.has_object_permission(UUID, UUID, TEXT);
DROP FUNCTION IF EXISTS storage.is_bucket_public(TEXT);

-- Drop all storage schema RLS policies
DROP POLICY IF EXISTS storage_object_permissions_view_shared ON storage.object_permissions;
DROP POLICY IF EXISTS storage_object_permissions_owner_manage ON storage.object_permissions;
DROP POLICY IF EXISTS storage_object_permissions_admin ON storage.object_permissions;
DROP POLICY IF EXISTS storage_objects_insert ON storage.objects;
DROP POLICY IF EXISTS storage_objects_shared_delete ON storage.objects;
DROP POLICY IF EXISTS storage_objects_shared_write ON storage.objects;
DROP POLICY IF EXISTS storage_objects_shared_read ON storage.objects;
DROP POLICY IF EXISTS storage_objects_public_read ON storage.objects;
DROP POLICY IF EXISTS storage_objects_owner ON storage.objects;
DROP POLICY IF EXISTS storage_objects_admin ON storage.objects;
DROP POLICY IF EXISTS storage_buckets_public_view ON storage.buckets;
DROP POLICY IF EXISTS storage_buckets_admin ON storage.buckets;
