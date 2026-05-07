import { api } from "./client";

export interface AIProvider {
  id: string;
  name: string;
  display_name: string;
  provider_type: "openai" | "azure" | "ollama";
  is_default: boolean;
  use_for_embeddings: boolean | null;
  embedding_model: string | null;
  config: Record<string, string>;
  enabled: boolean;
  from_config: boolean;
  created_at: string;
  updated_at: string;
  created_by?: string;
}

export interface UpdateAIProviderRequest {
  display_name?: string;
  config?: Record<string, string>;
  enabled?: boolean;
}

export const aiProvidersApi = {
  list: async (): Promise<AIProvider[]> => {
    const response = await api.get<{ providers: AIProvider[] }>(
      "/api/v1/admin/ai/providers",
    );
    return response.data.providers;
  },
};

export interface AIChatbotSummary {
  id: string;
  name: string;
  namespace: string;
  description?: string;
  model?: string;
  enabled: boolean;
  is_public: boolean;
  allowed_tables: string[];
  allowed_operations: string[];
  allowed_schemas: string[];
  version: number;
  source: string;
  created_at: string;
  updated_at: string;
}

export interface AIChatbot extends AIChatbotSummary {
  code: string;
  original_code?: string;
  max_tokens: number;
  temperature: number;
  provider_id?: string;
  persist_conversations: boolean;
  conversation_ttl_hours: number;
  max_conversation_turns: number;
  rate_limit_per_minute: number;
  daily_request_limit: number;
  daily_token_budget: number;
  allow_unauthenticated: boolean;
}

