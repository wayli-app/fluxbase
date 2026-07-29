---
editUrl: false
next: false
prev: false
title: "SignUpCredentials"
---

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="captchatoken"></a> `captchaToken?` | `string` | CAPTCHA token for bot protection (optional, required if CAPTCHA is enabled) |
| <a id="challengeid"></a> `challengeId?` | `string` | Challenge ID from pre-flight CAPTCHA check (for adaptive trust) |
| <a id="devicefingerprint"></a> `deviceFingerprint?` | `string` | Device fingerprint for trust tracking (optional) |
| <a id="email"></a> `email` | `string` | - |
| <a id="options"></a> `options?` | `object` | - |
| `options.data?` | `Record`\<`string`, `unknown`\> | User metadata to store in raw_user_meta_data (Supabase-compatible) |
| <a id="password"></a> `password` | `string` | - |
