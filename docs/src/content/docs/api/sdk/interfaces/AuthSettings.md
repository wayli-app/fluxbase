---
editUrl: false
next: false
prev: false
title: "AuthSettings"
---

Authentication settings configuration

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `_overrides?` | `Record`\<`string`, `SettingOverride`\> | Settings overridden by environment variables (read-only, cannot be modified via API) |
| `enable_magic_link` | `boolean` | - |
| `enable_signup` | `boolean` | - |
| `max_sessions_per_user` | `number` | - |
| `password_min_length` | `number` | - |
| `password_require_lowercase` | `boolean` | - |
| `password_require_number` | `boolean` | - |
| `password_require_special` | `boolean` | - |
| `password_require_uppercase` | `boolean` | - |
| `require_email_verification` | `boolean` | - |
| `session_timeout_minutes` | `number` | - |