export const chatbotsApi = {
  list: async (namespace?: string): Promise<AIChatbotSummary[]> => {
    const params = namespace ? `?namespace=${namespace}` : "";
    const response = await api.get<{
      chatbots: AIChatbotSummary[];
      count: number;
    }>(`/api/v1/admin/ai/chatbots${params}`);
    return response.data.chatbots || [];
  },

  get: async (id: string): Promise<AIChatbot> => {
    const response = await api.get<AIChatbot>(
      `/api/v1/admin/ai/chatbots/${id}`,
    );
    return response.data;
  },

  toggle: async (id: string, enabled: boolean): Promise<AIChatbot> => {
    const response = await api.put<AIChatbot>(
      `/api/v1/admin/ai/chatbots/${id}/toggle`,
      { enabled },
    );
    return response.data;
  },

  update: async (id: string, data: Partial<AIChatbot>): Promise<AIChatbot> => {
    const response = await api.put<AIChatbot>(
      `/api/v1/admin/ai/chatbots/${id}`,
      data,
    );
    return response.data;
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/v1/admin/ai/chatbots/${id}`);
  },

  sync: async (): Promise<{
    summary: {
      created: number;
      updated: number;
      deleted: number;
      errors: number;
    };
  }> => {
    const response = await api.post<{
      summary: {
        created: number;
        updated: number;
        deleted: number;
        errors: number;
      };
    }>("/api/v1/admin/ai/chatbots/sync", {});
    return response.data;
  },
};

export interface AIMetrics {
  total_requests: number;
  total_tokens: number;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  active_conversations: number;
  total_conversations: number;
  chatbot_stats: Array<{
    chatbot_id: string;
    chatbot_name: string;
    requests: number;
    tokens: number;
    error_count: number;
  }>;
  provider_stats: Array<{
    provider_id: string;
    provider_name: string;
    requests: number;
    avg_latency_ms: number;
  }>;
  error_rate: number;
  avg_response_time_ms: number;
}

export const aiMetricsApi = {
  getMetrics: async (): Promise<AIMetrics> => {
    const response = await api.get<AIMetrics>("/api/v1/admin/ai/metrics");
    return response.data;
  },
};

export interface ConversationSummary {
  id: string;
  chatbot_id: string;
  chatbot_name: string;
  user_id?: string;
  user_email?: string;
  session_id?: string;
  title?: string;
  status: string;
  turn_count: number;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  created_at: string;
  updated_at: string;
  last_message_at: string;
}

export interface MessageDetail {
  id: string;
  conversation_id: string;
  role: string;
  content: string;
  tool_call_id?: string;
  tool_name?: string;
  executed_sql?: string;
  sql_result_summary?: string;
  sql_row_count?: number;
  sql_error?: string;
  sql_duration_ms?: number;
  prompt_tokens?: number;
  completion_tokens?: number;
  created_at: string;
  sequence_number: number;
}

export const conversationsApi = {
  list: async (params?: {
    chatbot_id?: string;
    user_id?: string;
    status?: string;
    limit?: number;
    offset?: number;
  }): Promise<{
    conversations: ConversationSummary[];
    total: number;
    total_count: number;
  }> => {
    const queryParams = new URLSearchParams();
    if (params?.chatbot_id) queryParams.append("chatbot_id", params.chatbot_id);
    if (params?.user_id) queryParams.append("user_id", params.user_id);
    if (params?.status) queryParams.append("status", params.status);
    if (params?.limit) queryParams.append("limit", params.limit.toString());
    if (params?.offset) queryParams.append("offset", params.offset.toString());

    const response = await api.get<{
      conversations: ConversationSummary[];
      total: number;
      total_count: number;
    }>(`/api/v1/admin/ai/conversations?${queryParams.toString()}`);
    return response.data;
  },

  getMessages: async (
    conversationId: string,
  ): Promise<{ messages: MessageDetail[]; total: number }> => {
    const response = await api.get<{
      messages: MessageDetail[];
      total: number;
    }>(`/api/v1/admin/ai/conversations/${conversationId}/messages`);
    return response.data;
  },
};

export interface AuditLogEntry {
  id: string;
  chatbot_id?: string;
  chatbot_name?: string;
  conversation_id?: string;
  message_id?: string;
  user_id?: string;
  user_email?: string;
  generated_sql: string;
  sanitized_sql?: string;
  executed: boolean;
  validation_passed?: boolean;
  validation_errors?: string[];
  success?: boolean;
  error_message?: string;
  rows_returned?: number;
  execution_duration_ms?: number;
  tables_accessed?: string[];
  operations_used?: string[];
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}

export const auditLogApi = {
  list: async (params?: {
    chatbot_id?: string;
    user_id?: string;
    success?: boolean;
    limit?: number;
    offset?: number;
  }): Promise<{
    entries: AuditLogEntry[];
    total: number;
    total_count: number;
  }> => {
    const queryParams = new URLSearchParams();
    if (params?.chatbot_id) queryParams.append("chatbot_id", params.chatbot_id);
    if (params?.user_id) queryParams.append("user_id", params.user_id);
    if (params?.success !== undefined)
      queryParams.append("success", params.success.toString());
    if (params?.limit) queryParams.append("limit", params.limit.toString());
    if (params?.offset) queryParams.append("offset", params.offset.toString());

    const response = await api.get<{
      entries: AuditLogEntry[];
      total: number;
      total_count: number;
    }>(`/api/v1/admin/ai/audit?${queryParams.toString()}`);
    return response.data;
  },
};

export type KBVisibility = "private" | "shared" | "public";
export type KBPermission = "viewer" | "editor" | "owner";

export interface KnowledgeBaseSummary {
  id: string;
  name: string;
  namespace: string;
  description: string;
  enabled: boolean;
  document_count: number;
  total_chunks: number;
  embedding_model: string;
  created_at: string;
  updated_at: string;
  visibility: KBVisibility;
  user_permission?: KBPermission;
  owner_id?: string;
  tenant_id?: string;
  tenant_name?: string;
}

export interface KnowledgeBase extends KnowledgeBaseSummary {
  embedding_dimensions: number;
  chunk_size: number;
  chunk_overlap: number;
  chunk_strategy: string;
  source: string;
  created_by?: string;
}

export interface KBPermissionGrant {
  id: string;
  knowledge_base_id: string;
  user_id: string;
  permission: KBPermission;
  granted_by?: string;
  granted_at: string;
}

export interface CreateKnowledgeBaseRequest {
  name: string;
  namespace?: string;
  description?: string;
  visibility?: KBVisibility;
  embedding_model?: string;
  embedding_dimensions?: number;
  chunk_size?: number;
  chunk_overlap?: number;
  chunk_strategy?: string;
  initial_permissions?: Array<{
    user_id: string;
    permission: KBPermission;
  }>;
}

export interface UpdateKnowledgeBaseRequest {
  name?: string;
  description?: string;
  visibility?: KBVisibility;
  embedding_model?: string;
  embedding_dimensions?: number;
  chunk_size?: number;
  chunk_overlap?: number;
  chunk_strategy?: string;
  enabled?: boolean;
}

export type DocumentStatus = "pending" | "processing" | "indexed" | "failed";

export interface KnowledgeBaseDocument {
  id: string;
  knowledge_base_id: string;
  title: string;
  source_url?: string;
  source_type?: string;
  mime_type: string;
  content_hash: string;
  chunk_count: number;
  status: DocumentStatus;
  error_message?: string;
  metadata?: Record<string, string>;
  tags?: string[];
  owner_id?: string;
  created_at: string;
  updated_at: string;
}

export interface AddDocumentRequest {
  title?: string;
  content: string;
  source?: string;
  mime_type?: string;
  metadata?: Record<string, string>;
  tags?: string[];
}

export interface AddDocumentResponse {
  document_id: string;
  status: string;
  message: string;
}

export interface TableSummary {
  schema: string;
  name: string;
  columns: number;
  foreign_keys: number;
  last_export?: string;
}

export interface ExportTableOptions {
  schema: string;
  table: string;
  columns?: string[];
  include_sample_rows?: boolean;
  sample_row_count?: number;
  include_foreign_keys?: boolean;
  include_indexes?: boolean;
}

export interface ExportTableResult {
  document_id: string;
  entity_id: string;
  relationship_ids: string[];
}

export interface TableColumn {
  name: string;
  data_type: string;
  is_nullable: boolean;
  default_value?: string;
  is_primary_key: boolean;
  is_foreign_key: boolean;
  is_unique: boolean;
  max_length?: number;
  position: number;
}

export interface TableForeignKey {
  name: string;
  column_name: string;
  referenced_schema: string;
  referenced_table: string;
  referenced_column: string;
  on_delete: string;
  on_update: string;
}

export interface TableIndex {
  name: string;
  columns: string[];
  is_unique: boolean;
  is_primary: boolean;
}

export interface TableDetails {
  schema: string;
  name: string;
  type: string;
  columns: TableColumn[];
  primary_key: string[];
  foreign_keys: TableForeignKey[];
  indexes: TableIndex[];
  rls_enabled: boolean;
}

export interface ChatbotKnowledgeBaseLink {
  id: string;
  chatbot_id: string;
  knowledge_base_id: string;
  enabled: boolean;
  max_chunks: number;
  similarity_threshold: number;
  priority: number;
  created_at: string;
  chatbot_name?: string;
}

export interface SearchResult {
  chunk_id: string;
  document_id: string;
  document_title: string;
  knowledge_base_name?: string;
  content: string;
  similarity: number;
}

export interface DebugSearchResult {
  query: string;
  query_embedding_preview: number[];
  query_embedding_dims: number;
  stored_embedding_preview?: number[];
  raw_similarities: number[];
  embedding_model: string;
  kb_embedding_model: string;
  chunks_found: number;
  top_chunk_content_preview?: string;
  total_chunks: number;
  chunks_with_embedding: number;
  chunks_without_embedding: number;
  error_message?: string;
}

export type EntityType =
  | "person"
  | "organization"
  | "location"
  | "concept"
  | "product"
  | "event"
  | "table"
  | "url"
  | "api_endpoint"
  | "datetime"
  | "code_reference"
  | "error"
  | "other";

export interface Entity {
  id: string;
  knowledge_base_id: string;
  entity_type: EntityType;
  name: string;
  canonical_name: string;
  aliases: string[];
  metadata: Record<string, unknown>;
  document_count?: number;
  created_at: string;
}

export interface EntityRelationship {
  id: string;
  knowledge_base_id: string;
  source_entity_id: string;
  target_entity_id: string;
  relationship_type: string;
  confidence?: number;
  metadata: Record<string, unknown>;
  created_at: string;
  source_entity?: Entity;
  target_entity?: Entity;
}

export interface KnowledgeGraphData {
  entities: Entity[];
  relationships: EntityRelationship[];
  entity_count: number;
  relationship_count: number;
}

export const knowledgeBasesApi = {
  list: async (): Promise<KnowledgeBaseSummary[]> => {
    const response = await api.get<{
      knowledge_bases: KnowledgeBaseSummary[];
      count: number;
    }>("/api/v1/admin/ai/knowledge-bases");
    return response.data.knowledge_bases || [];
  },

  get: async (id: string): Promise<KnowledgeBase> => {
    const response = await api.get<KnowledgeBase>(
      `/api/v1/admin/ai/knowledge-bases/${id}`,
    );
    return response.data;
  },

  create: async (data: CreateKnowledgeBaseRequest): Promise<KnowledgeBase> => {
    const response = await api.post<KnowledgeBase>(
      "/api/v1/admin/ai/knowledge-bases",
      data,
    );
    return response.data;
  },

  update: async (
    id: string,
    data: UpdateKnowledgeBaseRequest,
  ): Promise<KnowledgeBase> => {
    const response = await api.put<KnowledgeBase>(
      `/api/v1/admin/ai/knowledge-bases/${id}`,
      data,
    );
    return response.data;
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/v1/admin/ai/knowledge-bases/${id}`);
  },

  listDocuments: async (kbId: string): Promise<KnowledgeBaseDocument[]> => {
    const response = await api.get<{
      documents: KnowledgeBaseDocument[];
      count: number;
    }>(`/api/v1/admin/ai/knowledge-bases/${kbId}/documents`);
    return response.data.documents || [];
  },

  getDocument: async (
    kbId: string,
    docId: string,
  ): Promise<KnowledgeBaseDocument> => {
    const response = await api.get<KnowledgeBaseDocument>(
      `/api/v1/admin/ai/knowledge-bases/${kbId}/documents/${docId}`,
    );
    return response.data;
  },

  addDocument: async (
    kbId: string,
    data: AddDocumentRequest,
  ): Promise<AddDocumentResponse> => {
    const response = await api.post<AddDocumentResponse>(
      `/api/v1/admin/ai/knowledge-bases/${kbId}/documents`,
      data,
    );
    return response.data;
  },

  deleteDocument: async (kbId: string, docId: string): Promise<void> => {
    await api.delete(`/api/v1/admin/ai/knowledge-bases/${kbId}/documents/${docId}`);
  },

  updateDocument: async (
    kbId: string,
    docId: string,
    data: {
      title?: string;
      metadata?: Record<string, string>;
      tags?: string[];
    },
  ): Promise<KnowledgeBaseDocument> => {
    const response = await api.patch<KnowledgeBaseDocument>(
      `/api/v1/admin/ai/knowledge-bases/${kbId}/documents/${docId}`,
      data,
    );
    return response.data;
  },

  uploadDocument: async (
    kbId: string,
    file: File,
    title?: string,
  ): Promise<{
    document_id: string;
    status: string;
    message: string;
    filename: string;
    extracted_length: number;
    mime_type: string;
  }> => {
    const formData = new FormData();
    formData.append("file", file);
    if (title) {
      formData.append("title", title);
    }
    const response = await api.post<{
      document_id: string;
      status: string;
      message: string;
      filename: string;
      extracted_length: number;
      mime_type: string;
    }>(`/api/v1/admin/ai/knowledge-bases/${kbId}/documents/upload`, formData, {
      headers: {
        "Content-Type": "multipart/form-data",
      },
    });
    return response.data;
  },

  search: async (
    kbId: string,
    query: string,
    options?: {
      max_chunks?: number;
      threshold?: number;
      mode?: "semantic" | "keyword" | "hybrid";
      semantic_weight?: number;
    },
  ): Promise<{
    results: SearchResult[];
    count: number;
    query: string;
    mode: string;
  }> => {
    const response = await api.post<{
      results: SearchResult[];
      count: number;
      query: string;
      mode: string;
    }>(`/api/v1/admin/ai/knowledge-bases/${kbId}/search`, {
      query,
      max_chunks: options?.max_chunks,
      threshold: options?.threshold,
      mode: options?.mode,
      semantic_weight: options?.semantic_weight,
    });
    return response.data;
  },

  debugSearch: async (
    kbId: string,
    query: string,
  ): Promise<DebugSearchResult> => {
    const response = await api.post<DebugSearchResult>(
      `/api/v1/admin/ai/knowledge-bases/${kbId}/debug-search`,
      { query },
    );
    return response.data;
  },

  listChatbotLinks: async (
    chatbotId: string,
  ): Promise<ChatbotKnowledgeBaseLink[]> => {
    const response = await api.get<{
      knowledge_bases: ChatbotKnowledgeBaseLink[];
      count: number;
    }>(`/api/v1/admin/ai/chatbots/${chatbotId}/knowledge-bases`);
    return response.data.knowledge_bases || [];
  },

  linkToChatbot: async (
    chatbotId: string,
    kbId: string,
    options?: {
      priority?: number;
      max_chunks?: number;
      similarity_threshold?: number;
    },
  ): Promise<ChatbotKnowledgeBaseLink> => {
    const response = await api.post<ChatbotKnowledgeBaseLink>(
      `/api/v1/admin/ai/chatbots/${chatbotId}/knowledge-bases`,
      {
        knowledge_base_id: kbId,
        ...options,
      },
    );
    return response.data;
  },

  updateChatbotLink: async (
    chatbotId: string,
    kbId: string,
    data: {
      priority?: number;
      max_chunks?: number;
      similarity_threshold?: number;
      enabled?: boolean;
    },
  ): Promise<ChatbotKnowledgeBaseLink> => {
    const response = await api.put<ChatbotKnowledgeBaseLink>(
      `/api/v1/admin/ai/chatbots/${chatbotId}/knowledge-bases/${kbId}`,
      data,
    );
    return response.data;
  },

  unlinkFromChatbot: async (chatbotId: string, kbId: string): Promise<void> => {
    await api.delete(
      `/api/v1/admin/ai/chatbots/${chatbotId}/knowledge-bases/${kbId}`,
    );
  },

  listEntities: async (
    kbId: string,
    entityType?: string,
  ): Promise<Entity[]> => {
    const url = entityType
      ? `/api/v1/admin/ai/knowledge-bases/${kbId}/entities?type=${entityType}`
      : `/api/v1/admin/ai/knowledge-bases/${kbId}/entities`;
    const response = await api.get<{ entities: Entity[]; count: number }>(url);
    return response.data.entities || [];
  },

  searchEntities: async (
    kbId: string,
    query: string,
    types?: string[],
  ): Promise<Entity[]> => {
    const params = new URLSearchParams({ q: query });
    if (types?.length) params.append("types", types.join(","));
    const response = await api.get<{ entities: Entity[]; count: number }>(
      `/api/v1/admin/ai/knowledge-bases/${kbId}/entities/search?${params}`,
    );
    return response.data.entities || [];
  },

  getEntityRelationships: async (
    kbId: string,
    entityId: string,
  ): Promise<EntityRelationship[]> => {
    const response = await api.get<{ relationships: EntityRelationship[] }>(
      `/api/v1/admin/ai/knowledge-bases/${kbId}/entities/${entityId}/relationships`,
    );
    return response.data.relationships || [];
  },

  getKnowledgeGraph: async (kbId: string): Promise<KnowledgeGraphData> => {
    const response = await api.get<KnowledgeGraphData>(
      `/api/v1/admin/ai/knowledge-bases/${kbId}/graph`,
    );
    return response.data;
  },

  listLinkedChatbots: async (
    kbId: string,
  ): Promise<ChatbotKnowledgeBaseLink[]> => {
    const response = await api.get<{
      chatbots: ChatbotKnowledgeBaseLink[];
      count: number;
    }>(`/api/v1/admin/ai/knowledge-bases/${kbId}/chatbots`);
    return response.data.chatbots || [];
  },

  deleteDocumentsByFilter: async (
    kbId: string,
    filter: { tags?: string[]; metadata?: Record<string, string> },
  ): Promise<{ deleted_count: number }> => {
    const response = await api.post<{ deleted_count: number }>(
      `/api/v1/admin/ai/knowledge-bases/${kbId}/documents/delete-by-filter`,
      filter,
    );
    return response.data;
  },

  listTables: async (
    kbId: string,
    schema?: string,
  ): Promise<TableSummary[]> => {
    const params: Record<string, string> = { knowledge_base_id: kbId };
    if (schema) params.schema = schema;
    const response = await api.get<{
      tables: TableSummary[];
      count: number;
    }>("/api/v1/admin/ai/tables", { params });
    return response.data.tables || [];
  },

  exportTable: async (
    kbId: string,
    options: ExportTableOptions,
  ): Promise<ExportTableResult> => {
    const response = await api.post<ExportTableResult>(
      `/api/v1/admin/ai/tables/${options.schema}/${options.table}/export`,
      { ...options, knowledge_base_id: kbId },
    );
    return response.data;
  },

  getTableDetails: async (
    schema: string,
    table: string,
  ): Promise<TableDetails> => {
    const response = await api.get<TableDetails>(
      `/api/v1/admin/ai/tables/${schema}/${table}`,
    );
    return response.data;
  },
};

