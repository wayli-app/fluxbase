/**
 * Storage client for file operations
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  FileObject,
  UploadOptions,
  UploadProgress,
  ListOptions,
  SignedUrlOptions,
  ShareFileOptions,
  FileShare,
  BucketSettings,
  Bucket,
} from "./types";

export class StorageBucket {
  private fetch: FluxbaseFetch;
  private bucketName: string;

  constructor(fetch: FluxbaseFetch, bucketName: string) {
    this.fetch = fetch;
    this.bucketName = bucketName;
  }

  /**
   * Upload a file to the bucket
   * @param path - The path/key for the file
   * @param file - The file to upload (File, Blob, or ArrayBuffer)
   * @param options - Upload options
   */
  async upload(
    path: string,
    file: File | Blob | ArrayBuffer,
    options?: UploadOptions,
  ): Promise<{ data: { id: string; path: string; fullPath: string } | null; error: Error | null }> {
    try {
      // Prepare FormData (common to both code paths)
      const formData = new FormData();

      // Convert to Blob if ArrayBuffer
      const blob = file instanceof ArrayBuffer ? new Blob([file]) : file;

      formData.append("file", blob);

      if (options?.contentType) {
        formData.append("content_type", options.contentType);
      }

      if (options?.metadata) {
        formData.append("metadata", JSON.stringify(options.metadata));
      }

      if (options?.cacheControl) {
        formData.append("cache_control", options.cacheControl);
      }

      if (options?.upsert !== undefined) {
        formData.append("upsert", String(options.upsert));
      }

      let response: any;

      // Use XMLHttpRequest for progress tracking if callback is provided
      if (options?.onUploadProgress) {
        response = await this.uploadWithProgress(path, formData, options.onUploadProgress);
      } else {
        // Use standard fetch for uploads without progress tracking
        response = await this.fetch.request<any>(
          `/api/v1/storage/${this.bucketName}/${path}`,
          {
            method: "POST",
            body: formData,
            headers: {}, // Let browser set Content-Type for FormData
          },
        );
      }

      // Return Supabase-compatible response format
      return {
        data: {
          id: response.id || response.key || path,
          path: path,
          fullPath: `${this.bucketName}/${path}`
        },
        error: null
      };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Upload with progress tracking using XMLHttpRequest
   * @private
   */
  private uploadWithProgress(
    path: string,
    formData: FormData,
    onProgress: (progress: UploadProgress) => void,
  ): Promise<any> {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      const url = `${this.fetch["baseUrl"]}/api/v1/storage/${this.bucketName}/${path}`;

      // Track upload progress
      xhr.upload.addEventListener('progress', (event) => {
        if (event.lengthComputable) {
          const percentage = Math.round((event.loaded / event.total) * 100);
          onProgress({
            loaded: event.loaded,
            total: event.total,
            percentage,
          });
        }
      });

      // Handle completion
      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            const response = JSON.parse(xhr.responseText);
            resolve(response);
          } catch (e) {
            resolve(xhr.responseText);
          }
        } else {
          try {
            const errorData = JSON.parse(xhr.responseText);
            reject(new Error(errorData.error || xhr.statusText));
          } catch (e) {
            reject(new Error(xhr.statusText));
          }
        }
      });

      // Handle errors
      xhr.addEventListener('error', () => {
        reject(new Error('Upload failed'));
      });

      xhr.addEventListener('abort', () => {
        reject(new Error('Upload aborted'));
      });

      // Open and send request
      xhr.open('POST', url);

      // Set authorization header if present
      const headers = this.fetch["defaultHeaders"];
      for (const [key, value] of Object.entries(headers)) {
        // Don't set Content-Type header - let browser handle it for FormData
        if (key.toLowerCase() !== 'content-type') {
          xhr.setRequestHeader(key, value);
        }
      }

      xhr.send(formData);
    });
  }

  /**
   * Download a file from the bucket
   * @param path - The path/key of the file
   */
  async download(
    path: string,
  ): Promise<{ data: Blob | null; error: Error | null }> {
    try {
      const response = await fetch(
        `${this.fetch["baseUrl"]}/api/v1/storage/${this.bucketName}/${path}`,
        {
          headers: this.fetch["defaultHeaders"],
        },
      );

      if (!response.ok) {
        throw new Error(`Failed to download file: ${response.statusText}`);
      }

      const blob = await response.blob();
      return { data: blob, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * List files in the bucket
   * Supports both Supabase-style list(path, options) and Fluxbase-style list(options)
   * @param pathOrOptions - The folder path or list options
   * @param maybeOptions - List options when first param is a path
   */
  async list(
    pathOrOptions?: string | ListOptions,
    maybeOptions?: ListOptions,
  ): Promise<{ data: FileObject[] | null; error: Error | null }> {
    try {
      const params = new URLSearchParams();

      // Determine if first arg is path or options
      let prefix: string | undefined;
      let options: ListOptions | undefined;

      if (typeof pathOrOptions === 'string') {
        // Supabase-style: list('path/', { limit: 10 })
        prefix = pathOrOptions;
        options = maybeOptions;
      } else {
        // Fluxbase-style: list({ prefix: 'path/', limit: 10 })
        options = pathOrOptions;
        prefix = options?.prefix;
      }

      if (prefix) {
        params.set("prefix", prefix);
      }

      if (options?.limit) {
        params.set("limit", String(options.limit));
      }

      if (options?.offset) {
        params.set("offset", String(options.offset));
      }

      const queryString = params.toString();
      const path = `/api/v1/storage/${this.bucketName}${queryString ? `?${queryString}` : ""}`;

      const response = await this.fetch.get<{ files: any[] }>(path);

      // Convert to FileObject format
      const files: FileObject[] = (response.files || []).map((file: any) => ({
        name: file.key || file.name,
        id: file.id,
        bucket_id: file.bucket || this.bucketName,
        created_at: file.last_modified || file.created_at,
        updated_at: file.updated_at,
        last_accessed_at: file.last_accessed_at,
        metadata: file.metadata,
      }));

      return { data: files, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Remove files from the bucket
   * @param paths - Array of file paths to remove
   */
  async remove(paths: string[]): Promise<{ data: FileObject[] | null; error: Error | null }> {
    try {
      const removedFiles: FileObject[] = [];

      // Delete files one by one (could be optimized with batch endpoint)
      for (const path of paths) {
        await this.fetch.delete(`/api/v1/storage/${this.bucketName}/${path}`);
        // Add to removed files list
        removedFiles.push({
          name: path,
          bucket_id: this.bucketName,
        });
      }

      return { data: removedFiles, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get a public URL for a file
   * @param path - The file path
   */
  getPublicUrl(path: string): { data: { publicUrl: string } } {
    const publicUrl = `${this.fetch["baseUrl"]}/api/v1/storage/${this.bucketName}/${path}`;
    return { data: { publicUrl } };
  }

  /**
   * Create a signed URL for temporary access to a file
   * @param path - The file path
   * @param options - Signed URL options
   */
  async createSignedUrl(
    path: string,
    options?: SignedUrlOptions,
  ): Promise<{ data: { signedUrl: string } | null; error: Error | null }> {
    try {
      const expiresIn = options?.expiresIn || 3600; // Default 1 hour

      const data = await this.fetch.post<{ signed_url: string }>(
        `/api/v1/storage/${this.bucketName}/sign/${path}`,
        { expires_in: expiresIn },
      );

      return { data: { signedUrl: data.signed_url }, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Move a file to a new location
   * @param fromPath - Current file path
   * @param toPath - New file path
   */
  async move(
    fromPath: string,
    toPath: string,
  ): Promise<{ data: { message: string } | null; error: Error | null }> {
    try {
      await this.fetch.post(
        `/api/v1/storage/${this.bucketName}/move`,
        {
          from_path: fromPath,
          to_path: toPath,
        },
      );

      return {
        data: { message: 'Successfully moved' },
        error: null
      };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Copy a file to a new location
   * @param fromPath - Source file path
   * @param toPath - Destination file path
   */
  async copy(
    fromPath: string,
    toPath: string,
  ): Promise<{ data: { path: string } | null; error: Error | null }> {
    try {
      await this.fetch.post(
        `/api/v1/storage/${this.bucketName}/copy`,
        {
          from_path: fromPath,
          to_path: toPath,
        },
      );

      return {
        data: { path: toPath },
        error: null
      };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Share a file with another user (RLS)
   * @param path - The file path
   * @param options - Share options (userId and permission)
   */
  async share(
    path: string,
    options: ShareFileOptions,
  ): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.post(
        `/api/v1/storage/${this.bucketName}/${path}/share`,
        {
          user_id: options.userId,
          permission: options.permission,
        },
      );

      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Revoke file access from a user (RLS)
   * @param path - The file path
   * @param userId - The user ID to revoke access from
   */
  async revokeShare(
    path: string,
    userId: string,
  ): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.delete(
        `/api/v1/storage/${this.bucketName}/${path}/share/${userId}`,
      );

      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * List users a file is shared with (RLS)
   * @param path - The file path
   */
  async listShares(
    path: string,
  ): Promise<{ data: FileShare[] | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<{ shares: FileShare[] }>(
        `/api/v1/storage/${this.bucketName}/${path}/shares`,
      );

      return { data: data.shares || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }
}

export class FluxbaseStorage {
  private fetch: FluxbaseFetch;

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch;
  }

  /**
   * Get a reference to a storage bucket
   * @param bucketName - The name of the bucket
   */
  from(bucketName: string): StorageBucket {
    return new StorageBucket(this.fetch, bucketName);
  }

  /**
   * List all buckets
   */
  async listBuckets(): Promise<{
    data: Array<{ name: string; created_at: string }> | null;
    error: Error | null;
  }> {
    try {
      const data = await this.fetch.get<{
        buckets: Array<{ name: string; created_at: string }>;
      }>("/api/v1/storage/buckets");

      return { data: data.buckets || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Create a new bucket
   * @param bucketName - The name of the bucket to create
   */
  async createBucket(
    bucketName: string,
  ): Promise<{ data: { name: string } | null; error: Error | null }> {
    try {
      await this.fetch.post(`/api/v1/storage/buckets/${bucketName}`);
      return { data: { name: bucketName }, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Delete a bucket
   * @param bucketName - The name of the bucket to delete
   */
  async deleteBucket(
    bucketName: string,
  ): Promise<{ data: { message: string } | null; error: Error | null }> {
    try {
      await this.fetch.delete(`/api/v1/storage/buckets/${bucketName}`);
      return { data: { message: 'Successfully deleted' }, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Empty a bucket (delete all files)
   * @param bucketName - The name of the bucket to empty
   */
  async emptyBucket(
    bucketName: string,
  ): Promise<{ data: { message: string } | null; error: Error | null }> {
    try {
      // List all files and delete them
      const bucket = this.from(bucketName);
      const { data: objects, error: listError } = await bucket.list();

      if (listError) {
        return { data: null, error: listError };
      }

      if (objects && objects.length > 0) {
        const paths = objects.map((obj) => obj.name);
        const { error: removeError } = await bucket.remove(paths);

        if (removeError) {
          return { data: null, error: removeError };
        }
      }

      return { data: { message: 'Successfully emptied' }, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Update bucket settings (RLS - requires admin or service key)
   * @param bucketName - The name of the bucket
   * @param settings - Bucket settings to update
   */
  async updateBucketSettings(
    bucketName: string,
    settings: BucketSettings,
  ): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.put(`/api/v1/storage/buckets/${bucketName}`, settings);
      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get bucket details
   * @param bucketName - The name of the bucket
   */
  async getBucket(
    bucketName: string,
  ): Promise<{ data: Bucket | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<Bucket>(
        `/api/v1/storage/buckets/${bucketName}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }
}
