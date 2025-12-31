---
editUrl: false
next: false
prev: false
title: "CreateBranchOptions"
---

Options for creating a new branch

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `dataCloneMode?` | [`DataCloneMode`](/api/sdk/type-aliases/dataclonemode/) | How to clone data |
| `expiresIn?` | `string` | Duration until branch expires (e.g., "24h", "7d") |
| `githubPRNumber?` | `number` | GitHub PR number (for preview branches) |
| `githubPRUrl?` | `string` | GitHub PR URL |
| `githubRepo?` | `string` | GitHub repository (owner/repo) |
| `parentBranchId?` | `string` | Parent branch to clone from (defaults to main) |
| `type?` | [`BranchType`](/api/sdk/type-aliases/branchtype/) | Branch type |
