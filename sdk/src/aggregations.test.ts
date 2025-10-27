/**
 * Tests for aggregation and batch operations
 */

import { describe, it, expect } from 'vitest'
import { QueryBuilder } from './query-builder'
import { FluxbaseFetch } from './fetch'

// Mock FluxbaseFetch
class MockFetch extends FluxbaseFetch {
  constructor() {
    super('http://localhost:8080', {})
  }

  // Override to capture the URL being called
  lastUrl: string = ''

  async get<T>(path: string): Promise<T> {
    this.lastUrl = path
    return [] as T
  }
}

describe('QueryBuilder Aggregations', () => {
  it('should build count query', () => {
    const fetch = new MockFetch()
    const builder = new QueryBuilder(fetch, 'products')

    builder.count('*').execute()

    expect(fetch.lastUrl).toContain('select=count')
    expect(fetch.lastUrl).toContain('*')
  })

  it('should build count with specific column', () => {
    const fetch = new MockFetch()
    const builder = new QueryBuilder(fetch, 'products')

    builder.count('id').execute()

    expect(fetch.lastUrl).toContain('select=count')
    expect(fetch.lastUrl).toContain('id')
  })

  it('should build sum query', () => {
    const fetch = new MockFetch()
    const builder = new QueryBuilder(fetch, 'products')

    builder.sum('price').execute()

    expect(fetch.lastUrl).toContain('select=sum')
    expect(fetch.lastUrl).toContain('price')
  })

  it('should build avg query', () => {
    const fetch = new MockFetch()
    const builder = new QueryBuilder(fetch, 'products')

    builder.avg('price').execute()

    expect(fetch.lastUrl).toContain('select=avg')
    expect(fetch.lastUrl).toContain('price')
  })

  it('should build min query', () => {
    const fetch = new MockFetch()
    const builder = new QueryBuilder(fetch, 'products')

    builder.min('price').execute()

    expect(fetch.lastUrl).toContain('select=min')
    expect(fetch.lastUrl).toContain('price')
  })

  it('should build max query', () => {
    const fetch = new MockFetch()
    const builder = new QueryBuilder(fetch, 'products')

    builder.max('price').execute()

    expect(fetch.lastUrl).toContain('select=max')
    expect(fetch.lastUrl).toContain('price')
  })

  it('should build group by query', () => {
    const fetch = new MockFetch()
    const builder = new QueryBuilder(fetch, 'products')

    builder.count('*').groupBy('category').execute()

    expect(fetch.lastUrl).toContain('select=count')
    expect(fetch.lastUrl).toContain('group_by=category')
  })

  it('should build group by with multiple columns', () => {
    const fetch = new MockFetch()
    const builder = new QueryBuilder(fetch, 'products')

    builder.count('*').groupBy(['category', 'status']).execute()

    expect(fetch.lastUrl).toContain('group_by=category')
    expect(fetch.lastUrl).toContain('status')
  })

  it('should combine aggregation with filters', () => {
    const fetch = new MockFetch()
    const builder = new QueryBuilder(fetch, 'products')

    builder.count('*').eq('active', true).groupBy('category').execute()

    expect(fetch.lastUrl).toContain('select=count')
    expect(fetch.lastUrl).toContain('active=eq.true')
    expect(fetch.lastUrl).toContain('group_by=category')
  })
})

describe('QueryBuilder Batch Operations', () => {
  it('should have insertMany alias', async () => {
    const fetch = new MockFetch()
    const builder = new QueryBuilder(fetch, 'products')

    // Mock post method
    fetch.post = async (path: string, body: unknown) => {
      expect(path).toBe('/api/v1/tables/products')
      expect(Array.isArray(body)).toBe(true)
      return [] as any
    }

    await builder.insertMany([{ name: 'Product 1' }, { name: 'Product 2' }])
  })

  it('should have updateMany alias', async () => {
    const fetch = new MockFetch()
    const builder = new QueryBuilder(fetch, 'products')

    // Mock patch method
    fetch.patch = async (path: string, body: unknown) => {
      expect(path).toContain('/api/v1/tables/products')
      expect(body).toEqual({ discount: 10 })
      return [] as any
    }

    await builder.eq('category', 'electronics').updateMany({ discount: 10 })
  })

  it('should have deleteMany alias', async () => {
    const fetch = new MockFetch()
    const builder = new QueryBuilder(fetch, 'products')

    // Mock delete method
    fetch.delete = async (path: string) => {
      expect(path).toContain('/api/v1/tables/products')
      expect(path).toContain('active=eq.false')
      return undefined as any
    }

    await builder.eq('active', false).deleteMany()
  })
})
