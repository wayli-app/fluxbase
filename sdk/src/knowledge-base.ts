/**
 * Knowledge Base module for Fluxbase SDK
 * Provides RAG document management, semantic search, and knowledge graph operations
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  KnowledgeBaseSummary,
  KnowledgeBase,
  CreateKnowledgeBaseRequest,
  UpdateKnowledgeBaseRequest,
  KnowledgeBaseDocument,
  AddDocumentRequest,
  AddDocumentResponse,
  UploadDocumentResponse,
  UpdateDocumentRequest,
  DeleteDocumentsByFilterRequest,
  DeleteDocumentsByFilterResponse,
  SearchKnowledgeBaseRequest,
  SearchKnowledgeBaseResponse,
  Entity,
  EntityType,
  EntityRelationship,
  KnowledgeGraphData,
  FluxbaseResponse,
} from "./types";

/**
 * FluxbaseKnowledgeBase provides knowledge base management for RAG applications
 *
 * @example
 * ```typescript
 * // List knowledge bases
 * const { data: kbs } = await client.knowledgeBase.list()
 *
 * // Create a knowledge base
 * const { data: kb } = await client.knowledgeBase.create({
 *   name: 'Product Docs',
 *   description: 'Product documentation for RAG'
 * })
 *
 * // Add a document
 * const { data } = await client.knowledgeBase.addDocument(kb.id, {
 *   title: 'Getting Started',
 *   content: 'Welcome to our product...'
 * })
 *
 * // Search the knowledge base
 * const { data: results } = await client.knowledgeBase.search(kb.id, {
 *   query: 'How to get started?'
 * })
 * ```
 *
 * @category AI
 */
