---
editUrl: false
next: false
prev: false
title: "QueryBuilder"
---

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

## Implements

- `PromiseLike`\<[`PostgrestResponse`](/api/sdk/interfaces/postgrestresponse/)\<`T`\>\>

## Constructors

### new QueryBuilder()

> **new QueryBuilder**\<`T`\>(`fetch`, `table`, `schema`?): [`QueryBuilder`](/api/sdk/classes/querybuilder/)\<`T`\>

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |
| `table` | `string` |
| `schema`? | `string` |

#### Returns

[`QueryBuilder`](/api/sdk/classes/querybuilder/)\<`T`\>

## Aggregation

### avg()

> **avg**(`column`): `this`

Calculate the average of a numeric column

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `column` | `string` | Column to average |

#### Returns

`this`

Query builder for chaining

#### Example

```typescript
// Average price
const { data } = await client.from('products').avg('price').execute()
// Returns: { avg_price: 129.99 }

// Average by category
const { data } = await client.from('products')
  .avg('price')
  .groupBy('category')
  .execute()
```

***

### count()

> **count**(`column`): `this`

Count rows or a specific column

#### Parameters

| Parameter | Type | Default value | Description |
| ------ | ------ | ------ | ------ |
| `column` | `string` | `"*"` | Column to count (default: '*' for row count) |

#### Returns

`this`

Query builder for chaining

#### Example

```typescript
// Count all rows
const { data } = await client.from('users').count().execute()
// Returns: { count: 150 }

// Count non-null values in a column
const { data } = await client.from('orders').count('completed_at').execute()

// Count with grouping
const { data } = await client.from('products')
  .count('*')
  .groupBy('category')
  .execute()
// Returns: [{ category: 'electronics', count: 45 }, { category: 'books', count: 23 }]
```

***

### groupBy()

> **groupBy**(`columns`): `this`

Group results by one or more columns (for use with aggregations)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `columns` | `string` \| `string`[] | Column name(s) to group by |

#### Returns

`this`

Query builder for chaining

#### Example

```typescript
// Group by single column
const { data } = await client.from('orders')
  .count('*')
  .groupBy('status')
  .execute()

// Group by multiple columns
const { data } = await client.from('sales')
  .sum('amount')
  .groupBy(['region', 'product_category'])
  .execute()
```

***

### max()

> **max**(`column`): `this`

Find the maximum value in a column

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `column` | `string` | Column to find maximum value |

#### Returns

`this`

Query builder for chaining

#### Example

```typescript
// Find highest price
const { data } = await client.from('products').max('price').execute()
// Returns: { max_price: 1999.99 }

// Find most recent order
const { data } = await client.from('orders').max('created_at').execute()
```

***

### min()

> **min**(`column`): `this`

Find the minimum value in a column

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `column` | `string` | Column to find minimum value |

#### Returns

`this`

Query builder for chaining

#### Example

```typescript
// Find lowest price
const { data } = await client.from('products').min('price').execute()
// Returns: { min_price: 9.99 }

// Find earliest date
const { data } = await client.from('orders').min('created_at').execute()
```

***

### sum()

> **sum**(`column`): `this`

Calculate the sum of a numeric column

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `column` | `string` | Column to sum |

#### Returns

`this`

Query builder for chaining

#### Example

```typescript
// Sum all prices
const { data } = await client.from('products').sum('price').execute()
// Returns: { sum_price: 15420.50 }

// Sum by category
const { data } = await client.from('orders')
  .sum('total')
  .groupBy('status')
  .execute()
// Returns: [{ status: 'completed', sum_total: 12500 }, { status: 'pending', sum_total: 3200 }]
```

## Batch Operations

### deleteMany()

> **deleteMany**(): `Promise`\<[`PostgrestResponse`](/api/sdk/interfaces/postgrestresponse/)\<`null`\>\>

Delete multiple rows matching the filters (batch delete)

Deletes all rows that match the current query filters.
This is a convenience method that explicitly shows intent for batch operations.

#### Returns

`Promise`\<[`PostgrestResponse`](/api/sdk/interfaces/postgrestresponse/)\<`null`\>\>

Promise confirming deletion

#### Example

```typescript
// Delete all inactive users
await client.from('users')
  .eq('active', false)
  .deleteMany()

// Delete old logs
await client.from('logs')
  .lt('created_at', '2024-01-01')
  .deleteMany()
```

