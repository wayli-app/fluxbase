---
editUrl: false
next: false
prev: false
title: "useGraphQLIntrospection"
---

> **useGraphQLIntrospection**(`options`?): `UseQueryResult`\<`undefined` \| `object`, `Error`\>

Hook to fetch the GraphQL schema via introspection

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options`? | `object` | Query options |
| `options.enabled`? | `boolean` | - |
| `options.requestOptions`? | [`GraphQLRequestOptions`](/api/sdk-react/interfaces/graphqlrequestoptions/) | - |
| `options.staleTime`? | `number` | - |

## Returns

`UseQueryResult`\<`undefined` \| `object`, `Error`\>

React Query result with schema introspection data

## Example

```tsx
function SchemaExplorer() {
  const { data, isLoading } = useGraphQLIntrospection()

  if (isLoading) return <div>Loading schema...</div>

  return (
    <div>
      <p>Query type: {data?.__schema.queryType.name}</p>
      <p>Types: {data?.__schema.types.length}</p>
    </div>
  )
}
```
