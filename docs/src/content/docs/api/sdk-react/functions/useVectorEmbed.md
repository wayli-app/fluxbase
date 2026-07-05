---
editUrl: false
next: false
prev: false
title: "useVectorEmbed"
---

> **useVectorEmbed**(): `UseMutationResult`\<`EmbedResponse`, `Error`, `EmbedRequest`, `unknown`\>

Hook to generate embeddings for text

## Returns

`UseMutationResult`\<`EmbedResponse`, `Error`, `EmbedRequest`, `unknown`\>

## Example

```tsx
const { mutateAsync } = useVectorEmbed()
const { data } = await mutateAsync({ text: 'Hello world' })
```
