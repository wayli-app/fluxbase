# React Hooks

The `@fluxbase/sdk-react` package provides React hooks built on top of [TanStack Query (React Query)](https://tanstack.com/query) for seamless integration with React applications.

## Installation

```bash
npm install @fluxbase/sdk @fluxbase/sdk-react @tanstack/react-query
```

## Setup

### 1. Create the Fluxbase Client

```typescript
// lib/fluxbase.ts
import { createClient } from '@fluxbase/sdk'

export const fluxbase = createClient({
  url: process.env.NEXT_PUBLIC_FLUXBASE_URL || 'http://localhost:8080',
  auth: {
    autoRefresh: true,
    persist: true
  }
})
```

### 2. Wrap Your App with FluxbaseProvider

```tsx
// app.tsx or _app.tsx (Next.js)
import { FluxbaseProvider } from '@fluxbase/sdk-react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fluxbase } from './lib/fluxbase'

const queryClient = new QueryClient()

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <FluxbaseProvider client={fluxbase}>
        <YourApp />
      </FluxbaseProvider>
    </QueryClientProvider>
  )
}
```

## Database Hooks

### useFluxbaseQuery

Query your database with automatic caching, refetching, and loading states.

```tsx
import { useFluxbaseQuery } from '@fluxbase/sdk-react'

interface User {
  id: number
  name: string
  email: string
}

function UsersList() {
  const { data: users, isLoading, error, refetch } = useFluxbaseQuery<User>({
    table: 'users',
    select: 'id, name, email',
    orderBy: { column: 'name', ascending: true }
  })

  if (isLoading) return <div>Loading...</div>
  if (error) return <div>Error: {error.message}</div>

  return (
    <div>
      <button onClick={() => refetch()}>Refresh</button>
      <ul>
        {users?.map(user => (
          <li key={user.id}>{user.name} - {user.email}</li>
        ))}
      </ul>
    </div>
  )
}
```

### With Filters

```tsx
function ActiveProducts() {
  const { data: products } = useFluxbaseQuery({
    table: 'products',
    select: '*',
    filters: [
      { column: 'status', operator: 'eq', value: 'active' },
      { column: 'price', operator: 'gt', value: 50 }
    ],
    orderBy: { column: 'price', ascending: false },
    limit: 20
  })

  return (
    <div>
      {products?.map(product => (
        <ProductCard key={product.id} product={product} />
      ))}
    </div>
  )
}
```

### With Pagination

```tsx
function PaginatedUsers() {
  const [page, setPage] = useState(0)
  const pageSize = 20

  const { data: users, isLoading } = useFluxbaseQuery({
    table: 'users',
    select: '*',
    limit: pageSize,
    offset: page * pageSize,
    orderBy: { column: 'created_at', ascending: false }
  })

  return (
    <div>
      {isLoading ? <Spinner /> : <UsersList users={users} />}
      <Pagination
        currentPage={page}
        onPageChange={setPage}
      />
    </div>
  )
}
```

### React Query Options

Pass any React Query options to customize behavior:

```tsx
const { data } = useFluxbaseQuery({
  table: 'products',
  select: '*'
}, {
  // React Query options
  staleTime: 5 * 60 * 1000, // Consider data fresh for 5 minutes
  cacheTime: 10 * 60 * 1000, // Keep in cache for 10 minutes
  refetchOnWindowFocus: false, // Don't refetch when window regains focus
  retry: 3, // Retry failed requests 3 times
  onSuccess: (data) => {
    console.log('Query succeeded:', data)
  },
  onError: (error) => {
    console.error('Query failed:', error)
  }
})
```

## Mutation Hooks

### useFluxbaseMutation

For insert, update, and delete operations with automatic cache invalidation.

#### Insert

```tsx
import { useFluxbaseMutation } from '@fluxbase/sdk-react'

interface CreateUserInput {
  name: string
  email: string
}

function CreateUserForm() {
  const createUser = useFluxbaseMutation<User, CreateUserInput>({
    table: 'users',
    operation: 'insert',
    invalidateQueries: ['users'] // Invalidate users queries on success
  })

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)

    try {
      await createUser.mutateAsync({
        name: formData.get('name') as string,
        email: formData.get('email') as string
      })
      alert('User created successfully!')
    } catch (error) {
      alert('Failed to create user')
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <input name="name" placeholder="Name" required />
      <input name="email" type="email" placeholder="Email" required />
      <button type="submit" disabled={createUser.isLoading}>
        {createUser.isLoading ? 'Creating...' : 'Create User'}
      </button>
      {createUser.error && <p>Error: {createUser.error.message}</p>}
    </form>
  )
}
```

#### Update

```tsx
interface UpdateProductInput {
  id: number
  price?: number
  stock?: number
}

function ProductEditor({ product }: { product: Product }) {
  const updateProduct = useFluxbaseMutation<Product, UpdateProductInput>({
    table: 'products',
    operation: 'update',
    invalidateQueries: ['products']
  })

  const handlePriceChange = async (newPrice: number) => {
    await updateProduct.mutateAsync({
      id: product.id,
      price: newPrice
    })
  }

  return (
    <div>
      <input
        type="number"
        defaultValue={product.price}
        onBlur={(e) => handlePriceChange(Number(e.target.value))}
      />
      {updateProduct.isLoading && <span>Saving...</span>}
    </div>
  )
}
```

#### Delete

```tsx
function DeleteUserButton({ userId }: { userId: number }) {
  const deleteUser = useFluxbaseMutation({
    table: 'users',
    operation: 'delete',
    invalidateQueries: ['users']
  })

  const handleDelete = async () => {
    if (confirm('Are you sure you want to delete this user?')) {
      await deleteUser.mutateAsync({ id: userId })
    }
  }

  return (
    <button
      onClick={handleDelete}
      disabled={deleteUser.isLoading}
    >
      {deleteUser.isLoading ? 'Deleting...' : 'Delete'}
    </button>
  )
}
```

### Optimistic Updates

```tsx
function TodoList() {
  const queryClient = useQueryClient()

  const toggleTodo = useFluxbaseMutation({
    table: 'todos',
    operation: 'update',
    onMutate: async (variables) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries(['todos'])

      // Snapshot previous value
      const previousTodos = queryClient.getQueryData(['todos'])

      // Optimistically update
      queryClient.setQueryData(['todos'], (old: any) =>
        old?.map((todo: any) =>
          todo.id === variables.id
            ? { ...todo, completed: !todo.completed }
            : todo
        )
      )

      return { previousTodos }
    },
    onError: (err, variables, context) => {
      // Rollback on error
      queryClient.setQueryData(['todos'], context?.previousTodos)
    },
    onSettled: () => {
      // Refetch after mutation
      queryClient.invalidateQueries(['todos'])
    }
  })

  return <div>...</div>
}
```

## RPC Hooks

Call PostgreSQL functions from your React components.

### useRPC

Query-style RPC hook with automatic caching:

```tsx
import { useRPC } from '@fluxbase/sdk-react'

interface StatsResult {
  total_users: number
  active_users: number
  total_revenue: number
}

function Dashboard() {
  const { data: stats, isLoading } = useRPC<StatsResult>('get_dashboard_stats')

  if (isLoading) return <Spinner />

  return (
    <div>
      <StatCard title="Total Users" value={stats?.total_users} />
      <StatCard title="Active Users" value={stats?.active_users} />
      <StatCard title="Revenue" value={stats?.total_revenue} />
    </div>
  )
}
```

### With Parameters

```typescript
function ProductDiscounts({ productId }: { productId: number }) {
  const { data: discount } = useRPC('calculate_discount', {
    product_id: productId,
    coupon_code: 'SAVE20'
  })

  return <div>You save: ${discount?.savings}</div>
}
```

### useRPCMutation

For RPC calls that modify data:

```tsx
import { useRPCMutation } from '@fluxbase/sdk-react'

function BulkImport() {
  const importProducts = useRPCMutation('import_products_from_csv')

  const handleImport = async (file: File) => {
    const reader = new FileReader()
    reader.onload = async (e) => {
      const csvData = e.target?.result as string
      try {
        const result = await importProducts.mutateAsync({ csv_data: csvData })
        alert(`Imported ${result.count} products`)
      } catch (error) {
        alert('Import failed')
      }
    }
    reader.readAsText(file)
  }

  return (
    <div>
      <input
        type="file"
        accept=".csv"
        onChange={(e) => e.target.files?.[0] && handleImport(e.target.files[0])}
      />
      {importProducts.isLoading && <Spinner />}
    </div>
  )
}
```

### useRPCBatch

Call multiple RPC functions in parallel:

```tsx
import { useRPCBatch } from '@fluxbase/sdk-react'

function Analytics() {
  const { data, isLoading } = useRPCBatch([
    { name: 'get_user_count' },
    { name: 'get_revenue_total' },
    { name: 'get_order_count', params: { status: 'completed' } },
    { name: 'get_top_products', params: { limit: 5 } }
  ])

  if (isLoading) return <Spinner />

  const [userCount, revenue, orderCount, topProducts] = data || []

  return (
    <div>
      <h2>Analytics Dashboard</h2>
      <p>Users: {userCount}</p>
      <p>Revenue: ${revenue}</p>
      <p>Orders: {orderCount}</p>
      <h3>Top Products:</h3>
      <ul>
        {topProducts?.map((product: any) => (
          <li key={product.id}>{product.name}</li>
        ))}
      </ul>
    </div>
  )
}
```

## Real-time Hooks

### useRealtime

Subscribe to real-time database changes:

```tsx
import { useRealtime } from '@fluxbase/sdk-react'

function LiveOrdersList() {
  const [orders, setOrders] = useState<Order[]>([])

  useRealtime({
    table: 'orders',
    event: '*', // Listen to all events (INSERT, UPDATE, DELETE)
    callback: (payload) => {
      console.log('Real-time event:', payload)

      if (payload.eventType === 'INSERT') {
        setOrders(prev => [...prev, payload.new as Order])
      } else if (payload.eventType === 'UPDATE') {
        setOrders(prev =>
          prev.map(order =>
            order.id === payload.new.id ? payload.new as Order : order
          )
        )
      } else if (payload.eventType === 'DELETE') {
        setOrders(prev =>
          prev.filter(order => order.id !== payload.old.id)
        )
      }
    }
  })

  return (
    <div>
      <h2>Live Orders</h2>
      {orders.map(order => (
        <OrderCard key={order.id} order={order} />
      ))}
    </div>
  )
}
```

### Filter Real-time Events

```tsx
function UserNotifications({ userId }: { userId: number }) {
  useRealtime({
    table: 'notifications',
    event: 'INSERT',
    filter: `user_id=eq.${userId}`, // Only get notifications for this user
    callback: (payload) => {
      showToast(payload.new.message)
    }
  })

  return <div>...</div>
}
```

## Authentication Hooks

### useAuth

Access authentication state and methods:

```tsx
import { useAuth } from '@fluxbase/sdk-react'

function AuthButtons() {
  const { user, isAuthenticated, login, logout, isLoading } = useAuth()

  if (isLoading) return <Spinner />

  if (isAuthenticated) {
    return (
      <div>
        <p>Welcome, {user?.email}</p>
        <button onClick={logout}>Logout</button>
      </div>
    )
  }

  return <button onClick={() => login({ email: 'user@example.com', password: 'pass' })}>
    Login
  </button>
}
```

## Storage Hooks

### useStorage

Upload and manage files:

```tsx
import { useStorage } from '@fluxbase/sdk-react'

function FileUploader() {
  const { upload, isUploading, error } = useStorage()

  const handleUpload = async (file: File) => {
    try {
      const result = await upload({
        bucket: 'avatars',
        path: `users/${userId}/${file.name}`,
        file
      })
      console.log('Uploaded:', result.url)
    } catch (err) {
      console.error('Upload failed:', err)
    }
  }

  return (
    <div>
      <input
        type="file"
        onChange={(e) => e.target.files?.[0] && handleUpload(e.target.files[0])}
        disabled={isUploading}
      />
      {isUploading && <Spinner />}
      {error && <p>Error: {error.message}</p>}
    </div>
  )
}
```

## Advanced Patterns

### Dependent Queries

```tsx
function UserProfile({ userId }: { userId: number }) {
  // First query: Get user
  const { data: user } = useFluxbaseQuery({
    table: 'users',
    select: '*',
    filters: [{ column: 'id', operator: 'eq', value: userId }]
  })

  // Second query: Get user's posts (depends on first query)
  const { data: posts } = useFluxbaseQuery({
    table: 'posts',
    select: '*',
    filters: [{ column: 'author_id', operator: 'eq', value: userId }]
  }, {
    enabled: !!user // Only run if user is loaded
  })

  return <div>...</div>
}
```

### Infinite Scrolling

```tsx
import { useInfiniteQuery } from '@tanstack/react-query'
import { useFluxbaseClient } from '@fluxbase/sdk-react'

function InfiniteProductList() {
  const client = useFluxbaseClient()

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage
  } = useInfiniteQuery({
    queryKey: ['products-infinite'],
    queryFn: async ({ pageParam = 0 }) => {
      const { data } = await client.from('products')
        .select('*')
        .order('created_at', { ascending: false })
        .range(pageParam, pageParam + 19)
        .execute()
      return data
    },
    getNextPageParam: (lastPage, pages) => {
      return lastPage.length === 20 ? pages.length * 20 : undefined
    }
  })

  return (
    <div>
      {data?.pages.map((page, i) => (
        <div key={i}>
          {page.map((product: any) => (
            <ProductCard key={product.id} product={product} />
          ))}
        </div>
      ))}
      <button
        onClick={() => fetchNextPage()}
        disabled={!hasNextPage || isFetchingNextPage}
      >
        {isFetchingNextPage ? 'Loading...' : 'Load More'}
      </button>
    </div>
  )
}
```

## Next Steps

- [Database Operations](./database.md) - Learn more about queries and filters
- [API Reference](/api/sdk-react/) - Complete React hooks API documentation
- [React Query Docs](https://tanstack.com/query) - Learn more about React Query
