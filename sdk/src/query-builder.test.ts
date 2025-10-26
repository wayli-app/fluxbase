/**
 * Query Builder Tests
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { QueryBuilder } from './query-builder'
import type { FluxbaseFetch } from './fetch'

describe('QueryBuilder', () => {
  let mockFetch: FluxbaseFetch
  let builder: QueryBuilder<any>

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    } as unknown as FluxbaseFetch

    builder = new QueryBuilder(mockFetch, 'users)))
  })

  describe('select()', () => {
    it('should build select query with all columns', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.select('*').execute()

      expect(mockFetch.get).toHaveBeenCalledWith('/api/tables/users)))
    })

    it('should build select query with specific columns', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.select('id,name,email').execute()

      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('select=id%2Cname%2Cemail)))
    })
  })

  describe('filters', () => {
    it('should build eq filter', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.eq('status', 'active').execute()

      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('status=eq.active)))
    })

    it('should build neq filter', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.neq('status', 'deleted').execute()

      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('status=neq.deleted)))
    })

    it('should build gt/gte filters', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.gt('age', 18).execute()
      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('age=gt.18)))

      await builder.gte('age', 21).execute()
      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('age=gte.21)))
    })

    it('should build lt/lte filters', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.lt('age', 65).execute()
      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('age=lt.65)))

      await builder.lte('age', 100).execute()
      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('age=lte.100)))
    })

    it('should build like/ilike filters', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.like('name', '%john%').execute()
      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('name=like.%25john%25)))

      await builder.ilike('email', '%@gmail.com').execute()
      expect(mockFetch.get).toHaveBeenCalledWith(
        '/api/tables/users?email=ilike.%25%40gmail.com'
      )
    })

    it('should build is filter for null', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.is('deleted_at', null).execute()

      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('deleted_at=is.null)))
    })

    it('should build in filter', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.in('status', ['active', 'pending']).execute()

      expect(mockFetch.get).toHaveBeenCalledWith(
        '/api/tables/users?status=in.(active%2Cpending)'
      )
    })

    it('should build contains filter', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.contains('tags', ['admin', 'user']).execute()

      expect(mockFetch.get).toHaveBeenCalledWith(
        expect.stringContaining('tags=cs.)))
      )
    })

    it('should build textSearch filter', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.textSearch('description', 'postgres & database').execute()

      expect(mockFetch.get).toHaveBeenCalledWith(
        expect.stringContaining('description=fts.postgres)))
      )
    })

    it('should chain multiple filters', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.eq('status', 'active').gt('age', 18).lt('age', 65).execute()

      const call = vi.mocked(mockFetch.get).mock.calls[0][0]
      expect(call).toContain('status=eq.active)))
      expect(call).toContain('age=gt.18)))
      expect(call).toContain('age=lt.65)))
    })
  })

  describe('ordering', () => {
    it('should build ascending order', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.order('created_at', { ascending: true }).execute()

      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('order=created_at.asc)))
    })

    it('should build descending order', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.order('created_at', { ascending: false }).execute()

      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('order=created_at.desc)))
    })

    it('should default to ascending', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.order('name').execute()

      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('order=name.asc.nullslast)))
    })

    it('should support multiple order columns', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.order('status').order('created_at', { ascending: false }).execute()

      expect(mockFetch.get).toHaveBeenCalledWith(
        '/api/tables/users?order=status.asc.nullslast%2Ccreated_at.desc.nullslast'
      )
    })
  })

  describe('pagination', () => {
    it('should build limit query', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.limit(10).execute()

      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('limit=10)))
    })

    it('should build offset query', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.offset(20).execute()

      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('offset=20)))
    })

    it('should build limit and offset together', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.limit(10).offset(20).execute()

      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('limit=10&offset=20)))
    })

    it('should build range query', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder.range(0, 9).execute()

      expect(mockFetch.get).toHaveBeenCalledWith(expect.stringContaining('limit=10&offset=0)))
    })
  })

  describe('single()', () => {
    it('should return single row when found', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([{ id: 1, name: 'John' }])

      const result = await builder.eq('id', 1).single().execute()

      expect(result.data).toEqual({ id: 1, name: 'John' })
      expect(result.error).toBeNull()
    })

    it('should return error when no rows found', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      const result = await builder.eq('id', 999).single().execute()

      expect(result.data).toBeNull()
      expect(result.error).toEqual({ message: 'No rows found', code: 'PGRST116' })
      expect(result.status).toBe(404)
    })
  })

  describe('insert()', () => {
    it('should insert single row', async () => {
      const newUser = { name: 'John', email: 'john@example.com' }
      vi.mocked(mockFetch.post).mockResolvedValue({ id: 1, ...newUser })

      const result = await builder.insert(newUser)

      expect(mockFetch.post).toHaveBeenCalledWith('/api/tables/users', newUser)
      expect(result.data).toEqual({ id: 1, ...newUser })
      expect(result.error).toBeNull()
      expect(result.count).toBe(1)
    })

    it('should insert multiple rows', async () => {
      const newUsers = [
        { name: 'John', email: 'john@example.com' },
        { name: 'Jane', email: 'jane@example.com' },
      ]
      vi.mocked(mockFetch.post).mockResolvedValue([
        { id: 1, ...newUsers[0] },
        { id: 2, ...newUsers[1] },
      ])

      const result = await builder.insert(newUsers)

      expect(mockFetch.post).toHaveBeenCalledWith('/api/tables/users', newUsers)
      expect(result.count).toBe(2)
    })
  })

  describe('upsert()', () => {
    it('should upsert with merge-duplicates header', async () => {
      const user = { id: 1, name: 'John Updated' }
      vi.mocked(mockFetch.post).mockResolvedValue(user)

      await builder.upsert(user)

      expect(mockFetch.post).toHaveBeenCalledWith('/api/tables/users', user, {
        headers: { Prefer: 'resolution=merge-duplicates' },
      })
    })
  })

  describe('update()', () => {
    it('should update matching rows', async () => {
      const updates = { status: 'inactive' }
      vi.mocked(mockFetch.patch).mockResolvedValue([])

      await builder.eq('age', 100).update(updates)

      expect(mockFetch.patch).toHaveBeenCalledWith(expect.stringContaining('age=eq.100', updates)
    })
  })

  describe('delete()', () => {
    it('should delete matching rows', async () => {
      vi.mocked(mockFetch.delete).mockResolvedValue(undefined)

      const result = await builder.eq('status', 'deleted').delete()

      expect(mockFetch.delete).toHaveBeenCalledWith(expect.stringContaining('status=eq.deleted)))
      expect(result.status).toBe(204)
    })
  })

  describe('complex queries', () => {
    it('should build complex query with all features', async () => {
      vi.mocked(mockFetch.get).mockResolvedValue([])

      await builder
        .select('id,name,email,created_at)))
        .eq('status', 'active)))
        .gt('age', 18)
        .ilike('name', '%john%)))
        .order('created_at', { ascending: false })
        .limit(10)
        .offset(20)
        .execute()

      const call = vi.mocked(mockFetch.get).mock.calls[0][0]
      expect(call).toContain('select=)))
      expect(call).toContain('status=eq.active)))
      expect(call).toContain('age=gt.18)))
      expect(call).toContain('name=ilike.)))
      expect(call).toContain('order=created_at.desc)))
      expect(call).toContain('limit=10)))
      expect(call).toContain('offset=20)))
    })
  })

  describe('error handling', () => {
    it('should handle fetch errors gracefully', async () => {
      vi.mocked(mockFetch.get).mockRejectedValue(new Error('Network error'))

      const result = await builder.execute()

      expect(result.data).toBeNull()
      expect(result.error).toEqual({
        message: 'Network error',
        code: 'PGRST000',
      })
      expect(result.status).toBe(500)
    })
  })
})
