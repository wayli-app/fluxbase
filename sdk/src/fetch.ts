/**
 * HTTP client for making requests to the Fluxbase API
 */

import type { FluxbaseError, HttpMethod } from "./types";

export interface FetchOptions {
  method: HttpMethod;
  headers?: Record<string, string>;
  body?: unknown;
  timeout?: number;
  /** Skip automatic token refresh on 401 (used for auth endpoints) */
  skipAutoRefresh?: boolean;
}

/**
 * Response with headers included (for count queries)
 */
export interface FetchResponseWithHeaders<T> {
  data: T;
  headers: Headers;
  status: number;
}

/** Callback type for automatic token refresh on 401 errors */
export type RefreshTokenCallback = () => Promise<boolean>;

/** Callback type for modifying headers before each request */
export type BeforeRequestCallback = (headers: Record<string, string>) => void;

export class FluxbaseFetch {
  private baseUrl: string;
  private defaultHeaders: Record<string, string>;
  private timeout: number;
  private debug: boolean;
  private refreshTokenCallback: RefreshTokenCallback | null = null;
  private isRefreshing = false;
  private refreshPromise: Promise<boolean> | null = null;
  private anonKey: string | null = null;
  private beforeRequestCallback: BeforeRequestCallback | null = null;

  constructor(
    baseUrl: string,
    options: {
      headers?: Record<string, string>;
      timeout?: number;
      debug?: boolean;
    } = {},
  ) {
    this.baseUrl = baseUrl.replace(/\/$/, ""); // Remove trailing slash
    this.defaultHeaders = {
      "Content-Type": "application/json",
      ...options.headers,
    };
    this.timeout = options.timeout ?? 30000;
    this.debug = options.debug ?? false;
  }

  getBaseUrl(): string { return this.baseUrl }

  getDefaultHeaders(): Record<string, string> { return { ...this.defaultHeaders } }

  /**
   * Register a callback to refresh the token when a 401 error occurs
   * The callback should return true if refresh was successful, false otherwise
   */
  setRefreshTokenCallback(callback: RefreshTokenCallback | null) {
    this.refreshTokenCallback = callback;
  }

  /**
   * Register a callback to be called before every request.
   * The callback receives the headers object and can modify it in place.
   * This is useful for dynamically injecting headers at request time.
   */
  setBeforeRequestCallback(callback: BeforeRequestCallback | null) {
    this.beforeRequestCallback = callback;
  }

  /**
   * Set the anon key for fallback authentication
   * When setAuthToken(null) is called, the Authorization header will be
   * restored to use this anon key instead of being deleted
   */
  setAnonKey(key: string) {
    this.anonKey = key;
  }

  /**
   * Update the authorization header
   * When token is null, restores to anon key if available
   */
  setAuthToken(token: string | null) {
    if (token) {
      this.defaultHeaders["Authorization"] = `Bearer ${token}`;
    } else if (this.anonKey) {
      // Restore anon key auth instead of deleting header
      this.defaultHeaders["Authorization"] = `Bearer ${this.anonKey}`;
    } else {
      delete this.defaultHeaders["Authorization"];
    }
  }

  /**
   * Set a custom header on all requests
   */
  setHeader(name: string, value: string) {
    this.defaultHeaders[name] = value;
  }

  /**
   * Remove a custom header
   */
  removeHeader(name: string) {
    delete this.defaultHeaders[name];
  }

  /**
   * Make an HTTP request
   */
  async request<T = unknown>(path: string, options: FetchOptions): Promise<T> {
    return this.requestInternal<T>(path, options, false);
  }

