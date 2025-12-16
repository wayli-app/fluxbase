---
editUrl: false
next: false
prev: false
title: "OrderBy"
---

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `column` | `string` | - |
| `direction` | [`OrderDirection`](/api/sdk/type-aliases/orderdirection/) | - |
| `nulls?` | `"first"` \| `"last"` | - |
| `vectorOp?` | `"vec_l2"` \| `"vec_cos"` \| `"vec_ip"` | Vector operator for similarity ordering (vec_l2, vec_cos, vec_ip) |
| `vectorValue?` | `number`[] | Vector value for similarity ordering |
