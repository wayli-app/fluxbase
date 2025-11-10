# Database Operations

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
const { data: user } = await client.from('users')
  .eq('id', 123)
  .single()
```

### Nested Relations

```typescript
// Select with related data
const { data } = await client.from('posts')
  .select('id, title, content, author(id, name, email)')
```

## Filtering

The SDK supports all PostgREST filter operators:

### Equality

```typescript
// Equal to
const { data } = await client.from('products')
  .eq('category', 'electronics')

// Not equal to
const { data } = await client.from('products')
  .neq('status', 'discontinued')
```

### Comparisons

```typescript
// Greater than
const { data } = await client.from('products').gt('price', 100)

// Greater than or equal
const { data } = await client.from('products').gte('rating', 4)

// Less than
const { data } = await client.from('products').lt('stock', 10)

// Less than or equal
const { data } = await client.from('products').lte('price', 500)
```

### Pattern Matching

```typescript
// LIKE (case-sensitive)
const { data } = await client.from('users').like('email', '%@gmail.com')

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
// Insert or update if unique constraint conflict
const { data } = await client.from('users').upsert({
  id: 123, // Will update if ID exists, insert if not
  name: 'Updated Name',
  email: 'updated@example.com'
})
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
