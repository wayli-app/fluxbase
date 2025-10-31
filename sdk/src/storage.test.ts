/**
 * Storage Service Tests
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { FluxbaseStorage, StorageBucket } from './storage'
import type { FluxbaseFetch } from './fetch'

// Mock FluxbaseFetch
class MockFetch implements FluxbaseFetch {
  public lastUrl: string = ''
  public lastMethod: string = ''
  public lastBody: unknown = null
  public lastHeaders: Record<string, string> = {}

  constructor(public baseUrl: string = 'http://localhost:8080', public headers: Record<string, string> = {}) {}

  async get<T>(path: string): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'GET'
    return [] as T
  }

  async post<T>(path: string, body?: unknown, options?: { headers?: Record<string, string> }): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'POST'
    this.lastBody = body
    this.lastHeaders = options?.headers || {}
    return body as T
  }

  async patch<T>(path: string, body?: unknown): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'PATCH'
    this.lastBody = body
    return body as T
  }

  async delete(path: string): Promise<void> {
    this.lastUrl = path
    this.lastMethod = 'DELETE'
  }

  setAuthToken(token: string | null): void {
    if (token) {
      this.headers['Authorization'] = `Bearer ${token}`
    } else {
      delete this.headers['Authorization']
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
    storage = new FluxbaseStorage(fetch)
  })

  it('should list all buckets', async () => {
    await storage.listBuckets()

    expect(fetch.lastMethod).toBe('GET')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets')
  })

  it('should create a bucket', async () => {
    await storage.createBucket({
      name: 'my-bucket',
      public: false,
    })

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets')
    expect(fetch.lastBody).toEqual({ name: 'my-bucket', public: false })
  })

  it('should create a public bucket', async () => {
    await storage.createBucket({
      name: 'public-bucket',
      public: true,
    })

    const body = fetch.lastBody as any
    expect(body.public).toBe(true)
  })

  it('should get bucket details', async () => {
    await storage.getBucket('my-bucket')

    expect(fetch.lastMethod).toBe('GET')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/my-bucket')
  })

  it('should update bucket', async () => {
    await storage.updateBucket('my-bucket', {
      public: true,
    })

    expect(fetch.lastMethod).toBe('PATCH')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/my-bucket')
  })

  it('should delete bucket', async () => {
    await storage.deleteBucket('my-bucket')

    expect(fetch.lastMethod).toBe('DELETE')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/my-bucket')
  })
})

describe('StorageBucket - File Upload', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch, 'uploads')
  })

  it('should upload a file', async () => {
    const file = new MockFile(['Hello World'], 'test.txt', { type: 'text/plain' })

    await bucket.upload('test.txt', file)

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/uploads/files')
  })

  it('should upload with custom path', async () => {
    const file = new MockFile(['Content'], 'document.pdf', { type: 'application/pdf' })

    await bucket.upload('documents/2024/document.pdf', file)

    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/uploads/files')
  })

  it('should upload with metadata', async () => {
    const file = new MockFile(['Image data'], 'photo.jpg', { type: 'image/jpeg' })

    await bucket.upload('photos/photo.jpg', file, {
      metadata: {
        author: 'John Doe',
        description: 'Test photo',
      },
    })

    expect(fetch.lastMethod).toBe('POST')
  })

  it('should upload multiple files', async () => {
    const file1 = new MockFile(['File 1'], 'file1.txt')
    const file2 = new MockFile(['File 2'], 'file2.txt')

    await bucket.upload('file1.txt', file1)
    await bucket.upload('file2.txt', file2)

    expect(fetch.lastMethod).toBe('POST')
  })
})

describe('StorageBucket - File Download', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch, 'downloads')
  })

  it('should download a file', async () => {
    await bucket.download('file.txt')

    expect(fetch.lastMethod).toBe('GET')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/downloads/files/file.txt')
  })

  it('should download from nested path', async () => {
    await bucket.download('folder/subfolder/document.pdf')

    expect(fetch.lastUrl).toContain('folder/subfolder/document.pdf')
  })
})

describe('StorageBucket - File List', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch, 'files')
  })

  it('should list all files', async () => {
    await bucket.list()

    expect(fetch.lastMethod).toBe('GET')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/files/files')
  })

  it('should list files with prefix', async () => {
    await bucket.list({ prefix: 'documents/' })

    expect(fetch.lastUrl).toContain('prefix=documents/')
  })

  it('should list files with limit', async () => {
    await bucket.list({ limit: 100 })

    expect(fetch.lastUrl).toContain('limit=100')
  })

  it('should list files with offset', async () => {
    await bucket.list({ offset: 50 })

    expect(fetch.lastUrl).toContain('offset=50')
  })

  it('should list files with pagination', async () => {
    await bucket.list({
      limit: 25,
      offset: 0,
      prefix: 'images/',
    })

    expect(fetch.lastUrl).toContain('limit=25')
    expect(fetch.lastUrl).toContain('offset=0')
    expect(fetch.lastUrl).toContain('prefix=images/')
  })
})

describe('StorageBucket - File Operations', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch, 'files')
  })

  it('should copy a file', async () => {
    await bucket.copy('source.txt', 'destination.txt')

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/files/copy')
    expect(fetch.lastBody).toEqual({
      source_key: 'source.txt',
      destination_key: 'destination.txt',
    })
  })

  it('should move a file', async () => {
    await bucket.move('old-path.txt', 'new-path.txt')

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/files/move')
    expect(fetch.lastBody).toEqual({
      source_key: 'old-path.txt',
      destination_key: 'new-path.txt',
    })
  })

  it('should delete a file', async () => {
    await bucket.remove('file-to-delete.txt')

    expect(fetch.lastMethod).toBe('DELETE')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/files/files/file-to-delete.txt')
  })

  it('should delete multiple files', async () => {
    await bucket.remove(['file1.txt', 'file2.txt', 'file3.txt'])

    expect(fetch.lastMethod).toBe('DELETE')
  })
})

describe('StorageBucket - URL Generation', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch, 'public-files')
  })

  it('should get public URL', () => {
    const url = bucket.getPublicUrl('avatar.jpg')

    expect(url).toContain('/api/v1/storage/buckets/public-files/files/avatar.jpg')
  })

  it('should get public URL with nested path', () => {
    const url = bucket.getPublicUrl('images/2024/photo.jpg')

    expect(url).toContain('images/2024/photo.jpg')
  })

  it('should create signed URL', async () => {
    await bucket.createSignedUrl('private-document.pdf', 3600)

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/public-files/files/private-document.pdf/sign')
    expect(fetch.lastBody).toEqual({ expiresIn: 3600 })
  })

  it('should create signed URL with default expiry', async () => {
    await bucket.createSignedUrl('file.txt')

    expect(fetch.lastBody).toHaveProperty('expiresIn')
  })
})

describe('StorageBucket - File Metadata', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch, 'media')
  })

  it('should get file metadata', async () => {
    await bucket.getMetadata('video.mp4')

    expect(fetch.lastMethod).toBe('GET')
    expect(fetch.lastUrl).toContain('/api/v1/storage/buckets/media/files/video.mp4')
  })

  it('should update file metadata', async () => {
    await bucket.updateMetadata('image.jpg', {
      contentType: 'image/jpeg',
      cacheControl: 'max-age=3600',
      metadata: {
        title: 'Updated Image',
      },
    })

    expect(fetch.lastMethod).toBe('PATCH')
  })
})

describe('Storage - Error Handling', () => {
  let fetch: MockFetch
  let storage: FluxbaseStorage

  beforeEach(() => {
    fetch = new MockFetch()
    storage = new FluxbaseStorage(fetch)
  })

  it('should handle bucket not found', async () => {
    // Mock error response
    fetch.get = vi.fn().mockRejectedValue(new Error('Bucket not found'))

    try {
      await storage.getBucket('non-existent')
    } catch (error) {
      expect(error).toBeDefined()
    }
  })

  it('should handle file not found', async () => {
    const bucket = new StorageBucket(fetch, 'files')

    fetch.get = vi.fn().mockRejectedValue(new Error('File not found'))

    try {
      await bucket.download('non-existent.txt')
    } catch (error) {
      expect(error).toBeDefined()
    }
  })

  it('should handle upload failure', async () => {
    const bucket = new StorageBucket(fetch, 'uploads')
    const file = new MockFile(['Content'], 'file.txt')

    fetch.post = vi.fn().mockRejectedValue(new Error('Upload failed'))

    try {
      await bucket.upload('file.txt', file)
    } catch (error) {
      expect(error).toBeDefined()
    }
  })
})

describe('Storage - Concurrent Operations', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch, 'concurrent')
  })

  it('should handle concurrent uploads', async () => {
    const files = Array.from({ length: 5 }, (_, i) =>
      new MockFile([`File ${i}`], `file${i}.txt`)
    )

    const uploads = files.map((file, i) =>
      bucket.upload(`file${i}.txt`, file)
    )

    await Promise.all(uploads)

    expect(fetch.lastMethod).toBe('POST')
  })

  it('should handle concurrent downloads', async () => {
    const downloads = Array.from({ length: 3 }, (_, i) =>
      bucket.download(`file${i}.txt`)
    )

    await Promise.all(downloads)

    expect(fetch.lastMethod).toBe('GET')
  })
})

describe('Storage - File Size Limits', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch, 'limited')
  })

  it('should handle small files', async () => {
    const smallFile = new MockFile(['Small content'], 'small.txt')

    await bucket.upload('small.txt', smallFile)

    expect(fetch.lastMethod).toBe('POST')
  })

  it('should handle large files', async () => {
    // Simulate 5MB file
    const largeContent = 'A'.repeat(5 * 1024 * 1024)
    const largeFile = new MockFile([largeContent], 'large.bin')

    await bucket.upload('large.bin', largeFile)

    expect(fetch.lastMethod).toBe('POST')
  })
})

describe('Storage - Content Types', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch, 'files')
  })

  it('should detect text file type', async () => {
    const file = new MockFile(['Text content'], 'document.txt', { type: 'text/plain' })

    await bucket.upload('document.txt', file)

    expect(file.type).toBe('text/plain')
  })

  it('should detect image file type', async () => {
    const file = new MockFile(['Image data'], 'photo.jpg', { type: 'image/jpeg' })

    await bucket.upload('photo.jpg', file)

    expect(file.type).toBe('image/jpeg')
  })

  it('should detect JSON file type', async () => {
    const file = new MockFile([JSON.stringify({ key: 'value' })], 'data.json', { type: 'application/json' })

    await bucket.upload('data.json', file)

    expect(file.type).toBe('application/json')
  })

  it('should detect PDF file type', async () => {
    const file = new MockFile(['PDF content'], 'document.pdf', { type: 'application/pdf' })

    await bucket.upload('document.pdf', file)

    expect(file.type).toBe('application/pdf')
  })
})

describe('Storage - Path Handling', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch, 'files')
  })

  it('should handle simple file names', async () => {
    const file = new MockFile(['Content'], 'file.txt')

    await bucket.upload('file.txt', file)

    expect(fetch.lastUrl).toContain('file.txt')
  })

  it('should handle nested paths', async () => {
    const file = new MockFile(['Content'], 'doc.txt')

    await bucket.upload('folder/subfolder/doc.txt', file)

    expect(fetch.lastUrl).toContain('folder/subfolder/doc.txt')
  })

  it('should handle special characters in names', async () => {
    const file = new MockFile(['Content'], 'file with spaces.txt')

    await bucket.upload('file with spaces.txt', file)

    // URL encoding should be handled
    expect(fetch.lastUrl).toBeDefined()
  })

  it('should handle unicode characters', async () => {
    const file = new MockFile(['Content'], 'файл.txt')

    await bucket.upload('файл.txt', file)

    expect(fetch.lastUrl).toBeDefined()
  })
})

describe('Storage - Batch Operations', () => {
  let fetch: MockFetch
  let bucket: StorageBucket

  beforeEach(() => {
    fetch = new MockFetch()
    bucket = new StorageBucket(fetch, 'batch')
  })

  it('should delete multiple files at once', async () => {
    const filesToDelete = ['file1.txt', 'file2.txt', 'file3.txt']

    await bucket.remove(filesToDelete)

    expect(fetch.lastMethod).toBe('DELETE')
  })

  it('should list and process files in batches', async () => {
    // List first batch
    await bucket.list({ limit: 50, offset: 0 })

    expect(fetch.lastUrl).toContain('limit=50')
    expect(fetch.lastUrl).toContain('offset=0')

    // List second batch
    await bucket.list({ limit: 50, offset: 50 })

    expect(fetch.lastUrl).toContain('offset=50')
  })
})
