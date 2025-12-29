---
title: GraphQL API
description: Auto-generated GraphQL API for querying and mutating your database tables
---

Fluxbase provides an auto-generated GraphQL API that exposes your PostgreSQL tables as a fully typed GraphQL schema. The schema is automatically generated from your database tables, including relationships, filters, and mutations.

## Overview

The GraphQL API provides:
- **Auto-generated types** from your PostgreSQL tables
- **Query support** with filtering, ordering, and pagination
- **Mutation support** for insert, update, and delete operations
- **Nested queries** following foreign key relationships
- **Row Level Security (RLS)** enforcement on all operations
- **Introspection** for schema discovery (configurable)

## Endpoint

```
POST /api/v1/graphql
```

## Authentication

The GraphQL endpoint uses the same authentication as the REST API. Include a JWT token in the `Authorization` header:

```bash
curl -X POST http://localhost:8080/api/v1/graphql \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query": "{ users { id email } }"}'
```

## Configuration

Configure GraphQL in your `fluxbase.yaml` or via environment variables:

```yaml
# fluxbase.yaml
graphql:
  enabled: true
  max_depth: 10
  max_complexity: 1000
  introspection: true  # Disable in production
```

| Setting | Env Variable | Default | Description |
|---------|--------------|---------|-------------|
| `enabled` | `FLUXBASE_GRAPHQL_ENABLED` | `true` | Enable/disable GraphQL endpoint |
| `max_depth` | `FLUXBASE_GRAPHQL_MAX_DEPTH` | `10` | Maximum query nesting depth |
| `max_complexity` | `FLUXBASE_GRAPHQL_MAX_COMPLEXITY` | `1000` | Maximum query complexity score |
| `introspection` | `FLUXBASE_GRAPHQL_INTROSPECTION` | `true` | Allow schema introspection |

## Query Syntax

### Basic Query

```graphql
query {
  users {
    id
    email
    name
    created_at
  }
}
```

### With Filtering

```graphql
query {
  users(where: { email: { _eq: "john@example.com" } }) {
    id
    email
  }
}
```

### With Ordering and Pagination

```graphql
query {
  users(
    order_by: { created_at: desc }
    limit: 10
    offset: 0
  ) {
    id
    email
    created_at
  }
}
```

### Nested Queries (Relationships)

```graphql
query {
  users {
    id
    email
    posts {
      id
      title
      comments {
        id
        content
        author {
          email
        }
      }
    }
  }
}
```

## Filter Operators

The GraphQL API supports PostgREST-compatible filter operators:

| Operator | Description | Example |
|----------|-------------|---------|
| `_eq` | Equal | `{ status: { _eq: "active" } }` |
| `_neq` | Not equal | `{ status: { _neq: "deleted" } }` |
| `_gt` | Greater than | `{ age: { _gt: 18 } }` |
| `_gte` | Greater than or equal | `{ age: { _gte: 18 } }` |
| `_lt` | Less than | `{ price: { _lt: 100 } }` |
| `_lte` | Less than or equal | `{ price: { _lte: 100 } }` |
| `_like` | Pattern match | `{ name: { _like: "John%" } }` |
| `_ilike` | Case-insensitive match | `{ name: { _ilike: "john%" } }` |
| `_in` | In list | `{ status: { _in: ["active", "pending"] } }` |
| `_is_null` | Is null | `{ deleted_at: { _is_null: true } }` |
| `_and` | Logical AND | `{ _and: [{ age: { _gte: 18 } }, { status: { _eq: "active" } }] }` |
| `_or` | Logical OR | `{ _or: [{ role: { _eq: "admin" } }, { role: { _eq: "moderator" } }] }` |

## Mutations

### Insert

```graphql
mutation {
  insert_users(objects: [
    { email: "new@example.com", name: "New User" }
  ]) {
    returning {
      id
      email
    }
  }
}
```

### Update

```graphql
mutation {
  update_users(
    where: { id: { _eq: "user-uuid" } }
    _set: { name: "Updated Name" }
  ) {
    affected_rows
    returning {
      id
      name
    }
  }
}
```

### Delete

```graphql
mutation {
  delete_users(where: { id: { _eq: "user-uuid" } }) {
    affected_rows
  }
}
```

### Upsert (Insert or Update)

```graphql
mutation {
  insert_users(
    objects: [{ id: "existing-uuid", email: "user@example.com", name: "User" }]
    on_conflict: {
      constraint: users_pkey
      update_columns: [name]
    }
  ) {
    returning {
      id
      name
    }
  }
}
```

## Type Mapping

PostgreSQL types are automatically mapped to GraphQL types:

| PostgreSQL | GraphQL |
|------------|---------|
| `text`, `varchar`, `char` | `String` |
| `integer`, `smallint` | `Int` |
| `bigint` | `String` (to preserve precision) |
| `boolean` | `Boolean` |
| `numeric`, `real`, `double precision` | `Float` |
| `uuid` | `ID` |
| `json`, `jsonb` | `JSON` (custom scalar) |
| `timestamp`, `timestamptz` | `DateTime` (custom scalar) |
| `date` | `Date` (custom scalar) |
| `array` types | `[Type]` (List) |

## Introspection

When enabled, you can query the schema:

```graphql
query {
  __schema {
    types {
      name
      fields {
        name
        type {
          name
        }
      }
    }
  }
}
```

Or get type details:

```graphql
query {
  __type(name: "users") {
    name
    fields {
      name
      type {
        name
        kind
      }
    }
  }
}
```

## CLI Usage

The Fluxbase CLI provides a `graphql` command for executing queries and mutations from the command line.

