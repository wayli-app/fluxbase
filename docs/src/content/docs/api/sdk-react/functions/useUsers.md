---
editUrl: false
next: false
prev: false
title: "useUsers"
---

> **useUsers**(`options`): `UseUsersReturn`

Hook for managing users

Provides user list with pagination, search, and management functions.

## Parameters

| Parameter | Type |
| ------ | ------ |
| `options` | `UseUsersOptions` |

## Returns

`UseUsersReturn`

## Example

```tsx
function UserList() {
  const { users, total, isLoading, refetch, inviteUser, deleteUser } = useUsers({
    limit: 20,
    search: searchTerm
  })

  return (
    <div>
      {isLoading ? <Spinner /> : (
        <ul>
          {users.map(user => (
            <li key={user.id}>
              {user.email} - {user.role}
              <button onClick={() => deleteUser(user.id)}>Delete</button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
```
