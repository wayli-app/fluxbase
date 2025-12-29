/**
 * Storage hooks for Fluxbase SDK
 */

import { useState } from "react";
import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type {
  ListOptions,
  UploadOptions,
  UploadProgress,
  TransformOptions,
  SignedUrlOptions,
} from "@fluxbase/sdk";

/**
 * Hook to list files in a bucket
 */
export function useStorageList(
  bucket: string,
  options?: ListOptions &
    Omit<UseQueryOptions<any[], Error>, "queryKey" | "queryFn">,
) {
  const client = useFluxbaseClient();
  const { prefix, limit, offset, ...queryOptions } = options || {};

  return useQuery({
    queryKey: [
      "fluxbase",
      "storage",
      bucket,
      "list",
      { prefix, limit, offset },
    ],
    queryFn: async () => {
      const { data, error } = await client.storage
        .from(bucket)
        .list({ prefix, limit, offset });

      if (error) {
        throw error;
      }

      return data || [];
    },
    ...queryOptions,
  });
}

/**
 * Hook to upload a file to a bucket
 *
 * Note: You can track upload progress by passing an `onUploadProgress` callback in the options:
 *
 * @example
 * ```tsx
 * const upload = useStorageUpload('avatars')
 *
 * upload.mutate({
 *   path: 'user.jpg',
 *   file: file,
 *   options: {
 *     onUploadProgress: (progress) => {
 *       console.log(`${progress.percentage}% uploaded`)
 *     }
 *   }
 * })
 * ```
 *
 * For automatic progress state management, use `useStorageUploadWithProgress` instead.
 */
export function useStorageUpload(bucket: string) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: {
      path: string;
      file: File | Blob | ArrayBuffer;
      options?: UploadOptions;
    }) => {
      const { path, file, options } = params;
      const { data, error } = await client.storage
        .from(bucket)
        .upload(path, file, options);

      if (error) {
        throw error;
      }

      return data;
    },
    onSuccess: () => {
      // Invalidate list queries for this bucket
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "storage", bucket, "list"],
      });
    },
  });
}

/**
 * Hook to upload a file to a bucket with built-in progress tracking
 *
 * @example
 * ```tsx
 * const { upload, progress, reset } = useStorageUploadWithProgress('avatars')
 *
 * // Upload with automatic progress tracking
 * upload.mutate({
 *   path: 'user.jpg',
 *   file: file
 * })
 *
 * // Display progress
 * console.log(progress) // { loaded: 1024, total: 2048, percentage: 50 }
 * ```
 */
export function useStorageUploadWithProgress(bucket: string) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();
  const [progress, setProgress] = useState<UploadProgress | null>(null);

  const mutation = useMutation({
    mutationFn: async (params: {
      path: string;
      file: File | Blob | ArrayBuffer;
      options?: Omit<UploadOptions, "onUploadProgress">;
    }) => {
      const { path, file, options } = params;

      // Reset progress at the start of upload
      setProgress({ loaded: 0, total: 0, percentage: 0 });

      const { data, error } = await client.storage
        .from(bucket)
        .upload(path, file, {
          ...options,
          onUploadProgress: (p: import("@fluxbase/sdk").UploadProgress) => {
            setProgress(p);
          },
        });

      if (error) {
        throw error;
      }

      return data;
    },
    onSuccess: () => {
      // Invalidate list queries for this bucket
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "storage", bucket, "list"],
      });
    },
    onError: () => {
      // Reset progress on error
      setProgress(null);
    },
  });

  return {
    upload: mutation,
    progress,
    reset: () => setProgress(null),
  };
}

/**
 * Hook to download a file from a bucket
 */
export function useStorageDownload(
  bucket: string,
  path: string | null,
  enabled = true,
) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "storage", bucket, "download", path],
    queryFn: async () => {
      if (!path) {
        return null;
      }

      const { data, error } = await client.storage.from(bucket).download(path);

      if (error) {
        throw error;
      }

      return data;
    },
    enabled: enabled && !!path,
  });
}

/**
 * Hook to delete files from a bucket
 */
export function useStorageDelete(bucket: string) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (paths: string[]) => {
      const { error } = await client.storage.from(bucket).remove(paths);

      if (error) {
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "storage", bucket, "list"],
      });
    },
  });
}

/**
 * Hook to get a public URL for a file
 */
export function useStoragePublicUrl(bucket: string, path: string | null) {
  const client = useFluxbaseClient();

  if (!path) {
    return null;
  }

  const { data } = client.storage.from(bucket).getPublicUrl(path);
  return data.publicUrl;
}

