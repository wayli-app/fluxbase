---
editUrl: false
next: false
prev: false
title: "Branch"
---

Database branch information

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `created_at` | `string` | When the branch was created |
| `created_by?` | `string` | User ID who created the branch |
| `data_clone_mode` | [`DataCloneMode`](/api/sdk/type-aliases/dataclonemode/) | How data was cloned when branch was created |
| `database_name` | `string` | Actual database name |
| `error_message?` | `string` | Error message if status is 'error' |
| `expires_at?` | `string` | When the branch will automatically expire |
| `github_pr_number?` | `number` | GitHub PR number if this is a preview branch |
| `github_pr_url?` | `string` | GitHub PR URL |
| `github_repo?` | `string` | GitHub repository (owner/repo) |
| `id` | `string` | Unique branch identifier |
| `name` | `string` | Display name of the branch |
| `parent_branch_id?` | `string` | Parent branch ID (for feature branches) |
| `slug` | `string` | URL-safe slug for the branch |
| `status` | [`BranchStatus`](/api/sdk/type-aliases/branchstatus/) | Current status of the branch |
| `type` | [`BranchType`](/api/sdk/type-aliases/branchtype/) | Type of branch |
| `updated_at` | `string` | When the branch was last updated |
