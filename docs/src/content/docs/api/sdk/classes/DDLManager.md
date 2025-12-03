---
editUrl: false
next: false
prev: false
title: "DDLManager"
---

DDL (Data Definition Language) Manager

Provides methods for managing database schemas and tables programmatically.
This includes creating schemas, creating tables with custom columns, listing
schemas and tables, and deleting tables.

## Example

```typescript
const ddl = client.admin.ddl

// Create a new schema
await ddl.createSchema('analytics')

// Create a table with columns
await ddl.createTable('analytics', 'events', [
  { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
  { name: 'user_id', type: 'UUID', nullable: false },
  { name: 'event_name', type: 'TEXT', nullable: false },
  { name: 'event_data', type: 'JSONB' },
  { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
])

// List all schemas
const { schemas } = await ddl.listSchemas()

// List all tables in a schema
const { tables } = await ddl.listTables('analytics')

// Delete a table
await ddl.deleteTable('analytics', 'events')
```

## Constructors

### new DDLManager()

> **new DDLManager**(`fetch`): [`DDLManager`](/api/sdk/classes/ddlmanager/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`DDLManager`](/api/sdk/classes/ddlmanager/)

## Methods

### createSchema()

> **createSchema**(`name`): `Promise`\<[`CreateSchemaResponse`](/api/sdk/interfaces/createschemaresponse/)\>

Create a new database schema

Creates a new schema in the database. Schemas are used to organize tables
into logical groups and provide namespace isolation.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `name` | `string` | Schema name (must be valid PostgreSQL identifier) |

#### Returns

`Promise`\<[`CreateSchemaResponse`](/api/sdk/interfaces/createschemaresponse/)\>

Promise resolving to CreateSchemaResponse

#### Example

```typescript
// Create a schema for analytics data
const result = await client.admin.ddl.createSchema('analytics')
console.log(result.message) // "Schema created successfully"
console.log(result.schema)  // "analytics"
```

***

### createTable()

> **createTable**(`schema`, `name`, `columns`): `Promise`\<[`CreateTableResponse`](/api/sdk/interfaces/createtableresponse/)\>

Create a new table in a schema

Creates a new table with the specified columns. Supports various column
options including primary keys, nullability, data types, and default values.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `schema` | `string` | Schema name where the table will be created |
| `name` | `string` | Table name (must be valid PostgreSQL identifier) |
| `columns` | [`CreateColumnRequest`](/api/sdk/interfaces/createcolumnrequest/)[] | Array of column definitions |

#### Returns

`Promise`\<[`CreateTableResponse`](/api/sdk/interfaces/createtableresponse/)\>

Promise resolving to CreateTableResponse

#### Examples

```typescript
// Create a users table
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
    type: 'TEXT'
  },
  {
    name: 'created_at',
    type: 'TIMESTAMPTZ',
    nullable: false,
    defaultValue: 'NOW()'
  }
])
```

```typescript
// Create a products table with JSONB metadata
await client.admin.ddl.createTable('public', 'products', [
  { name: 'id', type: 'SERIAL', primaryKey: true },
  { name: 'name', type: 'TEXT', nullable: false },
  { name: 'price', type: 'DECIMAL(10,2)', nullable: false },
  { name: 'metadata', type: 'JSONB' },
  { name: 'in_stock', type: 'BOOLEAN', defaultValue: 'true' }
])
```

***

### deleteTable()

> **deleteTable**(`schema`, `name`): `Promise`\<[`DeleteTableResponse`](/api/sdk/interfaces/deletetableresponse/)\>

Delete a table from a schema

Permanently deletes a table and all its data. This operation cannot be undone.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `schema` | `string` | Schema name containing the table |
| `name` | `string` | Table name to delete |

#### Returns

`Promise`\<[`DeleteTableResponse`](/api/sdk/interfaces/deletetableresponse/)\>

Promise resolving to DeleteTableResponse

#### Examples

```typescript
// Delete a table
const result = await client.admin.ddl.deleteTable('public', 'old_data')
console.log(result.message) // "Table deleted successfully"
```

```typescript
// Safe deletion with confirmation
const confirm = await askUser('Are you sure you want to delete this table?')
if (confirm) {
  await client.admin.ddl.deleteTable('analytics', 'events')
  console.log('Table deleted')
}
```

***

### listSchemas()

> **listSchemas**(): `Promise`\<[`ListSchemasResponse`](/api/sdk/interfaces/listschemasresponse/)\>

List all database schemas

Retrieves a list of all schemas in the database. This includes both
system schemas (like 'public', 'pg_catalog') and user-created schemas.

#### Returns

`Promise`\<[`ListSchemasResponse`](/api/sdk/interfaces/listschemasresponse/)\>

Promise resolving to ListSchemasResponse

#### Example

```typescript
const { schemas } = await client.admin.ddl.listSchemas()

schemas.forEach(schema => {
  console.log(`Schema: ${schema.name}, Owner: ${schema.owner}`)
})
```

***

### listTables()

> **listTables**(`schema`?): `Promise`\<[`ListTablesResponse`](/api/sdk/interfaces/listtablesresponse/)\>

List all tables in the database or a specific schema

Retrieves a list of all tables. If a schema is specified, only tables
from that schema are returned. Otherwise, all tables from all schemas
are returned.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `schema`? | `string` | Optional schema name to filter tables |

#### Returns

`Promise`\<[`ListTablesResponse`](/api/sdk/interfaces/listtablesresponse/)\>

Promise resolving to ListTablesResponse

#### Examples

```typescript
// List all tables in the public schema
const { tables } = await client.admin.ddl.listTables('public')

tables.forEach(table => {
  console.log(`Table: ${table.schema}.${table.name}`)
  table.columns?.forEach(col => {
    console.log(`  - ${col.name}: ${col.type}`)
  })
})
```

```typescript
// List all tables across all schemas
const { tables } = await client.admin.ddl.listTables()

const tablesBySchema = tables.reduce((acc, table) => {
  if (!acc[table.schema]) acc[table.schema] = []
  acc[table.schema].push(table.name)
  return acc
}, {} as Record<string, string[]>)

console.log(tablesBySchema)
```