/**
 * Hook to get a public URL for an image with transformations applied
 *
 * Only works for image files (JPEG, PNG, WebP, GIF, AVIF, etc.)
 *
 * @param bucket - The storage bucket name
 * @param path - The file path (or null to disable)
 * @param transform - Transformation options (width, height, format, quality, fit)
 *
 * @example
 * ```tsx
 * function ImageThumbnail({ path }: { path: string }) {
 *   const url = useStorageTransformUrl('images', path, {
 *     width: 300,
 *     height: 200,
 *     format: 'webp',
 *     quality: 85,
 *     fit: 'cover'
 *   });
 *
 *   return <img src={url || ''} alt="Thumbnail" />;
 * }
 * ```
 */
export function useStorageTransformUrl(
  bucket: string,
  path: string | null,
  transform: TransformOptions,
): string | null {
  const client = useFluxbaseClient();

  if (!path) {
    return null;
  }

  return client.storage.from(bucket).getTransformUrl(path, transform);
}

/**
 * Hook to create a signed URL
 *
 * @deprecated Use useStorageSignedUrlWithOptions for more control including transforms
 */
export function useStorageSignedUrl(
  bucket: string,
  path: string | null,
  expiresIn?: number,
) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "storage", bucket, "signed-url", path, expiresIn],
    queryFn: async () => {
      if (!path) {
        return null;
      }

      const { data, error } = await client.storage
        .from(bucket)
        .createSignedUrl(path, { expiresIn });

      if (error) {
        throw error;
      }

      return data?.signedUrl || null;
    },
    enabled: !!path,
    staleTime: expiresIn ? expiresIn * 1000 - 60000 : 1000 * 60 * 50, // Refresh 1 minute before expiry
  });
}

/**
 * Hook to create a signed URL with full options including image transformations
 *
 * @param bucket - The storage bucket name
 * @param path - The file path (or null to disable)
 * @param options - Signed URL options including expiration and transforms
 *
 * @example
 * ```tsx
 * function SecureThumbnail({ path }: { path: string }) {
 *   const { data: url } = useStorageSignedUrlWithOptions('images', path, {
 *     expiresIn: 3600,
 *     transform: {
 *       width: 400,
 *       height: 300,
 *       format: 'webp',
 *       quality: 85,
 *       fit: 'cover'
 *     }
 *   });
 *
 *   return <img src={url || ''} alt="Secure Thumbnail" />;
 * }
 * ```
 */
export function useStorageSignedUrlWithOptions(
  bucket: string,
  path: string | null,
  options?: SignedUrlOptions,
) {
  const client = useFluxbaseClient();
  const expiresIn = options?.expiresIn;

  // Create a stable cache key from transform options
  const transformKey = options?.transform
    ? JSON.stringify(options.transform)
    : null;

  return useQuery({
    queryKey: [
      "fluxbase",
      "storage",
      bucket,
      "signed-url",
      path,
      expiresIn,
      transformKey,
    ],
    queryFn: async () => {
      if (!path) {
        return null;
      }

      const { data, error } = await client.storage
        .from(bucket)
        .createSignedUrl(path, options);

      if (error) {
        throw error;
      }

      return data?.signedUrl || null;
    },
    enabled: !!path,
    staleTime: expiresIn ? expiresIn * 1000 - 60000 : 1000 * 60 * 50, // Refresh 1 minute before expiry
  });
}

/**
 * Hook to move a file
 */
export function useStorageMove(bucket: string) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { fromPath: string; toPath: string }) => {
      const { fromPath, toPath } = params;
      const { data, error } = await client.storage
        .from(bucket)
        .move(fromPath, toPath);

      if (error) {
        throw error;
      }

      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "storage", bucket, "list"],
      });
    },
  });
}

/**
 * Hook to copy a file
 */
export function useStorageCopy(bucket: string) {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (params: { fromPath: string; toPath: string }) => {
      const { fromPath, toPath } = params;
      const { data, error } = await client.storage
        .from(bucket)
        .copy(fromPath, toPath);

      if (error) {
        throw error;
      }

      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "storage", bucket, "list"],
      });
    },
  });
}

/**
 * Hook to manage buckets
 */
export function useStorageBuckets() {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "storage", "buckets"],
    queryFn: async () => {
      const { data, error } = await client.storage.listBuckets();

      if (error) {
        throw error;
      }

      return data || [];
    },
  });
}

/**
 * Hook to create a bucket
 */
export function useCreateBucket() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (bucketName: string) => {
      const { error } = await client.storage.createBucket(bucketName);

      if (error) {
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "storage", "buckets"],
      });
    },
  });
}

/**
 * Hook to delete a bucket
 */
export function useDeleteBucket() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (bucketName: string) => {
      const { error } = await client.storage.deleteBucket(bucketName);

      if (error) {
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "storage", "buckets"],
      });
    },
  });
}
