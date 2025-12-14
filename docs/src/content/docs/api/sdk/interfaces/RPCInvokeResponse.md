---
editUrl: false
next: false
prev: false
title: "RPCInvokeResponse"
---

RPC invocation response

## Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

## Properties

| Property | Type |
| ------ | ------ |
| `duration_ms?` | `number` |
| `error?` | `string` |
| `execution_id` | `string` |
| `result?` | `T` |
| `rows_returned?` | `number` |
| `status` | [`RPCExecutionStatus`](/api/sdk/type-aliases/rpcexecutionstatus/) |