export class FluxbaseKnowledgeBase {
  private fetch: FluxbaseFetch;

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch;
  }

  // ===========================================================================
  // KNOWLEDGE BASE CRUD
  // ===========================================================================

  /**
   * List all knowledge bases the user has access to
   *
   * @example
   * ```typescript
   * const { data, error } = await client.knowledgeBase.list()
   * ```
   */
  async list(): Promise<FluxbaseResponse<KnowledgeBaseSummary[]>> {
    try {
      const data = await this.fetch.request<KnowledgeBaseSummary[]>(
        "/api/v1/ai/knowledge-bases",
        { method: "GET" },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get a knowledge base by ID
   *
   * @param id - Knowledge base ID
   */
  async get(id: string): Promise<FluxbaseResponse<KnowledgeBase>> {
    try {
      const data = await this.fetch.request<KnowledgeBase>(
        `/api/v1/ai/knowledge-bases/${id}`,
        { method: "GET" },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Create a new knowledge base
   *
   * @param request - Knowledge base configuration
   * @example
   * ```typescript
   * const { data, error } = await client.knowledgeBase.create({
   *   name: 'Product Docs',
   *   description: 'Product documentation',
   *   embedding_model: 'text-embedding-3-small',
   *   chunk_size: 1000,
   *   chunk_overlap: 200
   * })
   * ```
   */
  async create(
    request: CreateKnowledgeBaseRequest,
  ): Promise<FluxbaseResponse<KnowledgeBase>> {
    try {
      const data = await this.fetch.request<KnowledgeBase>(
        "/api/v1/ai/knowledge-bases",
        { method: "POST", body: request },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Update a knowledge base
   *
   * @param id - Knowledge base ID
   * @param updates - Fields to update
   */
  async update(
    id: string,
    updates: UpdateKnowledgeBaseRequest,
  ): Promise<FluxbaseResponse<KnowledgeBase>> {
    try {
      const data = await this.fetch.request<KnowledgeBase>(
        `/api/v1/ai/knowledge-bases/${id}`,
        { method: "PATCH", body: updates },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Delete a knowledge base
   *
   * @param id - Knowledge base ID
   */
  async delete(id: string): Promise<{ data: boolean; error: Error | null }> {
    try {
      await this.fetch.request<void>(`/api/v1/ai/knowledge-bases/${id}`, {
        method: "DELETE",
      });
      return { data: true, error: null };
    } catch (error) {
      return { data: false, error: error as Error };
    }
  }

  // ===========================================================================
  // DOCUMENT MANAGEMENT
  // ===========================================================================

  /**
   * List documents in a knowledge base
   *
   * @param kbId - Knowledge base ID
   */
  async listDocuments(
    kbId: string,
  ): Promise<FluxbaseResponse<KnowledgeBaseDocument[]>> {
    try {
      const data = await this.fetch.request<KnowledgeBaseDocument[]>(
        `/api/v1/ai/knowledge-bases/${kbId}/documents`,
        { method: "GET" },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get a single document
   *
   * @param kbId - Knowledge base ID
   * @param docId - Document ID
   */
  async getDocument(
    kbId: string,
    docId: string,
  ): Promise<FluxbaseResponse<KnowledgeBaseDocument>> {
    try {
      const data = await this.fetch.request<KnowledgeBaseDocument>(
        `/api/v1/ai/knowledge-bases/${kbId}/documents/${docId}`,
        { method: "GET" },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Add a document with text content
   *
   * @param kbId - Knowledge base ID
   * @param request - Document content and metadata
   * @example
   * ```typescript
   * const { data } = await client.knowledgeBase.addDocument(kbId, {
   *   title: 'API Reference',
   *   content: 'The API supports REST and GraphQL...',
   *   metadata: { category: 'reference' },
   *   tags: ['api', 'reference']
   * })
   * ```
   */
  async addDocument(
    kbId: string,
    request: AddDocumentRequest,
  ): Promise<FluxbaseResponse<AddDocumentResponse>> {
    try {
      const data = await this.fetch.request<AddDocumentResponse>(
        `/api/v1/ai/knowledge-bases/${kbId}/documents`,
        { method: "POST", body: request },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Upload a file as a document
   *
   * Supports: PDF, TXT, MD, HTML, CSV, DOCX, XLSX, RTF, EPUB, JSON (max 50MB)
   *
   * @param kbId - Knowledge base ID
   * @param file - File to upload (File, Blob, or ArrayBuffer)
   * @param filename - Name of the file
   * @param metadata - Optional metadata
   * @example
   * ```typescript
   * const file = new File(['content'], 'guide.pdf', { type: 'application/pdf' })
   * const { data } = await client.knowledgeBase.uploadDocument(kbId, file, 'guide.pdf')
   * ```
   */
  async uploadDocument(
    kbId: string,
    file: File | Blob | ArrayBuffer,
    filename: string,
    metadata?: Record<string, string>,
  ): Promise<FluxbaseResponse<UploadDocumentResponse>> {
    try {
      const formData = new FormData();
      formData.append("file", file instanceof Blob ? file : new Blob([file]), filename);
      if (metadata) {
        for (const [key, value] of Object.entries(metadata)) {
          formData.append(`metadata[${key}]`, value);
        }
      }
      const data = await this.fetch.request<UploadDocumentResponse>(
        `/api/v1/ai/knowledge-bases/${kbId}/documents/upload`,
        { method: "POST", body: formData },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Update a document's metadata
   *
   * @param kbId - Knowledge base ID
   * @param docId - Document ID
   * @param updates - Fields to update
   */
  async updateDocument(
    kbId: string,
    docId: string,
    updates: UpdateDocumentRequest,
  ): Promise<FluxbaseResponse<KnowledgeBaseDocument>> {
    try {
      const data = await this.fetch.request<KnowledgeBaseDocument>(
        `/api/v1/ai/knowledge-bases/${kbId}/documents/${docId}`,
        { method: "PATCH", body: updates },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Delete a document
   *
   * @param kbId - Knowledge base ID
   * @param docId - Document ID
   */
  async deleteDocument(
    kbId: string,
    docId: string,
  ): Promise<{ data: boolean; error: Error | null }> {
    try {
      await this.fetch.request<void>(
        `/api/v1/ai/knowledge-bases/${kbId}/documents/${docId}`,
        { method: "DELETE" },
      );
      return { data: true, error: null };
    } catch (error) {
      return { data: false, error: error as Error };
    }
  }

  /**
   * Delete documents matching a filter (bulk operation)
   *
   * @param kbId - Knowledge base ID
   * @param filter - Filter criteria (by tags and/or metadata)
   */
  async deleteDocumentsByFilter(
    kbId: string,
    filter: DeleteDocumentsByFilterRequest,
  ): Promise<FluxbaseResponse<DeleteDocumentsByFilterResponse>> {
    try {
      const data = await this.fetch.request<DeleteDocumentsByFilterResponse>(
        `/api/v1/ai/knowledge-bases/${kbId}/documents/delete-by-filter`,
        { method: "POST", body: filter },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  // ===========================================================================
  // SEARCH
  // ===========================================================================

  /**
   * Search a knowledge base using semantic similarity
   *
   * Automatically embeds the query text and returns matching chunks.
   *
   * @param kbId - Knowledge base ID
   * @param request - Search parameters
   * @example
   * ```typescript
   * const { data, error } = await client.knowledgeBase.search(kbId, {
   *   query: 'How to configure authentication?',
   *   max_chunks: 5,
   *   threshold: 0.8
   * })
   *
   * data.results.forEach(result => {
   *   console.log(result.document_title, result.similarity)
   *   console.log(result.content)
   * })
   * ```
   */
  async search(
    kbId: string,
    request: SearchKnowledgeBaseRequest,
  ): Promise<FluxbaseResponse<SearchKnowledgeBaseResponse>> {
    try {
      const data = await this.fetch.request<SearchKnowledgeBaseResponse>(
        `/api/v1/ai/knowledge-bases/${kbId}/search`,
        { method: "POST", body: request },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  // ===========================================================================
  // ENTITIES & KNOWLEDGE GRAPH
  // ===========================================================================

  /**
   * List entities in a knowledge base
   *
   * @param kbId - Knowledge base ID
   * @param type - Optional entity type filter
   */
  async listEntities(
    kbId: string,
    type?: EntityType,
  ): Promise<FluxbaseResponse<Entity[]>> {
    try {
      const params = type ? `?type=${type}` : "";
      const data = await this.fetch.request<Entity[]>(
        `/api/v1/ai/knowledge-bases/${kbId}/entities${params}`,
        { method: "GET" },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Search entities by name
   *
   * @param kbId - Knowledge base ID
   * @param query - Search query
   */
  async searchEntities(
    kbId: string,
    query: string,
  ): Promise<FluxbaseResponse<Entity[]>> {
    try {
      const data = await this.fetch.request<Entity[]>(
        `/api/v1/ai/knowledge-bases/${kbId}/entities/search?q=${encodeURIComponent(query)}`,
        { method: "GET" },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get relationships for an entity
   *
   * @param kbId - Knowledge base ID
   * @param entityId - Entity ID
   */
  async getEntityRelationships(
    kbId: string,
    entityId: string,
  ): Promise<FluxbaseResponse<EntityRelationship[]>> {
    try {
      const data = await this.fetch.request<EntityRelationship[]>(
        `/api/v1/ai/knowledge-bases/${kbId}/entities/${entityId}/relationships`,
        { method: "GET" },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get the full knowledge graph for a knowledge base
   *
   * Returns all entities and relationships for visualization.
   *
   * @param kbId - Knowledge base ID
   */
  async getKnowledgeGraph(
    kbId: string,
  ): Promise<FluxbaseResponse<KnowledgeGraphData>> {
    try {
      const data = await this.fetch.request<KnowledgeGraphData>(
        `/api/v1/ai/knowledge-bases/${kbId}/graph`,
        { method: "GET" },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }
}
