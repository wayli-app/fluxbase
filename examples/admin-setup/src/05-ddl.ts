/**
 * Example 5: DDL Operations (Database Schema Management)
 *
 * This example demonstrates:
 * - Creating database schemas
 * - Listing schemas
 * - Creating tables with columns
 * - Listing tables
 * - Multi-tenant architecture
 * - Column types and constraints
 */

import { getClient, authenticateAdmin } from './utils/client.js'
import { logger } from './utils/logger.js'

async function main() {
  logger.section('DDL Operations Example')

  const client = getClient()

  try {
    await authenticateAdmin(client)
    logger.success('Admin authenticated')

    // Step 1: List Existing Schemas
    logger.step(1, 'List Existing Schemas')
    const schemas = await client.admin.ddl.listSchemas()
    logger.success(`Found ${schemas.length} schemas`)
    schemas.forEach(s => logger.item(`${s.schema_name}`))

    // Step 2: Create New Schema for Tenant
    logger.step(2, 'Create Multi-Tenant Schema')
    const tenantName = `demo_tenant_${Date.now()}`
    const schemaName = `tenant_${tenantName}`

    try {
      await client.admin.ddl.createSchema(schemaName)
      logger.success(`Schema created: ${schemaName}`)
    } catch (error: any) {
      if (error.status === 409) {
        logger.warn(`Schema ${schemaName} already exists`)
      } else {
        throw error
      }
    }

    // Step 3: Create Users Table
    logger.step(3, 'Create Users Table')
    try {
      await client.admin.ddl.createTable(schemaName, 'users', [
        { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
        { name: 'email', type: 'CITEXT', nullable: false },
        { name: 'name', type: 'TEXT', nullable: false },
        { name: 'role', type: 'TEXT', defaultValue: "'member'" },
        { name: 'metadata', type: 'JSONB', defaultValue: "'{}'" },
        { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' },
        { name: 'updated_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
      ])
      logger.success('Users table created')
    } catch (error: any) {
      if (error.status === 409) {
        logger.warn('Users table already exists')
      } else {
        throw error
      }
    }

    // Step 4: Create Projects Table
    logger.step(4, 'Create Projects Table')
    try {
      await client.admin.ddl.createTable(schemaName, 'projects', [
        { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
        { name: 'name', type: 'TEXT', nullable: false },
        { name: 'description', type: 'TEXT' },
        { name: 'owner_id', type: 'UUID', nullable: false },
        { name: 'status', type: 'TEXT', defaultValue: "'active'" },
        { name: 'settings', type: 'JSONB', defaultValue: "'{}'" },
        { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
      ])
      logger.success('Projects table created')
    } catch (error: any) {
      if (error.status === 409) {
        logger.warn('Projects table already exists')
      } else {
        throw error
      }
    }

    // Step 5: Create Events Table (Analytics)
    logger.step(5, 'Create Events Table (Analytics)')
    try {
      await client.admin.ddl.createTable(schemaName, 'events', [
        { name: 'id', type: 'BIGSERIAL', primaryKey: true },
        { name: 'user_id', type: 'UUID' },
        { name: 'event_name', type: 'TEXT', nullable: false },
        { name: 'event_data', type: 'JSONB', defaultValue: "'{}'" },
        { name: 'ip_address', type: 'TEXT' },
        { name: 'user_agent', type: 'TEXT' },
        { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
      ])
      logger.success('Events table created')
    } catch (error: any) {
      if (error.status === 409) {
        logger.warn('Events table already exists')
      } else {
        throw error
      }
    }

    // Step 6: List Tables in Schema
    logger.step(6, 'List Tables in Schema')
    const tables = await client.admin.ddl.listTables(schemaName)
    logger.success(`Found ${tables.length} tables in ${schemaName}`)
    tables.forEach(t => {
      logger.item(`${t.table_schema}.${t.table_name}`)
    })

    // Step 7: Supported Data Types Reference
    logger.step(7, 'Supported Data Types Reference')
    logger.info('Available PostgreSQL data types:')
    logger.item('Text: TEXT, VARCHAR, CITEXT')
    logger.item('Numbers: INTEGER, BIGINT, SERIAL, BIGSERIAL, DECIMAL, REAL')
    logger.item('Date/Time: DATE, TIME, TIMESTAMP, TIMESTAMPTZ')
    logger.item('JSON: JSON, JSONB')
    logger.item('Boolean: BOOLEAN')
    logger.item('UUID: UUID')
    logger.item('Arrays: TEXT[], INTEGER[], etc.')

    // Step 8: Column Constraints
    logger.step(8, 'Column Constraints')
    logger.info('Supported constraints:')
    logger.item('primaryKey: true - Makes column a primary key')
    logger.item('nullable: false - Makes column required')
    logger.item('defaultValue: "value" - Sets default value')
    logger.item('Example: { name: "id", type: "UUID", primaryKey: true, defaultValue: "gen_random_uuid()" }')

    logger.section('✅ DDL Operations Example Complete')
    logger.info('Key takeaways:')
    logger.item('Create schemas for multi-tenant isolation')
    logger.item('Define tables with full column specifications')
    logger.item('Use appropriate data types for your data')
    logger.item('Set primary keys and defaults')
    logger.item('JSONB for flexible metadata storage')
    logger.item('Timestamps for audit trails')

  } catch (error) {
    logger.error('DDL operations failed', error)
    process.exit(1)
  }
}

main().catch(console.error)