***

### insertMany()

> **insertMany**(`rows`): `Promise`\<[`PostgrestResponse`](/api/sdk/interfaces/postgrestresponse/)\<`T`\>\>

Insert multiple rows in a single request (batch insert)

This is a convenience method that explicitly shows intent for batch operations.
Internally calls `insert()` with an array.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `rows` | `Partial`\<`T`\>[] | Array of row objects to insert |

#### Returns

`Promise`\<[`PostgrestResponse`](/api/sdk/interfaces/postgrestresponse/)\<`T`\>\>

Promise with the inserted rows

#### Example

```typescript
// Insert multiple users at once
const { data } = await client.from('users').insertMany([
  { name: 'Alice', email: 'alice@example.com' },
  { name: 'Bob', email: 'bob@example.com' },
  { name: 'Charlie', email: 'charlie@example.com' }
])
```

***

### updateMany()

> **updateMany**(`data`): `Promise`\<[`PostgrestResponse`](/api/sdk/interfaces/postgrestresponse/)\<`T`\>\>

Update multiple rows matching the filters (batch update)

Updates all rows that match the current query filters.
This is a convenience method that explicitly shows intent for batch operations.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `data` | `Partial`\<`T`\> | Data to update matching rows with |

#### Returns

`Promise`\<[`PostgrestResponse`](/api/sdk/interfaces/postgrestresponse/)\<`T`\>\>

Promise with the updated rows

#### Example

```typescript
// Apply discount to all electronics
const { data } = await client.from('products')
  .eq('category', 'electronics')
  .updateMany({ discount: 10, updated_at: new Date() })

// Mark all pending orders as processing
const { data } = await client.from('orders')
  .eq('status', 'pending')
  .updateMany({ status: 'processing' })
```

## Other

### and()

> **and**(`filters`): `this`

Apply AND logic to filters (Supabase-compatible)
Groups multiple conditions that must all be true

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `filters` | `string` |

#### Returns

`this`

#### Examples

```ts
and('status.eq.active,verified.eq.true')
```

```ts
and('age.gte.18,age.lte.65')
```

***

### containedBy()

> **containedBy**(`column`, `value`): `this`

Check if column is contained by value (Supabase-compatible)
For arrays and JSONB

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `value` | `unknown` |

#### Returns

`this`

#### Example

```ts
containedBy('tags', '["news","update"]')
```

***

### contains()

> **contains**(`column`, `value`): `this`

Contains (for arrays and JSONB)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `value` | `unknown` |

#### Returns

`this`

***

### crosses()

> **crosses**(`column`, `geojson`): `this`

Check if geometries cross (PostGIS ST_Crosses)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `column` | `string` | Column containing geometry/geography data |
| `geojson` | `unknown` | GeoJSON object to test crossing |

#### Returns

`this`

#### Example

```ts
crosses('road', { type: 'LineString', coordinates: [[...]] })
```

***

### delete()

> **delete**(): `this`

Delete rows matching the filters

#### Returns

`this`

***

### eq()

> **eq**(`column`, `value`): `this`

Equal to

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `value` | `unknown` |

#### Returns

`this`

***

### execute()

> **execute**(): `Promise`\<[`PostgrestResponse`](/api/sdk/interfaces/postgrestresponse/)\<`T`\>\>

Execute the query and return results

#### Returns

`Promise`\<[`PostgrestResponse`](/api/sdk/interfaces/postgrestresponse/)\<`T`\>\>

***

### filter()

> **filter**(`column`, `operator`, `value`): `this`

Generic filter method using PostgREST syntax (Supabase-compatible)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `operator` | [`FilterOperator`](/api/sdk/type-aliases/filteroperator/) |
| `value` | `unknown` |

#### Returns

`this`

#### Examples

```ts
filter('name', 'in', '("Han","Yoda")')
```

```ts
filter('age', 'gte', '18')
```

***

### gt()

> **gt**(`column`, `value`): `this`

Greater than

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `value` | `unknown` |

#### Returns

`this`

***

### gte()

> **gte**(`column`, `value`): `this`

Greater than or equal to

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `value` | `unknown` |

#### Returns

`this`

***

### ilike()

> **ilike**(`column`, `pattern`): `this`

