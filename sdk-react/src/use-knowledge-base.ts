/**
 * Knowledge Base hooks for Fluxbase React SDK
 * Provides hooks for RAG document management, semantic search, and knowledge graph
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useFluxbaseClient } from "./context";
import type {
  CreateKnowledgeBaseRequest,
  UpdateKnowledgeBaseRequest,
  AddDocumentRequest,
  SearchKnowledgeBaseRequest,
} from "@nimbleflux/fluxbase-sdk";

/**
 * Hook to list all knowledge bases the user has access to
 */
export function useKnowledgeBases() {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "knowledge-base", "list"],
    queryFn: async () => {
      const { data, error } = await client.knowledgeBase.list();
      if (error) throw error;
      return data || [];
    },
  });
}

/**
 * Hook to get a single knowledge base by ID
 */
export function useKnowledgeBase(kbId: string | null) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "knowledge-base", kbId],
    queryFn: async () => {
      if (!kbId) return null;
      const { data, error } = await client.knowledgeBase.get(kbId);
      if (error) throw error;
      return data;
    },
    enabled: !!kbId,
  });
}

/**
 * Hook to create a new knowledge base
 */
export function useCreateKnowledgeBase() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (request: CreateKnowledgeBaseRequest) => {
      const { data, error } = await client.knowledgeBase.create(request);
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "knowledge-base", "list"],
      });
    },
  });
}

/**
 * Hook to update a knowledge base
 */
export function useUpdateKnowledgeBase() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      kbId,
      updates,
    }: {
      kbId: string;
      updates: UpdateKnowledgeBaseRequest;
    }) => {
      const { data, error } = await client.knowledgeBase.update(kbId, updates);
      if (error) throw error;
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "knowledge-base", "list"],
      });
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "knowledge-base", variables.kbId],
      });
    },
  });
}

/**
 * Hook to delete a knowledge base
 */
export function useDeleteKnowledgeBase() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (kbId: string) => {
      const { error } = await client.knowledgeBase.delete(kbId);
      if (error) throw error;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "knowledge-base", "list"],
      });
    },
  });
}

/**
 * Hook to list documents in a knowledge base
 */
export function useKBDocuments(kbId: string | null) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "knowledge-base", kbId, "documents"],
    queryFn: async () => {
      if (!kbId) return [];
      const { data, error } = await client.knowledgeBase.listDocuments(kbId);
      if (error) throw error;
      return data || [];
    },
    enabled: !!kbId,
  });
}

/**
 * Hook to add a document to a knowledge base
 */
export function useAddDocument() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      kbId,
      request,
    }: {
      kbId: string;
      request: AddDocumentRequest;
    }) => {
      const { data, error } = await client.knowledgeBase.addDocument(
        kbId,
        request,
      );
      if (error) throw error;
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "knowledge-base", variables.kbId, "documents"],
      });
    },
  });
}

/**
 * Hook to upload a file document to a knowledge base
 */
export function useUploadDocument() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      kbId,
      file,
      filename,
      metadata,
    }: {
      kbId: string;
      file: File | Blob | ArrayBuffer;
      filename: string;
      metadata?: Record<string, string>;
    }) => {
      const { data, error } = await client.knowledgeBase.uploadDocument(
        kbId,
        file,
        filename,
        metadata,
      );
      if (error) throw error;
      return data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "knowledge-base", variables.kbId, "documents"],
      });
    },
  });
}

/**
 * Hook to delete a document
 */
export function useDeleteDocument() {
  const client = useFluxbaseClient();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      kbId,
      docId,
    }: {
      kbId: string;
      docId: string;
    }) => {
      const { error } = await client.knowledgeBase.deleteDocument(kbId, docId);
      if (error) throw error;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["fluxbase", "knowledge-base", variables.kbId, "documents"],
      });
    },
  });
}

/**
 * Hook to search a knowledge base with semantic similarity
 *
 * @example
 * ```tsx
 * const { data, isPending } = useKBSearch(kbId, {
 *   query: 'How to configure auth?',
 *   max_chunks: 5,
 * })
 * ```
 */
export function useKBSearch(
  kbId: string | null,
  request: SearchKnowledgeBaseRequest | null,
) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "knowledge-base", kbId, "search", request],
    queryFn: async () => {
      if (!kbId || !request) return null;
      const { data, error } = await client.knowledgeBase.search(kbId, request);
      if (error) throw error;
      return data;
    },
    enabled: !!kbId && !!request,
  });
}

/**
 * Hook to list entities in a knowledge base
 */
export function useKBEntities(kbId: string | null, type?: string) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "knowledge-base", kbId, "entities", type],
    queryFn: async () => {
      if (!kbId) return [];
      const { data, error } = await client.knowledgeBase.listEntities(
        kbId,
        type as any,
      );
      if (error) throw error;
      return data || [];
    },
    enabled: !!kbId,
  });
}

/**
 * Hook to get the knowledge graph (entities + relationships)
 */
export function useKnowledgeGraph(kbId: string | null) {
  const client = useFluxbaseClient();

  return useQuery({
    queryKey: ["fluxbase", "knowledge-base", kbId, "graph"],
    queryFn: async () => {
      if (!kbId) return null;
      const { data, error } = await client.knowledgeBase.getKnowledgeGraph(kbId);
      if (error) throw error;
      return data;
    },
    enabled: !!kbId,
  });
}
