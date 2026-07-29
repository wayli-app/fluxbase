/**
 * Storage stores for the Fluxbase Svelte SDK.
 *
 * `storageList` / `storageBuckets` are reactive queries; uploads, downloads,
 * and deletes are mutations that invalidate the relevant list cache.
 */

import { createQuery, createMutation } from "@tanstack/svelte-query";
import type { FluxbaseClient } from "@nimbleflux/fluxbase-sdk";
import type { StoreDeps } from "../auth/store";

/** List files in a bucket reactively. */
export function createStorageList(
  { client, queryClient }: StoreDeps,
  bucket: string,
  options?: { queryKey?: unknown[] },
) {
  const tenantId = client.getTenantId();
  return createQuery(
    {
      queryKey: [
        "fluxbase",
        tenantId ?? null,
        "storage",
        bucket,
        ...(options?.queryKey ?? []),
      ],
      queryFn: async () => {
        const { data, error } = await client.storage.from(bucket).list();
        if (error) throw error;
        return data ?? [];
      },
    },
    queryClient,
  );
}

/** List storage buckets reactively. */
export function createStorageBuckets({ client, queryClient }: StoreDeps) {
  const tenantId = client.getTenantId();
  return createQuery(
    {
      queryKey: ["fluxbase", tenantId ?? null, "storage", "buckets"],
      queryFn: async () => {
        const { data, error } = await client.storage.listBuckets();
        if (error) throw error;
        return data ?? [];
      },
    },
    queryClient,
  );
}

/** Upload a file, then invalidate the bucket's file list. */
export function createStorageUpload(
  { client, queryClient }: StoreDeps,
  bucket: string,
) {
  const tenantId = client.getTenantId();
  return createMutation(
    {
      mutationFn: async (params: { path: string; file: Blob | File }) => {
        const { data, error } = await client.storage
          .from(bucket)
          .upload(params.path, params.file);
        if (error) throw error;
        return data;
      },
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: ["fluxbase", tenantId ?? null, "storage", bucket],
        });
      },
    },
    queryClient,
  );
}

/** Download a file. */
export function createStorageDownload(
  { client, queryClient }: StoreDeps,
  bucket: string,
) {
  return createMutation(
    {
      mutationFn: async (path: string) => {
        const { data, error } = await client.storage
          .from(bucket)
          .download(path);
        if (error) throw error;
        return data;
      },
    },
    queryClient,
  );
}

/** Delete a file, then invalidate the bucket's file list. */
export function createStorageDelete(
  { client, queryClient }: StoreDeps,
  bucket: string,
) {
  const tenantId = client.getTenantId();
  return createMutation(
    {
      mutationFn: async (paths: string | string[]) => {
        const { error } = await client.storage.from(bucket).remove(
          Array.isArray(paths) ? paths : [paths],
        );
        if (error) throw error;
      },
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: ["fluxbase", tenantId ?? null, "storage", bucket],
        });
      },
    },
    queryClient,
  );
}

/** Get a public URL for an object (synchronous helper, not a store). */
export function storagePublicUrl(
  client: FluxbaseClient,
  bucket: string,
  path: string,
): string {
  return client.storage.from(bucket).getPublicUrl(path).data.publicUrl;
}
