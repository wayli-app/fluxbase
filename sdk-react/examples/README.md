# React Admin Hooks Examples

This directory contains complete, production-ready examples demonstrating the Fluxbase React admin hooks.

## Examples

### AdminDashboard.tsx

A complete admin dashboard application demonstrating all admin hooks in action.

**Features:**
- Admin authentication with protected routes
- User management with pagination and search
- Real-time statistics dashboard
- Modern UI with Tailwind CSS
- Full TypeScript support

**Hooks demonstrated:**
- `useAdminAuth()` - Authentication state management
- `useUsers()` - User CRUD operations with pagination
- `useAPIKeys()` - API key management
- `useWebhooks()` - Webhook configuration
- `useAppSettings()` - Application settings
- `useSystemSettings()` - System settings

**Run the example:**

```bash
# Install dependencies
npm install @fluxbase/sdk @fluxbase/sdk-react @tanstack/react-query

# Copy AdminDashboard.tsx to your project
# Add Tailwind CSS to your project (optional, for styling)
```

**Basic usage:**

```tsx
import { createClient } from '@fluxbase/sdk'
import { FluxbaseProvider } from '@fluxbase/sdk-react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import AdminDashboard from './AdminDashboard'

const client = createClient({ url: 'http://localhost:8080' })
const queryClient = new QueryClient()

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <FluxbaseProvider client={client}>
        <AdminDashboard />
      </FluxbaseProvider>
    </QueryClientProvider>
  )
}
```

## Key Patterns

### 1. Authentication Flow

```tsx
const { isAuthenticated, isLoading, login, logout } = useAdminAuth({
  autoCheck: true // Automatically check auth status on mount
})

if (isLoading) return <LoadingSpinner />
if (!isAuthenticated) return <LoginPage />
return <Dashboard />
```

### 2. Data Fetching with Pagination

```tsx
const [page, setPage] = useState(0)
const limit = 20

const { users, total, isLoading } = useUsers({
  autoFetch: true,
  limit,
  offset: page * limit
})
```

### 3. Search and Filters

```tsx
const [searchEmail, setSearchEmail] = useState('')
const [roleFilter, setRoleFilter] = useState<'admin' | 'user' | ''>('')

const { users, refetch } = useUsers({
  autoFetch: true,
  email: searchEmail || undefined,
  role: roleFilter || undefined
})

// Refetch when filters change
useEffect(() => {
  refetch()
}, [searchEmail, roleFilter, refetch])
```

### 4. Optimistic Updates

All mutation functions automatically refetch data:

```tsx
const { users, inviteUser, deleteUser } = useUsers({ autoFetch: true })

// These automatically refetch the user list after success
await inviteUser('new@example.com', 'user')
await deleteUser(userId)
// users state is now updated
```

### 5. Error Handling

```tsx
const { data, error, isLoading, refetch } = useUsers({ autoFetch: true })

if (isLoading) return <LoadingSpinner />
if (error) return <ErrorMessage error={error} onRetry={refetch} />
return <DataDisplay data={data} />
```

## Styling

The example uses Tailwind CSS utility classes, but you can easily adapt it to your preferred styling solution:

- **CSS Modules**: Replace `className` with module imports
- **Styled Components**: Replace elements with styled components
- **Material-UI**: Use MUI components instead of native HTML
- **Chakra UI**: Use Chakra components for rapid development

## TypeScript

All examples are fully typed. The hooks provide complete type safety:

```tsx
import type { EnrichedUser, APIKey, Webhook } from '@fluxbase/sdk-react'

const { users }: { users: EnrichedUser[] } = useUsers({ autoFetch: true })
const { keys }: { keys: APIKey[] } = useAPIKeys({ autoFetch: true })
```

## Next Steps

1. **Customize the UI** - Adapt the styling to match your design system
2. **Add more features** - Implement API keys, webhooks, and settings tabs
3. **Add validation** - Add form validation for user inputs
4. **Add loading states** - Improve loading and error states
5. **Add notifications** - Show success/error toasts for mutations
6. **Add search** - Enhance search with debouncing and advanced filters

## Documentation

- [Complete Admin Hooks Guide](../README-ADMIN.md) - Full API reference and examples
- [React Hooks Documentation](../../docs/docs/sdks/react-hooks.md) - Core hooks documentation
- [Admin API Documentation](../../docs/docs/sdk/admin.md) - Admin operations reference

## License

MIT
