/**
 * Functions Service Tests
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { FluxbaseFunctions } from './functions'
import type { FluxbaseFetch } from './fetch'

// Mock FluxbaseFetch
class MockFetch implements Partial<FluxbaseFetch> {
  public lastUrl: string = ''
  public lastMethod: string = ''
  public lastBody: unknown = null
  public lastHeaders: Record<string, string> = {}
  public mockResponse: any = null

  async get<T>(path: string, options?: { headers?: Record<string, string> }): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'GET'
    this.lastHeaders = options?.headers || {}
    return this.mockResponse as T
  }

  async post<T>(path: string, body?: unknown, options?: { headers?: Record<string, string> }): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'POST'
    this.lastBody = body
    this.lastHeaders = options?.headers || {}
    return this.mockResponse as T
  }

  async put<T>(path: string, body?: unknown, options?: { headers?: Record<string, string> }): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'PUT'
    this.lastBody = body
    this.lastHeaders = options?.headers || {}
    return this.mockResponse as T
  }

  async patch<T>(path: string, body?: unknown, options?: { headers?: Record<string, string> }): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'PATCH'
    this.lastBody = body
    this.lastHeaders = options?.headers || {}
    return this.mockResponse as T
  }

  async delete<T>(path: string, options?: { headers?: Record<string, string> }): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'DELETE'
    this.lastHeaders = options?.headers || {}
    return this.mockResponse as T
  }

  setAuthToken(token: string | null): void {}
}

describe('FluxbaseFunctions', () => {
  let mockFetch: MockFetch
  let functions: FluxbaseFunctions

  beforeEach(() => {
    mockFetch = new MockFetch()
    functions = new FluxbaseFunctions(mockFetch as FluxbaseFetch)
  })

  describe('invoke', () => {
    it('should invoke a function with POST by default', async () => {
      mockFetch.mockResponse = { result: 'success' }

      const { data, error } = await functions.invoke('test-function', {
        body: { key: 'value' }
      })

      expect(mockFetch.lastUrl).toBe('/api/v1/functions/test-function/invoke')
      expect(mockFetch.lastMethod).toBe('POST')
      expect(mockFetch.lastBody).toEqual({ key: 'value' })
      expect(data).toEqual({ result: 'success' })
      expect(error).toBeNull()
    })

    it('should support GET method', async () => {
      mockFetch.mockResponse = { data: 'test' }

      const { data, error } = await functions.invoke('get-function', {
        method: 'GET'
      })

      expect(mockFetch.lastMethod).toBe('GET')
      expect(data).toEqual({ data: 'test' })
      expect(error).toBeNull()
    })

    it('should support custom headers', async () => {
      mockFetch.mockResponse = { ok: true }

      await functions.invoke('test-function', {
        body: { test: true },
        headers: {
          'X-Custom-Header': 'custom-value'
        }
      })

      expect(mockFetch.lastHeaders).toEqual({
        'X-Custom-Header': 'custom-value'
      })
    })

    it('should support all HTTP methods', async () => {
      mockFetch.mockResponse = { ok: true }

      const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'] as const

      for (const method of methods) {
        await functions.invoke('test', { method, body: {} })
        expect(mockFetch.lastMethod).toBe(method)
      }
    })

    it('should handle errors', async () => {
      mockFetch.post = async () => {
        throw new Error('Function not found')
      }

      const { data, error } = await functions.invoke('non-existent')

      expect(data).toBeNull()
      expect(error).toBeDefined()
      expect(error?.message).toBe('Function not found')
    })

    it('should support namespace parameter', async () => {
      mockFetch.mockResponse = { result: 'namespaced' }

      const { data, error } = await functions.invoke('test-function', {
        body: { key: 'value' },
        namespace: 'my-app'
      })

      expect(mockFetch.lastUrl).toContain('namespace=my-app')
      expect(error).toBeNull()
    })
  })

  describe('list', () => {
    it('should list functions', async () => {
      const mockFunctions = [
        { id: '1', name: 'func1', version: 1 },
        { id: '2', name: 'func2', version: 1 }
      ]
      mockFetch.mockResponse = mockFunctions

      const { data, error } = await functions.list()

      expect(mockFetch.lastUrl).toBe('/api/v1/functions')
      expect(mockFetch.lastMethod).toBe('GET')
      expect(data).toEqual(mockFunctions)
      expect(error).toBeNull()
    })

    it('should handle list errors', async () => {
      mockFetch.get = async () => {
        throw new Error('Permission denied')
      }

      const { data, error } = await functions.list()

      expect(data).toBeNull()
      expect(error).toBeDefined()
      expect(error?.message).toBe('Permission denied')
    })
  })

  describe('get', () => {
    it('should get a function by name', async () => {
      const mockFunction = { id: '1', name: 'test-func', version: 1 }
      mockFetch.mockResponse = mockFunction

      const { data, error } = await functions.get('test-func')

      expect(mockFetch.lastUrl).toBe('/api/v1/functions/test-func')
      expect(mockFetch.lastMethod).toBe('GET')
      expect(data).toEqual(mockFunction)
      expect(error).toBeNull()
    })

    it('should handle get errors', async () => {
      mockFetch.get = async () => {
        throw new Error('Function not found')
      }

      const { data, error } = await functions.get('non-existent')

      expect(data).toBeNull()
      expect(error).toBeDefined()
      expect(error?.message).toBe('Function not found')
    })
  })
})
