/**
 * Complete Admin Dashboard Example
 *
 * This example demonstrates all admin hooks in a real-world dashboard interface.
 *
 * Features:
 * - Admin authentication with protected routes
 * - User management with pagination
 * - API key management
 * - Webhook configuration
 * - App and system settings
 * - Real-time statistics
 */

import React, { useState } from 'react'
import {
  useAdminAuth,
  useUsers,
  useAPIKeys,
  useWebhooks,
  useAppSettings,
  useSystemSettings
} from '@fluxbase/sdk-react'
import type { EnrichedUser } from '@fluxbase/sdk'

// ============================================================================
// Admin Login Component
// ============================================================================

function AdminLogin() {
  const { isAuthenticated, isLoading, error, login } = useAdminAuth({ autoCheck: true })
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loginError, setLoginError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoginError(null)

    try {
      await login(email, password)
    } catch (err) {
      setLoginError((err as Error).message)
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (isAuthenticated) {
    return null // Will be redirected by parent
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8 p-8 bg-white rounded-lg shadow">
        <div>
          <h2 className="text-center text-3xl font-extrabold text-gray-900">
            Admin Login
          </h2>
        </div>
        <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
          <div className="rounded-md shadow-sm -space-y-px">
            <div>
              <input
                type="email"
                required
                className="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-t-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
                placeholder="Email address"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div>
              <input
                type="password"
                required
                className="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-b-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
                placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
          </div>

          {(loginError || error) && (
            <div className="text-red-600 text-sm">
              {loginError || error?.message}
            </div>
          )}

          <div>
            <button
              type="submit"
              className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              Sign in
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

// ============================================================================
// Statistics Overview
// ============================================================================

function Overview() {
  const { users, total: totalUsers, isLoading: loadingUsers } = useUsers({
    autoFetch: true,
    limit: 5
  })
  const { keys, isLoading: loadingKeys } = useAPIKeys({ autoFetch: true })
  const { webhooks, isLoading: loadingWebhooks } = useWebhooks({ autoFetch: true })
  const { settings, isLoading: loadingSettings } = useAppSettings({ autoFetch: true })

  const isLoading = loadingUsers || loadingKeys || loadingWebhooks || loadingSettings

  if (isLoading) {
    return <div className="text-center py-8">Loading overview...</div>
  }

  const enabledWebhooks = webhooks.filter(w => w.enabled).length
  const activeKeys = keys.filter(k => !k.expires_at || new Date(k.expires_at) > new Date()).length

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Dashboard Overview</h2>

      {/* Statistics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-white rounded-lg shadow p-6">
          <div className="text-sm font-medium text-gray-500">Total Users</div>
          <div className="mt-2 text-3xl font-semibold text-gray-900">{totalUsers}</div>
          <div className="mt-2 text-xs text-gray-500">Registered accounts</div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="text-sm font-medium text-gray-500">client keys</div>
          <div className="mt-2 text-3xl font-semibold text-gray-900">{activeKeys}</div>
          <div className="mt-2 text-xs text-gray-500">{keys.length - activeKeys} expired</div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="text-sm font-medium text-gray-500">Webhooks</div>
          <div className="mt-2 text-3xl font-semibold text-gray-900">{enabledWebhooks}</div>
          <div className="mt-2 text-xs text-gray-500">{webhooks.length - enabledWebhooks} disabled</div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="text-sm font-medium text-gray-500">Features</div>
          <div className="mt-2 space-y-1">
            <div className="flex items-center text-sm">
              <span className={`w-2 h-2 rounded-full mr-2 ${settings?.features?.enable_realtime ? 'bg-green-500' : 'bg-gray-300'}`}></span>
              Realtime
            </div>
            <div className="flex items-center text-sm">
              <span className={`w-2 h-2 rounded-full mr-2 ${settings?.features?.enable_storage ? 'bg-green-500' : 'bg-gray-300'}`}></span>
              Storage
            </div>
            <div className="flex items-center text-sm">
              <span className={`w-2 h-2 rounded-full mr-2 ${settings?.features?.enable_functions ? 'bg-green-500' : 'bg-gray-300'}`}></span>
              Functions
            </div>
          </div>
        </div>
      </div>

      {/* Recent Users */}
      <div className="bg-white rounded-lg shadow">
        <div className="p-6">
          <h3 className="text-lg font-medium mb-4">Recent Users</h3>
          <div className="space-y-3">
            {users.slice(0, 5).map((user) => (
              <div key={user.id} className="flex items-center justify-between">
                <div>
                  <div className="font-medium">{user.email}</div>
                  <div className="text-sm text-gray-500">
                    {new Date(user.created_at).toLocaleDateString()}
                  </div>
                </div>
                <div className="flex items-center space-x-2">
                  <span className={`px-2 py-1 text-xs rounded ${
                    user.role === 'admin' ? 'bg-purple-100 text-purple-800' : 'bg-gray-100 text-gray-800'
                  }`}>
                    {user.role}
                  </span>
                  <span className={`px-2 py-1 text-xs rounded ${
                    user.email_confirmed ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                  }`}>
                    {user.email_confirmed ? 'Verified' : 'Pending'}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

// ============================================================================
// User Management
// ============================================================================

function UserManagement() {
  const [page, setPage] = useState(0)
  const [searchEmail, setSearchEmail] = useState('')
  const [roleFilter, setRoleFilter] = useState<'admin' | 'user' | ''>('')
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
    offset: page * limit,
    email: searchEmail || undefined,
    role: roleFilter || undefined
  })

  const handleInvite = async () => {
    const email = prompt('Enter user email:')
    if (!email) return

    const isAdmin = confirm('Grant admin privileges?')
    try {
      await inviteUser(email, isAdmin ? 'admin' : 'user')
      alert('User invited successfully!')
    } catch (err) {
      alert('Failed to invite user: ' + (err as Error).message)
    }
  }

  const handleRoleToggle = async (userId: string, currentRole: string) => {
    const newRole = currentRole === 'admin' ? 'user' : 'admin'
    if (confirm(`Change role to ${newRole}?`)) {
      await updateUserRole(userId, newRole)
    }
  }

  const handleDelete = async (userId: string, email: string) => {
    if (confirm(`Delete user ${email}? This cannot be undone.`)) {
      await deleteUser(userId)
    }
  }

  const handleResetPassword = async (userId: string) => {
    try {
      const newPassword = await resetPassword(userId)
      alert(`New password: ${newPassword}\n\nMake sure to save this - it won't be shown again!`)
    } catch (err) {
      alert('Failed to reset password: ' + (err as Error).message)
    }
  }

  if (error) {
    return <div className="text-red-600">Error: {error.message}</div>
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h2 className="text-2xl font-bold">User Management</h2>
        <button
          onClick={handleInvite}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          Invite User
        </button>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg shadow p-4">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <input
            type="text"
            placeholder="Search by email..."
            value={searchEmail}
            onChange={(e) => {
              setSearchEmail(e.target.value)
              setPage(0)
            }}
            className="px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <select
            value={roleFilter}
            onChange={(e) => {
              setRoleFilter(e.target.value as any)
              setPage(0)
            }}
            className="px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">All Roles</option>
            <option value="admin">Admin</option>
            <option value="user">User</option>
          </select>
          <button
            onClick={refetch}
            className="px-4 py-2 border border-gray-300 rounded hover:bg-gray-50"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Users Table */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        {isLoading ? (
          <div className="text-center py-8">Loading users...</div>
        ) : (
          <>
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    User
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Role
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Created
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {users.map((user) => (
                  <tr key={user.id}>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-gray-900">{user.email}</div>
                      <div className="text-xs text-gray-500">{user.id.substring(0, 8)}</div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 py-1 text-xs rounded ${
                        user.role === 'admin' ? 'bg-purple-100 text-purple-800' : 'bg-gray-100 text-gray-800'
                      }`}>
                        {user.role}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`px-2 py-1 text-xs rounded ${
                        user.email_confirmed ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                      }`}>
                        {user.email_confirmed ? 'Verified' : 'Pending'}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {new Date(user.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2">
                      <button
                        onClick={() => handleRoleToggle(user.id, user.role)}
                        className="text-blue-600 hover:text-blue-900"
                      >
                        Toggle Role
                      </button>
                      <button
                        onClick={() => handleResetPassword(user.id)}
                        className="text-yellow-600 hover:text-yellow-900"
                      >
                        Reset PW
                      </button>
                      <button
                        onClick={() => handleDelete(user.id, user.email)}
                        className="text-red-600 hover:text-red-900"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {/* Pagination */}
            <div className="bg-gray-50 px-6 py-3 flex items-center justify-between">
              <div className="text-sm text-gray-700">
                Showing {page * limit + 1} to {Math.min((page + 1) * limit, total)} of {total} users
              </div>
              <div className="space-x-2">
                <button
                  onClick={() => setPage(page - 1)}
                  disabled={page === 0}
                  className="px-3 py-1 border border-gray-300 rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100"
                >
                  Previous
                </button>
                <button
                  onClick={() => setPage(page + 1)}
                  disabled={(page + 1) * limit >= total}
                  className="px-3 py-1 border border-gray-300 rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100"
                >
                  Next
                </button>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  )
}

// ============================================================================
// Main Dashboard
// ============================================================================

type TabType = 'overview' | 'users' | 'keys' | 'webhooks' | 'settings'

export default function AdminDashboard() {
  const { user, isAuthenticated, isLoading, logout } = useAdminAuth({ autoCheck: true })
  const [activeTab, setActiveTab] = useState<TabType>('overview')

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return <AdminLogin />
  }

  const tabs: { id: TabType; label: string }[] = [
    { id: 'overview', label: 'Overview' },
    { id: 'users', label: 'Users' },
    { id: 'keys', label: 'client keys' },
    { id: 'webhooks', label: 'Webhooks' },
    { id: 'settings', label: 'Settings' }
  ]

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Header */}
      <header className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex justify-between items-center">
            <h1 className="text-2xl font-bold text-gray-900">Admin Dashboard</h1>
            <div className="flex items-center space-x-4">
              <div className="text-sm text-gray-700">
                <span className="font-medium">{user?.email}</span>
                <span className="ml-2 px-2 py-1 text-xs rounded bg-purple-100 text-purple-800">
                  {user?.role}
                </span>
              </div>
              <button
                onClick={logout}
                className="px-4 py-2 text-sm font-medium text-gray-700 hover:text-gray-900"
              >
                Logout
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation Tabs */}
      <nav className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex space-x-8">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`py-4 px-1 border-b-2 font-medium text-sm ${
                  activeTab === tab.id
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab === 'overview' && <Overview />}
        {activeTab === 'users' && <UserManagement />}
        {activeTab === 'keys' && <div className="text-center py-8">client keys management (implement using useAPIKeys)</div>}
        {activeTab === 'webhooks' && <div className="text-center py-8">Webhooks management (implement using useWebhooks)</div>}
        {activeTab === 'settings' && <div className="text-center py-8">Settings management (implement using useAppSettings and useSystemSettings)</div>}
      </main>
    </div>
  )
}
