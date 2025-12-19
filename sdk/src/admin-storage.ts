import type { FluxbaseFetch } from "./fetch";
import type {
  AdminListBucketsResponse,
  AdminListObjectsResponse,
  AdminStorageObject,
  SignedUrlResponse,
  DataResponse,
  VoidResponse,
} from "./types";
import { wrapAsync, wrapAsyncVoid } from "./utils/error-handling";

/**
 * Admin storage manager for bucket and object management
 */
export class FluxbaseAdminStorage {
  private fetch: FluxbaseFetch;

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch;
  }

  // ============================================================================
  // Bucket Operations
  // ============================================================================

  /**
   * List all storage buckets
   *
   * @returns List of buckets
   *
   * @example
   * ```typescript
   * const { data, error } = await admin.storage.listBuckets();
   * if (data) {
   *   console.log(`Found ${data.buckets.length} buckets`);
   * }
   * ```
   */
  async listBuckets(): Promise<DataResponse<AdminListBucketsResponse>> {
    return wrapAsync(async () => {
      return await this.fetch.get<AdminListBucketsResponse>(
        "/api/v1/storage/buckets"
      );
    });
  }

  /**
   * Create a new storage bucket
   *
   * @param name - Bucket name
   * @returns Success message
   *
   * @example
   * ```typescript
   * const { error } = await admin.storage.createBucket('my-bucket');
   * if (!error) {
   *   console.log('Bucket created');
   * }
   * ```
   */
  async createBucket(name: string): Promise<DataResponse<{ message: string }>> {
    return wrapAsync(async () => {
      return await this.fetch.post<{ message: string }>(
        `/api/v1/storage/buckets/${encodeURIComponent(name)}`
      );
    });
  }

  /**
   * Delete a storage bucket
   *
   * @param name - Bucket name
   * @returns Success message
   *
   * @example
   * ```typescript
   * const { error } = await admin.storage.deleteBucket('my-bucket');
   * if (!error) {
   *   console.log('Bucket deleted');
   * }
   * ```
   */
  async deleteBucket(name: string): Promise<DataResponse<{ message: string }>> {
    return wrapAsync(async () => {
      return await this.fetch.delete<{ message: string }>(
        `/api/v1/storage/buckets/${encodeURIComponent(name)}`
      );
    });
  }

  // ============================================================================
  // Object Operations
  // ============================================================================

  /**
   * List objects in a bucket
   *
   * @param bucket - Bucket name
   * @param prefix - Optional path prefix to filter results
   * @param delimiter - Optional delimiter for hierarchical listing (usually '/')
   * @returns List of objects and prefixes (folders)
   *
   * @example
   * ```typescript
   * // List all objects in bucket
   * const { data } = await admin.storage.listObjects('my-bucket');
   *
   * // List objects in a folder
   * const { data } = await admin.storage.listObjects('my-bucket', 'folder/', '/');
   * ```
   */
  async listObjects(
    bucket: string,
    prefix?: string,
    delimiter?: string
  ): Promise<DataResponse<AdminListObjectsResponse>> {
    return wrapAsync(async () => {
      const params = new URLSearchParams();
      if (prefix) params.append("prefix", prefix);
      if (delimiter) params.append("delimiter", delimiter);

      const queryString = params.toString();
      const url = `/api/v1/storage/${encodeURIComponent(bucket)}${queryString ? `?${queryString}` : ""}`;

      return await this.fetch.get<AdminListObjectsResponse>(url);
    });
  }

  /**
   * Get object metadata
   *
   * @param bucket - Bucket name
   * @param key - Object key (path)
   * @returns Object metadata
   *
   * @example
   * ```typescript
   * const { data } = await admin.storage.getObjectMetadata('my-bucket', 'path/to/file.txt');
   * if (data) {
   *   console.log(`File size: ${data.size} bytes`);
   * }
   * ```
   */
  async getObjectMetadata(
    bucket: string,
    key: string
  ): Promise<DataResponse<AdminStorageObject>> {
    return wrapAsync(async () => {
      const encodedKey = key
        .split("/")
        .map((s) => encodeURIComponent(s))
        .join("/");

      return await this.fetch.get<AdminStorageObject>(
        `/api/v1/storage/${encodeURIComponent(bucket)}/${encodedKey}`,
        {
          headers: { "X-Metadata-Only": "true" },
        }
      );
    });
  }

  /**
   * Download an object as a Blob
   *
   * @param bucket - Bucket name
   * @param key - Object key (path)
   * @returns Object data as Blob
   *
   * @example
   * ```typescript
   * const { data: blob } = await admin.storage.downloadObject('my-bucket', 'file.pdf');
   * if (blob) {
   *   // Use the blob
   *   const url = URL.createObjectURL(blob);
   * }
   * ```
   */
  async downloadObject(
    bucket: string,
    key: string
  ): Promise<DataResponse<Blob>> {
    return wrapAsync(async () => {
      const encodedKey = key
        .split("/")
        .map((s) => encodeURIComponent(s))
        .join("/");

      const response = await this.fetch.getBlob(
        `/api/v1/storage/${encodeURIComponent(bucket)}/${encodedKey}`
      );
      return response;
    });
  }

  /**
   * Delete an object
   *
   * @param bucket - Bucket name
   * @param key - Object key (path)
   *
   * @example
   * ```typescript
   * const { error } = await admin.storage.deleteObject('my-bucket', 'path/to/file.txt');
   * if (!error) {
   *   console.log('Object deleted');
   * }
   * ```
   */
  async deleteObject(bucket: string, key: string): Promise<VoidResponse> {
    return wrapAsyncVoid(async () => {
      const encodedKey = key
        .split("/")
        .map((s) => encodeURIComponent(s))
        .join("/");

      await this.fetch.delete(
        `/api/v1/storage/${encodeURIComponent(bucket)}/${encodedKey}`
      );
    });
  }

  /**
   * Create a folder (empty object with directory content type)
   *
   * @param bucket - Bucket name
   * @param folderPath - Folder path (should end with /)
   *
   * @example
   * ```typescript
   * const { error } = await admin.storage.createFolder('my-bucket', 'new-folder/');
   * ```
   */
  async createFolder(bucket: string, folderPath: string): Promise<VoidResponse> {
    return wrapAsyncVoid(async () => {
      const encodedPath = folderPath
        .split("/")
        .map((s) => encodeURIComponent(s))
        .join("/");

      await this.fetch.post(
        `/api/v1/storage/${encodeURIComponent(bucket)}/${encodedPath}`,
        null,
        {
          headers: { "Content-Type": "application/x-directory" },
        }
      );
    });
  }

  /**
   * Generate a signed URL for temporary access
   *
   * @param bucket - Bucket name
   * @param key - Object key (path)
   * @param expiresIn - Expiration time in seconds
   * @returns Signed URL and expiration info
   *
   * @example
   * ```typescript
   * const { data } = await admin.storage.generateSignedUrl('my-bucket', 'file.pdf', 3600);
   * if (data) {
   *   console.log(`Download at: ${data.url}`);
   *   console.log(`Expires in: ${data.expires_in} seconds`);
   * }
   * ```
   */
  async generateSignedUrl(
    bucket: string,
    key: string,
    expiresIn: number
  ): Promise<DataResponse<SignedUrlResponse>> {
    return wrapAsync(async () => {
      const encodedKey = key
        .split("/")
        .map((s) => encodeURIComponent(s))
        .join("/");

      return await this.fetch.post<SignedUrlResponse>(
        `/api/v1/storage/${encodeURIComponent(bucket)}/${encodedKey}/signed-url`,
        { expires_in: expiresIn }
      );
    });
  }
}
