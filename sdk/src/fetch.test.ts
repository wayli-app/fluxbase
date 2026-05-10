/**
 * Fetch Module Tests
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { FluxbaseFetch, type FetchOptions, type RefreshTokenCallback } from './fetch'

// Mock global fetch
const mockFetch = vi.fn()
global.fetch = mockFetch

describe('FluxbaseFetch', () => {
  let fluxFetch: FluxbaseFetch

  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
    fluxFetch = new FluxbaseFetch('http://localhost:8080')
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('constructor', () => {
    it('should initialize with base URL', () => {
      const fetch = new FluxbaseFetch('http://localhost:8080')
      expect(fetch).toBeDefined()
    })

    it('should remove trailing slash from base URL', async () => {
      const fetch = new FluxbaseFetch('http://localhost:8080/')

      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ data: 'test' }),
      })

      await fetch.get('/test')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/test',
        expect.anything()
      )
    })

    it('should accept custom headers', async () => {
      const fetch = new FluxbaseFetch('http://localhost:8080', {
        headers: { 'X-Custom': 'header' },
      })

      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({}),
      })

      await fetch.get('/test')

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            'X-Custom': 'header',
          }),
        })
      )
    })

    it('should accept custom timeout', () => {
      const fetch = new FluxbaseFetch('http://localhost:8080', {
        timeout: 60000,
      })
      expect(fetch).toBeDefined()
    })

    it('should accept debug flag', () => {
      const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {})

      const fetch = new FluxbaseFetch('http://localhost:8080', {
        debug: true,
      })

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ data: 'test' }),
      })

      fetch.get('/test')

      // Debug log should be called
      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })
  })

  describe('setAuthToken', () => {
    it('should set authorization header', async () => {
      fluxFetch.setAuthToken('my-token')

      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({}),
      })

      await fluxFetch.get('/test')

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': 'Bearer my-token',
          }),
        })
      )
    })

    it('should restore anon key when token is null and anon key is set', async () => {
      fluxFetch.setAnonKey('anon-key')
      fluxFetch.setAuthToken('user-token')
      fluxFetch.setAuthToken(null)

      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({}),
      })

      await fluxFetch.get('/test')

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': 'Bearer anon-key',
          }),
        })
      )
    })

    it('should remove authorization header when token is null and no anon key', async () => {
      fluxFetch.setAuthToken('token')
      fluxFetch.setAuthToken(null)

      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({}),
      })

      await fluxFetch.get('/test')

      const callArgs = mockFetch.mock.calls[0][1]
      expect(callArgs.headers['Authorization']).toBeUndefined()
    })
  })

  describe('request', () => {
    it('should make GET request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ data: 'test' }),
      })

      const result = await fluxFetch.request('/api/test', { method: 'GET' })

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/test',
        expect.objectContaining({
          method: 'GET',
        })
      )
      expect(result).toEqual({ data: 'test' })
    })

    it('should make POST request with JSON body', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ success: true }),
      })

      await fluxFetch.request('/api/test', {
        method: 'POST',
        body: { key: 'value' },
      })

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ key: 'value' }),
        })
      )
    })

    it('should handle FormData body', async () => {
      const formData = new FormData()
      formData.append('file', new Blob(['test']))

      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ uploaded: true }),
      })

      await fluxFetch.request('/api/upload', {
        method: 'POST',
        body: formData,
      })

      const callArgs = mockFetch.mock.calls[0][1]
      expect(callArgs.body).toBe(formData)
      // Content-Type should be omitted for FormData
      expect(callArgs.headers['Content-Type']).toBeUndefined()
    })

    it('should handle text response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: new Headers({ 'content-type': 'text/plain' }),
        text: async () => 'plain text response',
      })

      const result = await fluxFetch.request('/api/text', { method: 'GET' })

      expect(result).toBe('plain text response')
    })

    it('should throw error for non-ok response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ error: 'Resource not found' }),
      })

      await expect(fluxFetch.request('/api/missing', { method: 'GET' }))
        .rejects.toThrow('Resource not found')
    })

    it('should throw error with status text if no error in response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ data: 'no error field' }),
      })

      await expect(fluxFetch.request('/api/error', { method: 'GET' }))
        .rejects.toThrow('Internal Server Error')
    })

    it('should include custom headers in request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({}),
      })

      await fluxFetch.request('/api/test', {
        method: 'GET',
        headers: { 'X-Request-Header': 'custom' },
      })

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            'X-Request-Header': 'custom',
          }),
        })
      )
    })
  })

  describe('auto token refresh', () => {
    it('should call refresh callback on 401 and retry', async () => {
      const refreshCallback: RefreshTokenCallback = vi.fn().mockResolvedValue(true)
      fluxFetch.setRefreshTokenCallback(refreshCallback)

      // First call returns 401
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ error: 'Token expired' }),
      })

      // Retry succeeds
      mockFetch.mockResolvedValueOnce({
        ok: true,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ data: 'success' }),
      })

      const result = await fluxFetch.get('/api/protected')

      expect(refreshCallback).toHaveBeenCalled()
      expect(mockFetch).toHaveBeenCalledTimes(2)
      expect(result).toEqual({ data: 'success' })
    })

    it('should not refresh when skipAutoRefresh is true', async () => {
      const refreshCallback: RefreshTokenCallback = vi.fn().mockResolvedValue(true)
      fluxFetch.setRefreshTokenCallback(refreshCallback)

      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ error: 'Token expired' }),
      })

      await expect(fluxFetch.request('/api/auth', {
        method: 'POST',
        skipAutoRefresh: true,
      })).rejects.toThrow()

      expect(refreshCallback).not.toHaveBeenCalled()
    })

    it('should not retry when refresh fails', async () => {
      const refreshCallback: RefreshTokenCallback = vi.fn().mockResolvedValue(false)
      fluxFetch.setRefreshTokenCallback(refreshCallback)

      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ error: 'Token expired' }),
      })

      await expect(fluxFetch.get('/api/protected')).rejects.toThrow()

      expect(refreshCallback).toHaveBeenCalled()
      expect(mockFetch).toHaveBeenCalledTimes(1) // No retry
    })

    it('should deduplicate concurrent refresh requests', async () => {
      vi.useRealTimers() // Use real timers for async operations

      let refreshCount = 0
      const refreshCallback: RefreshTokenCallback = vi.fn().mockImplementation(async () => {
        refreshCount++
        await new Promise(resolve => setTimeout(resolve, 50))
        return true
      })
      fluxFetch.setRefreshTokenCallback(refreshCallback)

      // Both calls return 401
      mockFetch.mockResolvedValue({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ error: 'Token expired' }),
      })

      // Make concurrent requests
      const promise1 = fluxFetch.get('/api/endpoint1')
      const promise2 = fluxFetch.get('/api/endpoint2')

      // Both should fail, but refresh should only be called once due to deduplication
      await expect(promise1).rejects.toThrow()
      await expect(promise2).rejects.toThrow()

      expect(refreshCount).toBe(1)
    })

    it('should handle refresh callback throwing error', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      const debugFetch = new FluxbaseFetch('http://localhost:8080', { debug: true })

      const refreshCallback: RefreshTokenCallback = vi.fn().mockRejectedValue(new Error('Refresh failed'))
      debugFetch.setRefreshTokenCallback(refreshCallback)

      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ error: 'Token expired' }),
      })

      await expect(debugFetch.get('/api/protected')).rejects.toThrow()

      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })

    it('should clear refresh callback when set to null', async () => {
      const refreshCallback: RefreshTokenCallback = vi.fn().mockResolvedValue(true)
      fluxFetch.setRefreshTokenCallback(refreshCallback)
      fluxFetch.setRefreshTokenCallback(null)

      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ error: 'Token expired' }),
      })

      await expect(fluxFetch.get('/api/protected')).rejects.toThrow()

      expect(refreshCallback).not.toHaveBeenCalled()
    })
  })

  describe('HTTP method shortcuts', () => {
    beforeEach(() => {
      mockFetch.mockResolvedValue({
        ok: true,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ success: true }),
      })
    })

    it('should make GET request', async () => {
      await fluxFetch.get('/api/resource')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/resource',
        expect.objectContaining({ method: 'GET' })
      )
    })

    it('should make POST request', async () => {
      await fluxFetch.post('/api/resource', { data: 'test' })

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/resource',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ data: 'test' }),
        })
      )
    })

    it('should make PUT request', async () => {
      await fluxFetch.put('/api/resource/1', { data: 'updated' })

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/resource/1',
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify({ data: 'updated' }),
        })
      )
    })

    it('should make PATCH request', async () => {
      await fluxFetch.patch('/api/resource/1', { field: 'value' })

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/resource/1',
        expect.objectContaining({
          method: 'PATCH',
          body: JSON.stringify({ field: 'value' }),
        })
      )
    })

    it('should make DELETE request', async () => {
      await fluxFetch.delete('/api/resource/1')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/resource/1',
        expect.objectContaining({ method: 'DELETE' })
      )
    })
  })

  describe('getWithHeaders', () => {
    it('should return data with headers', async () => {
      const responseHeaders = new Headers({
        'content-type': 'application/json',
        'x-total-count': '100',
      })

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: responseHeaders,
        json: async () => [{ id: 1 }],
      })

      const result = await fluxFetch.getWithHeaders('/api/items')

      expect(result.data).toEqual([{ id: 1 }])
      expect(result.headers.get('x-total-count')).toBe('100')
      expect(result.status).toBe(200)
    })
  })

  describe('postWithHeaders', () => {
    it('should return data with headers for POST', async () => {
      const responseHeaders = new Headers({
        'content-type': 'application/json',
        'location': '/api/items/new-id',
      })

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 201,
        headers: responseHeaders,
        json: async () => ({ id: 'new-id' }),
      })

      const result = await fluxFetch.postWithHeaders('/api/items', { name: 'Test' })

      expect(result.data).toEqual({ id: 'new-id' })
      expect(result.headers.get('location')).toBe('/api/items/new-id')
      expect(result.status).toBe(201)
    })
  })

  describe('head', () => {
    it('should make HEAD request and return headers', async () => {
      const responseHeaders = new Headers({
        'content-length': '1234',
        'last-modified': 'Wed, 01 Jan 2025 00:00:00 GMT',
      })

      mockFetch.mockResolvedValueOnce({
        headers: responseHeaders,
      })

      const headers = await fluxFetch.head('/api/resource')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/resource',
        expect.objectContaining({ method: 'HEAD' })
      )
      expect(headers.get('content-length')).toBe('1234')
    })
  })

  describe('getBlob', () => {
    it('should download file as blob', async () => {
      const mockBlob = new Blob(['file content'], { type: 'text/plain' })

      mockFetch.mockResolvedValueOnce({
        ok: true,
        blob: async () => mockBlob,
      })

      const blob = await fluxFetch.getBlob('/api/files/doc.txt')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/files/doc.txt',
        expect.objectContaining({ method: 'GET' })
      )
      expect(blob).toBe(mockBlob)
    })

    it('should throw on non-ok response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
      })

      await expect(fluxFetch.getBlob('/api/files/missing.txt'))
        .rejects.toThrow('Not Found')
    })

    it('should log debug message when debug enabled', async () => {
      const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {})
      const debugFetch = new FluxbaseFetch('http://localhost:8080', { debug: true })

      const mockBlob = new Blob(['content'])
      mockFetch.mockResolvedValueOnce({
        ok: true,
        blob: async () => mockBlob,
      })

      await debugFetch.getBlob('/api/files/test.txt')

      expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining('GET (blob)'))
      consoleSpy.mockRestore()
    })

    it('should handle AbortError (timeout)', async () => {
      const abortError = new Error('Aborted')
      abortError.name = 'AbortError'
      mockFetch.mockRejectedValueOnce(abortError)

      await expect(fluxFetch.getBlob('/api/files/large.bin'))
        .rejects.toThrow('Request timeout')
    })

    it('should handle non-Error exceptions', async () => {
      mockFetch.mockRejectedValueOnce('string error')

      await expect(fluxFetch.getBlob('/api/files/test.txt'))
        .rejects.toThrow('string error')
    })
  })

  describe('error handling', () => {
    it('should handle AbortError', async () => {
      const abortError = new Error('Aborted')
      abortError.name = 'AbortError'
      mockFetch.mockRejectedValueOnce(abortError)

      await expect(fluxFetch.get('/api/test'))
        .rejects.toThrow('Request timeout')
    })

    it('should rethrow other errors', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'))

      await expect(fluxFetch.get('/api/test'))
        .rejects.toThrow('Network error')
    })

    it('should handle non-Error exceptions', async () => {
      mockFetch.mockRejectedValueOnce('string error')

      await expect(fluxFetch.get('/api/test'))
        .rejects.toThrow('string error')
    })

    it('should include status and details on error', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ error: 'Invalid input', details: { field: 'name' } }),
      })

      try {
        await fluxFetch.get('/api/test')
        expect.fail('Should have thrown')
      } catch (err: any) {
        expect(err.status).toBe(400)
        expect(err.details).toEqual({ error: 'Invalid input', details: { field: 'name' } })
      }
    })
  })

  describe('requestWithHeaders auto refresh', () => {
    it('should retry requestWithHeaders on successful token refresh', async () => {
      const refreshCallback: RefreshTokenCallback = vi.fn().mockResolvedValue(true)
      fluxFetch.setRefreshTokenCallback(refreshCallback)

      const responseHeaders = new Headers({
        'content-type': 'application/json',
        'x-custom': 'header',
      })

      // First call returns 401
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ error: 'Token expired' }),
      })

      // Retry succeeds
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: responseHeaders,
        json: async () => ({ data: 'success' }),
      })

      const result = await fluxFetch.getWithHeaders('/api/protected')

      expect(refreshCallback).toHaveBeenCalled()
      expect(mockFetch).toHaveBeenCalledTimes(2)
      expect(result.data).toEqual({ data: 'success' })
    })

    it('should handle FormData body in requestWithHeaders', async () => {
      const formData = new FormData()
      formData.append('file', new Blob(['test']))

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ uploaded: true }),
      })

      const result = await fluxFetch.postWithHeaders('/api/upload', formData)

      const callArgs = mockFetch.mock.calls[0][1]
      expect(callArgs.body).toBe(formData)
      expect(callArgs.headers['Content-Type']).toBeUndefined()
      expect(result.data).toEqual({ uploaded: true })
    })

    it('should handle text response in requestWithHeaders', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-type': 'text/plain' }),
        text: async () => 'plain text response',
      })

      const result = await fluxFetch.getWithHeaders('/api/text')

      expect(result.data).toBe('plain text response')
      expect(result.status).toBe(200)
    })

    it('should handle AbortError in requestWithHeaders', async () => {
      const abortError = new Error('Aborted')
      abortError.name = 'AbortError'
      mockFetch.mockRejectedValueOnce(abortError)

      await expect(fluxFetch.getWithHeaders('/api/test'))
        .rejects.toThrow('Request timeout')
    })

    it('should handle non-Error exceptions in requestWithHeaders', async () => {
      mockFetch.mockRejectedValueOnce('string error')

      await expect(fluxFetch.getWithHeaders('/api/test'))
        .rejects.toThrow('string error')
    })

    it('should not refresh when skipAutoRefresh is true in requestWithHeaders', async () => {
      const refreshCallback: RefreshTokenCallback = vi.fn().mockResolvedValue(true)
      fluxFetch.setRefreshTokenCallback(refreshCallback)

      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ error: 'Token expired' }),
      })

      await expect(fluxFetch.requestWithHeaders('/api/auth', {
        method: 'POST',
        skipAutoRefresh: true,
      })).rejects.toThrow()

      expect(refreshCallback).not.toHaveBeenCalled()
    })

    it('should log debug messages in requestWithHeaders', async () => {
      const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {})
      const debugFetch = new FluxbaseFetch('http://localhost:8080', { debug: true })

      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ data: 'test' }),
      })

      await debugFetch.getWithHeaders('/api/test')

      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })

    it('should include error status and details in requestWithHeaders', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => ({ error: 'Invalid input', details: { field: 'name' } }),
      })

      try {
        await fluxFetch.getWithHeaders('/api/test')
        expect.fail('Should have thrown')
      } catch (err: any) {
        expect(err.status).toBe(400)
        expect(err.details).toEqual({ error: 'Invalid input', details: { field: 'name' } })
      }
    })
  })
})
