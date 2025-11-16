/**
 * Comprehensive Query Builder Tests
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { QueryBuilder } from './query-builder'
import type { FluxbaseFetch } from './fetch'

// Mock FluxbaseFetch
class MockFetch implements FluxbaseFetch {
  public lastUrl: string = ''
  public lastMethod: string = ''
  public lastBody: unknown = null
  public lastHeaders: Record<string, string> = {}
  public mockResponse: unknown = []
  public mockError: Error | null = null

  constructor(public baseUrl: string = 'http://localhost:8080', public headers: Record<string, string> = {}) {}

  async get<T>(path: string): Promise<T> {
    this.lastUrl = path
    this.lastMethod = 'GET'
    if (this.mockError) {
      throw this.mockError
    }
    return this.mockResponse as T
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

describe('QueryBuilder - Select Operations', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'users')
  })

  it('should select all columns by default', async () => {
    await builder.execute()
    expect(fetch.lastUrl).toContain('select=*')
  })

  it('should select specific columns', async () => {
    await builder.select('id, name, email').execute()
    expect(fetch.lastUrl).toContain('select=id')
    expect(fetch.lastUrl).toContain('name')
    expect(fetch.lastUrl).toContain('email')
  })

  it('should select with aggregations', async () => {
    await builder.select('count, sum(price), avg(rating)').execute()
    expect(fetch.lastUrl).toContain('count')
    expect(fetch.lastUrl).toContain('sum')
    expect(fetch.lastUrl).toContain('avg')
  })
})

describe('QueryBuilder - Filter Operators', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'products')
  })

  it('should filter with eq (equals)', async () => {
    await builder.eq('price', 29.99).execute()
    expect(fetch.lastUrl).toContain('price=eq.29.99')
  })

  it('should filter with neq (not equals)', async () => {
    await builder.neq('status', 'deleted').execute()
    expect(fetch.lastUrl).toContain('status=neq.deleted')
  })

  it('should filter with gt (greater than)', async () => {
    await builder.gt('stock', 10).execute()
    expect(fetch.lastUrl).toContain('stock=gt.10')
  })

  it('should filter with gte (greater than or equal)', async () => {
    await builder.gte('price', 50).execute()
    expect(fetch.lastUrl).toContain('price=gte.50')
  })

  it('should filter with lt (less than)', async () => {
    await builder.lt('discount', 0.5).execute()
    expect(fetch.lastUrl).toContain('discount=lt.0.5')
  })

  it('should filter with lte (less than or equal)', async () => {
    await builder.lte('rating', 3).execute()
    expect(fetch.lastUrl).toContain('rating=lte.3')
  })

  it('should filter with like (pattern matching)', async () => {
    await builder.like('name', '%Product%').execute()
    expect(fetch.lastUrl).toContain('name=like.%25Product%25')
  })

  it('should filter with ilike (case-insensitive like)', async () => {
    await builder.ilike('email', '%@gmail.com').execute()
    expect(fetch.lastUrl).toContain('email=ilike')
  })

  it('should filter with in (list)', async () => {
    await builder.in('category', ['electronics', 'books', 'clothing']).execute()
    expect(fetch.lastUrl).toContain('category=in.')
  })

  it('should filter with is null', async () => {
    await builder.is('deleted_at', null).execute()
    expect(fetch.lastUrl).toContain('deleted_at=is.null')
  })

  it('should chain multiple filters', async () => {
    await builder
      .eq('status', 'active')
      .gte('price', 10)
      .lte('price', 100)
      .execute()

    expect(fetch.lastUrl).toContain('status=eq.active')
    expect(fetch.lastUrl).toContain('price=gte.10')
    expect(fetch.lastUrl).toContain('price=lte.100')
  })
})

describe('QueryBuilder - Ordering', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'posts')
  })

  it('should order by column ascending', async () => {
    await builder.order('created_at', { ascending: true }).execute()
    expect(fetch.lastUrl).toContain('order=created_at.asc')
  })

  it('should order by column descending', async () => {
    await builder.order('views', { ascending: false }).execute()
    expect(fetch.lastUrl).toContain('order=views.desc')
  })

  it('should support multiple order by', async () => {
    await builder
      .order('category', { ascending: true })
      .order('price', { ascending: false })
      .execute()

    expect(fetch.lastUrl).toContain('order=category.asc')
    expect(fetch.lastUrl).toContain('price.desc')
  })
})

describe('QueryBuilder - Pagination', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'articles')
  })

  it('should limit results', async () => {
    await builder.limit(10).execute()
    expect(fetch.lastUrl).toContain('limit=10')
  })

  it('should offset results', async () => {
    await builder.offset(20).execute()
    expect(fetch.lastUrl).toContain('offset=20')
  })

  it('should combine limit and offset', async () => {
    await builder.limit(10).offset(20).execute()
    expect(fetch.lastUrl).toContain('limit=10')
    expect(fetch.lastUrl).toContain('offset=20')
  })

  it('should support pagination pattern', async () => {
    const page = 3
    const pageSize = 25
    await builder.limit(pageSize).offset((page - 1) * pageSize).execute()

    expect(fetch.lastUrl).toContain('limit=25')
    expect(fetch.lastUrl).toContain('offset=50')
  })
})

describe('QueryBuilder - Insert Operations', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'users')
  })

  it('should insert a single row', async () => {
    const user = { name: 'John Doe', email: 'john@example.com' }
    await builder.insert(user)

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastUrl).toContain('/api/v1/tables/users')
    expect(fetch.lastBody).toEqual(user)
  })

  it('should insert multiple rows', async () => {
    const users = [
      { name: 'Alice', email: 'alice@example.com' },
      { name: 'Bob', email: 'bob@example.com' },
    ]
    await builder.insert(users)

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastBody).toEqual(users)
  })
})

describe('QueryBuilder - Upsert Operations', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'products')
  })

  it('should upsert with merge-duplicates header', async () => {
    const product = { id: 1, name: 'Product', price: 29.99 }
    await builder.upsert(product)

    expect(fetch.lastMethod).toBe('POST')
    expect(fetch.lastHeaders['Prefer']).toBe('resolution=merge-duplicates')
  })

  it('should upsert multiple rows', async () => {
    const products = [
      { id: 1, name: 'Product 1', price: 19.99 },
      { id: 2, name: 'Product 2', price: 29.99 },
    ]
    await builder.upsert(products)

    expect(fetch.lastBody).toEqual(products)
    expect(fetch.lastHeaders['Prefer']).toBe('resolution=merge-duplicates')
  })
})

describe('QueryBuilder - Update Operations', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'posts')
  })

  it('should update with filters', async () => {
    await builder.eq('id', 123).update({ title: 'Updated Title' })

    expect(fetch.lastMethod).toBe('PATCH')
    expect(fetch.lastUrl).toContain('id=eq.123')
    expect(fetch.lastBody).toEqual({ title: 'Updated Title' })
  })

  it('should update multiple fields', async () => {
    const updates = {
      title: 'New Title',
      content: 'New Content',
      updated_at: new Date().toISOString(),
    }

    await builder.eq('status', 'draft').update(updates)

    expect(fetch.lastUrl).toContain('status=eq.draft')
    expect(fetch.lastBody).toEqual(updates)
  })
})

describe('QueryBuilder - Delete Operations', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'comments')
  })

  it('should delete with filters', async () => {
    await builder.eq('id', 456).delete()

    expect(fetch.lastMethod).toBe('DELETE')
    expect(fetch.lastUrl).toContain('id=eq.456')
  })

  it('should delete multiple rows', async () => {
    await builder.eq('spam', true).delete()

    expect(fetch.lastMethod).toBe('DELETE')
    expect(fetch.lastUrl).toContain('spam=eq.true')
  })
})

describe('QueryBuilder - Single Row', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'profiles')
  })

  it('should fetch single row', async () => {
    await builder.eq('user_id', 'abc-123').single().execute()

    expect(fetch.lastUrl).toContain('user_id=eq.abc-123')
    expect(fetch.lastUrl).toContain('limit=1')
  })
})

describe('QueryBuilder - Complex Queries', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'orders')
  })

  it('should build complex query with multiple operations', async () => {
    await builder
      .select('id, total, status, user(name, email)')
      .eq('status', 'pending')
      .gte('total', 100)
      .order('created_at', { ascending: false })
      .limit(20)
      .offset(0)
      .execute()

    const url = fetch.lastUrl
    expect(url).toContain('select=')
    expect(url).toContain('status=eq.pending')
    expect(url).toContain('total=gte.100')
    expect(url).toContain('order=created_at.desc')
    expect(url).toContain('limit=20')
    expect(url).toContain('offset=0')
  })

  it('should support filtering with JSON operators', async () => {
    await builder.eq('metadata->theme', 'dark').execute()

    expect(fetch.lastUrl).toContain('metadata')
    expect(fetch.lastUrl).toContain('theme')
  })
})

describe('QueryBuilder - Group By', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'sales')
  })

  it('should group by single column', async () => {
    await builder.select('category, count').groupBy('category').execute()

    expect(fetch.lastUrl).toContain('group_by=category')
  })

  it('should group by multiple columns', async () => {
    await builder.select('category, status, count').groupBy('category,status').execute()

    expect(fetch.lastUrl).toContain('group_by=')
    expect(fetch.lastUrl).toContain('category')
  })
})

describe('QueryBuilder - Text Search', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'documents')
  })

  it('should perform full-text search', async () => {
    await builder.textSearch('content', 'search terms').execute()

    expect(fetch.lastUrl).toContain('content=fts')
  })
})

describe('QueryBuilder - Batch Operations', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'tasks')
  })

  it('should batch insert', async () => {
    const tasks = [
      { title: 'Task 1', completed: false },
      { title: 'Task 2', completed: false },
      { title: 'Task 3', completed: false },
    ]

    await builder.insert(tasks)

    expect(fetch.lastBody).toEqual(tasks)
    expect(Array.isArray(fetch.lastBody)).toBe(true)
  })

  it('should batch update with filters', async () => {
    await builder.eq('status', 'pending').update({ status: 'completed' })

    expect(fetch.lastUrl).toContain('status=eq.pending')
    expect(fetch.lastBody).toEqual({ status: 'completed' })
  })

  it('should batch delete with filters', async () => {
    await builder.lt('created_at', '2023-01-01').delete()

    expect(fetch.lastUrl).toContain('created_at=lt.2023-01-01')
  })
})

describe('QueryBuilder - Error Handling', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'items')
  })

  it('should handle empty filters gracefully', async () => {
    await builder.execute()

    expect(fetch.lastUrl).toContain('/api/v1/tables/items')
    expect(fetch.lastUrl).toContain('select=*')
  })

  it('should handle undefined values', async () => {
    await builder.eq('name', undefined as any).execute()

    // Should still build URL even with undefined
    expect(fetch.lastUrl).toContain('name=eq')
  })

  it('should handle null values', async () => {
    await builder.is('deleted_at', null).execute()

    expect(fetch.lastUrl).toContain('deleted_at=is.null')
  })
})

describe('QueryBuilder - Advanced Features', () => {
  let fetch: MockFetch
  let builder: QueryBuilder

  beforeEach(() => {
    fetch = new MockFetch()
    builder = new QueryBuilder(fetch, 'analytics')
  })

  it('should support NOT operator', async () => {
    await builder.not('status', 'eq', 'deleted').execute()

    expect(fetch.lastUrl).toContain('status=not.eq.deleted')
  })

  it('should support OR operator', async () => {
    await builder.or('status.eq.active,status.eq.pending').execute()

    expect(fetch.lastUrl).toContain('or=')
    expect(fetch.lastUrl).toContain('status')
  })

  it('should chain complex filters', async () => {
    await builder
      .select('*')
      .or('status.eq.active,priority.eq.high')
      .gte('score', 80)
      .order('created_at', { ascending: false })
      .limit(50)
      .execute()

    const url = fetch.lastUrl
    expect(url).toContain('or=')
    expect(url).toContain('score=gte.80')
    expect(url).toContain('limit=50')
  })

  it('should support match() for multiple exact matches', async () => {
    await builder.match({ id: 1, status: 'active', role: 'admin' }).execute()

    const url = fetch.lastUrl
    expect(url).toContain('id=eq.1')
    expect(url).toContain('status=eq.active')
    expect(url).toContain('role=eq.admin')
  })

  it('should support filter() generic method', async () => {
    await builder.filter('age', 'gte', '18').execute()

    expect(fetch.lastUrl).toContain('age=gte.18')
  })

  it('should support containedBy() for arrays', async () => {
    await builder.containedBy('tags', '["news","update"]').execute()

    expect(fetch.lastUrl).toContain('tags=cd.')
  })

  it('should support overlaps() for arrays', async () => {
    await builder.overlaps('tags', '["news","sports"]').execute()

    expect(fetch.lastUrl).toContain('tags=ov.')
  })

  it('should support and() operator for grouped conditions', async () => {
    await builder.and('status.eq.active,verified.eq.true').execute()

    const url = fetch.lastUrl
    expect(url).toContain('and=')
    expect(url).toContain('status.eq.active')
    expect(url).toContain('verified.eq.true')
  })

  it('should support maybeSingle() returning null for no results', async () => {
    fetch.mockResponse = []

    const { data, error } = await builder.eq('id', 999).maybeSingle().execute()

    expect(data).toBeNull()
    expect(error).toBeNull()
  })

  it('should support maybeSingle() returning single row', async () => {
    const mockUser = { id: 1, name: 'Alice' }
    fetch.mockResponse = [mockUser]

    const { data, error } = await builder.eq('id', 1).maybeSingle().execute()

    expect(data).toEqual(mockUser)
    expect(error).toBeNull()
  })

  it('should support throwOnError() returning data directly', async () => {
    const mockUsers = [{ id: 1, name: 'Alice' }, { id: 2, name: 'Bob' }]
    fetch.mockResponse = mockUsers

    const data = await builder.throwOnError()

    expect(data).toEqual(mockUsers)
  })

  it('should support throwOnError() throwing on error', async () => {
    fetch.mockError = new Error('Network error')

    await expect(builder.throwOnError()).rejects.toThrow('Network error')
  })

  it('should support upsert() with onConflict option', async () => {
    await builder.upsert({ id: 1, email: 'alice@example.com' }, { onConflict: 'email' })

    const url = fetch.lastUrl
    expect(url).toContain('on_conflict=email')
    expect(fetch.lastHeaders?.Prefer).toContain('resolution=merge-duplicates')
  })

  it('should support upsert() with ignoreDuplicates option', async () => {
    await builder.upsert({ id: 1, email: 'alice@example.com' }, { ignoreDuplicates: true })

    expect(fetch.lastHeaders?.Prefer).toContain('resolution=ignore-duplicates')
  })

  it('should support upsert() with defaultToNull option', async () => {
    await builder.upsert({ id: 1, name: 'Alice' }, { defaultToNull: true })

    expect(fetch.lastHeaders?.Prefer).toContain('missing=default')
  })
})

describe('QueryBuilder - RPC (Remote Procedure Call)', () => {
  let fetch: MockFetch

  beforeEach(() => {
    fetch = new MockFetch()
  })

  it('should call RPC function', async () => {
    await fetch.post('/api/v1/rpc/calculate_total', { order_id: 123 })

    expect(fetch.lastUrl).toContain('/api/v1/rpc/calculate_total')
    expect(fetch.lastBody).toEqual({ order_id: 123 })
  })

  it('should call RPC with no parameters', async () => {
    await fetch.post('/api/v1/rpc/get_stats')

    expect(fetch.lastUrl).toContain('/api/v1/rpc/get_stats')
  })
})
