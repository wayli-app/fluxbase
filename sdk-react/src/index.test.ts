/**
 * Tests for module exports
 */

import { describe, it, expect } from 'vitest';
import * as exports from './index';

describe('Module Exports', () => {
  describe('Context exports', () => {
    it('should export FluxbaseProvider', () => {
      expect(exports.FluxbaseProvider).toBeDefined();
    });

    it('should export useFluxbaseClient', () => {
      expect(exports.useFluxbaseClient).toBeDefined();
    });
  });

  describe('Auth hook exports', () => {
    it('should export useAuth', () => {
      expect(exports.useAuth).toBeDefined();
    });

    it('should export useUser', () => {
      expect(exports.useUser).toBeDefined();
    });

    it('should export useSession', () => {
      expect(exports.useSession).toBeDefined();
    });

    it('should export useSignIn', () => {
      expect(exports.useSignIn).toBeDefined();
    });

    it('should export useSignUp', () => {
      expect(exports.useSignUp).toBeDefined();
    });

    it('should export useSignOut', () => {
      expect(exports.useSignOut).toBeDefined();
    });

    it('should export useUpdateUser', () => {
      expect(exports.useUpdateUser).toBeDefined();
    });
  });

  describe('CAPTCHA hook exports', () => {
    it('should export useCaptchaConfig', () => {
      expect(exports.useCaptchaConfig).toBeDefined();
    });

    it('should export useCaptcha', () => {
      expect(exports.useCaptcha).toBeDefined();
    });

    it('should export isCaptchaRequiredForEndpoint', () => {
      expect(exports.isCaptchaRequiredForEndpoint).toBeDefined();
    });
  });

  describe('Auth config exports', () => {
    it('should export useAuthConfig', () => {
      expect(exports.useAuthConfig).toBeDefined();
    });
  });

  describe('SAML hook exports', () => {
    it('should export useSAMLProviders', () => {
      expect(exports.useSAMLProviders).toBeDefined();
    });

    it('should export useGetSAMLLoginUrl', () => {
      expect(exports.useGetSAMLLoginUrl).toBeDefined();
    });

    it('should export useSignInWithSAML', () => {
      expect(exports.useSignInWithSAML).toBeDefined();
    });

    it('should export useHandleSAMLCallback', () => {
      expect(exports.useHandleSAMLCallback).toBeDefined();
    });

    it('should export useSAMLMetadataUrl', () => {
      expect(exports.useSAMLMetadataUrl).toBeDefined();
    });
  });

  describe('GraphQL hook exports', () => {
    it('should export useGraphQLQuery', () => {
      expect(exports.useGraphQLQuery).toBeDefined();
    });

    it('should export useGraphQLMutation', () => {
      expect(exports.useGraphQLMutation).toBeDefined();
    });

    it('should export useGraphQLIntrospection', () => {
      expect(exports.useGraphQLIntrospection).toBeDefined();
    });

    it('should export useGraphQL', () => {
      expect(exports.useGraphQL).toBeDefined();
    });
  });

  describe('Database query hook exports', () => {
    it('should export useFluxbaseQuery', () => {
      expect(exports.useFluxbaseQuery).toBeDefined();
    });

    it('should export useTable', () => {
      expect(exports.useTable).toBeDefined();
    });

    it('should export useInsert', () => {
      expect(exports.useInsert).toBeDefined();
    });

    it('should export useUpdate', () => {
      expect(exports.useUpdate).toBeDefined();
    });

    it('should export useUpsert', () => {
      expect(exports.useUpsert).toBeDefined();
    });

    it('should export useDelete', () => {
      expect(exports.useDelete).toBeDefined();
    });
  });

  describe('Realtime hook exports', () => {
    it('should export useRealtime', () => {
      expect(exports.useRealtime).toBeDefined();
    });

    it('should export useTableSubscription', () => {
      expect(exports.useTableSubscription).toBeDefined();
    });

    it('should export useTableInserts', () => {
      expect(exports.useTableInserts).toBeDefined();
    });

    it('should export useTableUpdates', () => {
      expect(exports.useTableUpdates).toBeDefined();
    });

    it('should export useTableDeletes', () => {
      expect(exports.useTableDeletes).toBeDefined();
    });
  });

  describe('Storage hook exports', () => {
    it('should export useStorageList', () => {
      expect(exports.useStorageList).toBeDefined();
    });

    it('should export useStorageUpload', () => {
      expect(exports.useStorageUpload).toBeDefined();
    });

    it('should export useStorageUploadWithProgress', () => {
      expect(exports.useStorageUploadWithProgress).toBeDefined();
    });

    it('should export useStorageDownload', () => {
      expect(exports.useStorageDownload).toBeDefined();
    });

    it('should export useStorageDelete', () => {
      expect(exports.useStorageDelete).toBeDefined();
    });

    it('should export useStoragePublicUrl', () => {
      expect(exports.useStoragePublicUrl).toBeDefined();
    });

    it('should export useStorageTransformUrl', () => {
      expect(exports.useStorageTransformUrl).toBeDefined();
    });

    it('should export useStorageSignedUrl', () => {
      expect(exports.useStorageSignedUrl).toBeDefined();
    });

    it('should export useStorageSignedUrlWithOptions', () => {
      expect(exports.useStorageSignedUrlWithOptions).toBeDefined();
    });

    it('should export useStorageBuckets', () => {
      expect(exports.useStorageBuckets).toBeDefined();
    });

    it('should export useCreateBucket', () => {
      expect(exports.useCreateBucket).toBeDefined();
    });

    it('should export useDeleteBucket', () => {
      expect(exports.useDeleteBucket).toBeDefined();
    });
  });

  describe('Admin hook exports', () => {
    it('should export useAdminAuth', () => {
      expect(exports.useAdminAuth).toBeDefined();
    });

    it('should export useUsers', () => {
      expect(exports.useUsers).toBeDefined();
    });

    it('should export useClientKeys', () => {
      expect(exports.useClientKeys).toBeDefined();
    });

    it('should export useAPIKeys (deprecated alias)', () => {
      expect(exports.useAPIKeys).toBeDefined();
      expect(exports.useAPIKeys).toBe(exports.useClientKeys);
    });

    it('should export useWebhooks', () => {
      expect(exports.useWebhooks).toBeDefined();
    });

    it('should export useAppSettings', () => {
      expect(exports.useAppSettings).toBeDefined();
    });

    it('should export useSystemSettings', () => {
      expect(exports.useSystemSettings).toBeDefined();
    });
  });

  describe('Function hook exports', () => {
    it('should export useFunctions', () => {
      expect(exports.useFunctions).toBeDefined();
    });
  });

  describe('Job hook exports', () => {
    it('should export useJobs', () => {
      expect(exports.useJobs).toBeDefined();
    });
    it('should export useCancelJob', () => {
      expect(exports.useCancelJob).toBeDefined();
    });
    it('should export useRetryJob', () => {
      expect(exports.useRetryJob).toBeDefined();
    });
  });

  describe('Branching hook exports', () => {
    it('should export useCreateBranch', () => {
      expect(exports.useCreateBranch).toBeDefined();
    });
    it('should export useDeleteBranch', () => {
      expect(exports.useDeleteBranch).toBeDefined();
    });
    it('should export useResetBranch', () => {
      expect(exports.useResetBranch).toBeDefined();
    });
  });

  describe('DDL hook exports', () => {
    it('should export useSchemas', () => {
      expect(exports.useSchemas).toBeDefined();
    });
    it('should export useTables', () => {
      expect(exports.useTables).toBeDefined();
    });
    it('should export useCreateSchema', () => {
      expect(exports.useCreateSchema).toBeDefined();
    });
    it('should export useDeleteTable', () => {
      expect(exports.useDeleteTable).toBeDefined();
    });
  });

  describe('Impersonation hook exports', () => {
    it('should export useImpersonationSessions', () => {
      expect(exports.useImpersonationSessions).toBeDefined();
    });
    it('should export useCurrentImpersonation', () => {
      expect(exports.useCurrentImpersonation).toBeDefined();
    });
    it('should export useImpersonateUser', () => {
      expect(exports.useImpersonateUser).toBeDefined();
    });
    it('should export useStopImpersonation', () => {
      expect(exports.useStopImpersonation).toBeDefined();
    });
  });

  describe('Migration hook exports', () => {
    it('should export useMigrations', () => {
      expect(exports.useMigrations).toBeDefined();
    });
    it('should export useApplyMigration', () => {
      expect(exports.useApplyMigration).toBeDefined();
    });
    it('should export useRollbackMigration', () => {
      expect(exports.useRollbackMigration).toBeDefined();
    });
    it('should export useSyncMigrations', () => {
      expect(exports.useSyncMigrations).toBeDefined();
    });
  });

  describe('Total export count', () => {
    it('should export the expected number of items', () => {
      // Count the exports (functions and types are counted)
      const exportKeys = Object.keys(exports);

      // We expect at least 50 exports (hooks, types, etc.)
      expect(exportKeys.length).toBeGreaterThanOrEqual(50);
    });
  });
});
