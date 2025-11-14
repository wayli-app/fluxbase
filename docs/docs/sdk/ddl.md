---
title: DDL SDK
sidebar_position: 6
---

# DDL SDK

The DDL (Data Definition Language) SDK provides programmatic control over database schemas and tables. Use this module to create schemas, manage tables, and modify your database structure directly from your TypeScript application.

:::info
DDL operations require admin authentication. All operations in this guide assume you have logged in as an admin user.
:::

:::warning
DDL operations directly modify your database structure. Use with caution in production environments. Always test schema changes in a development environment first.
:::

## Installation

The DDL module is included with the Fluxbase SDK:

```bash
npm install @fluxbase/sdk
```

## Quick Start

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient(
  'http://localhost:8080',
  'your-service-role-key'
)

// Authenticate as admin
await client.admin.login({
  email: 'admin@example.com',
  password: 'admin-password'
})

// Create a new schema
await client.admin.ddl.createSchema('analytics')

// Create a table with columns
await client.admin.ddl.createTable('analytics', 'events', [
  {
    name: 'id',
    type: 'UUID',
    primaryKey: true,
    defaultValue: 'gen_random_uuid()'
  },
  {
    name: 'user_id',
    type: 'UUID',
    nullable: false
  },
  {
    name: 'event_name',
    type: 'TEXT',
    nullable: false
  },
  {
    name: 'event_data',
    type: 'JSONB'
  },
  {
    name: 'created_at',
    type: 'TIMESTAMPTZ',
    defaultValue: 'NOW()'
  }
])

// List all tables in the schema
const { tables } = await client.admin.ddl.listTables('analytics')
console.log('Tables:', tables)

// Delete a table when no longer needed
await client.admin.ddl.deleteTable('analytics', 'old_events')
```

---

## Schema Operations

Schemas provide logical grouping and namespace isolation for database tables.

### Create Schema

Create a new database schema.

```typescript
const result = await client.admin.ddl.createSchema('analytics')

console.log(result.message) // "Schema created successfully"
console.log(result.schema)  // "analytics"
```

**Parameters:**
- `name` (required): Schema name - Must be a valid PostgreSQL identifier

**Returns:** `CreateSchemaResponse` with message and schema name

**Example Use Cases:**
```typescript
// Create schema for multi-tenant data
await client.admin.ddl.createSchema('tenant_abc')

// Create schema for feature modules
await client.admin.ddl.createSchema('reporting')

// Create schema for testing
await client.admin.ddl.createSchema('test_data')
```

### List Schemas

Retrieve all database schemas.

```typescript
const { schemas } = await client.admin.ddl.listSchemas()

schemas.forEach(schema => {
  console.log(`Schema: ${schema.name}, Owner: ${schema.owner}`)
})

// Example output:
// Schema: public, Owner: postgres
// Schema: analytics, Owner: admin
// Schema: auth, Owner: admin
```

**Returns:** `ListSchemasResponse` containing:
- `schemas`: Array of schema objects with name and owner

**Filtering Schemas:**
```typescript
const { schemas } = await client.admin.ddl.listSchemas()

// Filter out system schemas
const userSchemas = schemas.filter(s =>
  !['pg_catalog', 'information_schema', 'pg_toast'].includes(s.name)
)

