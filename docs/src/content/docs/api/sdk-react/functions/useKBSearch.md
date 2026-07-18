---
editUrl: false
next: false
prev: false
title: "useKBSearch"
---

> **useKBSearch**(`kbId`, `request`): `UseQueryResult`\<`NoInfer`\<`SearchKnowledgeBaseResponse` \| `null`\>, `Error`\>

Hook to search a knowledge base with semantic similarity

## Parameters

| Parameter | Type |
| ------ | ------ |
| `kbId` | `string` \| `null` |
| `request` | `SearchKnowledgeBaseRequest` \| `null` |

## Returns

`UseQueryResult`\<`NoInfer`\<`SearchKnowledgeBaseResponse` \| `null`\>, `Error`\>

## Example

```tsx
const { data, isPending } = useKBSearch(kbId, {
  query: 'How to configure auth?',
  max_chunks: 5,
})
```
