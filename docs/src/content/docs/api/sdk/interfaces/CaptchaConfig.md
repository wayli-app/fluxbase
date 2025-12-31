---
editUrl: false
next: false
prev: false
title: "CaptchaConfig"
---

Public CAPTCHA configuration returned from the server
Used by clients to know which CAPTCHA provider to load

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `cap_server_url?` | `string` | Cap server URL - only present when provider is 'cap' |
| `enabled` | `boolean` | Whether CAPTCHA is enabled |
| `endpoints?` | `string`[] | Endpoints that require CAPTCHA verification |
| `provider?` | [`CaptchaProvider`](/api/sdk/type-aliases/captchaprovider/) | CAPTCHA provider name |
| `site_key?` | `string` | Public site key for the CAPTCHA widget (hcaptcha, recaptcha, turnstile) |
