# Fluxbase React Admin Hooks

Comprehensive React hooks for building admin dashboards and management interfaces with Fluxbase.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Admin Authentication](#admin-authentication)
- [User Management](#user-management)
- [API Keys](#api-keys)
- [Webhooks](#webhooks)
- [Settings Management](#settings-management)
- [Complete Examples](#complete-examples)

## Installation

```bash
npm install @fluxbase/sdk @fluxbase/sdk-react @tanstack/react-query
```

## Quick Start

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

## Admin Authentication

### useAdminAuth

Hook for managing admin authentication state.

```tsx
import { useAdminAuth } from '@fluxbase/sdk-react'

function AdminLogin() {
  const { user, isAuthenticated, isLoading, error, login, logout } = useAdminAuth({
    autoCheck: true // Automatically check if admin is authenticated on mount
  })

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await login(email, password)
      // Redirect to admin dashboard
    } catch (err) {
      console.error('Login failed:', err)
    }
  }

  if (isLoading) {
    return <div>Checking authentication...</div>
  }

  if (isAuthenticated) {
    return (
      <div>
        <p>Logged in as: {user?.email}</p>
        <p>Role: {user?.role}</p>
        <button onClick={logout}>Logout</button>
      </div>
    )
  }

  return (
    <form onSubmit={handleLogin}>
      <input
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="Admin email"
        required
      />
      <input
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        placeholder="Password"
        required
      />
      <button type="submit">Login</button>
      {error && <p className="error">{error.message}</p>}
    </form>
  )
}
```

### Protected Admin Routes

```tsx
import { useAdminAuth } from '@fluxbase/sdk-react'
import { Navigate } from 'react-router-dom'

function ProtectedAdminRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAdminAuth({ autoCheck: true })

  if (isLoading) {
    return <div>Loading...</div>
  }

  if (!isAuthenticated) {
    return <Navigate to="/admin/login" replace />
  }

  return <>{children}</>
}

// Usage
<Routes>
  <Route path="/admin/login" element={<AdminLogin />} />
  <Route
    path="/admin/*"
    element={
      <ProtectedAdminRoute>
        <AdminDashboard />
      </ProtectedAdminRoute>
    }
  />
</Routes>
```

## User Management

### useUsers

Hook for managing users with pagination and CRUD operations.

```tsx
import { useUsers } from '@fluxbase/sdk-react'
import { useState } from 'react'

function UserManagement() {
  const [page, setPage] = useState(0)
  const limit = 20

  const {
    users,
    total,
    isLoading,
    error,
    refetch,
    inviteUser,
    updateUserRole,
    deleteUser,
    resetPassword
  } = useUsers({
    autoFetch: true,
    limit,
    offset: page * limit
  })

  const handleInvite = async () => {
    const email = prompt('Enter email:')
    const role = confirm('Admin role?') ? 'admin' : 'user'
    if (email) {
      await inviteUser(email, role)
    }
  }

  const handleRoleChange = async (userId: string, currentRole: string) => {
    const newRole = currentRole === 'admin' ? 'user' : 'admin'
    await updateUserRole(userId, newRole)
  }

  const handleDelete = async (userId: string) => {
    if (confirm('Delete this user?')) {
      await deleteUser(userId)
    }
  }

  const handleResetPassword = async (userId: string) => {
    const newPassword = await resetPassword(userId)
    alert(`New password: ${newPassword}`)
  }

  if (isLoading) return <div>Loading users...</div>
  if (error) return <div>Error: {error.message}</div>

  return (
    <div>
      <div className="header">
        <h2>User Management ({total} users)</h2>
        <button onClick={handleInvite}>Invite User</button>
        <button onClick={refetch}>Refresh</button>
      </div>

      <table>
        <thead>
          <tr>
            <th>Email</th>
            <th>Role</th>
            <th>Status</th>
            <th>Created</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {users.map((user) => (
            <tr key={user.id}>
              <td>{user.email}</td>
              <td>
                <span className={`badge ${user.role}`}>{user.role}</span>
              </td>
              <td>
                <span className={`status ${user.email_confirmed ? 'confirmed' : 'pending'}`}>
                  {user.email_confirmed ? 'Confirmed' : 'Pending'}
                </span>
              </td>
              <td>{new Date(user.created_at).toLocaleDateString()}</td>
              <td>
                <button onClick={() => handleRoleChange(user.id, user.role)}>
                  Toggle Role
                </button>
                <button onClick={() => handleResetPassword(user.id)}>
                  Reset Password
                </button>
                <button onClick={() => handleDelete(user.id)}>Delete</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <div className="pagination">
        <button disabled={page === 0} onClick={() => setPage(page - 1)}>
          Previous
        </button>
        <span>
          Page {page + 1} of {Math.ceil(total / limit)}
        </span>
        <button
          disabled={(page + 1) * limit >= total}
          onClick={() => setPage(page + 1)}
        >
          Next
        </button>
      </div>
    </div>
  )
}
```

### User Search and Filters

```tsx
import { useUsers } from '@fluxbase/sdk-react'
import { useState, useEffect } from 'react'

function UserSearch() {
  const [searchEmail, setSearchEmail] = useState('')
  const [roleFilter, setRoleFilter] = useState<'admin' | 'user' | undefined>()

  const { users, isLoading, refetch } = useUsers({
    autoFetch: true,
    email: searchEmail || undefined,
    role: roleFilter
  })

  // Refetch when filters change
  useEffect(() => {
    refetch()
  }, [searchEmail, roleFilter, refetch])

  return (
    <div>
      <input
        type="text"
        placeholder="Search by email..."
        value={searchEmail}
        onChange={(e) => setSearchEmail(e.target.value)}
      />
      <select value={roleFilter || ''} onChange={(e) => setRoleFilter(e.target.value as any)}>
        <option value="">All Roles</option>
        <option value="admin">Admin</option>
        <option value="user">User</option>
      </select>

      {isLoading ? (
        <div>Searching...</div>
      ) : (
        <ul>
          {users.map((user) => (
            <li key={user.id}>
              {user.email} - {user.role}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
```

## API Keys

### useAPIKeys

Hook for managing API keys.

```tsx
import { useAPIKeys } from '@fluxbase/sdk-react'
import { useState } from 'react'

function APIKeyManagement() {
  const { keys, isLoading, error, createKey, updateKey, revokeKey, deleteKey } = useAPIKeys({
    autoFetch: true
  })

  const [showCreateForm, setShowCreateForm] = useState(false)
  const [newKeyData, setNewKeyData] = useState<{
    name: string
    description: string
    expiresInDays: number
  }>({ name: '', description: '', expiresInDays: 365 })

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const expiresAt = new Date()
      expiresAt.setDate(expiresAt.getDate() + newKeyData.expiresInDays)

      const result = await createKey({
        name: newKeyData.name,
        description: newKeyData.description,
        expires_at: expiresAt.toISOString()
      })

      // Show the key (only time it's visible)
      alert(`API Key created!\n\nKey: ${result.key}\n\nSave this securely - it won't be shown again!`)

      setShowCreateForm(false)
      setNewKeyData({ name: '', description: '', expiresInDays: 365 })
    } catch (err) {
      console.error('Failed to create key:', err)
    }
  }

  const handleRevoke = async (keyId: string) => {
    if (confirm('Revoke this API key? It will immediately stop working.')) {
      await revokeKey(keyId)
    }
  }

  const handleDelete = async (keyId: string) => {
    if (confirm('Permanently delete this API key?')) {
      await deleteKey(keyId)
    }
  }

  if (isLoading) return <div>Loading API keys...</div>
  if (error) return <div>Error: {error.message}</div>

  return (
    <div>
      <div className="header">
        <h2>API Keys ({keys.length})</h2>
        <button onClick={() => setShowCreateForm(!showCreateForm)}>
          {showCreateForm ? 'Cancel' : 'Create New Key'}
        </button>
      </div>

      {showCreateForm && (
        <form onSubmit={handleCreate} className="create-form">
          <h3>Create New API Key</h3>
          <input
            type="text"
            placeholder="Key name (e.g., Backend Service)"
            value={newKeyData.name}
            onChange={(e) => setNewKeyData({ ...newKeyData, name: e.target.value })}
            required
          />
          <textarea
            placeholder="Description (optional)"
            value={newKeyData.description}
            onChange={(e) => setNewKeyData({ ...newKeyData, description: e.target.value })}
          />
          <label>
            Expires in:
            <input
              type="number"
              min="1"
              max="3650"
              value={newKeyData.expiresInDays}
              onChange={(e) => setNewKeyData({ ...newKeyData, expiresInDays: parseInt(e.target.value) })}
            />
            days
          </label>
          <button type="submit">Create Key</button>
        </form>
      )}

      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Description</th>
            <th>Created</th>
            <th>Expires</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {keys.map((key) => (
            <tr key={key.id}>
              <td>{key.name}</td>
              <td>{key.description}</td>
              <td>{new Date(key.created_at).toLocaleDateString()}</td>
              <td>
                {key.expires_at ? (
                  <span className={new Date(key.expires_at) < new Date() ? 'expired' : ''}>
                    {new Date(key.expires_at).toLocaleDateString()}
                  </span>
                ) : (
                  'Never'
                )}
              </td>
              <td>
                <button onClick={() => handleRevoke(key.id)}>Revoke</button>
                <button onClick={() => handleDelete(key.id)}>Delete</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
```

## Webhooks

### useWebhooks

Hook for managing webhooks and monitoring deliveries.

```tsx
import { useWebhooks } from '@fluxbase/sdk-react'
import { useState } from 'react'

function WebhookManagement() {
  const {
    webhooks,
    isLoading,
    error,
    createWebhook,
    updateWebhook,
    deleteWebhook,
    testWebhook,
    getDeliveries,
    retryDelivery
  } = useWebhooks({ autoFetch: true })

  const [showCreateForm, setShowCreateForm] = useState(false)
  const [selectedWebhook, setSelectedWebhook] = useState<string | null>(null)
  const [deliveries, setDeliveries] = useState<any[]>([])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    const formData = new FormData(e.target as HTMLFormElement)

    await createWebhook({
      name: formData.get('name') as string,
      url: formData.get('url') as string,
      events: ['INSERT', 'UPDATE', 'DELETE'],
      table: formData.get('table') as string,
      schema: 'public',
      enabled: true,
      secret: `webhook_secret_${Math.random().toString(36).substring(7)}`
    })

    setShowCreateForm(false)
  }

  const handleTest = async (webhookId: string) => {
    try {
      await testWebhook(webhookId)
      alert('Test webhook sent! Check your endpoint.')
    } catch (err) {
      alert('Test failed: ' + (err as Error).message)
    }
  }

  const handleToggle = async (webhookId: string, currentlyEnabled: boolean) => {
    await updateWebhook(webhookId, { enabled: !currentlyEnabled })
  }

  const viewDeliveries = async (webhookId: string) => {
    const result = await getDeliveries(webhookId, { limit: 20 })
    setDeliveries(result.deliveries)
    setSelectedWebhook(webhookId)
  }

  if (isLoading) return <div>Loading webhooks...</div>
  if (error) return <div>Error: {error.message}</div>

  return (
    <div>
      <div className="header">
        <h2>Webhooks ({webhooks.length})</h2>
        <button onClick={() => setShowCreateForm(!showCreateForm)}>
          {showCreateForm ? 'Cancel' : 'Create Webhook'}
        </button>
      </div>

      {showCreateForm && (
        <form onSubmit={handleCreate} className="create-form">
          <h3>Create New Webhook</h3>
          <input name="name" placeholder="Webhook name" required />
          <input name="url" type="url" placeholder="https://example.com/webhook" required />
          <input name="table" placeholder="Table name (e.g., users)" required />
          <button type="submit">Create</button>
        </form>
      )}

      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>URL</th>
            <th>Table</th>
            <th>Events</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {webhooks.map((webhook) => (
            <tr key={webhook.id}>
              <td>{webhook.name}</td>
              <td className="url">{webhook.url}</td>
              <td>{webhook.schema}.{webhook.table}</td>
              <td>{webhook.events.join(', ')}</td>
              <td>
                <span className={`status ${webhook.enabled ? 'enabled' : 'disabled'}`}>
                  {webhook.enabled ? 'Enabled' : 'Disabled'}
                </span>
              </td>
              <td>
                <button onClick={() => handleToggle(webhook.id, webhook.enabled)}>
                  {webhook.enabled ? 'Disable' : 'Enable'}
                </button>
                <button onClick={() => handleTest(webhook.id)}>Test</button>
                <button onClick={() => viewDeliveries(webhook.id)}>Deliveries</button>
                <button onClick={() => deleteWebhook(webhook.id)}>Delete</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {selectedWebhook && (
        <div className="deliveries">
          <h3>Recent Deliveries</h3>
          <button onClick={() => setSelectedWebhook(null)}>Close</button>
          <table>
            <thead>
              <tr>
                <th>Status</th>
                <th>Response Code</th>
                <th>Attempt</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {deliveries.map((delivery) => (
                <tr key={delivery.id}>
                  <td>
                    <span className={`status ${delivery.status}`}>{delivery.status}</span>
                  </td>
                  <td>{delivery.response_status_code || 'N/A'}</td>
                  <td>{delivery.attempt_count}</td>
                  <td>{new Date(delivery.created_at).toLocaleString()}</td>
                  <td>
                    {delivery.status === 'failed' && (
                      <button onClick={() => retryDelivery(selectedWebhook, delivery.id)}>
                        Retry
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
```

## Settings Management

### useAppSettings

Hook for managing application-wide settings.

```tsx
import { useAppSettings } from '@fluxbase/sdk-react'

function AppSettingsPanel() {
  const { settings, isLoading, error, updateSettings } = useAppSettings({
    autoFetch: true
  })

  const handleToggleFeature = async (feature: string, enabled: boolean) => {
    await updateSettings({
      features: {
        ...settings?.features,
        [feature]: enabled
      }
    })
  }

  const handleUpdateSecurity = async (e: React.FormEvent) => {
    e.preventDefault()
    const formData = new FormData(e.target as HTMLFormElement)

    await updateSettings({
      security: {
        enable_rate_limiting: formData.get('rateLimiting') === 'on',
        rate_limit_requests_per_minute: parseInt(formData.get('rateLimit') as string)
      }
    })
  }

  if (isLoading) return <div>Loading settings...</div>
  if (error) return <div>Error: {error.message}</div>
  if (!settings) return <div>No settings found</div>

  return (
    <div>
      <h2>Application Settings</h2>

      <section>
        <h3>Features</h3>
        <label>
          <input
            type="checkbox"
            checked={settings.features?.enable_realtime ?? false}
            onChange={(e) => handleToggleFeature('enable_realtime', e.target.checked)}
          />
          Enable Realtime
        </label>
        <label>
          <input
            type="checkbox"
            checked={settings.features?.enable_storage ?? false}
            onChange={(e) => handleToggleFeature('enable_storage', e.target.checked)}
          />
          Enable Storage
        </label>
        <label>
          <input
            type="checkbox"
            checked={settings.features?.enable_functions ?? false}
            onChange={(e) => handleToggleFeature('enable_functions', e.target.checked)}
          />
          Enable Functions
        </label>
      </section>

      <section>
        <h3>Security</h3>
        <form onSubmit={handleUpdateSecurity}>
          <label>
            <input
              type="checkbox"
              name="rateLimiting"
              defaultChecked={settings.security?.enable_rate_limiting ?? false}
            />
            Enable Rate Limiting
          </label>
          <label>
            Requests per minute:
            <input
              type="number"
              name="rateLimit"
              defaultValue={settings.security?.rate_limit_requests_per_minute ?? 60}
              min="1"
            />
          </label>
          <button type="submit">Update Security</button>
        </form>
      </section>
    </div>
  )
}
```

### useSystemSettings

Hook for managing system-wide key-value settings.

```tsx
import { useSystemSettings } from '@fluxbase/sdk-react'
import { useState } from 'react'

function SystemSettingsPanel() {
  const { settings, isLoading, error, getSetting, updateSetting, deleteSetting } = useSystemSettings({
    autoFetch: true
  })

  const [editingKey, setEditingKey] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')

  const handleEdit = (key: string, currentValue: any) => {
    setEditingKey(key)
    setEditValue(JSON.stringify(currentValue, null, 2))
  }

  const handleSave = async () => {
    if (!editingKey) return

    try {
      const parsedValue = JSON.parse(editValue)
      await updateSetting(editingKey, { value: parsedValue })
      setEditingKey(null)
    } catch (err) {
      alert('Invalid JSON')
    }
  }

  const handleCreate = async () => {
    const key = prompt('Setting key:')
    const valueStr = prompt('Setting value (JSON):')
    const description = prompt('Description:')

    if (key && valueStr) {
      try {
        const value = JSON.parse(valueStr)
        await updateSetting(key, { value, description: description || undefined })
      } catch (err) {
        alert('Invalid JSON')
      }
    }
  }

  const handleDelete = async (key: string) => {
    if (confirm(`Delete setting "${key}"?`)) {
      await deleteSetting(key)
    }
  }

  if (isLoading) return <div>Loading settings...</div>
  if (error) return <div>Error: {error.message}</div>

  return (
    <div>
      <div className="header">
        <h2>System Settings ({settings.length})</h2>
        <button onClick={handleCreate}>Create Setting</button>
      </div>

      <table>
        <thead>
          <tr>
            <th>Key</th>
            <th>Value</th>
            <th>Description</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {settings.map((setting) => (
            <tr key={setting.key}>
              <td><code>{setting.key}</code></td>
              <td>
                {editingKey === setting.key ? (
                  <textarea
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    rows={5}
                  />
                ) : (
                  <pre>{JSON.stringify(setting.value, null, 2)}</pre>
                )}
              </td>
              <td>{setting.description}</td>
              <td>
                {editingKey === setting.key ? (
                  <>
                    <button onClick={handleSave}>Save</button>
                    <button onClick={() => setEditingKey(null)}>Cancel</button>
                  </>
                ) : (
                  <>
                    <button onClick={() => handleEdit(setting.key, setting.value)}>Edit</button>
                    <button onClick={() => handleDelete(setting.key)}>Delete</button>
                  </>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
```

## Complete Examples

### Full Admin Dashboard

```tsx
import { useAdminAuth, useUsers, useAPIKeys, useWebhooks, useAppSettings } from '@fluxbase/sdk-react'
import { useState } from 'react'

function AdminDashboard() {
  const { user, isAuthenticated, logout } = useAdminAuth({ autoCheck: true })
  const { users, total: totalUsers } = useUsers({ autoFetch: true, limit: 5 })
  const { keys } = useAPIKeys({ autoFetch: true })
  const { webhooks } = useWebhooks({ autoFetch: true })
  const { settings } = useAppSettings({ autoFetch: true })

  const [activeTab, setActiveTab] = useState<'overview' | 'users' | 'keys' | 'webhooks' | 'settings'>('overview')

  if (!isAuthenticated) {
    return <div>Please log in to access admin dashboard</div>
  }

  return (
    <div className="admin-dashboard">
      <header>
        <h1>Admin Dashboard</h1>
        <div className="user-info">
          <span>{user?.email}</span>
          <button onClick={logout}>Logout</button>
        </div>
      </header>

      <nav>
        <button onClick={() => setActiveTab('overview')}>Overview</button>
        <button onClick={() => setActiveTab('users')}>Users</button>
        <button onClick={() => setActiveTab('keys')}>API Keys</button>
        <button onClick={() => setActiveTab('webhooks')}>Webhooks</button>
        <button onClick={() => setActiveTab('settings')}>Settings</button>
      </nav>

      <main>
        {activeTab === 'overview' && (
          <div className="overview">
            <h2>Overview</h2>
            <div className="stats">
              <div className="stat-card">
                <h3>Total Users</h3>
                <p className="stat-value">{totalUsers}</p>
              </div>
              <div className="stat-card">
                <h3>API Keys</h3>
                <p className="stat-value">{keys.length}</p>
              </div>
              <div className="stat-card">
                <h3>Webhooks</h3>
                <p className="stat-value">{webhooks.length}</p>
              </div>
              <div className="stat-card">
                <h3>Realtime</h3>
                <p className="stat-value">
                  {settings?.features?.enable_realtime ? 'Enabled' : 'Disabled'}
                </p>
              </div>
            </div>

            <div className="recent-users">
              <h3>Recent Users</h3>
              <ul>
                {users.slice(0, 5).map((u) => (
                  <li key={u.id}>
                    {u.email} - {u.role}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        )}

        {activeTab === 'users' && <UserManagement />}
        {activeTab === 'keys' && <APIKeyManagement />}
        {activeTab === 'webhooks' && <WebhookManagement />}
        {activeTab === 'settings' && <AppSettingsPanel />}
      </main>
    </div>
  )
}
```

### Multi-Tab Admin Interface

```tsx
import { Tabs, TabList, Tab, TabPanels, TabPanel } from '@reach/tabs'
import {
  useUsers,
  useAPIKeys,
  useWebhooks,
  useAppSettings,
  useSystemSettings
} from '@fluxbase/sdk-react'

function AdminTabs() {
  return (
    <Tabs>
      <TabList>
        <Tab>Users</Tab>
        <Tab>API Keys</Tab>
        <Tab>Webhooks</Tab>
        <Tab>App Settings</Tab>
        <Tab>System Settings</Tab>
      </TabList>

      <TabPanels>
        <TabPanel>
          <UserManagement />
        </TabPanel>
        <TabPanel>
          <APIKeyManagement />
        </TabPanel>
        <TabPanel>
          <WebhookManagement />
        </TabPanel>
        <TabPanel>
          <AppSettingsPanel />
        </TabPanel>
        <TabPanel>
          <SystemSettingsPanel />
        </TabPanel>
      </TabPanels>
    </Tabs>
  )
}
```

### Real-time Updates

All hooks support automatic refetching with the `refetchInterval` option:

```tsx
function LiveUserList() {
  const { users, total } = useUsers({
    autoFetch: true,
    refetchInterval: 5000 // Refetch every 5 seconds
  })

  return (
    <div>
      <h2>Live Users ({total})</h2>
      <ul>
        {users.map((user) => (
          <li key={user.id}>{user.email}</li>
        ))}
      </ul>
      <small>Updates every 5 seconds</small>
    </div>
  )
}
```

## Best Practices

### Error Handling

```tsx
function RobustComponent() {
  const { users, error, isLoading, refetch } = useUsers({ autoFetch: true })

  if (isLoading) {
    return <LoadingSpinner />
  }

  if (error) {
    return (
      <ErrorState
        message={error.message}
        onRetry={refetch}
      />
    )
  }

  return <UserList users={users} />
}
```

### Optimistic Updates

All mutation functions automatically refetch data after successful operations:

```tsx
const { users, inviteUser } = useUsers({ autoFetch: true })

// This will automatically refetch the user list after inviting
await inviteUser('new@example.com', 'user')
// users state is now updated with the new user
```

### Manual Refetch

```tsx
const { users, refetch } = useUsers({ autoFetch: false })

// Manually fetch when needed
useEffect(() => {
  refetch()
}, [someCondition])
```

### Performance Optimization

```tsx
// Don't auto-fetch on mount if data isn't immediately needed
const { users, refetch } = useUsers({ autoFetch: false })

// Fetch only when tab is active
useEffect(() => {
  if (isTabActive) {
    refetch()
  }
}, [isTabActive, refetch])
```

## TypeScript Support

All hooks are fully typed with comprehensive TypeScript interfaces:

```tsx
import type { EnrichedUser, APIKey, Webhook } from '@fluxbase/sdk-react'

function TypedComponent() {
  const { users }: { users: EnrichedUser[] } = useUsers({ autoFetch: true })
  const { keys }: { keys: APIKey[] } = useAPIKeys({ autoFetch: true })
  const { webhooks }: { webhooks: Webhook[] } = useWebhooks({ autoFetch: true })

  // Full type safety
}
```

## API Reference

### Common Hook Options

All hooks support these common options:

- `autoFetch?: boolean` - Automatically fetch data on component mount (default: `true`)
- `refetchInterval?: number` - Automatically refetch data every N milliseconds (default: `0` - disabled)

### Common Hook Returns

All hooks return these common fields:

- `isLoading: boolean` - Whether data is currently being fetched
- `error: Error | null` - Any error that occurred during fetch/mutation
- `refetch: () => Promise<void>` - Manually trigger a data refetch

## License

MIT
