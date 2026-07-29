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
| <a id="created_at"></a> `created_at` | `string` | When the branch was created |
| <a id="created_by"></a> `created_by?` | `string` | User ID who created the branch |
| <a id="data_clone_mode"></a> `data_clone_mode` | `DataCloneMode` | How data was cloned when branch was created |
| <a id="database_name"></a> `database_name` | `string` | Actual database name |
| <a id="error_message"></a> `error_message?` | `string` | Error message if status is 'error' |
| <a id="expires_at"></a> `expires_at?` | `string` | When the branch will automatically expire |
| <a id="github_pr_number"></a> `github_pr_number?` | `number` | GitHub PR number if this is a preview branch |
| <a id="github_pr_url"></a> `github_pr_url?` | `string` | GitHub PR URL |
| <a id="github_repo"></a> `github_repo?` | `string` | GitHub repository (owner/repo) |
| <a id="id"></a> `id` | `string` | Unique branch identifier |
| <a id="name"></a> `name` | `string` | Display name of the branch |
| <a id="parent_branch_id"></a> `parent_branch_id?` | `string` | Parent branch ID (for feature branches) |
| <a id="slug"></a> `slug` | `string` | URL-safe slug for the branch |
| <a id="status"></a> `status` | `BranchStatus` | Current status of the branch |
| <a id="type"></a> `type` | `BranchType` | Type of branch |
| <a id="updated_at"></a> `updated_at` | `string` | When the branch was last updated |
