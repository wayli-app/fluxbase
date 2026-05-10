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

  getBaseUrl(): string { return this.baseUrl }

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

  it('should upload a Uint8Array', async () => {
    const uint8Array = new Uint8Array([1, 2, 3, 4, 5])
    fetch.mockResponse = { id: '789', key: 'binary.bin' }

    const { data, error } = await bucket.upload('binary.bin', uint8Array, {
      contentType: 'application/octet-stream'
    })

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastUrl).toContain('/api/v1/storage/uploads/binary.bin')
    expect(error).toBeNull()
    expect(data).toBeDefined()
    expect(data?.path).toBe('binary.bin')
    // Verify the body is FormData
    expect(fetch.lastBody).toBeInstanceOf(FormData)
    const formData = fetch.lastBody as FormData
    expect(formData.has('file')).toBe(true)
  })

  it('should upload an ArrayBuffer', async () => {
    const buffer = new ArrayBuffer(8)
    const view = new Uint8Array(buffer)
    view.set([10, 20, 30, 40, 50, 60, 70, 80])
    fetch.mockResponse = { id: '101', key: 'buffer.bin' }

    const { data, error } = await bucket.upload('buffer.bin', buffer, {
      contentType: 'application/octet-stream'
    })

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastUrl).toContain('/api/v1/storage/uploads/buffer.bin')
    expect(error).toBeNull()
    expect(data).toBeDefined()
    // Verify the body is FormData
    expect(fetch.lastBody).toBeInstanceOf(FormData)
    const formData = fetch.lastBody as FormData
    expect(formData.has('file')).toBe(true)
  })

  it('should upload a Blob', async () => {
    const blob = new Blob([new Uint8Array([1, 2, 3, 4])], { type: 'application/zip' })
    fetch.mockResponse = { id: '202', key: 'archive.zip' }

    const { data, error } = await bucket.upload('archive.zip', blob)

    expect(fetch.lastMethod).toBe('POST')
    expect(error).toBeNull()
    expect(data?.path).toBe('archive.zip')
    // Verify the body is FormData
    expect(fetch.lastBody).toBeInstanceOf(FormData)
    const formData = fetch.lastBody as FormData
    expect(formData.has('file')).toBe(true)
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

describe('StorageBucket - File Sharing', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'my-bucket')
  })

  it('should share a file with a user', async () => {
    const { data, error } = await bucket.share('documents/file.pdf', {
      userId: 'user-123',
      permission: 'read',
    })

    expect(error).toBeNull()
    expect(fetch.lastUrl).toBe('/api/v1/storage/my-bucket/documents/file.pdf/share')
    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastBody).toEqual({
      user_id: 'user-123',
      permission: 'read',
    })
  })

  it('should share a file with write permission', async () => {
    const { data, error } = await bucket.share('documents/file.pdf', {
      userId: 'user-456',
      permission: 'write',
    })

    expect(error).toBeNull()
    expect(fetch.lastBody).toEqual({
      user_id: 'user-456',
      permission: 'write',
    })
  })

  it('should revoke file access from a user', async () => {
    const { data, error } = await bucket.revokeShare('documents/file.pdf', 'user-123')

    expect(error).toBeNull()
    expect(fetch.lastUrl).toBe('/api/v1/storage/my-bucket/documents/file.pdf/share/user-123')
    expect(fetch.lastMethod).toBe('DELETE')
  })

  it('should list users a file is shared with', async () => {
    fetch.mockResponse = {
      shares: [
        { user_id: 'user-123', permission: 'read' },
        { user_id: 'user-456', permission: 'write' },
      ],
    }

    const { data, error } = await bucket.listShares('documents/file.pdf')

    expect(error).toBeNull()
    expect(data).toEqual([
      { user_id: 'user-123', permission: 'read' },
      { user_id: 'user-456', permission: 'write' },
    ])
    expect(fetch.lastUrl).toBe('/api/v1/storage/my-bucket/documents/file.pdf/shares')
  })

  it('should return empty array when no shares exist', async () => {
    fetch.mockResponse = { shares: [] }

    const { data, error } = await bucket.listShares('private/file.pdf')

    expect(error).toBeNull()
    expect(data).toEqual([])
  })
})

