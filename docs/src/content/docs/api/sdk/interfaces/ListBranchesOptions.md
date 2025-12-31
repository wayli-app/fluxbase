---
editUrl: false
next: false
prev: false
title: "ListBranchesOptions"
---

Options for listing branches

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `githubRepo?` | `string` | Filter by GitHub repository |
| `limit?` | `number` | Maximum number of branches to return |
| `mine?` | `boolean` | Only show branches created by the current user |
| `offset?` | `number` | Offset for pagination |
| `status?` | [`BranchStatus`](/api/sdk/type-aliases/branchstatus/) | Filter by branch status |
| `type?` | [`BranchType`](/api/sdk/type-aliases/branchtype/) | Filter by branch type |