console.log('User schemas:', userSchemas.map(s => s.name))
```

---

## Table Operations

Create, list, and delete tables with custom column definitions.

### Create Table

Create a new table with column definitions.

```typescript
await client.admin.ddl.createTable('public', 'users', [
  {
    name: 'id',
    type: 'UUID',
    primaryKey: true,
    defaultValue: 'gen_random_uuid()'
  },
  {
    name: 'email',
    type: 'TEXT',
    nullable: false
  },
  {
    name: 'name',
    type: 'TEXT',
    nullable: true
  },
  {
    name: 'created_at',
    type: 'TIMESTAMPTZ',
    nullable: false,
    defaultValue: 'NOW()'
  }
])
```

**Parameters:**
- `schema` (required): Schema name where table will be created
- `name` (required): Table name - Must be a valid PostgreSQL identifier
- `columns` (required): Array of column definitions

**Column Definition:**
```typescript
interface CreateColumnRequest {
  name: string           // Column name
  type: string          // PostgreSQL data type
  nullable?: boolean    // Allow NULL values (default: true)
  primaryKey?: boolean  // Mark as primary key (default: false)
  defaultValue?: string // Default value expression
}
```

**Returns:** `CreateTableResponse` with message, schema, and table name

### Supported Data Types

Common PostgreSQL data types you can use:

```typescript
// Numeric types
await client.admin.ddl.createTable('public', 'products', [
  { name: 'id', type: 'SERIAL', primaryKey: true },
  { name: 'price', type: 'DECIMAL(10,2)', nullable: false },
  { name: 'quantity', type: 'INTEGER', defaultValue: '0' },
  { name: 'rating', type: 'REAL' }
])

// Text types
await client.admin.ddl.createTable('public', 'posts', [
  { name: 'id', type: 'UUID', primaryKey: true },
  { name: 'title', type: 'VARCHAR(255)', nullable: false },
  { name: 'content', type: 'TEXT' },
  { name: 'slug', type: 'CITEXT' } // Case-insensitive text
])