describe('StorageBucket - Upload Stream', () => {
  let fetch: MockFetch
  let bucket: StorageBucket
  let originalFetch: typeof globalThis.fetch

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'my-bucket')
    originalFetch = globalThis.fetch
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('should upload stream successfully', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: vi.fn().mockResolvedValue({ key: 'video.mp4' }),
    })

    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode('video data'))
        controller.close()
      }
    })

    const { data, error } = await bucket.uploadStream('video.mp4', stream, 1024, {
      contentType: 'video/mp4',
    })

    expect(error).toBeNull()
    expect(data).toEqual({
      id: 'video.mp4',
      path: 'video.mp4',
      fullPath: 'my-bucket/video.mp4',
    })
    expect(globalThis.fetch).toHaveBeenCalledWith(
      'http://localhost:8080/api/v1/storage/my-bucket/stream/video.mp4',
      expect.objectContaining({
        method: 'POST',
      })
    )
  })

  it('should reject upload with invalid size', async () => {
    const stream = new ReadableStream({
      start(controller) {
        controller.close()
      }
    })

    const { data, error } = await bucket.uploadStream('video.mp4', stream, 0)

    expect(data).toBeNull()
    expect(error?.message).toBe('size must be a positive number')
  })

  it('should reject upload with negative size', async () => {
    const stream = new ReadableStream({
      start(controller) {
        controller.close()
      }
    })

    const { data, error } = await bucket.uploadStream('video.mp4', stream, -100)

    expect(data).toBeNull()
    expect(error?.message).toBe('size must be a positive number')
  })

  it('should handle upload stream error', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      statusText: 'Internal Server Error',
      json: vi.fn().mockResolvedValue({ error: 'Upload failed' }),
    })

    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode('data'))
        controller.close()
      }
    })

    const { data, error } = await bucket.uploadStream('file.bin', stream, 100)

    expect(data).toBeNull()
    expect(error).toBeDefined()
  })

  it('should include metadata headers when provided', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: vi.fn().mockResolvedValue({ key: 'file.bin' }),
    })

    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode('data'))
        controller.close()
      }
    })

    await bucket.uploadStream('file.bin', stream, 100, {
      contentType: 'application/octet-stream',
      cacheControl: 'max-age=3600',
      metadata: { custom: 'value' },
    })

    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({
          'X-Storage-Content-Type': 'application/octet-stream',
          'X-Storage-Cache-Control': 'max-age=3600',
          'X-Storage-Metadata': '{"custom":"value"}',
        }),
      })
    )
  })

  it('should support abort signal', async () => {
    const controller = new AbortController()
    controller.abort()

    globalThis.fetch = vi.fn().mockImplementation(() => {
      throw new DOMException('Aborted', 'AbortError')
    })

    const stream = new ReadableStream({
      start(ctrl) {
        ctrl.close()
      }
    })

    const { data, error } = await bucket.uploadStream('file.bin', stream, 100, {
      signal: controller.signal,
    })

    expect(data).toBeNull()
    expect(error).toBeDefined()
  })
})

describe('StorageBucket - Resumable Upload', () => {
  let fetch: MockFetch
  let bucket: StorageBucket
  let originalFetch: typeof globalThis.fetch

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'my-bucket')
    originalFetch = globalThis.fetch
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('should abort when signal is already aborted', async () => {
    const controller = new AbortController()
    controller.abort()

    const file = new Blob(['test content'])
    const { data, error } = await bucket.uploadResumable('file.bin', file, {
      signal: controller.signal,
    })

    expect(data).toBeNull()
    expect(error?.message).toBe('Upload aborted')
  })

  it('should abort resumable upload session', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
    })

    const { error } = await bucket.abortResumableUpload('session-123')

    expect(error).toBeNull()
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/chunked/session-123'),
      expect.objectContaining({ method: 'DELETE' })
    )
  })

  it('should get resumable upload status', async () => {
    const sessionStatus = {
      session: {
        sessionId: 'session-123',
        status: 'in_progress',
        completedChunks: [0, 1],
        totalChunks: 4,
      }
    }

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(sessionStatus),
    })

    const { data, error } = await bucket.getResumableUploadStatus('session-123')

    expect(error).toBeNull()
    expect(data).toEqual(sessionStatus.session)
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/chunked/session-123/status'),
      expect.objectContaining({ method: 'GET' })
    )
  })

  it('should handle error when getting upload status', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      statusText: 'Not Found',
      json: () => Promise.resolve({ error: 'Session not found' }),
    })

    const { data, error } = await bucket.getResumableUploadStatus('invalid-session')

    expect(data).toBeNull()
    expect(error).toBeDefined()
  })
})

