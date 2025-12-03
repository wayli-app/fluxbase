---
editUrl: false
next: false
prev: false
title: "EmailSettings"
---

Email configuration settings

## Properties

| Property | Type |
| ------ | ------ |
| `enabled` | `boolean` |
| `from_address?` | `string` |
| `from_name?` | `string` |
| `mailgun?` | [`MailgunSettings`](/api/sdk/interfaces/mailgunsettings/) |
| `provider` | `"smtp"` \| `"sendgrid"` \| `"mailgun"` \| `"ses"` |
| `reply_to_address?` | `string` |
| `sendgrid?` | [`SendGridSettings`](/api/sdk/interfaces/sendgridsettings/) |
| `ses?` | [`SESSettings`](/api/sdk/interfaces/sessettings/) |
| `smtp?` | [`SMTPSettings`](/api/sdk/interfaces/smtpsettings/) |
