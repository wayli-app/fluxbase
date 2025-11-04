import { describe, it, expect, beforeEach, vi } from 'vitest'
import { DDLManager } from './ddl'
import type { FluxbaseFetch } from './fetch'
import type {
  CreateSchemaResponse,
  CreateTableResponse,
  DeleteTableResponse,
  ListSchemasResponse,
  ListTablesResponse,
} from './types'

describe('DDLManager', () => {
  let manager: DDLManager
  let mockFetch: any

  beforeEach(() => {
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    }
    manager = new DDLManager(mockFetch as unknown as FluxbaseFetch)
  })

  describe('createSchema', () => {
    it('should create a new schema', async () => {
      const mockResponse: CreateSchemaResponse = {
        message: 'Schema created successfully',
        schema: 'analytics',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      const result = await manager.createSchema('analytics')

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/admin/ddl/schemas', {
        name: 'analytics',
      })
      expect(result).toEqual(mockResponse)
      expect(result.schema).toBe('analytics')
    })

    it('should handle schema creation errors', async () => {
      vi.mocked(mockFetch.post).mockRejectedValue(new Error('Schema already exists'))

      await expect(manager.createSchema('existing')).rejects.toThrow('Schema already exists')
    })
  })

  describe('listSchemas', () => {
    it('should list all schemas', async () => {
      const mockResponse: ListSchemasResponse = {
        schemas: [
          { name: 'public', owner: 'postgres' },
          { name: 'analytics', owner: 'admin' },
          { name: 'auth', owner: 'admin' },
        ],
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      const result = await manager.listSchemas()

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/admin/ddl/schemas')
      expect(result.schemas).toHaveLength(3)
      expect(result.schemas[0].name).toBe('public')
    })

    it('should handle empty schemas list', async () => {
      const mockResponse: ListSchemasResponse = {
        schemas: [],
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      const result = await manager.listSchemas()

      expect(result.schemas).toEqual([])
    })
  })

  describe('createTable', () => {
    it('should create a table with columns', async () => {
      const mockResponse: CreateTableResponse = {
        message: 'Table created successfully',
        schema: 'public',
        table: 'users',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      const result = await manager.createTable('public', 'users', [
        {
          name: 'id',
          type: 'UUID',
          primaryKey: true,
          defaultValue: 'gen_random_uuid()',
        },
        {
          name: 'email',
          type: 'TEXT',
          nullable: false,
        },
        {
          name: 'name',
          type: 'TEXT',
        },
      ])

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/admin/ddl/tables', {
        schema: 'public',
        name: 'users',
        columns: [
          {
            name: 'id',
            type: 'UUID',
            primaryKey: true,
            defaultValue: 'gen_random_uuid()',
          },
          {
            name: 'email',
            type: 'TEXT',
            nullable: false,
          },
          {
            name: 'name',
            type: 'TEXT',
          },
        ],
      })
      expect(result).toEqual(mockResponse)
      expect(result.table).toBe('users')
    })

    it('should create a table with various column types', async () => {
      const mockResponse: CreateTableResponse = {
        message: 'Table created successfully',
        schema: 'analytics',
        table: 'events',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      await manager.createTable('analytics', 'events', [
        {
          name: 'id',
          type: 'SERIAL',
          primaryKey: true,
        },
        {
          name: 'user_id',
          type: 'UUID',
          nullable: false,
        },
        {
          name: 'event_name',
          type: 'TEXT',
          nullable: false,
        },
        {
          name: 'event_data',
          type: 'JSONB',
        },
        {
          name: 'created_at',
          type: 'TIMESTAMPTZ',
          defaultValue: 'NOW()',
        },
      ])

      expect(mockFetch.post).toHaveBeenCalledWith(
        '/api/v1/admin/ddl/tables',
        expect.objectContaining({
          schema: 'analytics',
          name: 'events',
        })
      )
    })

    it('should handle table creation errors', async () => {
      vi.mocked(mockFetch.post).mockRejectedValue(new Error('Table already exists'))

      await expect(
        manager.createTable('public', 'existing_table', [
          { name: 'id', type: 'SERIAL', primaryKey: true },
        ])
      ).rejects.toThrow('Table already exists')
    })

    it('should create table with nullable columns', async () => {
      const mockResponse: CreateTableResponse = {
        message: 'Table created successfully',
        schema: 'public',
        table: 'products',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      await manager.createTable('public', 'products', [
        { name: 'id', type: 'SERIAL', primaryKey: true },
        { name: 'name', type: 'TEXT', nullable: false },
        { name: 'description', type: 'TEXT', nullable: true },
        { name: 'price', type: 'DECIMAL(10,2)', nullable: false },
      ])

      expect(mockFetch.post).toHaveBeenCalledWith(
        '/api/v1/admin/ddl/tables',
        expect.objectContaining({
          columns: expect.arrayContaining([
            expect.objectContaining({ name: 'description', nullable: true }),
          ]),
        })
      )
    })
  })

  describe('listTables', () => {
    it('should list all tables without schema filter', async () => {
      const mockResponse: ListTablesResponse = {
        tables: [
          { schema: 'public', name: 'users' },
          { schema: 'public', name: 'posts' },
          { schema: 'analytics', name: 'events' },
        ],
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      const result = await manager.listTables()

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/admin/ddl/tables')
      expect(result.tables).toHaveLength(3)
    })

    it('should list tables filtered by schema', async () => {
      const mockResponse: ListTablesResponse = {
        tables: [
          { schema: 'public', name: 'users' },
          { schema: 'public', name: 'posts' },
        ],
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      const result = await manager.listTables('public')

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/admin/ddl/tables?schema=public')
      expect(result.tables).toHaveLength(2)
      expect(result.tables.every((t) => t.schema === 'public')).toBe(true)
    })

    it('should handle empty tables list', async () => {
      const mockResponse: ListTablesResponse = {
        tables: [],
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      const result = await manager.listTables()

      expect(result.tables).toEqual([])
    })

    it('should properly encode schema name in URL', async () => {
      const mockResponse: ListTablesResponse = {
        tables: [],
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      await manager.listTables('my-schema')

      expect(mockFetch.get).toHaveBeenCalledWith('/api/v1/admin/ddl/tables?schema=my-schema')
    })

    it('should return tables with column information', async () => {
      const mockResponse: ListTablesResponse = {
        tables: [
          {
            schema: 'public',
            name: 'users',
            columns: [
              {
                name: 'id',
                type: 'uuid',
                nullable: false,
                is_primary_key: true,
              },
              {
                name: 'email',
                type: 'text',
                nullable: false,
              },
              {
                name: 'name',
                type: 'text',
                nullable: true,
              },
            ],
          },
        ],
      }

      vi.mocked(mockFetch.get).mockResolvedValue(mockResponse)

      const result = await manager.listTables('public')

      expect(result.tables[0].columns).toBeDefined()
      expect(result.tables[0].columns).toHaveLength(3)
      expect(result.tables[0].columns![0].is_primary_key).toBe(true)
    })
  })

  describe('deleteTable', () => {
    it('should delete a table', async () => {
      const mockResponse: DeleteTableResponse = {
        message: 'Table deleted successfully',
      }

      vi.mocked(mockFetch.delete).mockResolvedValue(mockResponse)

      const result = await manager.deleteTable('public', 'old_table')

      expect(mockFetch.delete).toHaveBeenCalledWith('/api/v1/admin/ddl/tables/public/old_table')
      expect(result.message).toBe('Table deleted successfully')
    })

    it('should properly encode schema and table names in URL', async () => {
      const mockResponse: DeleteTableResponse = {
        message: 'Table deleted successfully',
      }

      vi.mocked(mockFetch.delete).mockResolvedValue(mockResponse)

      await manager.deleteTable('my-schema', 'my-table')

      expect(mockFetch.delete).toHaveBeenCalledWith(
        '/api/v1/admin/ddl/tables/my-schema/my-table'
      )
    })

    it('should handle delete errors', async () => {
      vi.mocked(mockFetch.delete).mockRejectedValue(new Error('Table not found'))

      await expect(manager.deleteTable('public', 'nonexistent')).rejects.toThrow('Table not found')
    })

    it('should handle dependent object errors', async () => {
      vi.mocked(mockFetch.delete).mockRejectedValue(
        new Error('Cannot drop table due to dependent objects')
      )

      await expect(manager.deleteTable('public', 'users')).rejects.toThrow(
        'Cannot drop table due to dependent objects'
      )
    })
  })

  describe('Integration scenarios', () => {
    it('should support creating schema and table workflow', async () => {
      const schemaResponse: CreateSchemaResponse = {
        message: 'Schema created successfully',
        schema: 'analytics',
      }

      const tableResponse: CreateTableResponse = {
        message: 'Table created successfully',
        schema: 'analytics',
        table: 'events',
      }

      vi.mocked(mockFetch.post)
        .mockResolvedValueOnce(schemaResponse)
        .mockResolvedValueOnce(tableResponse)

      // Create schema
      await manager.createSchema('analytics')

      // Create table in that schema
      await manager.createTable('analytics', 'events', [
        { name: 'id', type: 'UUID', primaryKey: true },
        { name: 'data', type: 'JSONB' },
      ])

      expect(mockFetch.post).toHaveBeenCalledTimes(2)
    })

    it('should support list and delete workflow', async () => {
      const listResponse: ListTablesResponse = {
        tables: [
          { schema: 'public', name: 'temp_table' },
          { schema: 'public', name: 'users' },
        ],
      }

      const deleteResponse: DeleteTableResponse = {
        message: 'Table deleted successfully',
      }

      vi.mocked(mockFetch.get).mockResolvedValue(listResponse)
      vi.mocked(mockFetch.delete).mockResolvedValue(deleteResponse)

      // List tables
      const { tables } = await manager.listTables('public')

      // Delete temp table
      const tempTable = tables.find((t) => t.name === 'temp_table')
      if (tempTable) {
        await manager.deleteTable(tempTable.schema, tempTable.name)
      }

      expect(mockFetch.delete).toHaveBeenCalledWith('/api/v1/admin/ddl/tables/public/temp_table')
    })
  })
})
