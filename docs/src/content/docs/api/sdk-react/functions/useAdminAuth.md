---
editUrl: false
next: false
prev: false
title: "useAdminAuth"
---

> **useAdminAuth**(`options`): `UseAdminAuthReturn`

Hook for admin authentication

Manages admin login state, authentication checks, and user info.

## Parameters

| Parameter | Type |
| ------ | ------ |
| `options` | `UseAdminAuthOptions` |

## Returns

`UseAdminAuthReturn`

## Example

```tsx
function AdminLogin() {
  const { user, isAuthenticated, isLoading, login, logout } = useAdminAuth()

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    await login(email, password)
  }

  if (isLoading) return <div>Loading...</div>
  if (isAuthenticated) return <div>Welcome {user?.email}</div>

  return <form onSubmit={handleLogin}>...</form>
}
```
