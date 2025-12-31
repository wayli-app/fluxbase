---
editUrl: false
next: false
prev: false
title: "FunctionInvokeOptions"
---

Options for invoking an edge function

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `body?` | `unknown` | Request body to send to the function |
| `headers?` | `Record`\<`string`, `string`\> | Custom headers to include in the request |
| `method?` | `"GET"` \| `"POST"` \| `"PUT"` \| `"PATCH"` \| `"DELETE"` | HTTP method to use **Default** `'POST'` |
| `namespace?` | `string` | Namespace of the function to invoke If not provided, the first function with the given name is used (alphabetically by namespace) |
