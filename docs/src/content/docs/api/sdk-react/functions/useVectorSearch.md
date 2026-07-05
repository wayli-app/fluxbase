---
editUrl: false
next: false
prev: false
title: "useVectorSearch"
---

> **useVectorSearch**(): `UseMutationResult`\<`VectorSearchResult`\<`Record`\<`string`, `unknown`\>\>, `Error`, `VectorSearchOptions`, `unknown`\>

Hook for vector similarity search with automatic text embedding

## Returns

`UseMutationResult`\<`VectorSearchResult`\<`Record`\<`string`, `unknown`\>\>, `Error`, `VectorSearchOptions`, `unknown`\>

## Example

```tsx
const { mutateAsync } = useVectorSearch()
const { data } = await mutateAsync({
  table: 'documents',
  column: 'embedding',
  query: 'How to use TypeScript?',
  match_count: 10,
})
```
