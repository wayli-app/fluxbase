/**
 * Storage client for file operations
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  FileObject,
  UploadOptions,
  UploadProgress,
  StreamUploadOptions,
  ListOptions,
  SignedUrlOptions,
  DownloadOptions,
  StreamDownloadData,
  ResumableDownloadOptions,
  ResumableDownloadData,
  ResumableUploadOptions,
  ChunkedUploadSession,
  ShareFileOptions,
  FileShare,
  BucketSettings,
  Bucket,
  TransformOptions,
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
   * @param file - The file to upload (File, Blob, ArrayBuffer, or ArrayBufferView like Uint8Array)
   * @param options - Upload options
   */
  async upload(
    path: string,
    file: File | Blob | ArrayBuffer | ArrayBufferView,
    options?: UploadOptions,
  ): Promise<{ data: { id: string; path: string; fullPath: string } | null; error: Error | null }> {
    try {
      // Prepare FormData (common to both code paths)
      const formData = new FormData();

      // Convert to Blob if ArrayBuffer or ArrayBufferView (Uint8Array, etc.)
      let blob: Blob | File;
      if (file instanceof ArrayBuffer) {
        blob = new Blob([file], { type: options?.contentType });
      } else if (ArrayBuffer.isView(file)) {
        // Cast needed because TypeScript's ArrayBufferView includes SharedArrayBuffer views
        blob = new Blob([file as BlobPart], { type: options?.contentType });
      } else {
        blob = file;
      }

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
   * Upload a file using streaming for reduced memory usage.
   * This method bypasses FormData buffering and streams data directly to the server.
   * Ideal for large files where memory efficiency is important.
   *
   * @param path - The path/key for the file
   * @param stream - ReadableStream of the file data
   * @param size - The size of the file in bytes (required for Content-Length header)
   * @param options - Upload options
   *
   * @example
   * ```typescript
   * // Upload from a File's stream
   * const file = new File([...], 'large-video.mp4');
   * const { data, error } = await storage
   *   .from('videos')
   *   .uploadStream('video.mp4', file.stream(), file.size, {
   *     contentType: 'video/mp4',
   *   });
   *
   * // Upload from a fetch response stream
   * const response = await fetch('https://example.com/data.zip');
   * const size = parseInt(response.headers.get('content-length') || '0');
   * const { data, error } = await storage
   *   .from('files')
   *   .uploadStream('data.zip', response.body!, size, {
   *     contentType: 'application/zip',
   *   });
   * ```
   */
  async uploadStream(
    path: string,
    stream: ReadableStream<Uint8Array>,
    size: number,
    options?: StreamUploadOptions,
  ): Promise<{ data: { id: string; path: string; fullPath: string } | null; error: Error | null }> {
    try {
      if (size <= 0) {
        return { data: null, error: new Error('size must be a positive number') };
      }

      // Build headers for streaming upload
      const headers: Record<string, string> = {
        ...this.fetch["defaultHeaders"],
        'Content-Length': String(size),
      };

      // Set content type
      if (options?.contentType) {
        headers['X-Storage-Content-Type'] = options.contentType;
      }

      // Set cache control
      if (options?.cacheControl) {
        headers['X-Storage-Cache-Control'] = options.cacheControl;
      }

      // Set metadata as JSON
      if (options?.metadata && Object.keys(options.metadata).length > 0) {
        headers['X-Storage-Metadata'] = JSON.stringify(options.metadata);
      }

      // Create a stream that tracks progress if callback provided
      let bodyStream: ReadableStream<Uint8Array> = stream;

      if (options?.onUploadProgress) {
        let uploadedBytes = 0;
        const progressCallback = options.onUploadProgress;
        const totalSize = size;

        const transformStream = new TransformStream<Uint8Array, Uint8Array>({
          transform(chunk, controller) {
            uploadedBytes += chunk.byteLength;
            const percentage = Math.round((uploadedBytes / totalSize) * 100);
            progressCallback({
              loaded: uploadedBytes,
              total: totalSize,
              percentage,
            });
            controller.enqueue(chunk);
          },
        });

        bodyStream = stream.pipeThrough(transformStream);
      }

      // Use fetch with streaming body
      // Note: duplex: 'half' is required for streaming request bodies
      const response = await fetch(
        `${this.fetch["baseUrl"]}/api/v1/storage/${this.bucketName}/stream/${path}`,
        {
          method: 'POST',
          headers,
          body: bodyStream,
          signal: options?.signal,
          // @ts-expect-error - duplex is not yet in TypeScript's RequestInit type
          duplex: 'half',
        },
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: response.statusText }));
        throw new Error(errorData.error || `Upload failed: ${response.statusText}`);
      }

      const result = await response.json();

      return {
        data: {
          id: result.key || path,
          path: path,
          fullPath: `${this.bucketName}/${path}`,
        },
        error: null,
      };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Upload a large file using streaming for reduced memory usage.
   * This is a convenience method that converts a File or Blob to a stream.
   *
   * @param path - The path/key for the file
   * @param file - The File or Blob to upload
   * @param options - Upload options
   *
   * @example
   * ```typescript
   * const file = new File([...], 'large-video.mp4');
   * const { data, error } = await storage
   *   .from('videos')
   *   .uploadLargeFile('video.mp4', file, {
   *     contentType: 'video/mp4',
   *     onUploadProgress: (p) => console.log(`${p.percentage}% complete`),
   *   });
   * ```
   */
  async uploadLargeFile(
    path: string,
    file: File | Blob,
    options?: StreamUploadOptions,
  ): Promise<{ data: { id: string; path: string; fullPath: string } | null; error: Error | null }> {
    // Use file's type if contentType not specified
    const opts: StreamUploadOptions = {
      ...options,
      contentType: options?.contentType || file.type || 'application/octet-stream',
    };

    return this.uploadStream(path, file.stream(), file.size, opts);
  }

  /**
   * Download a file from the bucket
   * @param path - The path/key of the file
   *
   * @example
   * ```typescript
   * // Default: returns Blob
   * const { data: blob } = await storage.from('bucket').download('file.pdf');
   *
   * // Streaming: returns { stream, size } for progress tracking
   * const { data } = await storage.from('bucket').download('large.json', { stream: true });
   * console.log(`File size: ${data.size} bytes`);
   * // Process data.stream...
   * ```
   */
  async download(
    path: string,
  ): Promise<{ data: Blob | null; error: Error | null }>;
  async download(
    path: string,
    options: { stream: true; timeout?: number; signal?: AbortSignal },
  ): Promise<{ data: StreamDownloadData | null; error: Error | null }>;
  async download(
    path: string,
    options: { stream?: false; timeout?: number; signal?: AbortSignal },
  ): Promise<{ data: Blob | null; error: Error | null }>;
  async download(
    path: string,
    options?: DownloadOptions,
  ): Promise<{
    data: Blob | StreamDownloadData | null;
    error: Error | null;
  }> {
    try {
      const controller = new AbortController();
      let timeoutId: ReturnType<typeof setTimeout> | undefined;

      // Forward external signal to our controller
      if (options?.signal) {
        if (options.signal.aborted) {
          return { data: null, error: new Error("Download aborted") };
        }
        options.signal.addEventListener("abort", () => controller.abort(), {
          once: true,
        });
      }

      // For streaming: no timeout by default (large files need time)
      // For non-streaming: 30s default
      const timeout = options?.timeout ?? (options?.stream ? 0 : 30000);

      if (timeout > 0) {
        timeoutId = setTimeout(() => controller.abort(), timeout);
      }

      try {
        const response = await fetch(
          `${this.fetch["baseUrl"]}/api/v1/storage/${this.bucketName}/${path}`,
          {
            headers: this.fetch["defaultHeaders"],
            signal: controller.signal,
          },
        );

        if (timeoutId) clearTimeout(timeoutId);

        if (!response.ok) {
          throw new Error(`Failed to download file: ${response.statusText}`);
        }

        // Return stream with size if requested
        if (options?.stream) {
          if (!response.body) {
            throw new Error("Response body is not available for streaming");
          }
          // Extract file size from Content-Length header
          const contentLength = response.headers.get("content-length");
          const size = contentLength ? parseInt(contentLength, 10) : null;
          return {
            data: { stream: response.body, size },
            error: null,
          };
        }

        // Default: return Blob
        const blob = await response.blob();
        return { data: blob, error: null };
      } catch (err) {
        if (timeoutId) clearTimeout(timeoutId);

        if (err instanceof Error && err.name === "AbortError") {
          // Check if it was user abort or timeout
          if (options?.signal?.aborted) {
            return { data: null, error: new Error("Download aborted") };
          }
          return { data: null, error: new Error("Download timeout") };
        }
        throw err;
      }
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Download a file with resumable chunked downloads for large files.
   * Returns a ReadableStream that abstracts the chunking internally.
   *
   * Features:
   * - Downloads file in chunks using HTTP Range headers
   * - Automatically retries failed chunks with exponential backoff
   * - Reports progress via callback
   * - Falls back to regular streaming if Range not supported
   *
   * @param path - The file path within the bucket
   * @param options - Download options including chunk size, retries, and progress callback
   * @returns A ReadableStream and file size (consumer doesn't need to know about chunking)
   *
   * @example
   * ```typescript
   * const { data, error } = await storage.from('bucket').downloadResumable('large.json', {
   *   chunkSize: 5 * 1024 * 1024, // 5MB chunks
   *   maxRetries: 3,
   *   onProgress: (progress) => console.log(`${progress.percentage}% complete`)
   * });
   * if (data) {
   *   console.log(`File size: ${data.size} bytes`);
   *   // Process data.stream...
   * }
   * ```
   */
  async downloadResumable(
    path: string,
    options?: ResumableDownloadOptions,
  ): Promise<{ data: ResumableDownloadData | null; error: Error | null }> {
    try {
      const chunkSize = options?.chunkSize ?? 5 * 1024 * 1024; // 5MB default
      const maxRetries = options?.maxRetries ?? 3;
      const retryDelayMs = options?.retryDelayMs ?? 1000;
      const chunkTimeout = options?.chunkTimeout ?? 30000;

      const url = `${this.fetch["baseUrl"]}/api/v1/storage/${this.bucketName}/${path}`;
      const headers = this.fetch["defaultHeaders"];

      // Check if already aborted
      if (options?.signal?.aborted) {
        return { data: null, error: new Error("Download aborted") };
      }

      // Get file info with HEAD request to determine size and Range support
      const headResponse = await fetch(url, {
        method: "HEAD",
        headers,
        signal: options?.signal,
      });

      if (!headResponse.ok) {
        throw new Error(`Failed to get file info: ${headResponse.statusText}`);
      }

      const contentLength = headResponse.headers.get("content-length");
      const acceptRanges = headResponse.headers.get("accept-ranges");
      const totalSize = contentLength ? parseInt(contentLength, 10) : null;

      // If server doesn't support Range requests, fall back to regular streaming
      if (acceptRanges !== "bytes") {
        const { data, error } = await this.download(path, {
          stream: true,
          timeout: 0,
          signal: options?.signal,
        });
        if (error) return { data: null, error };
        return {
          data: data as ResumableDownloadData,
          error: null,
        };
      }

      // Create a ReadableStream that fetches chunks internally
      let downloadedBytes = 0;
      let currentChunk = 0;
      const totalChunks = totalSize ? Math.ceil(totalSize / chunkSize) : null;
      let lastProgressTime = Date.now();
      let lastProgressBytes = 0;

      const stream = new ReadableStream<Uint8Array>({
        async pull(controller) {
          // Check if aborted
          if (options?.signal?.aborted) {
            controller.error(new Error("Download aborted"));
            return;
          }

          // Check if we've downloaded everything
          if (totalSize !== null && downloadedBytes >= totalSize) {
            controller.close();
            return;
          }

          const rangeStart = downloadedBytes;
          const rangeEnd =
            totalSize !== null
              ? Math.min(downloadedBytes + chunkSize - 1, totalSize - 1)
              : downloadedBytes + chunkSize - 1;

          let retryCount = 0;
          let chunk: Uint8Array | null = null;

          while (retryCount <= maxRetries && chunk === null) {
            try {
              // Check abort before each attempt
              if (options?.signal?.aborted) {
                controller.error(new Error("Download aborted"));
                return;
              }

              const chunkController = new AbortController();
              const timeoutId = setTimeout(
                () => chunkController.abort(),
                chunkTimeout,
              );

              // Forward external signal to chunk controller
              if (options?.signal) {
                options.signal.addEventListener(
                  "abort",
                  () => chunkController.abort(),
                  { once: true },
                );
              }

              const chunkResponse = await fetch(url, {
                headers: {
                  ...headers,
                  Range: `bytes=${rangeStart}-${rangeEnd}`,
                },
                signal: chunkController.signal,
              });

              clearTimeout(timeoutId);

              if (!chunkResponse.ok && chunkResponse.status !== 206) {
                throw new Error(
                  `Chunk download failed: ${chunkResponse.statusText}`,
                );
              }

              const arrayBuffer = await chunkResponse.arrayBuffer();
              chunk = new Uint8Array(arrayBuffer);

              // Check if we got less data than expected (end of file)
              if (totalSize === null && chunk.byteLength < chunkSize) {
                downloadedBytes += chunk.byteLength;
                currentChunk++;
                controller.enqueue(chunk);
                controller.close();
                return;
              }
            } catch (err) {
              // Check if it was user abort
              if (options?.signal?.aborted) {
                controller.error(new Error("Download aborted"));
                return;
              }

              retryCount++;
              if (retryCount > maxRetries) {
                controller.error(
                  new Error(
                    `Failed to download chunk after ${maxRetries} retries`,
                  ),
                );
                return;
              }

              // Exponential backoff: 1s, 2s, 4s...
              const delay = retryDelayMs * Math.pow(2, retryCount - 1);
              await new Promise((resolve) => setTimeout(resolve, delay));
            }
          }

          if (chunk) {
            downloadedBytes += chunk.byteLength;
            currentChunk++;

            // Report progress
            if (options?.onProgress) {
              const now = Date.now();
              const elapsed = (now - lastProgressTime) / 1000;
              const bytesPerSecond =
                elapsed > 0
                  ? (downloadedBytes - lastProgressBytes) / elapsed
                  : 0;

              lastProgressTime = now;
              lastProgressBytes = downloadedBytes;

              options.onProgress({
                loaded: downloadedBytes,
                total: totalSize,
                percentage: totalSize
                  ? Math.round((downloadedBytes / totalSize) * 100)
                  : null,
                currentChunk,
                totalChunks,
                bytesPerSecond,
              });
            }

            controller.enqueue(chunk);

            // Check if we're done
            if (totalSize !== null && downloadedBytes >= totalSize) {
              controller.close();
            }
          }
        },
      });

      return {
        data: { stream, size: totalSize },
        error: null,
      };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Upload a large file with resumable chunked uploads.
   *
   * Features:
   * - Uploads file in chunks for reliability
   * - Automatically retries failed chunks with exponential backoff
   * - Reports progress via callback with chunk-level granularity
   * - Can resume interrupted uploads using session ID
   *
   * @param path - The file path within the bucket
   * @param file - The File or Blob to upload
   * @param options - Upload options including chunk size, retries, and progress callback
   * @returns Upload result with file info
   *
   * @example
   * const { data, error } = await storage.from('uploads').uploadResumable('large.zip', file, {
   *   chunkSize: 5 * 1024 * 1024, // 5MB chunks
   *   maxRetries: 3,
   *   onProgress: (p) => {
   *     console.log(`${p.percentage}% (chunk ${p.currentChunk}/${p.totalChunks})`);
   *     console.log(`Speed: ${(p.bytesPerSecond / 1024 / 1024).toFixed(2)} MB/s`);
   *     console.log(`Session ID (for resume): ${p.sessionId}`);
   *   }
   * });
   *
   * // To resume an interrupted upload:
   * const { data, error } = await storage.from('uploads').uploadResumable('large.zip', file, {
   *   resumeSessionId: 'previous-session-id',
   * });
   */
  async uploadResumable(
    path: string,
    file: File | Blob,
    options?: ResumableUploadOptions,
  ): Promise<{
    data: { id: string; path: string; fullPath: string } | null;
    error: Error | null;
  }> {
    try {
      const chunkSize = options?.chunkSize ?? 5 * 1024 * 1024; // 5MB default
      const maxRetries = options?.maxRetries ?? 3;
      const retryDelayMs = options?.retryDelayMs ?? 1000;
      const chunkTimeout = options?.chunkTimeout ?? 60000; // 1 minute per chunk

      const totalSize = file.size;
      const totalChunks = Math.ceil(totalSize / chunkSize);

      // Check if already aborted
      if (options?.signal?.aborted) {
        return { data: null, error: new Error("Upload aborted") };
      }

      const baseUrl = this.fetch["baseUrl"];
      const headers = this.fetch["defaultHeaders"];

      // 1. Initialize or resume session
      let sessionId = options?.resumeSessionId;
      let session: ChunkedUploadSession;
      let completedChunks: number[] = [];

      if (!sessionId) {
        // Initialize new session
        const initResponse = await fetch(
          `${baseUrl}/api/v1/storage/${this.bucketName}/chunked/init`,
          {
            method: "POST",
            headers: {
              ...headers,
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              path,
              total_size: totalSize,
              chunk_size: chunkSize,
              content_type: options?.contentType || file.type || "application/octet-stream",
              metadata: options?.metadata,
              cache_control: options?.cacheControl,
            }),
            signal: options?.signal,
          },
        );

        if (!initResponse.ok) {
          const errorData = await initResponse.json().catch(() => ({}));
          throw new Error(
            errorData.error || `Failed to initialize upload: ${initResponse.statusText}`,
          );
        }

        const initData = await initResponse.json();
        session = {
          sessionId: initData.session_id,
          bucket: initData.bucket,
          path: initData.path,
          totalSize: initData.total_size,
          chunkSize: initData.chunk_size,
          totalChunks: initData.total_chunks,
          completedChunks: initData.completed_chunks || [],
          status: initData.status,
          expiresAt: initData.expires_at,
          createdAt: initData.created_at,
        };
        sessionId = session.sessionId;
      } else {
        // Resume existing session - get status
        const statusResponse = await fetch(
          `${baseUrl}/api/v1/storage/${this.bucketName}/chunked/${sessionId}/status`,
          {
            method: "GET",
            headers,
            signal: options?.signal,
          },
        );

        if (!statusResponse.ok) {
          const errorData = await statusResponse.json().catch(() => ({}));
          throw new Error(
            errorData.error || `Failed to get session status: ${statusResponse.statusText}`,
          );
        }

        const statusData = await statusResponse.json();
        session = statusData.session;
        completedChunks = session.completedChunks || [];
      }

      // Calculate already uploaded bytes for progress
      let uploadedBytes = 0;
      for (const chunkIdx of completedChunks) {
        const chunkStart = chunkIdx * chunkSize;
        const chunkEnd = Math.min(chunkStart + chunkSize, totalSize);
        uploadedBytes += chunkEnd - chunkStart;
      }

      let lastProgressTime = Date.now();
      let lastProgressBytes = uploadedBytes;

      // 2. Upload each chunk with retry logic
      for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
        // Check if aborted
        if (options?.signal?.aborted) {
          return { data: null, error: new Error("Upload aborted") };
        }

        // Skip already completed chunks
        if (completedChunks.includes(chunkIndex)) {
          continue;
        }

        const start = chunkIndex * chunkSize;
        const end = Math.min(start + chunkSize, totalSize);
        const chunk = file.slice(start, end);
        const chunkArrayBuffer = await chunk.arrayBuffer();

        let retryCount = 0;
        let chunkUploaded = false;

        while (retryCount <= maxRetries && !chunkUploaded) {
          try {
            // Check abort before each attempt
            if (options?.signal?.aborted) {
              return { data: null, error: new Error("Upload aborted") };
            }

            // Create per-chunk timeout controller
            const chunkController = new AbortController();
            const timeoutId = setTimeout(() => chunkController.abort(), chunkTimeout);

            // Forward external signal to chunk controller
            if (options?.signal) {
              options.signal.addEventListener(
                "abort",
                () => chunkController.abort(),
                { once: true },
              );
            }

            const chunkResponse = await fetch(
              `${baseUrl}/api/v1/storage/${this.bucketName}/chunked/${sessionId}/${chunkIndex}`,
              {
                method: "PUT",
                headers: {
                  ...headers,
                  "Content-Type": "application/octet-stream",
                  "Content-Length": String(chunkArrayBuffer.byteLength),
                },
                body: chunkArrayBuffer,
                signal: chunkController.signal,
              },
            );

            clearTimeout(timeoutId);

            if (!chunkResponse.ok) {
              const errorData = await chunkResponse.json().catch(() => ({}));
              throw new Error(
                errorData.error || `Chunk upload failed: ${chunkResponse.statusText}`,
              );
            }

            chunkUploaded = true;
          } catch (err) {
            // Check if it was user abort
            if (options?.signal?.aborted) {
              return { data: null, error: new Error("Upload aborted") };
            }

            retryCount++;
            if (retryCount > maxRetries) {
              throw new Error(
                `Failed to upload chunk ${chunkIndex} after ${maxRetries} retries: ${(err as Error).message}`,
              );
            }

            // Exponential backoff: 1s, 2s, 4s...
            const delay = retryDelayMs * Math.pow(2, retryCount - 1);
            await new Promise((resolve) => setTimeout(resolve, delay));
          }
        }

        // Update progress
        uploadedBytes += end - start;

        if (options?.onProgress) {
          const now = Date.now();
          const elapsed = (now - lastProgressTime) / 1000;
          const bytesPerSecond =
            elapsed > 0 ? (uploadedBytes - lastProgressBytes) / elapsed : 0;

          lastProgressTime = now;
          lastProgressBytes = uploadedBytes;

          options.onProgress({
            loaded: uploadedBytes,
            total: totalSize,
            percentage: Math.round((uploadedBytes / totalSize) * 100),
            currentChunk: chunkIndex + 1,
            totalChunks,
            bytesPerSecond,
            sessionId: sessionId!,
          });
        }
      }

      // 3. Complete the upload
      const completeResponse = await fetch(
        `${baseUrl}/api/v1/storage/${this.bucketName}/chunked/${sessionId}/complete`,
        {
          method: "POST",
          headers,
          signal: options?.signal,
        },
      );

      if (!completeResponse.ok) {
        const errorData = await completeResponse.json().catch(() => ({}));
        throw new Error(
          errorData.error || `Failed to complete upload: ${completeResponse.statusText}`,
        );
      }

      const result = await completeResponse.json();

      return {
        data: {
          id: result.id,
          path: result.path,
          fullPath: result.full_path,
        },
        error: null,
      };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Abort an in-progress resumable upload
   * @param sessionId - The upload session ID to abort
   */
  async abortResumableUpload(
    sessionId: string,
  ): Promise<{ error: Error | null }> {
    try {
      const baseUrl = this.fetch["baseUrl"];
      const headers = this.fetch["defaultHeaders"];

      const response = await fetch(
        `${baseUrl}/api/v1/storage/${this.bucketName}/chunked/${sessionId}`,
        {
          method: "DELETE",
          headers,
        },
      );

      if (!response.ok && response.status !== 204) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(
          errorData.error || `Failed to abort upload: ${response.statusText}`,
        );
      }

      return { error: null };
    } catch (error) {
      return { error: error as Error };
    }
  }

  /**
   * Get the status of a resumable upload session
   * @param sessionId - The upload session ID to check
   */
  async getResumableUploadStatus(
    sessionId: string,
  ): Promise<{ data: ChunkedUploadSession | null; error: Error | null }> {
    try {
      const baseUrl = this.fetch["baseUrl"];
      const headers = this.fetch["defaultHeaders"];

      const response = await fetch(
        `${baseUrl}/api/v1/storage/${this.bucketName}/chunked/${sessionId}/status`,
        {
          method: "GET",
          headers,
        },
      );

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(
          errorData.error || `Failed to get upload status: ${response.statusText}`,
        );
      }

      const data = await response.json();
      return {
        data: data.session,
        error: null,
      };
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
   * Build query string from transform options
   * @private
   */
  private buildTransformQuery(transform: TransformOptions): string {
    const params = new URLSearchParams();

    if (transform.width !== undefined && transform.width > 0) {
      params.set("w", String(transform.width));
    }
    if (transform.height !== undefined && transform.height > 0) {
      params.set("h", String(transform.height));
    }
    if (transform.format) {
      params.set("fmt", transform.format);
    }
    if (transform.quality !== undefined && transform.quality > 0) {
      params.set("q", String(transform.quality));
    }
    if (transform.fit) {
      params.set("fit", transform.fit);
    }

    return params.toString();
  }

  /**
   * Get a public URL for a file with image transformations applied
   * Only works for image files (JPEG, PNG, WebP, GIF, AVIF, etc.)
   *
   * @param path - The file path
   * @param transform - Transformation options (width, height, format, quality, fit)
   *
   * @example
   * ```typescript
   * // Get a 300x200 WebP thumbnail
   * const url = storage.from('images').getTransformUrl('photo.jpg', {
   *   width: 300,
   *   height: 200,
   *   format: 'webp',
   *   quality: 85,
   *   fit: 'cover'
   * });
   *
   * // Get a resized image maintaining aspect ratio
   * const url = storage.from('images').getTransformUrl('photo.jpg', {
   *   width: 800,
   *   format: 'webp'
   * });
   * ```
   */
  getTransformUrl(path: string, transform: TransformOptions): string {
    const baseUrl = `${this.fetch["baseUrl"]}/api/v1/storage/${this.bucketName}/${path}`;
    const queryString = this.buildTransformQuery(transform);
    return queryString ? `${baseUrl}?${queryString}` : baseUrl;
  }

  /**
   * Create a signed URL for temporary access to a file
   * Optionally include image transformation parameters
   *
   * @param path - The file path
   * @param options - Signed URL options including expiration and transforms
   *
   * @example
   * ```typescript
   * // Simple signed URL (1 hour expiry)
   * const { data, error } = await storage.from('images').createSignedUrl('photo.jpg');
   *
   * // Signed URL with custom expiry
   * const { data, error } = await storage.from('images').createSignedUrl('photo.jpg', {
   *   expiresIn: 7200 // 2 hours
   * });
   *
   * // Signed URL with image transformation
   * const { data, error } = await storage.from('images').createSignedUrl('photo.jpg', {
   *   expiresIn: 3600,
   *   transform: {
   *     width: 400,
   *     height: 300,
   *     format: 'webp',
   *     quality: 85,
   *     fit: 'cover'
   *   }
   * });
   * ```
   */
  async createSignedUrl(
    path: string,
    options?: SignedUrlOptions,
  ): Promise<{ data: { signedUrl: string } | null; error: Error | null }> {
    try {
      const expiresIn = options?.expiresIn || 3600; // Default 1 hour

      // Build request body with transform options if provided
      const requestBody: Record<string, unknown> = { expires_in: expiresIn };

      if (options?.transform) {
        const transform = options.transform;
        if (transform.width !== undefined && transform.width > 0) {
          requestBody.width = transform.width;
        }
        if (transform.height !== undefined && transform.height > 0) {
          requestBody.height = transform.height;
        }
        if (transform.format) {
          requestBody.format = transform.format;
        }
        if (transform.quality !== undefined && transform.quality > 0) {
          requestBody.quality = transform.quality;
        }
        if (transform.fit) {
          requestBody.fit = transform.fit;
        }
      }

      const data = await this.fetch.post<{ signed_url: string }>(
        `/api/v1/storage/${this.bucketName}/sign/${path}`,
        requestBody,
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
