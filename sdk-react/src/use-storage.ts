/**
 * Storage hooks for Fluxbase SDK
 */

import { useMutation, useQuery, useQueryClient, type UseQueryOptions } from '@tanstack/react-query'
import { useFluxbaseClient } from './context'
import type { ListOptions, UploadOptions } from '@fluxbase/sdk'

/**
 * Hook to list files in a bucket
 */
export function useStorageList(
  bucket: string,
  options?: ListOptions & Omit<UseQueryOptions<any[], Error>, 'queryKey' | 'queryFn'>
) {
  const client = useFluxbaseClient()
  const { prefix, limit, offset, ...queryOptions } = options || {}

  return useQuery({
    queryKey: ['fluxbase', 'storage', bucket, 'list', { prefix, limit, offset }],
    queryFn: async () => {
      const { data, error } = await client.storage.from(bucket).list({ prefix, limit, offset })

      if (error) {
        throw error
      }

      return data || []
    },
    ...queryOptions,
  })
}

/**
 * Hook to upload a file to a bucket
 */
export function useStorageUpload(bucket: string) {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (params: {
      path: string
      file: File | Blob | ArrayBuffer
      options?: UploadOptions
    }) => {
      const { path, file, options } = params
      const { data, error } = await client.storage.from(bucket).upload(path, file, options)

      if (error) {
        throw error
      }

      return data
    },
    onSuccess: () => {
      // Invalidate list queries for this bucket
      queryClient.invalidateQueries({ queryKey: ['fluxbase', 'storage', bucket, 'list'] })
    },
  })
}

/**
 * Hook to download a file from a bucket
 */
export function useStorageDownload(bucket: string, path: string | null, enabled = true) {
  const client = useFluxbaseClient()

  return useQuery({
    queryKey: ['fluxbase', 'storage', bucket, 'download', path],
    queryFn: async () => {
      if (!path) {
        return null
      }

      const { data, error } = await client.storage.from(bucket).download(path)

      if (error) {
        throw error
      }

      return data
    },
    enabled: enabled && !!path,
  })
}

/**
 * Hook to delete files from a bucket
 */
export function useStorageDelete(bucket: string) {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (paths: string[]) => {
      const { error } = await client.storage.from(bucket).remove(paths)

      if (error) {
        throw error
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fluxbase', 'storage', bucket, 'list'] })
    },
  })
}

/**
 * Hook to get a public URL for a file
 */
export function useStoragePublicUrl(bucket: string, path: string | null) {
  const client = useFluxbaseClient()

  if (!path) {
    return null
  }

  const { data } = client.storage.from(bucket).getPublicUrl(path)
  return data.publicUrl
}

/**
 * Hook to create a signed URL
 */
export function useStorageSignedUrl(bucket: string, path: string | null, expiresIn?: number) {
  const client = useFluxbaseClient()

  return useQuery({
    queryKey: ['fluxbase', 'storage', bucket, 'signed-url', path, expiresIn],
    queryFn: async () => {
      if (!path) {
        return null
      }

      const { data, error } = await client.storage.from(bucket).createSignedUrl(path, { expiresIn })

      if (error) {
        throw error
      }

      return data?.signedUrl || null
    },
    enabled: !!path,
    staleTime: expiresIn ? expiresIn * 1000 - 60000 : 1000 * 60 * 50, // Refresh 1 minute before expiry
  })
}

/**
 * Hook to move a file
 */
export function useStorageMove(bucket: string) {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (params: { fromPath: string; toPath: string }) => {
      const { fromPath, toPath } = params
      const { data, error } = await client.storage.from(bucket).move(fromPath, toPath)

      if (error) {
        throw error
      }

      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fluxbase', 'storage', bucket, 'list'] })
    },
  })
}

/**
 * Hook to copy a file
 */
export function useStorageCopy(bucket: string) {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (params: { fromPath: string; toPath: string }) => {
      const { fromPath, toPath } = params
      const { data, error } = await client.storage.from(bucket).copy(fromPath, toPath)

      if (error) {
        throw error
      }

      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fluxbase', 'storage', bucket, 'list'] })
    },
  })
}

/**
 * Hook to manage buckets
 */
export function useStorageBuckets() {
  const client = useFluxbaseClient()

  return useQuery({
    queryKey: ['fluxbase', 'storage', 'buckets'],
    queryFn: async () => {
      const { data, error } = await client.storage.listBuckets()

      if (error) {
        throw error
      }

      return data || []
    },
  })
}

/**
 * Hook to create a bucket
 */
export function useCreateBucket() {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (bucketName: string) => {
      const { error } = await client.storage.createBucket(bucketName)

      if (error) {
        throw error
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fluxbase', 'storage', 'buckets'] })
    },
  })
}

/**
 * Hook to delete a bucket
 */
export function useDeleteBucket() {
  const client = useFluxbaseClient()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (bucketName: string) => {
      const { error } = await client.storage.deleteBucket(bucketName)

      if (error) {
        throw error
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['fluxbase', 'storage', 'buckets'] })
    },
  })
}
