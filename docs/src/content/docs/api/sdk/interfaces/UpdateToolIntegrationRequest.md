---
editUrl: false
next: false
prev: false
title: "UpdateToolIntegrationRequest"
---

Request shape for updating a tool integration. All fields optional.
Updates that pass config.api_key = "***masked***" preserve the
existing encrypted value rather than overwriting it.

## Properties

| Property | Type |
| ------ | ------ |
| <a id="config"></a> `config?` | `Record`\<`string`, `string`\> |
| <a id="enabled"></a> `enabled?` | `boolean` |
| <a id="is_default"></a> `is_default?` | `boolean` |
| <a id="name"></a> `name?` | `string` |
