---
editUrl: false
next: false
prev: false
title: "CreateAIProviderRequest"
---

Request to create an AI provider
Note: config values can be strings, numbers, or booleans - they will be converted to strings automatically

## Properties

| Property | Type |
| ------ | ------ |
| `config` | `Record`\<`string`, `string` \| `number` \| `boolean`\> |
| `display_name` | `string` |
| `enabled?` | `boolean` |
| `is_default?` | `boolean` |
| `name` | `string` |
| `provider_type` | [`AIProviderType`](/api/sdk/type-aliases/aiprovidertype/) |