Pattern matching (case-insensitive)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `pattern` | `string` |

#### Returns

`this`

***

### in()

> **in**(`column`, `values`): `this`

Check if value is in array

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `values` | `unknown`[] |

#### Returns

`this`

***

### insert()

> **insert**(`data`): `this`

Insert a single row or multiple rows

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `data` | `Partial`\<`T`\> \| `Partial`\<`T`\>[] |

#### Returns

`this`

***

### intersects()

> **intersects**(`column`, `geojson`): `this`

Check if geometries intersect (PostGIS ST_Intersects)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `column` | `string` | Column containing geometry/geography data |
| `geojson` | `unknown` | GeoJSON object to test intersection with |

#### Returns

`this`

#### Example

```ts
intersects('location', { type: 'Point', coordinates: [-122.4, 37.8] })
```

***

### is()

> **is**(`column`, `value`): `this`

Check if value is null or not null

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `value` | `null` \| `boolean` |

#### Returns

`this`

***

### like()

> **like**(`column`, `pattern`): `this`

Pattern matching (case-sensitive)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `pattern` | `string` |

#### Returns

`this`

***

### limit()

> **limit**(`count`): `this`

Limit number of rows returned

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `count` | `number` |

#### Returns

`this`

***

### lt()

> **lt**(`column`, `value`): `this`

Less than

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `value` | `unknown` |

#### Returns

`this`

***

### lte()

> **lte**(`column`, `value`): `this`

Less than or equal to

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `value` | `unknown` |

#### Returns

`this`

***

### match()

> **match**(`conditions`): `this`

Match multiple columns with exact values (Supabase-compatible)
Shorthand for multiple .eq() calls

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `conditions` | `Record`\<`string`, `unknown`\> |

#### Returns

`this`

#### Example

```ts
match({ id: 1, status: 'active', role: 'admin' })
```

***

### maybeSingle()

> **maybeSingle**(): `this`

Return a single row or null (adds limit(1))
Does not error if no rows found (Supabase-compatible)

#### Returns

`this`

#### Example

```typescript
// Returns null instead of erroring when no row exists
const { data, error } = await client
  .from('users')
  .select('*')
  .eq('id', 999)
  .maybeSingle()
// data will be null if no row found
```

***

### neq()

> **neq**(`column`, `value`): `this`

Not equal to

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `value` | `unknown` |

#### Returns

`this`

***

### not()

> **not**(`column`, `operator`, `value`): `this`

Negate a filter condition (Supabase-compatible)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `operator` | [`FilterOperator`](/api/sdk/type-aliases/filteroperator/) |
| `value` | `unknown` |

#### Returns

`this`

#### Examples

```ts
not('status', 'eq', 'deleted')
```

```ts
not('completed_at', 'is', null)
```

***

### offset()

> **offset**(`count`): `this`

Skip rows

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `count` | `number` |

#### Returns

`this`

***

### or()

> **or**(`filters`): `this`

Apply OR logic to filters (Supabase-compatible)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `filters` | `string` |

#### Returns

`this`

#### Examples

```ts
or('status.eq.active,status.eq.pending')
```

```ts
or('id.eq.2,name.eq.Han')
```

***

### order()

> **order**(`column`, `options`?): `this`

Order results

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `options`? | `object` |
| `options.ascending`? | `boolean` |
| `options.nullsFirst`? | `boolean` |

#### Returns

`this`

***

### overlaps()

> **overlaps**(`column`, `value`): `this`

Check if arrays have common elements (Supabase-compatible)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `value` | `unknown` |

#### Returns

`this`

#### Example

```ts
overlaps('tags', '["news","sports"]')
```

***

### range()

> **range**(`from`, `to`): `this`

Range selection (pagination)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `from` | `number` |
| `to` | `number` |

#### Returns

`this`

***

### select()

> **select**(`columns`, `options`?): `this`

Select columns to return

#### Parameters

| Parameter | Type | Default value |
| ------ | ------ | ------ |
| `columns` | `string` | `"*"` |
| `options`? | `SelectOptions` | `undefined` |

#### Returns

`this`

#### Examples

```ts
select('*')
```

```ts
select('id, name, email')
```

```ts
select('id, name, posts(title, content)')
```

```ts
select('*', { count: 'exact' }) // Get exact count
```

