/**
 * Admin AI module for managing AI chatbots and providers
 * Provides administrative operations for chatbot lifecycle management
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
      const data = await this.fetch.post<AIProvider>(
        "/api/v1/admin/ai/providers",
        request,
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
      const data = await this.fetch.put<AIProvider>(
        `/api/v1/admin/ai/providers/${id}`,
        updates,
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
}
