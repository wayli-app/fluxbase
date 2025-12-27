/**
 * Admin AI module for managing AI chatbots, providers, and knowledge bases
 * Provides administrative operations for chatbot lifecycle management and RAG
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  AIChatbot,
  AIChatbotSummary,
  AIProvider,
  CreateAIProviderRequest,
  UpdateAIProviderRequest,
  SyncChatbotsOptions,
  SyncChatbotsResult,
  KnowledgeBase,
  KnowledgeBaseSummary,
  CreateKnowledgeBaseRequest,
  UpdateKnowledgeBaseRequest,
  KnowledgeBaseDocument,
  AddDocumentRequest,
  AddDocumentResponse,
  UploadDocumentResponse,
  ChatbotKnowledgeBaseLink,
  LinkKnowledgeBaseRequest,
  UpdateChatbotKnowledgeBaseRequest,
  SearchKnowledgeBaseResponse,
} from "./types";

/**
 * Admin AI manager for managing AI chatbots and providers
 * Provides create, update, delete, sync, and monitoring operations
 *
 * @category Admin
 */
export class FluxbaseAdminAI {
  private fetch: FluxbaseFetch;

  constructor(fetch: FluxbaseFetch) {
    this.fetch = fetch;
  }

  // ============================================================================
  // CHATBOT MANAGEMENT
  // ============================================================================

