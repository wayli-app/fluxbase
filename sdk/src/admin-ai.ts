/**
 * Admin AI module for managing AI chatbots and providers
 * Provides administrative operations for chatbot lifecycle management
 */

import type { FluxbaseFetch } from "./fetch";
import type {
  AIChatbot,
  AIChatbotSummary,
  AIProvider,
  SyncChatbotsOptions,
  SyncChatbotsResult,
  ChatbotKnowledgeBaseLink,
  LinkKnowledgeBaseRequest,
  UpdateChatbotKnowledgeBaseRequest,
  TableDetails,
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
   * Get details of a specific AI provider
   *
   * @param id - Provider ID
   * @returns Promise resolving to { data, error } tuple with provider details
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
   * @param params - Provider configuration including name, provider_type, and optional config
   * @returns Promise resolving to { data, error } tuple with created provider
   */
  async createProvider(params: {
    name: string;
    display_name?: string;
    provider_type: string;
    is_default?: boolean;
    config?: Record<string, unknown>;
  }): Promise<{ data: AIProvider | null; error: Error | null }> {
    try {
      const body: Record<string, unknown> = {
        name: params.name,
        provider_type: params.provider_type,
      };
      if (params.display_name !== undefined)
        body.display_name = params.display_name;
      if (params.is_default !== undefined) body.is_default = params.is_default;
      if (params.config) {
        body.config = normalizeConfig(params.config);
      }
      const data = await this.fetch.post<AIProvider>(
        "/api/v1/admin/ai/providers",
        body,
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
   */
  async updateProvider(
    id: string,
    updates: {
      display_name?: string;
      enabled?: boolean;
      config?: Record<string, unknown>;
      embedding_model?: string | null;
    },
  ): Promise<{ data: AIProvider | null; error: Error | null }> {
    try {
      const body: Record<string, unknown> = { ...updates };
      if (updates.config) {
        body.config = normalizeConfig(updates.config);
      }
      const data = await this.fetch.put<AIProvider>(
        `/api/v1/admin/ai/providers/${id}`,
        body,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Set a provider as the default provider
   *
   * @param id - Provider ID
   * @returns Promise resolving to { data, error } tuple with updated provider
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
   * Delete an AI provider
   *
   * @param id - Provider ID
   * @returns Promise resolving to { data, error } tuple
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

  /**
   * Set a provider as the embedding provider
   *
   * @param id - Provider ID
   * @returns Promise resolving to { data, error } tuple with updated provider
   */
  async setEmbeddingProvider(
    id: string,
  ): Promise<{
    data: { use_for_embeddings: boolean } | null;
    error: Error | null;
  }> {
    try {
      const data = await this.fetch.put<{ use_for_embeddings: boolean }>(
        `/api/v1/admin/ai/providers/${id}/embedding`,
        {},
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }

  /**
   * Clear the embedding provider assignment for a provider
   *
   * @param id - Provider ID
   * @returns Promise resolving to { data, error } tuple
   */
  async clearEmbeddingProvider(
    id: string,
  ): Promise<{
    data: { use_for_embeddings: boolean } | null;
    error: Error | null;
  }> {
    try {
      const data = await this.fetch.delete<{ use_for_embeddings: boolean }>(
        `/api/v1/admin/ai/providers/${id}/embedding`,
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

  // ============================================================================
  // TABLE DETAILS
  // ============================================================================

  /**
   * Get detailed table information including columns
   *
   * Use this to discover available columns before exporting.
   *
   * @param schema - Schema name (e.g., 'public')
   * @param table - Table name
   * @returns Promise resolving to { data, error } tuple with table details
   *
   * @example
   * ```typescript
   * const { data, error } = await client.admin.ai.getTableDetails('public', 'users')
   * if (data) {
   *   console.log('Columns:', data.columns.map(c => c.name))
   *   console.log('Primary key:', data.primary_key)
   * }
   * ```
   */
  async getTableDetails(
    schema: string,
    table: string,
  ): Promise<{ data: TableDetails | null; error: Error | null }> {
    try {
      const data = await this.fetch.get<TableDetails>(
        `/api/v1/admin/ai/tables/${schema}/${table}`,
      );
      return { data, error: null };
    } catch (error) {
      return { data: null, error: error as Error };
    }
  }
}

/**
 * Normalize config values: convert non-strings to strings, drop null/undefined entries.
 */
function normalizeConfig(
  config: Record<string, unknown>,
): Record<string, string> {
  const result: Record<string, string> = {};
  for (const [key, value] of Object.entries(config)) {
    if (value === undefined || value === null) continue;
    result[key] = String(value);
  }
  return result;
}
