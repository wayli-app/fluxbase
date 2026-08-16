---
editUrl: false
next: false
prev: false
title: "AuthConfig"
---

Comprehensive authentication configuration
Returns all public auth settings from the server

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="anon_key"></a> `anon_key?` | `string` | Publishable anon key for this instance (same key the web app exposes to browsers). Lets native clients connect without manual key entry. Omitted by servers that don't publish it. |
| <a id="captcha"></a> `captcha` | [`CaptchaConfig`](/api/sdk/interfaces/captchaconfig/) \| `null` | CAPTCHA configuration |
| <a id="magic_link_enabled"></a> `magic_link_enabled` | `boolean` | Whether magic link authentication is enabled |
| <a id="mfa_available"></a> `mfa_available` | `boolean` | Whether MFA/2FA is available (always true, users opt-in) |
| <a id="oauth_providers"></a> `oauth_providers` | [`OAuthProviderPublic`](/api/sdk/interfaces/oauthproviderpublic/)[] | Available OAuth providers for authentication |
| <a id="password_login_enabled"></a> `password_login_enabled` | `boolean` | Whether password login is enabled for app users |
| <a id="password_min_length"></a> `password_min_length` | `number` | Minimum password length requirement |
| <a id="password_require_lowercase"></a> `password_require_lowercase` | `boolean` | Whether passwords must contain lowercase letters |
| <a id="password_require_number"></a> `password_require_number` | `boolean` | Whether passwords must contain numbers |
| <a id="password_require_special"></a> `password_require_special` | `boolean` | Whether passwords must contain special characters |
| <a id="password_require_uppercase"></a> `password_require_uppercase` | `boolean` | Whether passwords must contain uppercase letters |
| <a id="require_email_verification"></a> `require_email_verification` | `boolean` | Whether email verification is required after signup |
| <a id="saml_providers"></a> `saml_providers` | `SAMLProviderInfo`[] | Available SAML providers for enterprise SSO |
| <a id="signup_enabled"></a> `signup_enabled` | `boolean` | Whether user signup is enabled |