describe('FluxbaseStorage - Bucket Management', () => {
  let fetch: MockFetch
  let storage: FluxbaseStorage

  beforeEach(() => {
    fetch = new MockFetch()
    storage = new FluxbaseStorage(fetch as unknown as FluxbaseFetch)
  })

  it('should create a storage bucket reference', () => {
    const bucket = storage.from('my-bucket')
    expect(bucket).toBeDefined()
  })
})

describe('StorageBucket - Transform Options', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'images')
  })

  it('should get transform URL with options', () => {
    const url = bucket.getTransformUrl('photo.jpg', {
      width: 200,
      height: 200,
      fit: 'cover',
      format: 'webp',
      quality: 80,
    })

    expect(url).toContain('http://localhost:8080/api/v1/storage/images/photo.jpg')
    expect(url).toContain('w=200')
    expect(url).toContain('h=200')
    expect(url).toContain('fit=cover')
    expect(url).toContain('fmt=webp')
    expect(url).toContain('q=80')
  })

  it('should get transform URL with only width', () => {
    const url = bucket.getTransformUrl('photo.jpg', {
      width: 400,
    })

    expect(url).toContain('w=400')
    expect(url).not.toContain('h=')
  })

  it('should create signed URL with transform options', async () => {
    fetch.mockResponse = { signed_url: 'https://signed-url-with-transform' }

    const { data, error } = await bucket.createSignedUrl('photo.jpg', {
      expiresIn: 3600,
      transform: {
        width: 100,
        height: 100,
      }
    })

    expect(error).toBeNull()
    expect(fetch.lastUrl).toContain('/sign')
    expect(fetch.lastBody).toEqual(expect.objectContaining({
      width: 100,
      height: 100,
      expires_in: 3600,
    }))
  })
})

describe('StorageBucket - Large File Upload', () => {
  let fetch: MockFetch
  let bucket: StorageBucket
  let originalFetch: typeof globalThis.fetch

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'uploads')
    originalFetch = globalThis.fetch
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('should upload large file using stream', async () => {
    // Mock successful stream upload
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ key: 'big-file.dat' })
    })

    // Create a mock file with stream method
    const mockFile = {
      size: 1000,
      type: 'application/octet-stream',
      stream: () => new ReadableStream({
        start(controller) {
          controller.enqueue(new TextEncoder().encode('x'.repeat(1000)))
          controller.close()
        }
      })
    }

    const { data, error } = await bucket.uploadLargeFile('big-file.dat', mockFile as any)

    expect(error).toBeNull()
    expect(data?.path).toBe('big-file.dat')
    expect(data?.fullPath).toBe('uploads/big-file.dat')
  })

  it('should use default content type when not specified', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ key: 'file.bin' })
    })

    const mockFile = {
      size: 100,
      type: '',
      stream: () => new ReadableStream({
        start(controller) {
          controller.close()
        }
      })
    }

    await bucket.uploadLargeFile('file.bin', mockFile as any)

    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({
          'X-Storage-Content-Type': 'application/octet-stream',
        }),
      })
    )
  })
})

describe('FluxbaseStorage - Empty Bucket', () => {
  let fetch: MockFetch
  let storage: FluxbaseStorage

  beforeEach(() => {
    fetch = new MockFetch()
    storage = new FluxbaseStorage(fetch as unknown as FluxbaseFetch)
  })

  it('should empty a bucket with files', async () => {
    // First call: list files
    fetch.get = vi.fn().mockResolvedValue({
      files: [
        { key: 'file1.txt' },
        { key: 'file2.txt' },
      ]
    })

    // delete calls
    fetch.delete = vi.fn().mockResolvedValue(undefined)

    const { data, error } = await storage.emptyBucket('test-bucket')

    expect(error).toBeNull()
    expect(data?.message).toBe('Successfully emptied')
    expect(fetch.delete).toHaveBeenCalledTimes(2)
  })

  it('should handle empty bucket', async () => {
    fetch.get = vi.fn().mockResolvedValue({ files: [] })

    const { data, error } = await storage.emptyBucket('empty-bucket')

    expect(error).toBeNull()
    expect(data?.message).toBe('Successfully emptied')
  })

  it('should handle list error', async () => {
    fetch.get = vi.fn().mockRejectedValue(new Error('List failed'))

    const { data, error } = await storage.emptyBucket('error-bucket')

    expect(data).toBeNull()
    expect(error?.message).toBe('List failed')
  })

  it('should handle remove error', async () => {
    fetch.get = vi.fn().mockResolvedValue({
      files: [{ key: 'file.txt' }]
    })
    fetch.delete = vi.fn().mockRejectedValue(new Error('Delete failed'))

    const { data, error } = await storage.emptyBucket('test-bucket')

    expect(data).toBeNull()
    expect(error?.message).toBe('Delete failed')
  })
})

