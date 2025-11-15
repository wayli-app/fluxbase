-- Drop storage tables in reverse dependency order
DROP TABLE IF EXISTS storage.object_permissions;
DROP TABLE IF EXISTS storage.objects;
DROP TABLE IF EXISTS storage.buckets;
