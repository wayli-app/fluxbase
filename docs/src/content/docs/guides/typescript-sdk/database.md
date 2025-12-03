---
title: "Database Operations"
---

The Fluxbase SDK provides a powerful query builder for interacting with your PostgreSQL database through a PostgREST-compatible API.

## Table of Contents

- [Query Execution](#query-execution)
- [Selecting Data](#selecting-data)
- [Filtering](#filtering)
- [Inserting Data](#inserting-data)
- [Updating Data](#updating-data)
- [Deleting Data](#deleting-data)
- [Aggregations](#aggregations)
- [Batch Operations](#batch-operations)
- [Sorting and Pagination](#sorting-and-pagination)
- [RPC (PostgreSQL Functions)](#rpc-postgresql-functions)

## Query Execution

The Fluxbase SDK query builder is **awaitable**, which means `.execute()` is optional. Both syntaxes work identically:

```typescript
// Supabase-compatible: await the query directly
const { data } = await client.from('users').select('*')

// Explicit execution: call .execute()
const { data } = await client.from('users').select('*').execute()
```

When you `await` a query without calling `.execute()`, the SDK automatically executes it for you. This makes Fluxbase queries compatible with Supabase's syntax.

**Note:** Mutation operations (`.insert()`, `.update()`, `.delete()`) and RPC calls (`.rpc()`) work the same way - `.execute()` is optional for all queries.

## Selecting Data

### Basic Select

```typescript
// Select all columns (both syntaxes work)
const { data } = await client.from('users').select('*')

// Select specific columns
const { data } = await client.from('users').select('id, name, email')
```

### Single Row

```typescript
// Get a single row (adds LIMIT 1)
// Errors if no rows found
const { data: user } = await client.from('users')
  .eq('id', 123)
  .single()

// Returns null if no rows found (doesn't error)
const { data: user } = await client.from('users')
  .eq('id', 999)
  .maybeSingle()
// data will be null if user doesn't exist, error will be null

// Throw error instead of returning { data, error }
try {
  const user = await client.from('users')
    .eq('id', 123)
    .single()
    .throwOnError() // Returns user directly or throws
  console.log('User:', user)
} catch (error) {
  console.error('Failed to fetch user:', error)
}
```

### Nested Relations

```typescript
// Select with related data
const { data } = await client.from('posts')
  .select('id, title, content, author(id, name, email)')
```

## Filtering

| Operator | Purpose | Example |
|----------|---------|---------|
| `eq()` | Equal | `.eq('status', 'active')` |
| `neq()` | Not equal | `.neq('status', 'deleted')` |
| `gt()` | Greater than | `.gt('price', 100)` |
| `gte()` | Greater than or equal | `.gte('rating', 4)` |
| `lt()` | Less than | `.lt('stock', 10)` |
| `lte()` | Less than or equal | `.lte('price', 500)` |
| `like()` | Pattern match (case-sensitive) | `.like('email', '%@gmail.com')` |
| `ilike()` | Pattern match (case-insensitive) | `.ilike('name', '%john%')` |
| `in()` | Value in list | `.in('status', ['active', 'pending'])` |
| `is()` | IS (for null checks) | `.is('deleted_at', null)` |

**Example:**

```typescript
const { data } = await client.from('products')
  .eq('category', 'electronics')
  .gt('price', 100)
  .lte('price', 500)

// ILIKE (case-insensitive)
const { data } = await client.from('users').ilike('name', '%john%')
```

### Arrays and Sets

```typescript
// IN (value in array)
const { data } = await client.from('products')
  .in('category', ['electronics', 'computers', 'phones'])

// NOT IN
const { data } = await client.from('products')
  .not('category', 'in', ['discontinued', 'archived'])
```

### Null Checks

```typescript
// IS NULL
const { data } = await client.from('tasks').is('completed_at', null)

// IS NOT NULL
const { data } = await client.from('tasks').not('completed_at', 'is', null)
```

### Range

```typescript
// Value within range
const { data } = await client.from('products')
  .gte('price', 100)
  .lte('price', 500)
```

### Combining Filters

All filter methods can be chained together:

```typescript
const { data } = await client.from('products')
  .eq('status', 'active')
  .gt('price', 50)
  .lt('price', 200)
  .in('category', ['electronics', 'accessories'])
  .ilike('name', '%phone%')
```

### Advanced Filtering

#### NOT Operator

Negate any filter condition:

```typescript
// NOT equal
const { data } = await client.from('products')
  .not('status', 'eq', 'deleted')

// NOT NULL
const { data } = await client.from('tasks')
  .not('completed_at', 'is', null)

// NOT IN
const { data } = await client.from('products')
  .not('category', 'in', ['discontinued', 'archived'])
```

#### OR Operator

Combine multiple conditions with OR logic:

```typescript
// Simple OR
const { data } = await client.from('products')
  .or('status.eq.active,status.eq.pending')

// Complex queries with OR and other filters
const { data } = await client.from('products')
  .or('status.eq.active,priority.eq.high')
  .gte('score', 80)
  .order('created_at', { ascending: false })
```

#### AND Operator

Group multiple conditions that must all be true:

```typescript
// Simple AND grouping
const { data } = await client.from('users')
  .and('status.eq.active,verified.eq.true')

// Complex AND with age range
const { data } = await client.from('users')
  .and('age.gte.18,age.lte.65')
  .eq('country', 'US')

// Combining AND and OR
const { data } = await client.from('products')
  .and('in_stock.eq.true,price.lte.1000')
  .or('category.eq.electronics,category.eq.computers')
```

#### Match Multiple Values

Shorthand for multiple exact matches:

```typescript
// Match multiple columns
const { data } = await client.from('users')
  .match({
    role: 'admin',
    status: 'active',
    department: 'engineering'
  })

// Equivalent to:
// .eq('role', 'admin').eq('status', 'active').eq('department', 'engineering')
```

#### Array and JSONB Operations

```typescript
// Contains (array/JSONB contains value)
const { data } = await client.from('posts')
  .contains('tags', '["typescript","javascript"]')

// Contained by (value is contained in array/JSONB)
const { data } = await client.from('posts')
  .containedBy('tags', '["news","update","feature"]')

// Overlaps (arrays have common elements)
const { data } = await client.from('posts')
  .overlaps('tags', '["typescript","react"]')
```

#### Generic Filter

Use raw PostgREST syntax:

```typescript
const { data } = await client.from('users')
  .filter('age', 'gte', '18')
  .filter('country', 'in', '("US","CA","UK")')
```

## Inserting Data

### Single Insert

```typescript
const { data, error } = await client.from('users').insert({
  name: 'Alice Smith',
  email: 'alice@example.com',
  age: 28
})

if (error) {
  console.error('Insert failed:', error)
} else {
  console.log('Created user:', data)
}
```

### Batch Insert (Multiple Rows)

```typescript
// Using insert() with an array
const { data } = await client.from('products').insert([
  { name: 'Product 1', price: 99.99, category: 'electronics' },
  { name: 'Product 2', price: 149.99, category: 'electronics' },
  { name: 'Product 3', price: 79.99, category: 'accessories' }
])

// Using insertMany() for clarity
const { data } = await client.from('products').insertMany([
  { name: 'Product 1', price: 99.99 },
  { name: 'Product 2', price: 149.99 },
  { name: 'Product 3', price: 79.99 }
])
```

### Upsert (Insert or Update)

```typescript
// Basic upsert - insert or update if unique constraint conflict
const { data } = await client.from('users').upsert({
  id: 123, // Will update if ID exists, insert if not
  name: 'Updated Name',
  email: 'updated@example.com'
})

// Upsert with conflict resolution on specific column
const { data } = await client.from('users').upsert(
  { email: 'alice@example.com', name: 'Alice', age: 30 },
  { onConflict: 'email' } // Resolve conflicts based on email column
)

// Upsert with multiple conflict columns
const { data } = await client.from('user_roles').upsert(
  { user_id: 1, tenant_id: 5, role: 'admin' },
  { onConflict: 'user_id,tenant_id' }
)

// Ignore duplicates instead of updating
const { data } = await client.from('logs').upsert(
  { id: 1, message: 'System started' },
  { ignoreDuplicates: true } // Don't update if row exists
)

// Set missing columns to null instead of keeping existing values
const { data } = await client.from('profiles').upsert(
  { user_id: 1, bio: 'New bio' }, // Only updating bio
  { defaultToNull: true } // Other columns will be set to null
)
```

## Updating Data

### Update with Filters

```typescript
// Update matching rows
const { data } = await client.from('users')
  .eq('id', 123)
  .update({
    name: 'John Updated',
    updated_at: new Date()
  })

// Update multiple rows
const { data } = await client.from('products')
  .eq('category', 'electronics')
  .update({ discount: 10 })
```

### Batch Update

```typescript
// Update all matching rows
const { data } = await client.from('orders')
  .eq('status', 'pending')
  .updateMany({ status: 'processing', processed_at: new Date() })
```

## Deleting Data

### Delete with Filters

```typescript
// Delete specific row
await client.from('users').eq('id', 123).delete()

// Delete multiple rows
await client.from('logs').lt('created_at', '2024-01-01').delete()
```

### Batch Delete

```typescript
// Delete all matching rows
await client.from('temp_data').eq('processed', true).deleteMany()
```

## Aggregations

The SDK supports SQL aggregation functions with GROUP BY support.

### Count

```typescript
// Count all rows
const { data } = await client.from('users').count()
// Returns: { count: 150 }

// Count specific column (non-null values)
const { data } = await client.from('orders').count('completed_at')

// Count with grouping
const { data } = await client.from('products')
  .count('*')
  .groupBy('category')
// Returns: [
//   { category: 'electronics', count: 45 },
//   { category: 'books', count: 23 },
//   { category: 'accessories', count: 12 }
// ]
```

### Sum

```typescript
// Sum a column
const { data } = await client.from('orders').sum('total')
// Returns: { sum_total: 125430.50 }

// Sum by category
const { data } = await client.from('sales')
  .sum('amount')
  .groupBy('region')
// Returns: [
//   { region: 'North', sum_amount: 45000 },
//   { region: 'South', sum_amount: 32000 }
// ]
```

### Average

```typescript
// Average price
const { data } = await client.from('products').avg('price')
// Returns: { avg_price: 129.99 }

// Average by category
const { data } = await client.from('products')
  .avg('price')
  .groupBy('category')
```

### Min/Max

```typescript
// Find minimum
const { data } = await client.from('products').min('price')
// Returns: { min_price: 9.99 }

// Find maximum
const { data } = await client.from('products').max('price')
// Returns: { max_price: 1999.99 }

// Min/Max with grouping
const { data } = await client.from('sales')
  .max('amount')
  .groupBy(['region', 'product_category'])
```

### Combining Aggregations with Filters

```typescript
// Count active users created this year
const { data } = await client.from('users')
  .count('*')
  .eq('status', 'active')
  .gte('created_at', '2024-01-01')

// Average order value by customer type
const { data } = await client.from('orders')
  .avg('total')
  .groupBy('customer_type')
  .gte('created_at', '2024-01-01')
```

## Batch Operations

Batch operations allow you to perform actions on multiple rows efficiently.

### Batch Insert

```typescript
// Insert 100 records at once
const users = Array.from({ length: 100 }, (_, i) => ({
  name: `User ${i}`,
  email: `user${i}@example.com`
}))

const { data } = await client.from('users').insertMany(users)
```

### Batch Update

```typescript
// Update all electronics products
await client.from('products')
  .eq('category', 'electronics')
  .updateMany({
    discount: 15,
    sale_ends: '2024-12-31'
  })

// Mark all old orders as archived
await client.from('orders')
  .lt('created_at', '2023-01-01')
  .updateMany({ archived: true })
```

### Batch Delete

```typescript
// Delete all processed temporary records
await client.from('temp_uploads')
  .eq('processed', true)
  .deleteMany()

// Delete old logs
await client.from('logs')
  .lt('created_at', '2024-01-01')
  .deleteMany()
```

## Sorting and Pagination

### Ordering

```typescript
// Order by single column (ascending by default)
const { data } = await client.from('users')
  .select('*')
  .order('name')

// Order descending
const { data } = await client.from('products')
  .select('*')
  .order('price', { ascending: false })

// Multiple ordering
const { data } = await client.from('products')
  .select('*')
  .order('category')
  .order('price', { ascending: false })

// Null handling
const { data } = await client.from('tasks')
  .select('*')
  .order('completed_at', { ascending: true, nullsFirst: true })
```

### Pagination

```typescript
// Using limit and offset
const { data } = await client.from('users')
  .select('*')
  .limit(10)
  .offset(20)

// Using range (page-based)
const page = 2
const pageSize = 10
const { data } = await client.from('users')
  .select('*')
  .range(page * pageSize, (page + 1) * pageSize - 1)
```

### Complete Pagination Example

```typescript
async function getPaginatedUsers(page: number = 0, pageSize: number = 20) {
  const { data, count } = await client.from('users')
    .select('id, name, email, created_at')
    .order('created_at', { ascending: false })
    .range(page * pageSize, (page + 1) * pageSize - 1)

  return {
    users: data,
    totalCount: count,
    totalPages: Math.ceil((count || 0) / pageSize),
    currentPage: page
  }
}
```

## RPC (PostgreSQL Functions)

Call PostgreSQL functions directly from your SDK:

### Simple RPC Call

```typescript
// Call function without parameters
const { data, error } = await client.rpc('get_user_count')

if (error) {
  console.error('RPC failed:', error)
} else {
  console.log('User count:', data)
}
```

### RPC with Parameters

```typescript
// Call function with parameters
const { data, error } = await client.rpc('calculate_discount', {
  product_id: 123,
  coupon_code: 'SAVE20'
})

// Complex example
const { data, error } = await client.rpc('search_products', {
  query: 'laptop',
  min_price: 500,
  max_price: 2000,
  category: 'electronics'
})
```

### Type-Safe RPC

```typescript
interface DiscountResult {
  original_price: number
  discounted_price: number
  savings: number
}

const { data, error } = await client.rpc<DiscountResult>('calculate_discount', {
  product_id: 123,
  coupon_code: 'SAVE20'
})

if (data) {
  console.log(`You save $${data.savings}!`)
}
```

## Error Handling

Always handle errors when performing database operations:

```typescript
try {
  const { data, error } = await client.from('users')
    .insert({ name: 'John', email: 'john@example.com' })

  if (error) {
    console.error('Database error:', error)
    return
  }

  console.log('Success:', data)
} catch (error) {
  console.error('Network or unexpected error:', error)
}
```

## TypeScript Support

The SDK is fully typed. Define your table schemas for type safety:

```typescript
interface User {
  id: number
  name: string
  email: string
  age: number
  created_at: string
}

// Type-safe query
const { data } = await client.from<User>('users')
  .select('id, name, email')

// TypeScript knows data is User[]
data?.forEach(user => {
  console.log(user.name) // ✅ Type-safe
  console.log(user.nonexistent) // ❌ TypeScript error
})
```

## Next Steps

- [React Hooks](./react-hooks.md) - Use these database operations in React
- [API Reference](../../api/sdk/) - Complete API documentation
