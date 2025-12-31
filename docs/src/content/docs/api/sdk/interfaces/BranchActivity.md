---
editUrl: false
next: false
prev: false
title: "BranchActivity"
---

Branch activity log entry

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `action` | `string` | Action performed |
| `branch_id` | `string` | Branch ID |
| `created_at` | `string` | When the activity occurred |
| `details?` | `Record`\<`string`, `unknown`\> | Additional details |
| `executed_by?` | `string` | User who performed the action |
| `id` | `string` | Activity ID |
| `status` | `"success"` \| `"pending"` \| `"failed"` | Activity status |
