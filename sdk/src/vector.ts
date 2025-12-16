/**
 * Vector search module for Fluxbase SDK
 * Provides convenience methods for vector similarity search using pgvector
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  EmbedRequest,
  EmbedResponse,
  FluxbaseResponse,
  VectorSearchOptions,
  VectorSearchResult,
} from "./types";

/**
 * FluxbaseVector provides vector search functionality using pgvector
 *
 * @example
 * ```typescript
 * // Embed text and search
 * const { data: results } = await client.vector.search({
 *   table: 'documents',
 *   column: 'embedding',
 *   query: 'How to use TypeScript?',
 *   match_count: 10
 * })
 *
 * // Embed text directly
 * const { data: embedding } = await client.vector.embed({ text: 'Hello world' })
 * ```
 */
export class FluxbaseVector {
  private fetch: FluxbaseFetch;

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch;
  }

  /**
   * Generate embeddings for text
   *
   * @example
   * ```typescript
   * // Single text
   * const { data } = await client.vector.embed({
   *   text: 'Hello world'
   * })
   * console.log(data.embeddings[0]) // [0.1, 0.2, ...]
   *
   * // Multiple texts
   * const { data } = await client.vector.embed({
   *   texts: ['Hello', 'World'],
   *   model: 'text-embedding-3-small'
   * })
   * ```
   */
  async embed(request: EmbedRequest): Promise<FluxbaseResponse<EmbedResponse>> {
    try {
      const response = await this.fetch.request<EmbedResponse>(
        "/api/v1/vector/embed",
        {
          method: "POST",
          body: request,
        }
      );

      return { data: response, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Search for similar vectors with automatic text embedding
   *
   * This is a convenience method that:
   * 1. Embeds the query text automatically (if `query` is provided)
   * 2. Performs vector similarity search
   * 3. Returns results with distance scores
   *
   * @example
   * ```typescript
   * // Search with text query (auto-embedded)
   * const { data } = await client.vector.search({
   *   table: 'documents',
   *   column: 'embedding',
   *   query: 'How to use TypeScript?',
   *   match_count: 10,
   *   match_threshold: 0.8
   * })
   *
   * // Search with pre-computed vector
   * const { data } = await client.vector.search({
   *   table: 'documents',
   *   column: 'embedding',
   *   vector: [0.1, 0.2, ...],
   *   metric: 'cosine',
   *   match_count: 10
   * })
   *
   * // With additional filters
   * const { data } = await client.vector.search({
   *   table: 'documents',
   *   column: 'embedding',
   *   query: 'TypeScript tutorial',
   *   filters: [
   *     { column: 'status', operator: 'eq', value: 'published' }
   *   ],
   *   match_count: 10
   * })
   * ```
   */
  async search<T = Record<string, unknown>>(
    options: VectorSearchOptions
  ): Promise<FluxbaseResponse<VectorSearchResult<T>>> {
    try {
      const response = await this.fetch.request<VectorSearchResult<T>>(
        "/api/v1/vector/search",
        {
          method: "POST",
          body: {
            table: options.table,
            column: options.column,
            query: options.query,
            vector: options.vector,
            metric: options.metric || "cosine",
            match_threshold: options.match_threshold,
            match_count: options.match_count,
            select: options.select,
            filters: options.filters,
          },
        }
      );

      return { data: response, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }
}
