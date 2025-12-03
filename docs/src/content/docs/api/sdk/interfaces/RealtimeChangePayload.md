---
editUrl: false
next: false
prev: false
title: "RealtimeChangePayload"
---

:::caution[Deprecated]
Use RealtimePostgresChangesPayload instead
:::

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| ~~`new_record?`~~ | `Record`\<`string`, `unknown`\> | :::caution[Deprecated] Use 'new' instead ::: |
| ~~`old_record?`~~ | `Record`\<`string`, `unknown`\> | :::caution[Deprecated] Use 'old' instead ::: |
| ~~`schema`~~ | `string` | - |
| ~~`table`~~ | `string` | - |
| ~~`timestamp`~~ | `string` | :::caution[Deprecated] Use commit_timestamp instead ::: |
| ~~`type`~~ | `"DELETE"` \| `"INSERT"` \| `"UPDATE"` | :::caution[Deprecated] Use eventType instead ::: |
