---
editUrl: false
next: false
prev: false
title: "EmbedResponse"
---

Response from vector embedding generation

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `dimensions` | `number` | Dimensions of the embeddings |
| `embeddings` | `number`[][] | Generated embeddings (one per input text) |
| `model` | `string` | Model used for embedding |
| `usage?` | `object` | Token usage information |
| `usage.prompt_tokens` | `number` | - |
| `usage.total_tokens` | `number` | - |
