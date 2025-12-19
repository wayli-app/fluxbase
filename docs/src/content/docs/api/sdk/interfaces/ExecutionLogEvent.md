---
editUrl: false
next: false
prev: false
title: "ExecutionLogEvent"
---

Execution log event received from realtime subscription

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `execution_id` | `string` | Unique execution ID |
| `execution_type` | [`ExecutionType`](/api/sdk/type-aliases/executiontype/) | Type of execution |
| `fields?` | `Record`\<`string`, `unknown`\> | Additional fields |
| `level` | [`ExecutionLogLevel`](/api/sdk/type-aliases/executionloglevel/) | Log level |
| `line_number` | `number` | Line number in the execution log |
| `message` | `string` | Log message content |
| `timestamp` | `string` | Timestamp of the log entry |
