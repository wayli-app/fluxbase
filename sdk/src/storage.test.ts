/**
 * Storage Service Tests
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { FluxbaseStorage, StorageBucket } from './storage'
import type { FluxbaseFetch } from './fetch'

// Mock FluxbaseFetch
class MockFetch {
  public baseUrl: string = 'http://localhost:8080'
  public defaultHeaders: Record<string, string> = {}
  public lastUrl: string = ''
  public lastMethod: string = ''
  public lastBody: unknown = null
  public lastHeaders: Record<string, string> = {}
  public mockResponse: any = null

  async get<T>(path: string): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'GET'
    return this.mockResponse as T
  }

  async post<T>(path: string, body?: unknown, options?: { headers?: Record<string, string> }): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'POST'
    this.lastBody = body
    this.lastHeaders = options?.headers || {}
    return this.mockResponse as T
  }

  async put<T>(path: string, body?: unknown): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'PUT'
    this.lastBody = body
    return this.mockResponse as T
  }

  async patch<T>(path: string, body?: unknown): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'PATCH'
    this.lastBody = body
    return this.mockResponse as T
  }

  async delete(path: string): Promise<void> {
    this.lastUrl = path
    this.lastMethod = 'DELETE'
  }

  async request<T>(path: string, options: { method: string; body?: any; headers?: Record<string, string> }): Promise<T> {
    this.lastUrl = path
    this.lastMethod = options.method
    this.lastBody = options.body
    this.lastHeaders = options.headers || {}
    return this.mockResponse as T
  }

  setAuthToken(token: string | null): void {
    if (token) {
      this.defaultHeaders['Authorization'] = `Bearer ${token}`
    } else {
      delete this.defaultHeaders['Authorization']
    }
  }
}

// Mock File
class MockFile {
  constructor(
    public chunks: BlobPart[],
    public name: string,
    public options?: FilePropertyBag
  ) {}

  get size(): number {
    return this.chunks.reduce((acc, chunk) => acc + (typeof chunk === 'string' ? chunk.length : 0), 0)
  }

  get type(): string {
    return this.options?.type || ''
  }
}

global.File = MockFile as any

describe('FluxbaseStorage - Bucket Operations', () => {
  let fetch: MockFetch
  let storage: FluxbaseStorage

  beforeEach(() => {
    fetch = new MockFetch()
    storage = new FluxbaseStorage(fetch as unknown as FluxbaseFetch)
  })

  it('should list all buckets', async () => {
    fetch.mockResponse = { buckets: [{ name: 'test', created_at: '2024-01-01' }] }

    const { data, error } = await storage.listBuckets()

    expect(fetch.lastMethod).toBe('GET')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets')
    expect(error).toBeNull()
    expect(data).toBeDefined()
  })

  it('should create a bucket', async () => {
    fetch.mockResponse = {}

    const { data, error } = await storage.createBucket('my-bucket')

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/my-bucket')
    expect(error).toBeNull()
    expect(data).toEqual({ name: 'my-bucket' })
  })

  it('should get bucket details', async () => {
    fetch.mockResponse = { name: 'my-bucket', public: false }

    const { data, error } = await storage.getBucket('my-bucket')

    expect(fetch.lastMethod).toBe('GET')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/my-bucket')
    expect(error).toBeNull()
  })

  it('should delete bucket', async () => {
    const { data, error } = await storage.deleteBucket('my-bucket')

    expect(fetch.lastMethod).toBe('DELETE')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/my-bucket')
    expect(error).toBeNull()
  })

  it('should update bucket settings', async () => {
    fetch.mockResponse = {}

    const { error } = await storage.updateBucketSettings('my-bucket', {
      public: true,
    })

    expect(fetch.lastMethod).toBe('PUT')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/my-bucket')
    expect(error).toBeNull()
  })
})

describe('StorageBucket - File Upload', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'uploads')
  })

  it('should upload a file', async () => {
    const file = new MockFile(['Hello World'], 'test.txt', { type: 'text/plain' })
    fetch.mockResponse = { id: '123', key: 'test.txt' }

    const { data, error } = await bucket.upload('test.txt', file)

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastUrl).toContain('/api/v1/storage/uploads/test.txt')
    expect(error).toBeNull()
    expect(data).toBeDefined()
    expect(data?.path).toBe('test.txt')
  })

  it('should upload with custom path', async () => {
    const file = new MockFile(['Content'], 'document.pdf', { type: 'application/pdf' })
    fetch.mockResponse = { id: '456', key: 'documents/2024/document.pdf' }

    const { data, error } = await bucket.upload('documents/2024/document.pdf', file)

    expect(fetch.lastUrl).toContain('/api/v1/storage/uploads/documents/2024/document.pdf')
    expect(error).toBeNull()
  })
})

describe('StorageBucket - File List', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'files')
  })

  it('should list all files', async () => {
    fetch.mockResponse = { files: [{ key: 'test.txt', id: '1' }] }

    const { data, error } = await bucket.list()

    expect(fetch.lastMethod).toBe('GET')
    expect(fetch.lastUrl).toContain('/api/v1/storage/files')
    expect(error).toBeNull()
  })

  it('should list files with prefix', async () => {
    fetch.mockResponse = { files: [] }

    await bucket.list({ prefix: 'documents/' })

    // URLSearchParams encodes '/' as '%2F'
    expect(fetch.lastUrl).toContain('prefix=documents%2F')
  })

  it('should list files with limit', async () => {
    fetch.mockResponse = { files: [] }

    await bucket.list({ limit: 100 })

    expect(fetch.lastUrl).toContain('limit=100')
  })

  it('should list files with offset', async () => {
    fetch.mockResponse = { files: [] }

    await bucket.list({ offset: 50 })

    expect(fetch.lastUrl).toContain('offset=50')
  })

  it('should list files with pagination', async () => {
    fetch.mockResponse = { files: [] }

    await bucket.list({
      limit: 25,
      offset: 0,
      prefix: 'images/',
    })

    expect(fetch.lastUrl).toContain('limit=25')
  })

  it('should support Supabase-style list(path, options)', async () => {
    fetch.mockResponse = { files: [] }

    await bucket.list('documents/', { limit: 10 })

    // URLSearchParams encodes '/' as '%2F'
    expect(fetch.lastUrl).toContain('prefix=documents%2F')
    expect(fetch.lastUrl).toContain('limit=10')
  })
})

describe('StorageBucket - File Operations', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'files')
  })

  it('should copy a file', async () => {
    fetch.mockResponse = {}

    const { data, error } = await bucket.copy('source.txt', 'destination.txt')

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastUrl).toContain('/api/v1/storage/files/copy')
    expect(fetch.lastBody).toEqual({
      from_path: 'source.txt',
      to_path: 'destination.txt',
    })
    expect(error).toBeNull()
  })

  it('should move a file', async () => {
    fetch.mockResponse = {}

    const { data, error } = await bucket.move('old-path.txt', 'new-path.txt')

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastUrl).toContain('/api/v1/storage/files/move')
    expect(fetch.lastBody).toEqual({
      from_path: 'old-path.txt',
      to_path: 'new-path.txt',
    })
    expect(error).toBeNull()
  })

  it('should delete files', async () => {
    const { data, error } = await bucket.remove(['file1.txt', 'file2.txt'])

    expect(fetch.lastMethod).toBe('DELETE')
    expect(error).toBeNull()
  })
})

describe('StorageBucket - URL Generation', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'public-files')
  })

  it('should get public URL', () => {
    const { data } = bucket.getPublicUrl('avatar.jpg')

    expect(data.publicUrl).toContain('/api/v1/storage/public-files/avatar.jpg')
  })

  it('should get public URL with nested path', () => {
    const { data } = bucket.getPublicUrl('images/2024/photo.jpg')

    expect(data.publicUrl).toContain('images/2024/photo.jpg')
  })

  it('should create signed URL', async () => {
    fetch.mockResponse = { signed_url: 'http://example.com/signed' }

    const { data, error } = await bucket.createSignedUrl('private-document.pdf', { expiresIn: 3600 })

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastUrl).toContain('/api/v1/storage/public-files/sign/private-document.pdf')
    expect(fetch.lastBody).toEqual({ expires_in: 3600 })
    expect(error).toBeNull()
    expect(data?.signedUrl).toBe('http://example.com/signed')
  })

  it('should create signed URL with default expiry', async () => {
    fetch.mockResponse = { signed_url: 'http://example.com/signed' }

    const { data, error } = await bucket.createSignedUrl('file.txt')

    expect(fetch.lastBody).toHaveProperty('expires_in')
    expect(error).toBeNull()
  })
})

describe('Storage - Error Handling', () => {
  let fetch: MockFetch
  let storage: FluxbaseStorage

  beforeEach(() => {
    fetch = new MockFetch()
    storage = new FluxbaseStorage(fetch as unknown as FluxbaseFetch)
  })

  it('should handle bucket not found', async () => {
    fetch.get = vi.fn().mockRejectedValue(new Error('Bucket not found'))

    const { data, error } = await storage.getBucket('non-existent')

    expect(error).toBeDefined()
    expect(error?.message).toBe('Bucket not found')
  })

  it('should handle file not found', async () => {
    const bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'files')
    fetch.get = vi.fn().mockRejectedValue(new Error('File not found'))

    // Note: download uses native fetch, not our mock
    // This test validates error handling in list()
    const { data, error } = await bucket.list()

    expect(error).toBeDefined()
  })
})

describe('Storage - Content Types', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'files')
  })

  it('should detect text file type', async () => {
    const file = new MockFile(['Text content'], 'document.txt', { type: 'text/plain' })
    fetch.mockResponse = { id: '1' }

    await bucket.upload('document.txt', file)

    expect(file.type).toBe('text/plain')
  })

  it('should detect image file type', async () => {
    const file = new MockFile(['Image data'], 'photo.jpg', { type: 'image/jpeg' })
    fetch.mockResponse = { id: '1' }

    await bucket.upload('photo.jpg', file)

    expect(file.type).toBe('image/jpeg')
  })

  it('should detect JSON file type', async () => {
    const file = new MockFile([JSON.stringify({ key: 'value' })], 'data.json', { type: 'application/json' })
    fetch.mockResponse = { id: '1' }

    await bucket.upload('data.json', file)

    expect(file.type).toBe('application/json')
  })

  it('should detect PDF file type', async () => {
    const file = new MockFile(['PDF content'], 'document.pdf', { type: 'application/pdf' })
    fetch.mockResponse = { id: '1' }

    await bucket.upload('document.pdf', file)

    expect(file.type).toBe('application/pdf')
  })
})

describe('Storage - Path Handling', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'files')
  })

  it('should handle simple file names', async () => {
    const file = new MockFile(['Content'], 'file.txt')
    fetch.mockResponse = { id: '1' }

    const { data, error } = await bucket.upload('file.txt', file)

    expect(fetch.lastUrl).toContain('file.txt')
    expect(error).toBeNull()
  })

  it('should handle nested paths', async () => {
    const file = new MockFile(['Content'], 'doc.txt')
    fetch.mockResponse = { id: '1' }

    const { data, error } = await bucket.upload('folder/subfolder/doc.txt', file)

    expect(fetch.lastUrl).toContain('folder/subfolder/doc.txt')
    expect(error).toBeNull()
  })

  it('should handle special characters in names', async () => {
    const file = new MockFile(['Content'], 'file with spaces.txt')
    fetch.mockResponse = { id: '1' }

    const { data, error } = await bucket.upload('file with spaces.txt', file)

    expect(fetch.lastUrl).toBeDefined()
    expect(error).toBeNull()
  })

  it('should handle unicode characters', async () => {
    const file = new MockFile(['Content'], 'файл.txt')
    fetch.mockResponse = { id: '1' }

    const { data, error } = await bucket.upload('файл.txt', file)

    expect(fetch.lastUrl).toBeDefined()
    expect(error).toBeNull()
  })
})

describe('Storage - Batch Operations', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'batch')
  })

  it('should delete multiple files at once', async () => {
    const filesToDelete = ['file1.txt', 'file2.txt', 'file3.txt']

    const { data, error } = await bucket.remove(filesToDelete)

    expect(fetch.lastMethod).toBe('DELETE')
    expect(error).toBeNull()
  })

  it('should list files with pagination', async () => {
    fetch.mockResponse = { files: [] }

    // List first batch
    await bucket.list({ limit: 50, offset: 0 })

    expect(fetch.lastUrl).toContain('limit=50')

    // List second batch
    await bucket.list({ limit: 50, offset: 50 })

    expect(fetch.lastUrl).toContain('offset=50')
  })
})

describe('StorageBucket - Stream Download with File Size', () => {
  let mockFetch: MockFetch
  let bucket: StorageBucket
  let originalFetch: typeof globalThis.fetch

  beforeEach(() => {
    mockFetch = new MockFetch()
    mockFetch.defaultHeaders = { 'Authorization': 'Bearer test-token' }
    bucket = new StorageBucket(mockFetch as unknown as FluxbaseFetch, 'downloads')
    originalFetch = globalThis.fetch
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('should return stream with file size from Content-Length header', async () => {
    const mockStream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode('test content'))
        controller.close()
      }
    })

    const mockResponse = {
      ok: true,
      body: mockStream,
      headers: new Headers({
        'content-length': '12345678'
      })
    }

    globalThis.fetch = vi.fn().mockResolvedValue(mockResponse)

    const { data, error } = await bucket.download('large-file.json', { stream: true })

    expect(error).toBeNull()
    expect(data).not.toBeNull()
    expect(data?.stream).toBe(mockStream)
    expect(data?.size).toBe(12345678)
  })

  it('should return null size when Content-Length header is missing', async () => {
    const mockStream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode('content'))
        controller.close()
      }
    })

    const mockResponse = {
      ok: true,
      body: mockStream,
      headers: new Headers({})
    }

    globalThis.fetch = vi.fn().mockResolvedValue(mockResponse)

    const { data, error } = await bucket.download('unknown-size.bin', { stream: true })

    expect(error).toBeNull()
    expect(data).not.toBeNull()
    expect(data?.stream).toBe(mockStream)
    expect(data?.size).toBeNull()
  })

  it('should still return Blob for non-stream downloads', async () => {
    const mockBlob = new Blob(['test content'], { type: 'text/plain' })

    const mockResponse = {
      ok: true,
      blob: vi.fn().mockResolvedValue(mockBlob),
      headers: new Headers({
        'content-length': '12'
      })
    }

    globalThis.fetch = vi.fn().mockResolvedValue(mockResponse)

    const { data, error } = await bucket.download('file.txt')

    expect(error).toBeNull()
    expect(data).toBe(mockBlob)
  })

  it('should handle large file sizes correctly', async () => {
    const mockStream = new ReadableStream()

    const mockResponse = {
      ok: true,
      body: mockStream,
      headers: new Headers({
        'content-length': '10737418240' // 10 GB
      })
    }

    globalThis.fetch = vi.fn().mockResolvedValue(mockResponse)

    const { data, error } = await bucket.download('huge-file.zip', { stream: true })

    expect(error).toBeNull()
    expect(data?.size).toBe(10737418240)
  })
})

describe('StorageBucket - Download Timeout and AbortSignal', () => {
  let mockFetch: MockFetch
  let bucket: StorageBucket
  let originalFetch: typeof globalThis.fetch

  beforeEach(() => {
    mockFetch = new MockFetch()
    mockFetch.defaultHeaders = { 'Authorization': 'Bearer test-token' }
    bucket = new StorageBucket(mockFetch as unknown as FluxbaseFetch, 'downloads')
    originalFetch = globalThis.fetch
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('should apply default 30s timeout for non-streaming downloads', async () => {
    let signalUsed: AbortSignal | undefined

    globalThis.fetch = vi.fn().mockImplementation(async (url, options) => {
      signalUsed = options?.signal
      return {
        ok: true,
        blob: vi.fn().mockResolvedValue(new Blob(['content']))
      }
    })

    await bucket.download('file.txt')

    expect(signalUsed).toBeDefined()
  })

  it('should not apply timeout for streaming downloads by default', async () => {
    const mockStream = new ReadableStream()

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: mockStream,
      headers: new Headers({ 'content-length': '100' })
    })

    const { data, error } = await bucket.download('large.json', { stream: true })

    expect(error).toBeNull()
    expect(data).not.toBeNull()
  })

  it('should return timeout error when download exceeds timeout', async () => {
    globalThis.fetch = vi.fn().mockImplementation(async (url, options) => {
      // Simulate slow response that will be aborted
      await new Promise((resolve, reject) => {
        const timeout = setTimeout(resolve, 5000)
        options?.signal?.addEventListener('abort', () => {
          clearTimeout(timeout)
          reject(new DOMException('Aborted', 'AbortError'))
        })
      })
      return { ok: true, blob: vi.fn().mockResolvedValue(new Blob()) }
    })

    const { data, error } = await bucket.download('file.txt', { timeout: 10 })

    expect(data).toBeNull()
    expect(error?.message).toBe('Download timeout')
  })

  it('should return abort error when external signal is aborted', async () => {
    const controller = new AbortController()

    globalThis.fetch = vi.fn().mockImplementation(async (url, options) => {
      await new Promise((resolve, reject) => {
        options?.signal?.addEventListener('abort', () => {
          reject(new DOMException('Aborted', 'AbortError'))
        })
      })
      return { ok: true }
    })

    // Abort immediately
    setTimeout(() => controller.abort(), 5)

    const { data, error } = await bucket.download('file.txt', { signal: controller.signal })

    expect(data).toBeNull()
    expect(error?.message).toBe('Download aborted')
  })

  it('should return abort error immediately if signal already aborted', async () => {
    const controller = new AbortController()
    controller.abort()

    const { data, error } = await bucket.download('file.txt', { signal: controller.signal })

    expect(data).toBeNull()
    expect(error?.message).toBe('Download aborted')
  })
})

describe('StorageBucket - Resumable Download', () => {
  let mockFetch: MockFetch
  let bucket: StorageBucket
  let originalFetch: typeof globalThis.fetch

  beforeEach(() => {
    mockFetch = new MockFetch()
    mockFetch.defaultHeaders = { 'Authorization': 'Bearer test-token' }
    bucket = new StorageBucket(mockFetch as unknown as FluxbaseFetch, 'downloads')
    originalFetch = globalThis.fetch
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('should return stream and size from downloadResumable', async () => {
    const fileSize = 10 * 1024 * 1024 // 10MB
    const chunkSize = 5 * 1024 * 1024 // 5MB

    globalThis.fetch = vi.fn().mockImplementation(async (url, options) => {
      if (options?.method === 'HEAD') {
        return {
          ok: true,
          headers: new Headers({
            'content-length': String(fileSize),
            'accept-ranges': 'bytes'
          })
        }
      }

      // Range request
      const rangeHeader = options?.headers?.Range || ''
      const match = rangeHeader.match(/bytes=(\d+)-(\d+)/)
      if (match) {
        const start = parseInt(match[1])
        const end = parseInt(match[2])
        const length = end - start + 1
        return {
          ok: true,
          status: 206,
          arrayBuffer: vi.fn().mockResolvedValue(new ArrayBuffer(length))
        }
      }

      return { ok: false, statusText: 'Bad Request' }
    })

    const { data, error } = await bucket.downloadResumable('large-file.json')

    expect(error).toBeNull()
    expect(data).not.toBeNull()
    expect(data?.size).toBe(fileSize)
    expect(data?.stream).toBeInstanceOf(ReadableStream)
  })

  it('should call onProgress callback during download', async () => {
    const fileSize = 15 * 1024 * 1024 // 15MB = 3 chunks at 5MB each
    const progressCalls: any[] = []

    globalThis.fetch = vi.fn().mockImplementation(async (url, options) => {
      if (options?.method === 'HEAD') {
        return {
          ok: true,
          headers: new Headers({
            'content-length': String(fileSize),
            'accept-ranges': 'bytes'
          })
        }
      }

      const rangeHeader = options?.headers?.Range || ''
      const match = rangeHeader.match(/bytes=(\d+)-(\d+)/)
      if (match) {
        const start = parseInt(match[1])
        const end = parseInt(match[2])
        const length = Math.min(end - start + 1, fileSize - start)
        return {
          ok: true,
          status: 206,
          arrayBuffer: vi.fn().mockResolvedValue(new ArrayBuffer(length))
        }
      }

      return { ok: false, statusText: 'Bad Request' }
    })

    const { data, error } = await bucket.downloadResumable('file.json', {
      chunkSize: 5 * 1024 * 1024,
      onProgress: (progress) => progressCalls.push({ ...progress })
    })

    expect(error).toBeNull()
    expect(data).not.toBeNull()

    // Read the stream to trigger progress callbacks
    const reader = data!.stream.getReader()
    while (true) {
      const { done } = await reader.read()
      if (done) break
    }

    expect(progressCalls.length).toBe(3) // 3 chunks
    expect(progressCalls[progressCalls.length - 1].percentage).toBe(100)
    expect(progressCalls[progressCalls.length - 1].currentChunk).toBe(3)
    expect(progressCalls[progressCalls.length - 1].totalChunks).toBe(3)
  })

  it('should fall back to regular streaming when Range not supported', async () => {
    const mockStream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode('content'))
        controller.close()
      }
    })

    globalThis.fetch = vi.fn().mockImplementation(async (url, options) => {
      if (options?.method === 'HEAD') {
        return {
          ok: true,
          headers: new Headers({
            'content-length': '1000'
            // No accept-ranges header
          })
        }
      }

      return {
        ok: true,
        body: mockStream,
        headers: new Headers({ 'content-length': '1000' })
      }
    })

    const { data, error } = await bucket.downloadResumable('file.json')

    expect(error).toBeNull()
    expect(data).not.toBeNull()
    expect(data?.size).toBe(1000)
  })

  it('should return abort error when signal is aborted', async () => {
    const controller = new AbortController()
    controller.abort()

    const { data, error } = await bucket.downloadResumable('file.json', {
      signal: controller.signal
    })

    expect(data).toBeNull()
    expect(error?.message).toBe('Download aborted')
  })

  it('should retry failed chunks with exponential backoff', async () => {
    const fileSize = 5 * 1024 * 1024 // 5MB = 1 chunk
    let attemptCount = 0

    globalThis.fetch = vi.fn().mockImplementation(async (url, options) => {
      if (options?.method === 'HEAD') {
        return {
          ok: true,
          headers: new Headers({
            'content-length': String(fileSize),
            'accept-ranges': 'bytes'
          })
        }
      }

      attemptCount++
      if (attemptCount <= 2) {
        throw new Error('Network error')
      }

      return {
        ok: true,
        status: 206,
        arrayBuffer: vi.fn().mockResolvedValue(new ArrayBuffer(fileSize))
      }
    })

    const { data, error } = await bucket.downloadResumable('file.json', {
      maxRetries: 3,
      retryDelayMs: 10 // Short delay for testing
    })

    expect(error).toBeNull()
    expect(data).not.toBeNull()

    // Read stream to completion
    const reader = data!.stream.getReader()
    await reader.read()

    expect(attemptCount).toBe(3) // 2 failures + 1 success
  })
})
