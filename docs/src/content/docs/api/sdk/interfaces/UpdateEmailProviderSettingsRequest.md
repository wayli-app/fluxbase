---
editUrl: false
next: false
prev: false
title: "UpdateEmailProviderSettingsRequest"
---

Request to update email provider settings

All fields are optional - only provided fields will be updated.
Secret fields (passwords, client keys) are only updated if provided.

## Properties

| Property | Type |
| ------ | ------ |
| `enabled?` | `boolean` |
| `from_address?` | `string` |
| `from_name?` | `string` |
| `mailgun_api_key?` | `string` |
| `mailgun_domain?` | `string` |
| `provider?` | `"smtp"` \| `"sendgrid"` \| `"mailgun"` \| `"ses"` |
| `sendgrid_api_key?` | `string` |
| `ses_access_key?` | `string` |
| `ses_region?` | `string` |
| `ses_secret_key?` | `string` |
| `smtp_host?` | `string` |
| `smtp_password?` | `string` |
| `smtp_port?` | `number` |
| `smtp_tls?` | `boolean` |
| `smtp_username?` | `string` |
