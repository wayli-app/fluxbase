---
editUrl: false
next: false
prev: false
title: "useGraphQLQuery"
---

> **useGraphQLQuery**\<`T`\>(`queryKey`, `query`, `options`?): `UseQueryResult`\<`NoInfer`\<`undefined` \| `T`\>, [`GraphQLError`](/api/sdk-react/interfaces/graphqlerror/)\>

Hook to execute GraphQL queries with React Query caching

## Type Parameters

| Type Parameter | Default type | Description |
| ------ | ------ | ------ |
| `T` | `unknown` | The expected response data type |

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `queryKey` | `string` \| readonly `unknown`[] | Unique key for caching (string or array) |
| `query` | `string` | The GraphQL query string |
| `options`? | [`UseGraphQLQueryOptions`](/api/sdk-react/interfaces/usegraphqlqueryoptions/)\<`T`\> | Query options including variables |

## Returns

`UseQueryResult`\<`NoInfer`\<`undefined` \| `T`\>, [`GraphQLError`](/api/sdk-react/interfaces/graphqlerror/)\>

React Query result object

## Examples

```tsx
interface UsersQuery {
  users: Array<{ id: string; email: string }>
}

function UsersList() {
  const { data, isLoading } = useGraphQLQuery<UsersQuery>(
    'users',
    `query { users { id email } }`
  )

  return <div>{data?.users.length} users</div>
}
```

```tsx
// With variables
const { data } = useGraphQLQuery<UserQuery>(
  ['user', userId],
  `query GetUser($id: ID!) { user(id: $id) { id email } }`,
  { variables: { id: userId } }
)
```