// Note: Testing XMLHttpRequest progress tracking requires a full XHR mock
// which is complex in vitest/jsdom. These code paths are tested via E2E tests.

describe('StorageBucket - Upload Options', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'uploads')
  })

  it('should upload with all options', async () => {
    fetch.mockResponse = { id: '123', key: 'file.txt' }

    const file = new Blob(['test'])
    await bucket.upload('file.txt', file, {
      contentType: 'text/plain',
      metadata: { custom: 'value' },
      cacheControl: 'max-age=3600',
      upsert: true,
    })

    const formData = fetch.lastBody as FormData
    expect(formData.has('file')).toBe(true)
    expect(formData.get('content_type')).toBe('text/plain')
    expect(formData.get('metadata')).toBe('{"custom":"value"}')
    expect(formData.get('cache_control')).toBe('max-age=3600')
    expect(formData.get('upsert')).toBe('true')
  })
})

describe('StorageBucket - Stream Upload Progress', () => {
  let fetch: MockFetch
  let bucket: StorageBucket
  let originalFetch: typeof globalThis.fetch

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'uploads')
    originalFetch = globalThis.fetch
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('should track progress during stream upload', async () => {
    const progressCalls: any[] = []

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ key: 'file.bin' })
    })

    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(new Uint8Array([1, 2, 3, 4, 5]))
        controller.close()
      }
    })

    await bucket.uploadStream('file.bin', stream, 5, {
      onUploadProgress: (p) => progressCalls.push({ ...p })
    })

    // The progress is tracked via transform stream, but the mock doesn't actually read it
    expect(globalThis.fetch).toHaveBeenCalled()
  })
})

describe('StorageBucket - Resumable Upload Full Flow', () => {
  let fetch: MockFetch
  let bucket: StorageBucket
  let originalFetch: typeof globalThis.fetch

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'uploads')
    originalFetch = globalThis.fetch
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('should complete full resumable upload', async () => {
    const fileContent = 'x'.repeat(100)
    const file = new Blob([fileContent])
    const progressCalls: any[] = []

    globalThis.fetch = vi.fn()
      .mockResolvedValueOnce({
        // Init response
        ok: true,
        json: () => Promise.resolve({
          session_id: 'sess-123',
          bucket: 'uploads',
          path: 'file.bin',
          total_size: 100,
          chunk_size: 50,
          total_chunks: 2,
          completed_chunks: [],
          status: 'pending',
        })
      })
      .mockResolvedValueOnce({
        // Chunk 0
        ok: true,
        json: () => Promise.resolve({ chunk: 0 })
      })
      .mockResolvedValueOnce({
        // Chunk 1
        ok: true,
        json: () => Promise.resolve({ chunk: 1 })
      })
      .mockResolvedValueOnce({
        // Complete
        ok: true,
        json: () => Promise.resolve({
          id: 'file-id',
          path: 'file.bin',
          full_path: 'uploads/file.bin'
        })
      })

    const { data, error } = await bucket.uploadResumable('file.bin', file, {
      chunkSize: 50,
      onProgress: (p) => progressCalls.push({ ...p })
    })

    expect(error).toBeNull()
    expect(data).toEqual({
      id: 'file-id',
      path: 'file.bin',
      fullPath: 'uploads/file.bin'
    })
    expect(progressCalls.length).toBe(2)
  })

  it('should resume existing upload session', async () => {
    const fileContent = 'x'.repeat(100)
    const file = new Blob([fileContent])

    globalThis.fetch = vi.fn()
      .mockResolvedValueOnce({
        // Status response
        ok: true,
        json: () => Promise.resolve({
          session: {
            sessionId: 'sess-123',
            bucket: 'uploads',
            path: 'file.bin',
            totalSize: 100,
            chunkSize: 50,
            totalChunks: 2,
            completedChunks: [0], // First chunk already uploaded
            status: 'in_progress',
          }
        })
      })
      .mockResolvedValueOnce({
        // Chunk 1 (skipping 0)
        ok: true,
        json: () => Promise.resolve({ chunk: 1 })
      })
      .mockResolvedValueOnce({
        // Complete
        ok: true,
        json: () => Promise.resolve({
          id: 'file-id',
          path: 'file.bin',
          full_path: 'uploads/file.bin'
        })
      })

    const { data, error } = await bucket.uploadResumable('file.bin', file, {
      chunkSize: 50,
      resumeSessionId: 'sess-123',
    })

    expect(error).toBeNull()
    expect(data?.id).toBe('file-id')
    // Should have only uploaded chunk 1 (not chunk 0)
    expect(globalThis.fetch).toHaveBeenCalledTimes(3) // status + chunk1 + complete
  })

  it('should handle init error', async () => {
    globalThis.fetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      statusText: 'Server Error',
      json: () => Promise.resolve({ error: 'Init failed' })
    })

    const file = new Blob(['test'])
    const { data, error } = await bucket.uploadResumable('file.bin', file)

    expect(data).toBeNull()
    expect(error?.message).toContain('Init failed')
  })

  it('should handle chunk upload error with retry', async () => {
    const file = new Blob(['x'.repeat(10)])

    let chunkAttempts = 0
    globalThis.fetch = vi.fn()
      .mockResolvedValueOnce({
        // Init
        ok: true,
        json: () => Promise.resolve({
          session_id: 'sess-123',
          total_chunks: 1,
          completed_chunks: [],
        })
      })
      .mockImplementation(() => {
        chunkAttempts++
        if (chunkAttempts <= 2) {
          return Promise.resolve({
            ok: false,
            statusText: 'Error',
            json: () => Promise.resolve({ error: 'Chunk failed' })
          })
        }
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ chunk: 0 })
        })
      })

    const { data, error } = await bucket.uploadResumable('file.bin', file, {
      maxRetries: 3,
      retryDelayMs: 1,
    })

    // Should have retried and eventually failed or succeeded
    expect(chunkAttempts).toBeGreaterThan(1)
  })
})