```bash
# Execute a query
fluxbase graphql query '{ users { id email } }'

# Query with variables
fluxbase graphql query 'query($id: ID!) { user(id: $id) { email } }' --var 'id=123'

# Execute from file
fluxbase graphql query --file ./query.graphql

# Execute a mutation
fluxbase graphql mutation 'mutation { insert_users(objects: [{email: "new@example.com"}]) { returning { id } } }'

# Introspect the schema
fluxbase graphql introspect

# List types only
fluxbase graphql introspect --types
```

See the [CLI Command Reference](/docs/cli/commands#graphql-commands) for complete documentation.

---

## SDK Usage

### TypeScript SDK

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient({ url: 'http://localhost:8080' })

// Execute a query
const { data, errors } = await client.graphql.query<UsersQuery>(`
  query GetUsers($limit: Int) {
    users(limit: $limit) {
      id
      email
    }
  }
`, { limit: 10 })

// Execute a mutation
const { data, errors } = await client.graphql.mutation<CreateUserMutation>(`
  mutation CreateUser($data: UserInput!) {
    insert_users(objects: [$data]) {
      returning {
        id
        email
      }
    }
  }
`, { data: { email: 'new@example.com' } })
```

### React SDK

```tsx
import { useGraphQLQuery, useGraphQLMutation } from '@fluxbase/sdk-react'

function UsersList() {
  const { data, isLoading, error } = useGraphQLQuery<UsersQuery>(
    'users-list',
    `query { users { id email } }`
  )

  if (isLoading) return <div>Loading...</div>
  if (error) return <div>Error: {error.message}</div>

  return (
    <ul>
      {data?.users.map(user => (
        <li key={user.id}>{user.email}</li>
      ))}
    </ul>
  )
}

function CreateUserForm() {
  const mutation = useGraphQLMutation<CreateUserMutation>(
    `mutation CreateUser($data: UserInput!) {
      insert_users(objects: [$data]) {
        returning { id email }
      }
    }`,
    {
      onSuccess: (data) => console.log('Created:', data),
      invalidateQueries: ['users-list']
    }
  )

  const handleSubmit = (email: string) => {
    mutation.mutate({ data: { email } })
  }

  return (
    <button onClick={() => handleSubmit('new@example.com')}>
      Create User
    </button>
  )
}
```

## Admin Dashboard

The Query Editor in the Admin Dashboard supports both SQL and GraphQL modes:

1. Navigate to **Query Editor** in the sidebar
2. Click the **GraphQL** tab at the top of the editor
3. Write your GraphQL query with auto-completion support
4. Press **Ctrl+Enter** (or **Cmd+Enter** on Mac) to execute

The editor provides:
- Syntax highlighting for GraphQL
- Auto-completion for types, fields, and operations
- Query history tracking for both SQL and GraphQL queries
- JSON result formatting with error display

## Row Level Security

The GraphQL API enforces Row Level Security (RLS) policies exactly like the REST API:

1. Anonymous users execute queries as the `anon` PostgreSQL role
2. Authenticated users execute as the `authenticated` role
3. Service role keys bypass RLS for admin operations

Session variables are available in your RLS policies:

```sql
-- Example RLS policy
CREATE POLICY "Users can view own data" ON users
  FOR SELECT
  USING (id = current_setting('request.jwt.claim.sub')::uuid);
```

## Security Best Practices

### Production Configuration

```yaml
graphql:
  enabled: true
  max_depth: 5        # Reduce depth in production
  max_complexity: 500  # Lower complexity limit
  introspection: false # Disable introspection in production
```

### Query Depth Limiting

The `max_depth` setting prevents deeply nested queries that could be expensive:

```graphql
# This query has depth 4 (users -> posts -> comments -> author)
query {
  users {           # depth 1
    posts {         # depth 2
      comments {    # depth 3
        author {    # depth 4
          email
        }
      }
    }
  }
}
```

### Complexity Limiting

Query complexity is calculated based on the number of fields and nesting. The `max_complexity` setting prevents resource-intensive queries.

## Error Handling

GraphQL errors are returned in the standard GraphQL error format:

```json
{
  "data": null,
  "errors": [
    {
      "message": "permission denied for table users",
      "locations": [{ "line": 2, "column": 3 }],
      "path": ["users"]
    }
  ]
}
```

Common error types:
- **Validation errors**: Invalid query syntax or unknown fields
- **Authorization errors**: RLS policy violations
- **Depth/complexity errors**: Query exceeds configured limits

## Comparison with REST API

| Feature | GraphQL | REST |
|---------|---------|------|
| Request format | POST with query body | GET/POST/PUT/DELETE |
| Response shape | Exactly what you request | Fixed per endpoint |
| Nested data | Single request | Multiple requests |
| Caching | Requires client setup | HTTP caching |
| Learning curve | Higher | Lower |

Choose GraphQL when you need:
- Complex nested queries
- Flexible response shapes
- Strong typing with introspection

Choose REST when you need:
- Simple CRUD operations
- HTTP caching
- Simpler integration

## Troubleshooting

### GraphQL endpoint returns 404

Ensure GraphQL is enabled in your configuration:

```bash
export FLUXBASE_GRAPHQL_ENABLED=true
```

### Query depth exceeded

Reduce the nesting in your query or increase `max_depth`:

```yaml
graphql:
  max_depth: 15
```

### Permission denied errors

Check that:
1. Your JWT token is valid and not expired
2. RLS policies allow the operation
3. You're using the correct role (anon vs authenticated)

### Introspection not working

Enable introspection (note: disable in production):

```yaml
graphql:
  introspection: true
```