// Date/Time types
await client.admin.ddl.createTable('public', 'events', [
  { name: 'id', type: 'BIGSERIAL', primaryKey: true },
  { name: 'event_date', type: 'DATE' },
  { name: 'event_time', type: 'TIME' },
  { name: 'created_at', type: 'TIMESTAMP', defaultValue: 'NOW()' },
  { name: 'updated_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
])

// JSON types
await client.admin.ddl.createTable('public', 'settings', [
  { name: 'id', type: 'UUID', primaryKey: true },
  { name: 'config', type: 'JSON' },
  { name: 'metadata', type: 'JSONB' } // Binary JSON (indexable, faster)
])

// Boolean and Arrays
await client.admin.ddl.createTable('public', 'features', [
  { name: 'id', type: 'SERIAL', primaryKey: true },
  { name: 'enabled', type: 'BOOLEAN', defaultValue: 'true' },
  { name: 'tags', type: 'TEXT[]' }, // Array of text
  { name: 'flags', type: 'INTEGER[]' } // Array of integers
])
```

### List Tables

List all tables, optionally filtered by schema.

```typescript
// List all tables across all schemas
const { tables } = await client.admin.ddl.listTables()

tables.forEach(table => {
  console.log(`${table.schema}.${table.name}`)
})

// List tables in specific schema
const { tables: analyticsTables } = await client.admin.ddl.listTables('analytics')

analyticsTables.forEach(table => {
  console.log(`Table: ${table.name}`)

  // Column information may be included
  table.columns?.forEach(col => {
    const pk = col.is_primary_key ? ' (PK)' : ''
    const nullable = col.nullable ? 'NULL' : 'NOT NULL'
    console.log(`  - ${col.name}: ${col.type} ${nullable}${pk}`)
  })
})
```

**Parameters:**
- `schema` (optional): Filter tables by schema name

**Returns:** `ListTablesResponse` containing:
- `tables`: Array of table objects with schema, name, and optionally columns

**Organizing Table Data:**
```typescript
const { tables } = await client.admin.ddl.listTables()

// Group tables by schema
const tablesBySchema = tables.reduce((acc, table) => {
  if (!acc[table.schema]) {
    acc[table.schema] = []
  }
  acc[table.schema].push(table.name)
  return acc
}, {} as Record<string, string[]>)

console.log('Tables by schema:', tablesBySchema)
// {
//   public: ['users', 'posts', 'comments'],
//   analytics: ['events', 'metrics'],
//   auth: ['sessions', 'tokens']
// }
```

### Delete Table

Permanently delete a table and all its data.

```typescript
const result = await client.admin.ddl.deleteTable('analytics', 'old_events')

console.log(result.message) // "Table deleted successfully"
```

**Parameters:**
- `schema` (required): Schema name containing the table
- `name` (required): Table name to delete

**Returns:** `DeleteTableResponse` with confirmation message

:::danger
**This operation is irreversible!** All data in the table will be permanently deleted. Always backup important data before deletion.
:::

**Safe Deletion Pattern:**
```typescript
// Verify table exists before deletion
const { tables } = await client.admin.ddl.listTables('analytics')
const tableExists = tables.some(t =>
  t.schema === 'analytics' && t.name === 'old_events'
)

if (tableExists) {
  // Optionally backup data first
  // await backupTable('analytics', 'old_events')

  // Delete the table
  await client.admin.ddl.deleteTable('analytics', 'old_events')
  console.log('Table deleted successfully')
} else {
  console.log('Table does not exist')
}
```

---

## Common Use Cases

### 1. Multi-Tenant Database Setup

Create isolated schemas for each tenant:

```typescript
async function setupTenant(tenantId: string) {
  const schemaName = `tenant_${tenantId}`

  // Create tenant schema
  await client.admin.ddl.createSchema(schemaName)

  // Create tables for this tenant
  await client.admin.ddl.createTable(schemaName, 'users', [
    { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
    { name: 'email', type: 'TEXT', nullable: false },
    { name: 'name', type: 'TEXT' }
  ])

  await client.admin.ddl.createTable(schemaName, 'data', [
    { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
    { name: 'user_id', type: 'UUID', nullable: false },
    { name: 'content', type: 'JSONB' },
    { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
  ])

  console.log(`Tenant ${tenantId} setup complete`)
}

// Setup multiple tenants
await setupTenant('acme-corp')
await setupTenant('widget-co')
```

### 2. Analytics Event Tracking

Create tables for analytics data:

```typescript
// Create analytics schema
await client.admin.ddl.createSchema('analytics')

// Page view tracking
await client.admin.ddl.createTable('analytics', 'page_views', [
  { name: 'id', type: 'BIGSERIAL', primaryKey: true },
  { name: 'user_id', type: 'UUID' },
  { name: 'page_url', type: 'TEXT', nullable: false },
  { name: 'referrer', type: 'TEXT' },
  { name: 'user_agent', type: 'TEXT' },
  { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
])

// Custom event tracking
await client.admin.ddl.createTable('analytics', 'events', [
  { name: 'id', type: 'BIGSERIAL', primaryKey: true },
  { name: 'user_id', type: 'UUID' },
  { name: 'event_name', type: 'TEXT', nullable: false },
  { name: 'event_properties', type: 'JSONB' },
  { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
])

// User sessions
await client.admin.ddl.createTable('analytics', 'sessions', [
  { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
  { name: 'user_id', type: 'UUID' },
  { name: 'started_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' },
  { name: 'ended_at', type: 'TIMESTAMPTZ' },
  { name: 'duration_seconds', type: 'INTEGER' }
])
```

### 3. Audit Log Tables

Create comprehensive audit logging:

```typescript
await client.admin.ddl.createTable('public', 'audit_logs', [
  { name: 'id', type: 'BIGSERIAL', primaryKey: true },
  { name: 'user_id', type: 'UUID' },
  { name: 'action', type: 'TEXT', nullable: false },
  { name: 'table_name', type: 'TEXT', nullable: false },
  { name: 'record_id', type: 'TEXT' },
  { name: 'old_values', type: 'JSONB' },
  { name: 'new_values', type: 'JSONB' },
  { name: 'ip_address', type: 'INET' },
  { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
])
```

### 4. Time-Series Data Tables

Tables optimized for time-series data:

```typescript
await client.admin.ddl.createSchema('metrics')

// System metrics
await client.admin.ddl.createTable('metrics', 'system_metrics', [
  { name: 'time', type: 'TIMESTAMPTZ', nullable: false },
  { name: 'metric_name', type: 'TEXT', nullable: false },
  { name: 'value', type: 'DOUBLE PRECISION', nullable: false },
  { name: 'tags', type: 'JSONB' }
])

// Application performance
await client.admin.ddl.createTable('metrics', 'api_performance', [
  { name: 'timestamp', type: 'TIMESTAMPTZ', nullable: false },
  { name: 'endpoint', type: 'TEXT', nullable: false },
  { name: 'method', type: 'TEXT', nullable: false },
  { name: 'status_code', type: 'INTEGER', nullable: false },
  { name: 'duration_ms', type: 'INTEGER', nullable: false },
  { name: 'user_id', type: 'UUID' }
])
```

### 5. Feature-Based Schema Organization

Organize tables by application feature:

```typescript
// E-commerce features
await client.admin.ddl.createSchema('ecommerce')

await client.admin.ddl.createTable('ecommerce', 'products', [
  { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
  { name: 'name', type: 'TEXT', nullable: false },
  { name: 'description', type: 'TEXT' },
  { name: 'price', type: 'DECIMAL(10,2)', nullable: false },
  { name: 'inventory', type: 'INTEGER', defaultValue: '0' },
  { name: 'metadata', type: 'JSONB' }
])

await client.admin.ddl.createTable('ecommerce', 'orders', [
  { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
  { name: 'user_id', type: 'UUID', nullable: false },
  { name: 'status', type: 'TEXT', nullable: false },
  { name: 'total', type: 'DECIMAL(10,2)', nullable: false },
  { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
])

await client.admin.ddl.createTable('ecommerce', 'order_items', [
  { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
  { name: 'order_id', type: 'UUID', nullable: false },
  { name: 'product_id', type: 'UUID', nullable: false },
  { name: 'quantity', type: 'INTEGER', nullable: false },
  { name: 'price', type: 'DECIMAL(10,2)', nullable: false }
])
```

### 6. Test Data Management

Create and cleanup test data schemas:

```typescript
async function setupTestEnvironment() {
  const testSchema = 'test_' + Date.now()

  try {
    // Create test schema
    await client.admin.ddl.createSchema(testSchema)

    // Create test tables
    await client.admin.ddl.createTable(testSchema, 'users', [
      { name: 'id', type: 'SERIAL', primaryKey: true },
      { name: 'email', type: 'TEXT', nullable: false }
    ])

    // Run tests...

    return testSchema
  } catch (error) {
    console.error('Test setup failed:', error)
    throw error
  }
}

async function cleanupTestEnvironment(testSchema: string) {
  // List and delete all test tables
  const { tables } = await client.admin.ddl.listTables(testSchema)

  for (const table of tables) {
    await client.admin.ddl.deleteTable(table.schema, table.name)
  }

  console.log(`Cleaned up ${tables.length} test tables`)
}

// Usage
const testSchema = await setupTestEnvironment()
// ... run tests ...
await cleanupTestEnvironment(testSchema)
```

### 7. Database Migration Script

Programmatic database migrations:

```typescript
async function migrateDatabase() {
  // Check if migration is needed
  const { tables } = await client.admin.ddl.listTables('public')
  const hasNewTable = tables.some(t => t.name === 'user_preferences')

  if (!hasNewTable) {
    console.log('Running migration: add user_preferences table')

    await client.admin.ddl.createTable('public', 'user_preferences', [
      { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
      { name: 'user_id', type: 'UUID', nullable: false },
      { name: 'preferences', type: 'JSONB', defaultValue: "'{}'" },
      { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' },
      { name: 'updated_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
    ])

    console.log('Migration complete')
  } else {
    console.log('Database already up to date')
  }
}

await migrateDatabase()
```

---

## Error Handling

Handle common DDL operation errors:

### Schema Already Exists

```typescript
try {
  await client.admin.ddl.createSchema('analytics')
} catch (error) {
  if (error.message.includes('already exists')) {
    console.log('Schema already exists, continuing...')
  } else {
    throw error
  }
}
```

### Table Already Exists

```typescript
try {
  await client.admin.ddl.createTable('public', 'users', columns)
} catch (error) {
  if (error.message.includes('already exists')) {
    console.log('Table already exists')
  } else if (error.message.includes('invalid')) {
    console.error('Invalid table or column definition')
  } else {
    throw error
  }
}
```

### Table Not Found

```typescript
try {
  await client.admin.ddl.deleteTable('public', 'old_table')
} catch (error) {
  if (error.status === 404) {
    console.log('Table does not exist')
  } else {
    throw error
  }
}
```

### Dependent Objects

```typescript
try {
  await client.admin.ddl.deleteTable('public', 'users')
} catch (error) {
  if (error.message.includes('dependent objects')) {
    console.error('Cannot delete table - other objects depend on it')
    console.error('Delete dependent tables first or use CASCADE')
  } else {
    throw error
  }
}
```

### Invalid Identifiers

```typescript
const isValidIdentifier = (name: string): boolean => {
  // PostgreSQL identifier rules
  return /^[a-zA-Z_][a-zA-Z0-9_]*$/.test(name)
}

const tableName = 'my-invalid-table' // Contains hyphen

if (!isValidIdentifier(tableName)) {
  console.error('Invalid table name - use letters, numbers, and underscores only')
} else {
  await client.admin.ddl.createTable('public', tableName, columns)
}
```

---

## Best Practices

### 1. Use Schemas for Organization

Group related tables into schemas:

```typescript
// Good: Organized by feature
await client.admin.ddl.createSchema('analytics')
await client.admin.ddl.createSchema('auth')
await client.admin.ddl.createSchema('ecommerce')

// Avoid: Everything in public schema
// All tables in 'public' - hard to organize
```

### 2. Always Specify Nullability

Be explicit about NULL constraints:

```typescript
// Good: Explicit nullability
await client.admin.ddl.createTable('public', 'users', [
  { name: 'id', type: 'UUID', primaryKey: true, nullable: false },
  { name: 'email', type: 'TEXT', nullable: false },
  { name: 'phone', type: 'TEXT', nullable: true }
])

// Avoid: Implicit nullability
await client.admin.ddl.createTable('public', 'users', [
  { name: 'id', type: 'UUID', primaryKey: true },
  { name: 'email', type: 'TEXT' }, // Nullable by default
  { name: 'phone', type: 'TEXT' }
])
```

### 3. Use Appropriate Data Types

Choose the right type for your data:

```typescript
// Good: Specific types
await client.admin.ddl.createTable('public', 'products', [
  { name: 'id', type: 'UUID' },                    // UUIDs for distributed IDs
  { name: 'price', type: 'DECIMAL(10,2)' },        // Precise decimals for money
  { name: 'created_at', type: 'TIMESTAMPTZ' },     // Timezone-aware timestamps
  { name: 'metadata', type: 'JSONB' }              // Indexable JSON
])

// Avoid: Generic types
await client.admin.ddl.createTable('public', 'products', [
  { name: 'id', type: 'TEXT' },                    // String instead of UUID
  { name: 'price', type: 'REAL' },                 // Floating point for money
  { name: 'created_at', type: 'TEXT' },            // String instead of timestamp
  { name: 'metadata', type: 'TEXT' }               // String instead of JSON
])
```

### 4. Include Audit Columns

Add standard audit columns to all tables:

```typescript
const withAuditColumns = (columns: CreateColumnRequest[]): CreateColumnRequest[] => {
  return [
    ...columns,
    { name: 'created_at', type: 'TIMESTAMPTZ', nullable: false, defaultValue: 'NOW()' },
    { name: 'updated_at', type: 'TIMESTAMPTZ', nullable: false, defaultValue: 'NOW()' },
    { name: 'created_by', type: 'UUID' },
    { name: 'updated_by', type: 'UUID' }
  ]
}

// Usage
await client.admin.ddl.createTable('public', 'posts', withAuditColumns([
  { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
  { name: 'title', type: 'TEXT', nullable: false },
  { name: 'content', type: 'TEXT' }
]))
```

### 5. Validate Before Deleting

Always verify before destructive operations:

```typescript
async function safeDeleteTable(schema: string, name: string) {
  // Verify table exists
  const { tables } = await client.admin.ddl.listTables(schema)
  const exists = tables.some(t => t.name === name)

  if (!exists) {
    throw new Error(`Table ${schema}.${name} does not exist`)
  }

  // Optionally check for data
  // const hasData = await checkTableHasData(schema, name)
  // if (hasData) {
  //   throw new Error('Table contains data - backup first')
  // }

  // Delete table
  await client.admin.ddl.deleteTable(schema, name)
}
```

### 6. Handle Idempotent Operations

Make DDL operations safe to run multiple times:

```typescript
async function ensureTableExists(
  schema: string,
  name: string,
  columns: CreateColumnRequest[]
) {
  try {
    await client.admin.ddl.createTable(schema, name, columns)
    console.log(`Created table ${schema}.${name}`)
  } catch (error) {
    if (error.message.includes('already exists')) {
      console.log(`Table ${schema}.${name} already exists`)
    } else {
      throw error
    }
  }
}
```

### 7. Use Naming Conventions

Establish consistent naming patterns:

```typescript
// Good: Consistent naming
const NAMING = {
  schemas: (feature: string) => feature.toLowerCase(),
  tables: (entity: string) => entity.toLowerCase() + 's',
  primaryKey: () => 'id',
  foreignKey: (table: string) => table + '_id',
  timestamp: {
    created: 'created_at',
    updated: 'updated_at'
  }
}

await client.admin.ddl.createTable(
  NAMING.schemas('analytics'),
  NAMING.tables('event'),
  [
    { name: NAMING.primaryKey(), type: 'UUID', primaryKey: true },
    { name: NAMING.foreignKey('user'), type: 'UUID' },
    { name: NAMING.timestamp.created, type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
  ]
)
```

---

## TypeScript Types

The DDL SDK is fully typed for TypeScript users.

```typescript
import type {
  CreateColumnRequest,
  CreateSchemaRequest,
  CreateSchemaResponse,
  CreateTableRequest,
  CreateTableResponse,
  DeleteTableResponse,
  Schema,
  ListSchemasResponse,
  Column,
  Table,
  ListTablesResponse
} from '@fluxbase/sdk'

// Type-safe column definitions
const columns: CreateColumnRequest[] = [
  { name: 'id', type: 'UUID', primaryKey: true },
  { name: 'email', type: 'TEXT', nullable: false }
]

// Type-safe responses
const schemaResult: CreateSchemaResponse = await client.admin.ddl.createSchema('test')
const tableResult: CreateTableResponse = await client.admin.ddl.createTable('test', 'users', columns)
```

---

## Security Considerations

### 1. Admin-Only Access

DDL operations require admin authentication:

```typescript
// Ensure admin authentication
if (!client.admin.getToken()) {
  throw new Error('Admin authentication required for DDL operations')
}

await client.admin.ddl.createTable(...)
```

### 2. SQL Injection Prevention

The SDK uses parameterized queries to prevent SQL injection:

```typescript
// Safe - parameters are properly escaped
await client.admin.ddl.createTable(
  userInputSchema,
  userInputTableName,
  userInputColumns
)

// The backend validates and sanitizes all inputs
```

### 3. Validation

Backend validates all identifiers and types:

```typescript
// Invalid identifier - will be rejected
try {
  await client.admin.ddl.createTable('public', '123invalid', [])
} catch (error) {
  // Error: Invalid table name
}

// Invalid type - will be rejected
try {
  await client.admin.ddl.createTable('public', 'test', [
    { name: 'col', type: 'INVALID_TYPE' }
  ])
} catch (error) {
  // Error: Invalid column type
}
```

---

## Limitations

Current limitations of the DDL SDK:

1. **No ALTER TABLE support** - Cannot modify existing table structure yet
2. **No schema deletion** - Cannot drop schemas programmatically
3. **No constraints** - Cannot add foreign keys, unique constraints, or check constraints
4. **No indexes** - Cannot create indexes programmatically

These features may be added in future releases based on user feedback.

---

## Next Steps

- Learn about [Admin SDK](/docs/sdk/admin) for user management
- Explore [Settings SDK](/docs/sdk/settings) for configuration management
- Read [Database](/docs/guides/typescript-sdk/database) docs for data operations
- Check out [Advanced Features](/docs/sdk/advanced-features) for schema design patterns