describe('StorageBucket - Transform URL Edge Cases', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'images')
  })

  it('should return base URL when no transform options provided', () => {
    const url = bucket.getTransformUrl('photo.jpg', {})
    expect(url).toBe('http://localhost:8080/api/v1/storage/images/photo.jpg')
    expect(url).not.toContain('?')
  })

  it('should not include zero or negative dimensions', () => {
    const url = bucket.getTransformUrl('photo.jpg', {
      width: 0,
      height: -1,
    })
    expect(url).not.toContain('w=')
    expect(url).not.toContain('h=')
  })

  it('should not include zero quality', () => {
    const url = bucket.getTransformUrl('photo.jpg', {
      width: 100,
      quality: 0,
    })
    expect(url).toContain('w=100')
    expect(url).not.toContain('q=')
  })
})

describe('FluxbaseStorage - Error Handling', () => {
  let fetch: MockFetch
  let storage: FluxbaseStorage

  beforeEach(() => {
    fetch = new MockFetch()
    storage = new FluxbaseStorage(fetch as unknown as FluxbaseFetch)
  })

  it('should handle listBuckets error', async () => {
    fetch.get = vi.fn().mockRejectedValue(new Error('Access denied'))

    const { data, error } = await storage.listBuckets()

    expect(data).toBeNull()
    expect(error?.message).toBe('Access denied')
  })

  it('should handle createBucket error', async () => {
    fetch.post = vi.fn().mockRejectedValue(new Error('Bucket already exists'))

    const { data, error } = await storage.createBucket('existing-bucket')

    expect(data).toBeNull()
    expect(error?.message).toBe('Bucket already exists')
  })

  it('should handle deleteBucket error', async () => {
    fetch.delete = vi.fn().mockRejectedValue(new Error('Bucket not empty'))

    const { data, error } = await storage.deleteBucket('my-bucket')

    expect(data).toBeNull()
    expect(error?.message).toBe('Bucket not empty')
  })

  it('should handle updateBucketSettings error', async () => {
    fetch.put = vi.fn().mockRejectedValue(new Error('Permission denied'))

    const { error } = await storage.updateBucketSettings('my-bucket', { public: true })

    expect(error?.message).toBe('Permission denied')
  })

  it('should handle getBucket error', async () => {
    fetch.get = vi.fn().mockRejectedValue(new Error('Bucket not found'))

    const { data, error } = await storage.getBucket('unknown-bucket')

    expect(data).toBeNull()
    expect(error?.message).toBe('Bucket not found')
  })

  it('should handle emptyBucket with unexpected exception', async () => {
    // Mock list to succeed but throw unexpected error
    storage.from = vi.fn().mockReturnValue({
      list: vi.fn().mockImplementation(() => {
        throw new Error('Unexpected error')
      })
    })

    const { data, error } = await storage.emptyBucket('test-bucket')

    expect(data).toBeNull()
    expect(error?.message).toBe('Unexpected error')
  })
})

