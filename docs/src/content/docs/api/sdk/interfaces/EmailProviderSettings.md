---
editUrl: false
next: false
prev: false
title: "EmailProviderSettings"
---

Email provider settings response from /api/v1/admin/email/settings

This is the flat structure returned by the admin API, which differs from
the nested EmailSettings structure used in AppSettings.

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `_overrides` | `Record`\<`string`, [`EmailSettingOverride`](/api/sdk/interfaces/emailsettingoverride/)\> | Settings overridden by environment variables |
| `enabled` | `boolean` | - |
| `from_address` | `string` | - |
| `from_name` | `string` | - |
| `mailgun_api_key_set` | `boolean` | - |
| `mailgun_domain` | `string` | - |
| `provider` | `"smtp"` \| `"sendgrid"` \| `"mailgun"` \| `"ses"` | - |
| `sendgrid_api_key_set` | `boolean` | - |
| `ses_access_key_set` | `boolean` | - |
| `ses_region` | `string` | - |
| `ses_secret_key_set` | `boolean` | - |
| `smtp_host` | `string` | - |
| `smtp_password_set` | `boolean` | - |
| `smtp_port` | `number` | - |
| `smtp_tls` | `boolean` | - |
| `smtp_username` | `string` | - |
