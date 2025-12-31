---
editUrl: false
next: false
prev: false
title: "SAMLSession"
---

SAML session information

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `attributes?` | `Record`\<`string`, `string`[]\> | SAML attributes |
| `created_at` | `string` | Session creation time |
| `expires_at?` | `string` | Session expiration time |
| `id` | `string` | Session ID |
| `name_id` | `string` | SAML NameID |
| `provider_name` | `string` | Provider name |
| `session_index?` | `string` | Session index from IdP |
| `user_id` | `string` | User ID |
