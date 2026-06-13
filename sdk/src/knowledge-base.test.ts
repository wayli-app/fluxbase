/**
 * Knowledge Base Module Tests
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { FluxbaseKnowledgeBase } from './knowledge-base'
import type { FluxbaseFetch } from './fetch'
import type {
  KnowledgeBaseSummary,
  KnowledgeBase,
  KnowledgeBaseDocument,
  AddDocumentResponse,
  UploadDocumentResponse,
  SearchKnowledgeBaseResponse,
  DeleteDocumentsByFilterResponse,
  Entity,
  EntityRelationship,
  KnowledgeGraphData,
} from './types'

class MockFetch implements Partial<FluxbaseFetch> {
  public lastUrl: string = ''
  public lastMethod: string = ''
  public lastBody: unknown = null
  public mockResponse: any = null
  public shouldThrow: boolean = false
  public errorMessage: string = 'Test error'

  async request<T>(path: string, options: { method: string; body?: unknown }): Promise<T> {
    this.lastUrl = path
    this.lastMethod = options.method
    this.lastBody = options.body
    if (this.shouldThrow) {
      throw new Error(this.errorMessage)
    }
    return this.mockResponse as T
  }
}

describe('FluxbaseKnowledgeBase', () => {
  let mockFetch: MockFetch
  let kb: FluxbaseKnowledgeBase

  beforeEach(() => {
    mockFetch = new MockFetch()
    kb = new FluxbaseKnowledgeBase(mockFetch as unknown as FluxbaseFetch)
  })

  // ===========================================================================
  // KNOWLEDGE BASE CRUD
  // ===========================================================================

  describe('list', () => {
    it('should list knowledge bases', async () => {
      const mockKbs: KnowledgeBaseSummary[] = [
        { id: 'kb1', name: 'Docs', namespace: 'default', description: '', enabled: true, document_count: 5, total_chunks: 100, embedding_model: 'text-embedding-3-small', created_at: '', updated_at: '' },
      ]
      mockFetch.mockResponse = mockKbs

      const { data, error } = await kb.list()

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases')
      expect(mockFetch.lastMethod).toBe('GET')
      expect(data).toEqual(mockKbs)
      expect(error).toBeNull()
    })

    it('should handle errors', async () => {
      mockFetch.shouldThrow = true
      mockFetch.errorMessage = 'Unauthorized'

      const { data, error } = await kb.list()

      expect(data).toBeNull()
      expect(error).toBeInstanceOf(Error)
      expect(error!.message).toBe('Unauthorized')
    })
  })

  describe('get', () => {
    it('should get a knowledge base by ID', async () => {
      const mockKb: KnowledgeBase = {
        id: 'kb1', name: 'Docs', namespace: 'default', description: '', enabled: true,
        document_count: 5, total_chunks: 100, embedding_model: 'text-embedding-3-small',
        embedding_dimensions: 1536, chunk_size: 1000, chunk_overlap: 200,
        chunk_strategy: 'recursive', source: '', created_at: '', updated_at: '',
      }
      mockFetch.mockResponse = mockKb

      const { data, error } = await kb.get('kb1')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases/kb1')
      expect(mockFetch.lastMethod).toBe('GET')
      expect(data).toEqual(mockKb)
      expect(error).toBeNull()
    })
  })

  describe('create', () => {
    it('should create a knowledge base', async () => {
      const mockKb: KnowledgeBase = {
        id: 'kb1', name: 'New KB', namespace: 'default', description: 'Test', enabled: true,
        document_count: 0, total_chunks: 0, embedding_model: 'text-embedding-3-small',
        embedding_dimensions: 1536, chunk_size: 1000, chunk_overlap: 200,
        chunk_strategy: 'recursive', source: '', created_at: '', updated_at: '',
      }
      mockFetch.mockResponse = mockKb

      const { data, error } = await kb.create({
        name: 'New KB',
        description: 'Test',
        embedding_model: 'text-embedding-3-small',
      })

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases')
      expect(mockFetch.lastMethod).toBe('POST')
      expect(mockFetch.lastBody).toEqual({
        name: 'New KB',
        description: 'Test',
        embedding_model: 'text-embedding-3-small',
      })
      expect(data).toEqual(mockKb)
      expect(error).toBeNull()
    })
  })

  describe('update', () => {
    it('should update a knowledge base', async () => {
      mockFetch.mockResponse = { id: 'kb1', name: 'Updated', enabled: false }

      const { data, error } = await kb.update('kb1', { name: 'Updated', enabled: false })

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases/kb1')
      expect(mockFetch.lastMethod).toBe('PATCH')
      expect(mockFetch.lastBody).toEqual({ name: 'Updated', enabled: false })
      expect(error).toBeNull()
    })
  })

  describe('delete', () => {
    it('should delete a knowledge base', async () => {
      const { data, error } = await kb.delete('kb1')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases/kb1')
      expect(mockFetch.lastMethod).toBe('DELETE')
      expect(data).toBe(true)
      expect(error).toBeNull()
    })

    it('should handle delete errors', async () => {
      mockFetch.shouldThrow = true

      const { data, error } = await kb.delete('kb1')

      expect(data).toBe(false)
      expect(error).toBeInstanceOf(Error)
    })
  })

  // ===========================================================================
  // DOCUMENT MANAGEMENT
  // ===========================================================================

  describe('listDocuments', () => {
    it('should list documents in a knowledge base', async () => {
      const mockDocs: KnowledgeBaseDocument[] = [
        { id: 'doc1', knowledge_base_id: 'kb1', title: 'Doc 1', mime_type: 'text/plain', content_hash: 'abc', chunk_count: 5, status: 'indexed', created_at: '', updated_at: '' },
      ]
      mockFetch.mockResponse = mockDocs

      const { data, error } = await kb.listDocuments('kb1')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases/kb1/documents')
      expect(mockFetch.lastMethod).toBe('GET')
      expect(data).toEqual(mockDocs)
      expect(error).toBeNull()
    })
  })

  describe('addDocument', () => {
    it('should add a text document', async () => {
      const mockResp: AddDocumentResponse = { document_id: 'doc1', status: 'processing', message: 'Document added' }
      mockFetch.mockResponse = mockResp

      const { data, error } = await kb.addDocument('kb1', {
        title: 'Test Doc',
        content: 'This is test content',
        metadata: { category: 'test' },
      })

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases/kb1/documents')
      expect(mockFetch.lastMethod).toBe('POST')
      expect(mockFetch.lastBody).toEqual({
        title: 'Test Doc',
        content: 'This is test content',
        metadata: { category: 'test' },
      })
      expect(data).toEqual(mockResp)
      expect(error).toBeNull()
    })
  })

  describe('uploadDocument', () => {
    it('should upload a file document', async () => {
      const mockResp: UploadDocumentResponse = {
        document_id: 'doc1', status: 'processing', message: 'File uploaded',
        filename: 'test.pdf', extracted_length: 5000, mime_type: 'application/pdf',
      }
      mockFetch.mockResponse = mockResp

      const file = new Blob(['content'], { type: 'application/pdf' })
      const { data, error } = await kb.uploadDocument('kb1', file, 'test.pdf')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases/kb1/documents/upload')
      expect(mockFetch.lastMethod).toBe('POST')
      expect(data).toEqual(mockResp)
      expect(error).toBeNull()
    })

    it('should upload with metadata', async () => {
      mockFetch.mockResponse = { document_id: 'doc1', status: 'ok', message: '', filename: 't.txt', extracted_length: 10, mime_type: 'text/plain' }

      await kb.uploadDocument('kb1', new Blob(['x']), 't.txt', { author: 'test' })

      expect(mockFetch.lastBody).toBeInstanceOf(FormData)
    })
  })

  describe('deleteDocument', () => {
    it('should delete a document', async () => {
      const { data, error } = await kb.deleteDocument('kb1', 'doc1')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases/kb1/documents/doc1')
      expect(mockFetch.lastMethod).toBe('DELETE')
      expect(data).toBe(true)
      expect(error).toBeNull()
    })
  })

  describe('deleteDocumentsByFilter', () => {
    it('should delete documents by filter', async () => {
      const mockResp: DeleteDocumentsByFilterResponse = { deleted_count: 3 }
      mockFetch.mockResponse = mockResp

      const { data, error } = await kb.deleteDocumentsByFilter('kb1', {
        tags: ['old'],
      })

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases/kb1/documents/delete-by-filter')
      expect(mockFetch.lastMethod).toBe('POST')
      expect(data).toEqual(mockResp)
      expect(error).toBeNull()
    })
  })

  // ===========================================================================
  // SEARCH
  // ===========================================================================

  describe('search', () => {
    it('should search a knowledge base', async () => {
      const mockResp: SearchKnowledgeBaseResponse = {
        results: [
          { chunk_id: 'c1', document_id: 'd1', document_title: 'Doc 1', content: '...', similarity: 0.95 },
        ],
        count: 1,
        query: 'test query',
      }
      mockFetch.mockResponse = mockResp

      const { data, error } = await kb.search('kb1', {
        query: 'test query',
        max_chunks: 5,
        threshold: 0.8,
      })

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases/kb1/search')
      expect(mockFetch.lastMethod).toBe('POST')
      expect(mockFetch.lastBody).toEqual({
        query: 'test query',
        max_chunks: 5,
        threshold: 0.8,
      })
      expect(data).toEqual(mockResp)
      expect(error).toBeNull()
    })

    it('should handle search errors', async () => {
      mockFetch.shouldThrow = true
      mockFetch.errorMessage = 'Search failed'

      const { data, error } = await kb.search('kb1', { query: 'test' })

      expect(data).toBeNull()
      expect(error!.message).toBe('Search failed')
    })
  })

  // ===========================================================================
  // ENTITIES & KNOWLEDGE GRAPH
  // ===========================================================================

  describe('listEntities', () => {
    it('should list entities without type filter', async () => {
      const mockEntities: Entity[] = [
        { id: 'e1', knowledge_base_id: 'kb1', entity_type: 'person', name: 'John', canonical_name: 'John', aliases: [], metadata: {}, created_at: '', updated_at: '' },
      ]
      mockFetch.mockResponse = mockEntities

      const { data, error } = await kb.listEntities('kb1')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases/kb1/entities')
      expect(mockFetch.lastMethod).toBe('GET')
      expect(data).toEqual(mockEntities)
      expect(error).toBeNull()
    })

    it('should list entities with type filter', async () => {
      mockFetch.mockResponse = []

      await kb.listEntities('kb1', 'organization')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases/kb1/entities?type=organization')
    })
  })

  describe('searchEntities', () => {
    it('should search entities by name', async () => {
      mockFetch.mockResponse = []

      await kb.searchEntities('kb1', 'John Smith')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases/kb1/entities/search?q=John%20Smith')
      expect(mockFetch.lastMethod).toBe('GET')
    })
  })

  describe('getKnowledgeGraph', () => {
    it('should get the knowledge graph', async () => {
      const mockGraph: KnowledgeGraphData = {
        entities: [],
        relationships: [],
        entity_count: 0,
        relationship_count: 0,
      }
      mockFetch.mockResponse = mockGraph

      const { data, error } = await kb.getKnowledgeGraph('kb1')

      expect(mockFetch.lastUrl).toBe('/api/v1/ai/knowledge-bases/kb1/graph')
      expect(mockFetch.lastMethod).toBe('GET')
      expect(data).toEqual(mockGraph)
      expect(error).toBeNull()
    })
  })
})
