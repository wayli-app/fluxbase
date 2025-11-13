/**
 * Storage client for file operations
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  StorageObject,
  UploadOptions,
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
  ): Promise<{ data: StorageObject | null; error: Error | null }> {
    try {
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

      const data = await this.fetch.request<StorageObject>(
        `/api/v1/storage/${this.bucketName}/${path}`,
        {
          method: "POST",
          body: formData,
          headers: {}, // Let browser set Content-Type for FormData
        },
      );

      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
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
   * @param options - List options (prefix, limit, offset)
   */
  async list(
    options?: ListOptions,
  ): Promise<{ data: StorageObject[] | null; error: Error | null }> {
    try {
      const params = new URLSearchParams();

      if (options?.prefix) {
        params.set("prefix", options.prefix);
      }

      if (options?.limit) {
        params.set("limit", String(options.limit));
      }

      if (options?.offset) {
        params.set("offset", String(options.offset));
      }

      const queryString = params.toString();
      const path = `/api/v1/storage/${this.bucketName}${queryString ? `?${queryString}` : ""}`;

      const data = await this.fetch.get<{ files: StorageObject[] }>(path);

      return { data: data.files || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Remove files from the bucket
   * @param paths - Array of file paths to remove
   */
  async remove(paths: string[]): Promise<{ data: null; error: Error | null }> {
    try {
      // Delete files one by one (could be optimized with batch endpoint)
      for (const path of paths) {
        await this.fetch.delete(`/api/v1/storage/${this.bucketName}/${path}`);
      }

      return { data: null, error: null };
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
  ): Promise<{ data: StorageObject | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<StorageObject>(
        `/api/v1/storage/${this.bucketName}/move`,
        {
          from_path: fromPath,
          to_path: toPath,
        },
      );

      return { data, error: null };
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
  ): Promise<{ data: StorageObject | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<StorageObject>(
        `/api/v1/storage/${this.bucketName}/copy`,
        {
          from_path: fromPath,
          to_path: toPath,
        },
      );

      return { data, error: null };
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
  ): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.post(`/api/v1/storage/buckets/${bucketName}`);
      return { data: null, error: null };
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
  ): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.delete(`/api/v1/storage/buckets/${bucketName}`);
      return { data: null, error: null };
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
  ): Promise<{ data: null; error: Error | null }> {
    try {
      // List all files and delete them
      const bucket = this.from(bucketName);
      const { data: objects, error: listError } = await bucket.list();

      if (listError) {
        return { data: null, error: listError };
      }

      if (objects && objects.length > 0) {
        const paths = objects.map((obj) => obj.key);
        const { error: removeError } = await bucket.remove(paths);

        if (removeError) {
          return { data: null, error: removeError };
        }
      }

      return { data: null, error: null };
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