  /**
   * List all chatbots (admin view)
   *
   * @param namespace - Optional namespace filter
   * @returns Promise resolving to { data, error } tuple with array of chatbot summaries
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.listChatbots()
   * if (data) {
   *   console.log('Chatbots:', data.map(c => c.name))
   * }
   * ```
   */
  async listChatbots(
    namespace?: string,
  ): Promise<{ data: AIChatbotSummary[] | null; error: Error | null }> {
    try {
      const params = namespace ? `?namespace=${namespace}` : "";
      const response = await this.fetch.get<{
        chatbots: AIChatbotSummary[];
        count: number;
      }>(`/api/v1/admin/ai/chatbots${params}`);
      return { data: response.chatbots || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get details of a specific chatbot
   *
   * @param id - Chatbot ID
   * @returns Promise resolving to { data, error } tuple with chatbot details
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.getChatbot('uuid')
   * if (data) {
   *   console.log('Chatbot:', data.name)
   * }
   * ```
   */
  async getChatbot(
    id: string,
  ): Promise<{ data: AIChatbot | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<AIChatbot>(
        `/api/v1/admin/ai/chatbots/${id}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Enable or disable a chatbot
   *
   * @param id - Chatbot ID
   * @param enabled - Whether to enable or disable
   * @returns Promise resolving to { data, error } tuple with updated chatbot
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.toggleChatbot('uuid', true)
   * ```
   */
  async toggleChatbot(
    id: string,
    enabled: boolean,
  ): Promise<{ data: AIChatbot | null; error: Error | null }> {
    try {
      const data = await this.fetch.put<AIChatbot>(
        `/api/v1/admin/ai/chatbots/${id}/toggle`,
        { enabled },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Delete a chatbot
   *
   * @param id - Chatbot ID
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.deleteChatbot('uuid')
   * ```
   */
  async deleteChatbot(
    id: string,
  ): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.delete(`/api/v1/admin/ai/chatbots/${id}`);
      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Sync chatbots from filesystem or API payload
   *
   * Can sync from:
   * 1. Filesystem (if no chatbots provided) - loads from configured chatbots directory
   * 2. API payload (if chatbots array provided) - syncs provided chatbot specifications
   *
   * Requires service_role or admin authentication.
   *
   * @param options - Sync options including namespace and optional chatbots array
   * @returns Promise resolving to { data, error } tuple with sync results
   *
   * @example
   * ```typescript
   * // Sync from filesystem
   * const { data, error } = await client.admin.ai.sync()
   *
   * // Sync with provided chatbot code
   * const { data, error } = await client.admin.ai.sync({
   *   namespace: 'default',
   *   chatbots: [{
   *     name: 'sql-assistant',
   *     code: myChatbotCode,
   *   }],
   *   options: {
   *     delete_missing: false, // Don't remove chatbots not in this sync
   *     dry_run: false,        // Preview changes without applying
   *   }
   * })
   *
   * if (data) {
   *   console.log(`Synced: ${data.summary.created} created, ${data.summary.updated} updated`)
   * }
   * ```
   */
  async sync(
    options?: SyncChatbotsOptions,
  ): Promise<{ data: SyncChatbotsResult | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<SyncChatbotsResult>(
        "/api/v1/admin/ai/chatbots/sync",
        {
          namespace: options?.namespace || "default",
          chatbots: options?.chatbots,
          options: {
            delete_missing: options?.options?.delete_missing ?? false,
            dry_run: options?.options?.dry_run ?? false,
          },
        },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  // ============================================================================
  // PROVIDER MANAGEMENT
  // ============================================================================

  /**
   * List all AI providers
   *
   * @returns Promise resolving to { data, error } tuple with array of providers
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.listProviders()
   * if (data) {
   *   console.log('Providers:', data.map(p => p.name))
   * }
   * ```
   */
  async listProviders(): Promise<{
    data: AIProvider[] | null;
    error: Error | null;
  }> {
    try {
      const response = await this.fetch.get<{
        providers: AIProvider[];
        count: number;
      }>("/api/v1/admin/ai/providers");
      return { data: response.providers || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get details of a specific provider
   *
   * @param id - Provider ID
   * @returns Promise resolving to { data, error } tuple with provider details
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.getProvider('uuid')
   * if (data) {
   *   console.log('Provider:', data.display_name)
   * }
   * ```
   */
  async getProvider(
    id: string,
  ): Promise<{ data: AIProvider | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<AIProvider>(
        `/api/v1/admin/ai/providers/${id}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Create a new AI provider
   *
   * @param request - Provider configuration
   * @returns Promise resolving to { data, error } tuple with created provider
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.createProvider({
   *   name: 'openai-main',
   *   display_name: 'OpenAI (Main)',
   *   provider_type: 'openai',
   *   is_default: true,
   *   config: {
   *     api_key: 'sk-...',
   *     model: 'gpt-4-turbo',
   *   }
   * })
   * ```
   */
  async createProvider(
    request: CreateAIProviderRequest,
  ): Promise<{ data: AIProvider | null; error: Error | null }> {
    try {
      // Convert all config values to strings (API requires map[string]string)
      // Skip undefined/null values as they shouldn't be sent to the API
      const normalizedConfig: Record<string, string> = {};
      if (request.config) {
        for (const [key, value] of Object.entries(request.config)) {
          if (value !== undefined && value !== null) {
            normalizedConfig[key] = String(value);
          }
        }
      }

      const data = await this.fetch.post<AIProvider>(
        "/api/v1/admin/ai/providers",
        {
          ...request,
          config: normalizedConfig,
        },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Update an existing AI provider
   *
   * @param id - Provider ID
   * @param updates - Fields to update
   * @returns Promise resolving to { data, error } tuple with updated provider
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.updateProvider('uuid', {
   *   display_name: 'Updated Name',
   *   config: {
   *     api_key: 'new-key',
   *     model: 'gpt-4-turbo',
   *   },
   *   enabled: true,
   * })
   * ```
   */
  async updateProvider(
    id: string,
    updates: UpdateAIProviderRequest,
  ): Promise<{ data: AIProvider | null; error: Error | null }> {
    try {
      // Convert all config values to strings (API requires map[string]string)
      // Skip undefined/null values as they shouldn't be sent to the API
      let normalizedUpdates = updates;
      if (updates.config) {
        const normalizedConfig: Record<string, string> = {};
        for (const [key, value] of Object.entries(updates.config)) {
          if (value !== undefined && value !== null) {
            normalizedConfig[key] = String(value);
          }
        }
        normalizedUpdates = { ...updates, config: normalizedConfig };
      }

      const data = await this.fetch.put<AIProvider>(
        `/api/v1/admin/ai/providers/${id}`,
        normalizedUpdates,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Set a provider as the default
   *
   * @param id - Provider ID
   * @returns Promise resolving to { data, error } tuple with updated provider
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.setDefaultProvider('uuid')
   * ```
   */
  async setDefaultProvider(
    id: string,
  ): Promise<{ data: AIProvider | null; error: Error | null }> {
    try {
      const data = await this.fetch.put<AIProvider>(
        `/api/v1/admin/ai/providers/${id}/default`,
        {},
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Delete a provider
   *
   * @param id - Provider ID
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.deleteProvider('uuid')
   * ```
   */
  async deleteProvider(
    id: string,
  ): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.delete(`/api/v1/admin/ai/providers/${id}`);
      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  // ============================================================================
  // KNOWLEDGE BASE MANAGEMENT (RAG)
  // ============================================================================

  /**
   * List all knowledge bases
   *
   * @returns Promise resolving to { data, error } tuple with array of knowledge base summaries
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.listKnowledgeBases()
   * if (data) {
   *   console.log('Knowledge bases:', data.map(kb => kb.name))
   * }
   * ```
   */
  async listKnowledgeBases(): Promise<{
    data: KnowledgeBaseSummary[] | null;
    error: Error | null;
  }> {
    try {
      const response = await this.fetch.get<{
        knowledge_bases: KnowledgeBaseSummary[];
        count: number;
      }>("/api/v1/admin/ai/knowledge-bases");
      return { data: response.knowledge_bases || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get a specific knowledge base
   *
   * @param id - Knowledge base ID
   * @returns Promise resolving to { data, error } tuple with knowledge base details
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.getKnowledgeBase('uuid')
   * if (data) {
   *   console.log('Knowledge base:', data.name)
   * }
   * ```
   */
  async getKnowledgeBase(
    id: string,
  ): Promise<{ data: KnowledgeBase | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<KnowledgeBase>(
        `/api/v1/admin/ai/knowledge-bases/${id}`,
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
   * @returns Promise resolving to { data, error } tuple with created knowledge base
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.createKnowledgeBase({
   *   name: 'product-docs',
   *   description: 'Product documentation',
   *   chunk_size: 512,
   *   chunk_overlap: 50,
   * })
   * ```
   */
  async createKnowledgeBase(
    request: CreateKnowledgeBaseRequest,
  ): Promise<{ data: KnowledgeBase | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<KnowledgeBase>(
        "/api/v1/admin/ai/knowledge-bases",
        request,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Update an existing knowledge base
   *
   * @param id - Knowledge base ID
   * @param updates - Fields to update
   * @returns Promise resolving to { data, error } tuple with updated knowledge base
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.updateKnowledgeBase('uuid', {
   *   description: 'Updated description',
   *   enabled: true,
   * })
   * ```
   */
  async updateKnowledgeBase(
    id: string,
    updates: UpdateKnowledgeBaseRequest,
  ): Promise<{ data: KnowledgeBase | null; error: Error | null }> {
    try {
      const data = await this.fetch.put<KnowledgeBase>(
        `/api/v1/admin/ai/knowledge-bases/${id}`,
        updates,
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
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.deleteKnowledgeBase('uuid')
   * ```
   */
  async deleteKnowledgeBase(
    id: string,
  ): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.delete(`/api/v1/admin/ai/knowledge-bases/${id}`);
      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  // ============================================================================
  // DOCUMENT MANAGEMENT
  // ============================================================================

  /**
   * List documents in a knowledge base
   *
   * @param knowledgeBaseId - Knowledge base ID
   * @returns Promise resolving to { data, error } tuple with array of documents
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.listDocuments('kb-uuid')
   * if (data) {
   *   console.log('Documents:', data.map(d => d.title))
   * }
   * ```
   */
  async listDocuments(
    knowledgeBaseId: string,
  ): Promise<{ data: KnowledgeBaseDocument[] | null; error: Error | null }> {
    try {
      const response = await this.fetch.get<{
        documents: KnowledgeBaseDocument[];
        count: number;
      }>(`/api/v1/admin/ai/knowledge-bases/${knowledgeBaseId}/documents`);
      return { data: response.documents || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Get a specific document
   *
   * @param knowledgeBaseId - Knowledge base ID
   * @param documentId - Document ID
   * @returns Promise resolving to { data, error } tuple with document details
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.getDocument('kb-uuid', 'doc-uuid')
   * ```
   */
  async getDocument(
    knowledgeBaseId: string,
    documentId: string,
  ): Promise<{ data: KnowledgeBaseDocument | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<KnowledgeBaseDocument>(
        `/api/v1/admin/ai/knowledge-bases/${knowledgeBaseId}/documents/${documentId}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Add a document to a knowledge base
   *
   * Document will be chunked and embedded asynchronously.
   *
   * @param knowledgeBaseId - Knowledge base ID
   * @param request - Document content and metadata
   * @returns Promise resolving to { data, error } tuple with document ID
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.addDocument('kb-uuid', {
   *   title: 'Getting Started Guide',
   *   content: 'This is the content of the document...',
   *   metadata: { category: 'guides' },
   * })
   * if (data) {
   *   console.log('Document ID:', data.document_id)
   * }
   * ```
   */
  async addDocument(
    knowledgeBaseId: string,
    request: AddDocumentRequest,
  ): Promise<{ data: AddDocumentResponse | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<AddDocumentResponse>(
        `/api/v1/admin/ai/knowledge-bases/${knowledgeBaseId}/documents`,
        request,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Upload a document file to a knowledge base
   *
   * Supported file types: PDF, TXT, MD, HTML, CSV, DOCX, XLSX, RTF, EPUB, JSON
   * Maximum file size: 50MB
   *
   * @param knowledgeBaseId - Knowledge base ID
   * @param file - File to upload (File or Blob)
   * @param title - Optional document title (defaults to filename without extension)
   * @returns Promise resolving to { data, error } tuple with upload result
   *
   * @example
   * ```typescript
   * // Browser
   * const fileInput = document.getElementById('file') as HTMLInputElement
   * const file = fileInput.files?.[0]
   * if (file) {
   *   const { data, error } = await client.admin.ai.uploadDocument('kb-uuid', file)
   *   if (data) {
   *     console.log('Document ID:', data.document_id)
   *     console.log('Extracted length:', data.extracted_length)
   *   }
   * }
   *
   * // Node.js (with node-fetch or similar)
   * import { Blob } from 'buffer'
   * const content = await fs.readFile('document.pdf')
   * const blob = new Blob([content], { type: 'application/pdf' })
   * const { data, error } = await client.admin.ai.uploadDocument('kb-uuid', blob, 'My Document')
   * ```
   */
  async uploadDocument(
    knowledgeBaseId: string,
    file: File | Blob,
    title?: string,
  ): Promise<{ data: UploadDocumentResponse | null; error: Error | null }> {
    try {
      const formData = new FormData();
      formData.append("file", file);
      if (title) {
        formData.append("title", title);
      }
      const data = await this.fetch.post<UploadDocumentResponse>(
        `/api/v1/admin/ai/knowledge-bases/${knowledgeBaseId}/documents/upload`,
        formData,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Delete a document from a knowledge base
   *
   * @param knowledgeBaseId - Knowledge base ID
   * @param documentId - Document ID
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.deleteDocument('kb-uuid', 'doc-uuid')
   * ```
   */
  async deleteDocument(
    knowledgeBaseId: string,
    documentId: string,
  ): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.delete(
        `/api/v1/admin/ai/knowledge-bases/${knowledgeBaseId}/documents/${documentId}`,
      );
      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Search a knowledge base
   *
   * @param knowledgeBaseId - Knowledge base ID
   * @param query - Search query
   * @param options - Search options
   * @returns Promise resolving to { data, error } tuple with search results
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.searchKnowledgeBase('kb-uuid', 'how to reset password', {
   *   max_chunks: 5,
   *   threshold: 0.7,
   * })
   * if (data) {
   *   console.log('Results:', data.results.map(r => r.content))
   * }
   * ```
   */
  async searchKnowledgeBase(
    knowledgeBaseId: string,
    query: string,
    options?: { max_chunks?: number; threshold?: number },
  ): Promise<{
    data: SearchKnowledgeBaseResponse | null;
    error: Error | null;
  }> {
    try {
      const data = await this.fetch.post<SearchKnowledgeBaseResponse>(
        `/api/v1/admin/ai/knowledge-bases/${knowledgeBaseId}/search`,
        {
          query,
          max_chunks: options?.max_chunks,
          threshold: options?.threshold,
        },
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  // ============================================================================
  // CHATBOT KNOWLEDGE BASE LINKING
  // ============================================================================

  /**
   * List knowledge bases linked to a chatbot
   *
   * @param chatbotId - Chatbot ID
   * @returns Promise resolving to { data, error } tuple with linked knowledge bases
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.listChatbotKnowledgeBases('chatbot-uuid')
   * if (data) {
   *   console.log('Linked KBs:', data.map(l => l.knowledge_base_id))
   * }
   * ```
   */
  async listChatbotKnowledgeBases(
    chatbotId: string,
  ): Promise<{ data: ChatbotKnowledgeBaseLink[] | null; error: Error | null }> {
    try {
      const response = await this.fetch.get<{
        knowledge_bases: ChatbotKnowledgeBaseLink[];
        count: number;
      }>(`/api/v1/admin/ai/chatbots/${chatbotId}/knowledge-bases`);
      return { data: response.knowledge_bases || [], error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Link a knowledge base to a chatbot
   *
   * @param chatbotId - Chatbot ID
   * @param request - Link configuration
   * @returns Promise resolving to { data, error } tuple with link details
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.linkKnowledgeBase('chatbot-uuid', {
   *   knowledge_base_id: 'kb-uuid',
   *   priority: 1,
   *   max_chunks: 5,
   *   similarity_threshold: 0.7,
   * })
   * ```
   */
  async linkKnowledgeBase(
    chatbotId: string,
    request: LinkKnowledgeBaseRequest,
  ): Promise<{ data: ChatbotKnowledgeBaseLink | null; error: Error | null }> {
    try {
      const data = await this.fetch.post<ChatbotKnowledgeBaseLink>(
        `/api/v1/admin/ai/chatbots/${chatbotId}/knowledge-bases`,
        request,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Update a chatbot-knowledge base link
   *
   * @param chatbotId - Chatbot ID
   * @param knowledgeBaseId - Knowledge base ID
   * @param updates - Fields to update
   * @returns Promise resolving to { data, error } tuple with updated link
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.updateChatbotKnowledgeBase(
   *   'chatbot-uuid',
   *   'kb-uuid',
   *   { max_chunks: 10, enabled: true }
   * )
   * ```
   */
  async updateChatbotKnowledgeBase(
    chatbotId: string,
    knowledgeBaseId: string,
    updates: UpdateChatbotKnowledgeBaseRequest,
  ): Promise<{ data: ChatbotKnowledgeBaseLink | null; error: Error | null }> {
    try {
      const data = await this.fetch.put<ChatbotKnowledgeBaseLink>(
        `/api/v1/admin/ai/chatbots/${chatbotId}/knowledge-bases/${knowledgeBaseId}`,
        updates,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Unlink a knowledge base from a chatbot
   *
   * @param chatbotId - Chatbot ID
   * @param knowledgeBaseId - Knowledge base ID
   * @returns Promise resolving to { data, error } tuple
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.unlinkKnowledgeBase('chatbot-uuid', 'kb-uuid')
   * ```
   */
  async unlinkKnowledgeBase(
    chatbotId: string,
    knowledgeBaseId: string,
  ): Promise<{ data: null; error: Error | null }> {
    try {
      await this.fetch.delete(
        `/api/v1/admin/ai/chatbots/${chatbotId}/knowledge-bases/${knowledgeBaseId}`,
      );
      return { data: null, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }
}
