---
editUrl: false
next: false
prev: false
title: "useGraphQL"
---

> **useGraphQL**(): `object`

Hook to execute raw GraphQL operations (query or mutation)

This is a lower-level hook that doesn't use React Query caching.
Useful for one-off operations or when you need full control.

## Returns

`object`

Functions to execute queries and mutations

| Name | Type | Description |
| ------ | ------ | ------ |
| `execute` | \<`T`\>(`document`, `variables`?, `operationName`?, `options`?) => `Promise`\<[`GraphQLResponse`](/api/sdk-react/interfaces/graphqlresponse/)\<`T`\>\> | Execute a GraphQL operation with an explicit operation name |
| `executeMutation` | \<`T`\>(`mutation`, `variables`?, `options`?) => `Promise`\<[`GraphQLResponse`](/api/sdk-react/interfaces/graphqlresponse/)\<`T`\>\> | Execute a GraphQL mutation |
| `executeQuery` | \<`T`\>(`query`, `variables`?, `options`?) => `Promise`\<[`GraphQLResponse`](/api/sdk-react/interfaces/graphqlresponse/)\<`T`\>\> | Execute a GraphQL query |
| `introspect` | (`options`?) => `Promise`\<[`GraphQLResponse`](/api/sdk-react/interfaces/graphqlresponse/)\<`object`\>\> | Fetch the GraphQL schema via introspection |

## Example

```tsx
function AdminPanel() {
  const { executeQuery, executeMutation } = useGraphQL()

  const handleExport = async () => {
    const result = await executeQuery<ExportData>(
      `query { exportAllData { url } }`
    )
    if (result.data) {
      window.open(result.data.exportAllData.url)
    }
  }

  return <button onClick={handleExport}>Export Data</button>
}
```