```ts
select('*', { count: 'exact', head: true }) // Get count only (no data)
```

***

### single()

> **single**(): `this`

Return a single row (adds limit(1))
Errors if no rows found

#### Returns

`this`

***

### stContains()

> **stContains**(`column`, `geojson`): `this`

Check if geometry A contains geometry B (PostGIS ST_Contains)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `column` | `string` | Column containing geometry/geography data |
| `geojson` | `unknown` | GeoJSON object to test containment |

#### Returns

`this`

#### Example

```ts
contains('region', { type: 'Point', coordinates: [-122.4, 37.8] })
```

***

### stOverlaps()

> **stOverlaps**(`column`, `geojson`): `this`

Check if geometries spatially overlap (PostGIS ST_Overlaps)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `column` | `string` | Column containing geometry/geography data |
| `geojson` | `unknown` | GeoJSON object to test overlap |

#### Returns

`this`

#### Example

```ts
stOverlaps('area', { type: 'Polygon', coordinates: [[...]] })
```

***

### textSearch()

> **textSearch**(`column`, `query`): `this`

Full-text search

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `column` | `string` |
| `query` | `string` |

#### Returns

`this`

***

### then()

> **then**\<`TResult1`, `TResult2`\>(`onfulfilled`?, `onrejected`?): `PromiseLike`\<`TResult1` \| `TResult2`\>

Make QueryBuilder awaitable (implements PromiseLike)
This allows using `await client.from('table').select()` without calling `.execute()`

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `TResult1` | [`PostgrestResponse`](/api/sdk/interfaces/postgrestresponse/)\<`T`\> |
| `TResult2` | `never` |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `onfulfilled`? | `null` \| (`value`) => `TResult1` \| `PromiseLike`\<`TResult1`\> |
| `onrejected`? | `null` \| (`reason`) => `TResult2` \| `PromiseLike`\<`TResult2`\> |

#### Returns

`PromiseLike`\<`TResult1` \| `TResult2`\>

#### Example

```typescript
// Without .execute() (new way)
const { data } = await client.from('users').select('*')

// With .execute() (old way, still supported)
const { data } = await client.from('users').select('*').execute()
```

#### Implementation of

`PromiseLike.then`

***

### throwOnError()

> **throwOnError**(): `Promise`\<`T`\>

Execute the query and throw an error if one occurs (Supabase-compatible)
Returns the data directly instead of { data, error } wrapper

#### Returns

`Promise`\<`T`\>

#### Throws

If the query fails or returns an error

#### Example

```typescript
// Throws error instead of returning { data, error }
try {
  const user = await client
    .from('users')
    .select('*')
    .eq('id', 1)
    .single()
    .throwOnError()
} catch (error) {
  console.error('Query failed:', error)
}
```

***

### touches()

> **touches**(`column`, `geojson`): `this`

Check if geometries touch (PostGIS ST_Touches)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `column` | `string` | Column containing geometry/geography data |
| `geojson` | `unknown` | GeoJSON object to test touching |

#### Returns

`this`

#### Example

```ts
touches('boundary', { type: 'LineString', coordinates: [[...]] })
```

***

### update()

> **update**(`data`): `this`

Update rows matching the filters

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `data` | `Partial`\<`T`\> |

#### Returns

`this`

***

### upsert()

> **upsert**(`data`, `options`?): `Promise`\<[`PostgrestResponse`](/api/sdk/interfaces/postgrestresponse/)\<`T`\>\>

Upsert (insert or update) rows (Supabase-compatible)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `data` | `Partial`\<`T`\> \| `Partial`\<`T`\>[] | Row(s) to upsert |
| `options`? | [`UpsertOptions`](/api/sdk/interfaces/upsertoptions/) | Upsert options (onConflict, ignoreDuplicates, defaultToNull) |

#### Returns

`Promise`\<[`PostgrestResponse`](/api/sdk/interfaces/postgrestresponse/)\<`T`\>\>

***

### within()

> **within**(`column`, `geojson`): `this`

Check if geometry A is within geometry B (PostGIS ST_Within)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `column` | `string` | Column containing geometry/geography data |
| `geojson` | `unknown` | GeoJSON object to test containment within |

#### Returns

`this`

#### Example

```ts
within('point', { type: 'Polygon', coordinates: [[...]] })
```
