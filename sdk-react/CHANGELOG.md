# Changelog

All notable changes to @fluxbase/sdk-react will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-10-26

### Added

- Initial release of Fluxbase React hooks library
- **Authentication Hooks**
  - `useAuth`: Complete auth state and methods
  - `useUser`: Current user data with auto-refetch
  - `useSession`: Access current session
  - `useSignIn`: Sign in mutation hook
  - `useSignUp`: Sign up mutation hook
  - `useSignOut`: Sign out mutation hook
  - `useUpdateUser`: Update user profile hook
- **Database Query Hooks**
  - `useTable`: Query table data with filters, ordering, pagination
  - `useFluxbaseQuery`: Lower-level query hook for custom queries
  - `useInsert`: Insert rows with automatic cache invalidation
  - `useUpdate`: Update rows
  - `useUpsert`: Insert or update (upsert)
  - `useDelete`: Delete rows
- **Realtime Hooks**
  - `useRealtime`: Subscribe to database changes
  - `useTableSubscription`: Auto-refetch queries on table changes
  - `useTableInserts`: Listen to INSERT events
  - `useTableUpdates`: Listen to UPDATE events
  - `useTableDeletes`: Listen to DELETE events
- **Storage Hooks**
  - `useStorageList`: List files in bucket
  - `useStorageUpload`: Upload files
  - `useStorageDownload`: Download files
  - `useStorageDelete`: Delete files
  - `useStoragePublicUrl`: Get public URL
  - `useStorageSignedUrl`: Generate signed URLs
  - `useStorageMove`: Move files
  - `useStorageCopy`: Copy files
  - `useStorageBuckets`: List buckets
  - `useCreateBucket`: Create bucket
  - `useDeleteBucket`: Delete bucket
- **Context & Provider**
  - `FluxbaseProvider`: Provide Fluxbase client to React tree
  - `useFluxbaseClient`: Access Fluxbase client from context
- **TypeScript Support**
  - Full type safety for all hooks
  - Generic type parameters for table data
  - Type inference from query builders
- **Built on TanStack Query**
  - Automatic caching and refetching
  - Optimistic updates support
  - Request deduplication
  - Background refetching
  - Stale-while-revalidate pattern
  - SSR support

### Peer Dependencies

- `@fluxbase/sdk`: ^0.1.0
- `@tanstack/react-query`: ^5.0.0
- `react`: ^18.0.0 || ^19.0.0

[0.1.0]: https://github.com/fluxbase-eu/fluxbase/releases/tag/sdk-react-v0.1.0