  /**
   * Full request implementation returning response with headers, status, and data.
   * Used by both requestInternal and requestWithHeadersInternal.
   */
  private async requestFull<T = unknown>(
    path: string,
    options: FetchOptions,
    isRetry: boolean,
  ): Promise<FetchResponseWithHeaders<T>> {
    const url = `${this.baseUrl}${path}`;
    const headers = { ...this.defaultHeaders, ...options.headers };
    if (this.beforeRequestCallback) {
      this.beforeRequestCallback(headers);
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(
      () => controller.abort(),
      options.timeout ?? this.timeout,
    );

    if (this.debug) {
      console.log(`[Fluxbase SDK] ${options.method} ${url}`, options.body);
    }

    try {
      const isFormData =
        options.body &&
        (options.body.constructor?.name === "FormData" ||
          options.body instanceof FormData);

      const requestHeaders = isFormData
        ? Object.fromEntries(
            Object.entries(headers).filter(
              ([key]) => key.toLowerCase() !== "content-type",
            ),
          )
        : headers;

      const response = await fetch(url, {
        method: options.method,
        headers: requestHeaders,
        body: isFormData
          ? (options.body as FormData)
          : options.body
            ? JSON.stringify(options.body)
            : undefined,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      const contentType = response.headers.get("content-type");
      let data: unknown;

      if (contentType?.includes("application/json")) {
        data = await response.json();
      } else {
        data = await response.text();
      }

      if (this.debug) {
        console.log(`[Fluxbase SDK] Response:`, response.status, data);
      }

      if (
        response.status === 401 &&
        !isRetry &&
        !options.skipAutoRefresh &&
        this.refreshTokenCallback
      ) {
        const refreshSuccess = await this.handleTokenRefresh();
        if (refreshSuccess) {
          return this.requestFull<T>(path, options, true);
        }
      }

      if (!response.ok) {
        const error = new Error(
          typeof data === "object" && data && "error" in data
            ? String(data.error)
            : response.statusText,
        ) as FluxbaseError;

        error.status = response.status;
        error.details = data;

        throw error;
      }

      return {
        data: data as T,
        headers: response.headers,
        status: response.status,
      };
    } catch (err) {
      clearTimeout(timeoutId);

      if (err instanceof Error) {
        if (err.name === "AbortError") {
          const timeoutError = new Error("Request timeout") as FluxbaseError;
          timeoutError.status = 408;
          throw timeoutError;
        }

        throw err;
      }

      throw new Error(err instanceof Error ? err.message : String(err) || "Unknown error occurred");
    }
  }

  /**
   * Handle token refresh with deduplication
   * Multiple concurrent requests that fail with 401 will share the same refresh operation
   */
  private async handleTokenRefresh(): Promise<boolean> {
    if (this.isRefreshing && this.refreshPromise) {
      return this.refreshPromise;
    }

    this.isRefreshing = true;
    this.refreshPromise = this.executeRefresh();

    try {
      return await this.refreshPromise;
    } finally {
      this.isRefreshing = false;
      this.refreshPromise = null;
    }
  }

  /**
   * Execute the actual token refresh
   */
  private async executeRefresh(): Promise<boolean> {
    if (!this.refreshTokenCallback) {
      return false;
    }

    try {
      return await this.refreshTokenCallback();
    } catch (error) {
      if (this.debug) {
        console.error("[Fluxbase SDK] Token refresh failed:", error);
      }
      return false;
    }
  }

  /**
   * Internal request implementation with retry capability
   */
  private async requestInternal<T = unknown>(
    path: string,
    options: FetchOptions,
    isRetry: boolean,
  ): Promise<T> {
    const { data } = await this.requestFull<T>(path, options, isRetry);
    return data;
  }

  /**
   * Internal request implementation that returns response with headers
   */
  private async requestWithHeadersInternal<T = unknown>(
    path: string,
    options: FetchOptions,
    isRetry: boolean,
  ): Promise<FetchResponseWithHeaders<T>> {
    return this.requestFull<T>(path, options, isRetry);
  }

  /**
   * GET request
   */
  async get<T = unknown>(
    path: string,
    options: Omit<FetchOptions, "method"> = {},
  ): Promise<T> {
    return this.request<T>(path, { ...options, method: "GET" });
  }

  /**
   * GET request that returns response with headers (for count queries)
   */
  async getWithHeaders<T = unknown>(
    path: string,
    options: Omit<FetchOptions, "method"> = {},
  ): Promise<FetchResponseWithHeaders<T>> {
    return this.requestWithHeaders<T>(path, { ...options, method: "GET" });
  }

  /**
   * POST request that returns response with headers (for POST-based queries with count)
   */
  async postWithHeaders<T = unknown>(
    path: string,
    body?: unknown,
    options: Omit<FetchOptions, "method" | "body"> = {},
  ): Promise<FetchResponseWithHeaders<T>> {
    return this.requestWithHeaders<T>(path, {
      ...options,
      method: "POST",
      body,
    });
  }

  /**
   * Make an HTTP request and return response with headers
   */
  async requestWithHeaders<T = unknown>(
    path: string,
    options: FetchOptions,
  ): Promise<FetchResponseWithHeaders<T>> {
    return this.requestWithHeadersInternal<T>(path, options, false);
  }

  /**
   * POST request
   */
  async post<T = unknown>(
    path: string,
    body?: unknown,
    options: Omit<FetchOptions, "method" | "body"> = {},
  ): Promise<T> {
    return this.request<T>(path, { ...options, method: "POST", body });
  }

  /**
   * PUT request
   */
  async put<T = unknown>(
    path: string,
    body?: unknown,
    options: Omit<FetchOptions, "method" | "body"> = {},
  ): Promise<T> {
    return this.request<T>(path, { ...options, method: "PUT", body });
  }

  /**
   * PATCH request
   */
  async patch<T = unknown>(
    path: string,
    body?: unknown,
    options: Omit<FetchOptions, "method" | "body"> = {},
  ): Promise<T> {
    return this.request<T>(path, { ...options, method: "PATCH", body });
  }

  /**
   * DELETE request
   */
  async delete<T = unknown>(
    path: string,
    options: Omit<FetchOptions, "method"> = {},
  ): Promise<T> {
    return this.request<T>(path, { ...options, method: "DELETE" });
  }

  /**
   * HEAD request
   */
  async head(
    path: string,
    options: Omit<FetchOptions, "method"> = {},
  ): Promise<Headers> {
    const url = `${this.baseUrl}${path}`;
    const headers = { ...this.defaultHeaders, ...options.headers };
    if (this.beforeRequestCallback) {
      this.beforeRequestCallback(headers);
    }

    const response = await fetch(url, {
      method: "HEAD",
      headers,
    });

    return response.headers;
  }

  /**
   * GET request that returns response as Blob (for file downloads)
   */
  async getBlob(
    path: string,
    options: Omit<FetchOptions, "method"> = {},
  ): Promise<Blob> {
    const url = `${this.baseUrl}${path}`;
    const headers = { ...this.defaultHeaders, ...options.headers };
    if (this.beforeRequestCallback) {
      this.beforeRequestCallback(headers);
    }
    // Remove Content-Type for blob downloads
    delete headers["Content-Type"];

    const controller = new AbortController();
    const timeoutId = setTimeout(
      () => controller.abort(),
      options.timeout ?? this.timeout,
    );

    if (this.debug) {
      console.log(`[Fluxbase SDK] GET (blob) ${url}`);
    }

    try {
      const response = await fetch(url, {
        method: "GET",
        headers,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        const error = new Error(response.statusText) as FluxbaseError;
        error.status = response.status;
        throw error;
      }

      return await response.blob();
    } catch (err) {
      clearTimeout(timeoutId);

      if (err instanceof Error) {
        if (err.name === "AbortError") {
          const timeoutError = new Error("Request timeout") as FluxbaseError;
          timeoutError.status = 408;
          throw timeoutError;
        }
        throw err;
      }

      throw new Error(err instanceof Error ? err.message : String(err) || "Unknown error occurred");
    }
  }
}