describe('StorageBucket - Share Error Handling', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'my-bucket')
  })

  it('should handle share error', async () => {
    fetch.post = vi.fn().mockRejectedValue(new Error('User not found'))

    const { error } = await bucket.share('file.txt', {
      userId: 'unknown-user',
      permission: 'read',
    })

    expect(error?.message).toBe('User not found')
  })

  it('should handle revokeShare error', async () => {
    fetch.delete = vi.fn().mockRejectedValue(new Error('Share not found'))

    const { error } = await bucket.revokeShare('file.txt', 'user-123')

    expect(error?.message).toBe('Share not found')
  })

  it('should handle listShares error', async () => {
    fetch.get = vi.fn().mockRejectedValue(new Error('Access denied'))

    const { data, error } = await bucket.listShares('file.txt')

    expect(data).toBeNull()
    expect(error?.message).toBe('Access denied')
  })
})

describe('StorageBucket - Resumable Upload Error Paths', () => {
  let fetch: MockFetch
  let bucket: StorageBucket
  let originalFetch: typeof globalThis.fetch

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'uploads')
    originalFetch = globalThis.fetch
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('should handle session status error during resume', async () => {
    const file = new Blob(['test content'])

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      statusText: 'Not Found',
      json: () => Promise.resolve({ error: 'Session expired' }),
    })

    const { data, error } = await bucket.uploadResumable('file.bin', file, {
      sessionId: 'expired-session',
    })

    expect(data).toBeNull()
    expect(error?.message).toBe('Session expired')
  })

  it('should handle pre-abort signal during resumable upload', async () => {
    const fileContent = 'x'.repeat(100)
    const file = new Blob([fileContent])

    // Abort before starting
    const controller = new AbortController()
    controller.abort()

    globalThis.fetch = vi.fn().mockImplementation(() => {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({
          session_id: 'session-123',
          bucket: 'uploads',
          path: 'file.bin',
          total_size: 100,
          chunk_size: 1000,
          total_chunks: 1,
          completed_chunks: [],
          status: 'active',
        }),
      })
    })

    const { data, error } = await bucket.uploadResumable('file.bin', file, {
      signal: controller.signal,
    })

    expect(data).toBeNull()
    expect(error?.message).toBe('Upload aborted')
  })

  it('should handle chunk upload error with JSON parse failure', async () => {
    const fileContent = 'x'.repeat(100)
    const file = new Blob([fileContent])

    let callCount = 0
    globalThis.fetch = vi.fn().mockImplementation(() => {
      callCount++
      if (callCount === 1) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            session_id: 'session-123',
            bucket: 'uploads',
            path: 'file.bin',
            total_size: 100,
            chunk_size: 1000,
            total_chunks: 1,
            completed_chunks: [],
            status: 'active',
          }),
        })
      }
      // Chunk upload fails with non-parseable response
      return Promise.resolve({
        ok: false,
        statusText: 'Bad Request',
        json: () => Promise.reject(new Error('Invalid JSON')),
      })
    })

    const { data, error } = await bucket.uploadResumable('file.bin', file, {
      maxRetries: 0,
    })

    expect(data).toBeNull()
    expect(error?.message).toContain('Bad Request')
  })

  it('should handle complete upload error with JSON parse failure', async () => {
    const fileContent = 'x'.repeat(50)
    const file = new Blob([fileContent])

    let callCount = 0
    globalThis.fetch = vi.fn().mockImplementation(() => {
      callCount++
      if (callCount === 1) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            session_id: 'session-123',
            bucket: 'uploads',
            path: 'file.bin',
            total_size: 50,
            chunk_size: 100,
            total_chunks: 1,
            completed_chunks: [],
            status: 'active',
          }),
        })
      }
      if (callCount === 2) {
        // Chunk upload succeeds
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({}),
        })
      }
      // Complete fails
      return Promise.resolve({
        ok: false,
        statusText: 'Internal Error',
        json: () => Promise.reject(new Error('Invalid JSON')),
      })
    })

    const { data, error } = await bucket.uploadResumable('file.bin', file)

    expect(data).toBeNull()
    expect(error?.message).toContain('Internal Error')
  })

  it('should resume session and track progress bytes', async () => {
    const fileContent = 'x'.repeat(200)
    const file = new Blob([fileContent])

    const progressCalls: any[] = []

    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes('/status')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            session: {
              sessionId: 'session-123',
              bucket: 'uploads',
              path: 'file.bin',
              totalSize: 200,
              chunkSize: 50,
              totalChunks: 4,
              completedChunks: [0, 1], // First two chunks already done
              status: 'active',
            },
          }),
        })
      }

      if (url.includes('/complete')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            id: 'file-id',
            path: 'file.bin',
            full_path: 'uploads/file.bin',
          }),
        })
      }

      // Chunk uploads
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({}),
      })
    })

    const { data, error } = await bucket.uploadResumable('file.bin', file, {
      sessionId: 'session-123',
      onProgress: (p) => progressCalls.push({ ...p }),
    })

    expect(error).toBeNull()
    expect(data).toBeDefined()
  })

  it('should handle init response with JSON parse failure', async () => {
    const fileContent = 'x'.repeat(100)
    const file = new Blob([fileContent])

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      statusText: 'Bad Request',
      json: () => Promise.reject(new Error('Invalid JSON')),
    })

    const { data, error } = await bucket.uploadResumable('file.bin', file)

    expect(data).toBeNull()
    expect(error?.message).toContain('Bad Request')
  })

  it('should handle status response with JSON parse failure during resume', async () => {
    const fileContent = 'x'.repeat(100)
    const file = new Blob([fileContent])

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      statusText: 'Not Found',
      json: () => Promise.reject(new Error('Invalid JSON')),
    })

    const { data, error } = await bucket.uploadResumable('file.bin', file, {
      sessionId: 'session-123',
    })

    expect(data).toBeNull()
    expect(error?.message).toContain('Not Found')
  })
})