export const userKnowledgeBasesApi = {
  create: async (data: CreateKnowledgeBaseRequest): Promise<KnowledgeBase> => {
    const response = await api.post<KnowledgeBase>(
      "/api/v1/ai/knowledge-bases",
      data,
    );
    return response.data;
  },

  list: async (): Promise<KnowledgeBaseSummary[]> => {
    const response = await api.get<{
      knowledge_bases: KnowledgeBaseSummary[];
      count: number;
    }>("/api/v1/ai/knowledge-bases");
    return response.data.knowledge_bases || [];
  },

  get: async (id: string): Promise<KnowledgeBase> => {
    const response = await api.get<KnowledgeBase>(
      `/api/v1/ai/knowledge-bases/${id}`,
    );
    return response.data;
  },

  share: async (
    kbId: string,
    userId: string,
    permission: KBPermission,
  ): Promise<KBPermissionGrant> => {
    const response = await api.post<KBPermissionGrant>(
      `/api/v1/ai/knowledge-bases/${kbId}/share`,
      { user_id: userId, permission },
    );
    return response.data;
  },

  listPermissions: async (kbId: string): Promise<KBPermissionGrant[]> => {
    const response = await api.get<KBPermissionGrant[]>(
      `/api/v1/ai/knowledge-bases/${kbId}/permissions`,
    );
    return response.data;
  },

  revokePermission: async (kbId: string, userId: string): Promise<void> => {
    await api.delete(
      `/api/v1/ai/knowledge-bases/${kbId}/permissions/${userId}`,
    );
  },
};
