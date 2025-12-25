---
editUrl: false
next: false
prev: false
title: "ExecutionLog"
---

Execution log entry (shared by jobs, RPC, and functions)

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `execution_id` | `string` | ID of the execution (job ID, RPC execution ID, or function execution ID) |
| `fields?` | `Record`\<`string`, `unknown`\> | Additional structured fields |
| `id` | `number` | Unique log entry ID |
| `level` | `string` | Log level (debug, info, warn, error) |
| `line_number` | `number` | Line number within the execution log |
| `message` | `string` | Log message content |
| `timestamp` | `string` | Timestamp of the log entry |