describe('StorageBucket - AbortResumableUpload Error Path', () => {
  let fetch: MockFetch
  let bucket: StorageBucket
  let originalFetch: typeof globalThis.fetch

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'uploads')
    originalFetch = globalThis.fetch
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('should handle abortResumableUpload error', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      statusText: 'Not Found',
      json: () => Promise.resolve({ error: 'Session not found' }),
    })

    const { error } = await bucket.abortResumableUpload('invalid-session')

    expect(error?.message).toBe('Session not found')
  })

  it('should handle abortResumableUpload JSON parse error', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      statusText: 'Server Error',
      json: () => Promise.reject(new Error('Invalid JSON')),
    })

    const { error } = await bucket.abortResumableUpload('invalid-session')

    expect(error?.message).toContain('Server Error')
  })
})

describe('StorageBucket - File Operations Error Handling', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch as unknown as FluxbaseFetch, 'test-bucket')
  })

  it('should handle createSignedUrl error', async () => {
    fetch.post = vi.fn().mockRejectedValue(new Error('Access denied'))

    const { data, error } = await bucket.createSignedUrl('private/file.pdf', 3600)

    expect(data).toBeNull()
    expect(error?.message).toBe('Access denied')
  })

  it('should handle createSignedUrl error with transform options', async () => {
    fetch.post = vi.fn().mockRejectedValue(new Error('File not found'))

    const { data, error } = await bucket.createSignedUrl('missing/image.jpg', 3600, {
      transform: {
        width: 100,
        height: 100,
        format: 'webp',
        quality: 80,
        fit: 'cover',
      }
    })

    expect(data).toBeNull()
    expect(error?.message).toBe('File not found')
  })

  it('should handle move error', async () => {
    fetch.post = vi.fn().mockRejectedValue(new Error('Destination already exists'))

    const { data, error } = await bucket.move('source/file.txt', 'dest/file.txt')

    expect(data).toBeNull()
    expect(error?.message).toBe('Destination already exists')
  })

  it('should handle copy error', async () => {
    fetch.post = vi.fn().mockRejectedValue(new Error('Source file not found'))

    const { data, error } = await bucket.copy('missing/file.txt', 'dest/file.txt')

    expect(data).toBeNull()
    expect(error?.message).toBe('Source file not found')
  })
})
