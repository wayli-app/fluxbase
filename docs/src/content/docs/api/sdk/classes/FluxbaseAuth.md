---
editUrl: false
next: false
prev: false
title: "FluxbaseAuth"
---

## Constructors

### Constructor

> **new FluxbaseAuth**(`fetch`, `autoRefresh?`, `persist?`, `storage?`): `FluxbaseAuth`

#### Parameters

| Parameter | Type | Default value |
| ------ | ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) | `undefined` |
| `autoRefresh` | `boolean` | `true` |
| `persist` | `boolean` | `true` |
| `storage?` | [`StorageAdapter`](/api/sdk/interfaces/storageadapter/) | `undefined` |

#### Returns

`FluxbaseAuth`

## Methods

### checkCaptcha()

> **checkCaptcha**(`request`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`CaptchaCheckResponse`\>\>

Check if CAPTCHA is required for an authentication action (adaptive trust)

This pre-flight check evaluates trust signals (known IP, device, previous CAPTCHA)
to determine if CAPTCHA verification is needed. Use this before showing auth forms
to provide a better user experience for trusted users.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | `CaptchaCheckRequest` | Check request with endpoint and optional trust signals |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`CaptchaCheckResponse`\>\>

Promise with whether CAPTCHA is required and challenge tracking info

#### Example

```typescript
// Check if CAPTCHA is needed for login
const { data, error } = await client.auth.checkCaptcha({
  endpoint: 'login',
  email: 'user@example.com'
});

if (data?.captcha_required) {
  // Show CAPTCHA widget using data.provider and data.site_key
  const captchaToken = await showCaptchaWidget(data.provider, data.site_key);

  // Include challenge_id and captcha token in sign in
  await client.auth.signIn({
    email: 'user@example.com',
    password: 'password',
    captchaToken,
    challengeId: data.challenge_id
  });
} else {
  // No CAPTCHA needed - trusted user
  await client.auth.signIn({
    email: 'user@example.com',
    password: 'password',
    challengeId: data?.challenge_id // Still include challenge_id
  });
}
```

***

### disable2FA()

> **disable2FA**(`password`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`TwoFactorDisableResponse`\>\>

Disable 2FA for the current user (Supabase-compatible)
Unenrolls the MFA factor

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `password` | `string` | User password for confirmation |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`TwoFactorDisableResponse`\>\>

Promise with unenrolled factor id

***

### enable2FA()

> **enable2FA**(`code`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`TwoFactorEnableResponse`](/api/sdk/interfaces/twofactorenableresponse/)\>\>

Enable 2FA after verifying the TOTP code (Supabase-compatible)
Verifies the TOTP code and returns new tokens with MFA session

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `code` | `string` | TOTP code from authenticator app |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`TwoFactorEnableResponse`](/api/sdk/interfaces/twofactorenableresponse/)\>\>

Promise with access_token, refresh_token, and user

***

### exchangeCodeForSession()

> **exchangeCodeForSession**(`code`, `state?`): `Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Exchange OAuth authorization code for session
This is typically called in your OAuth callback handler

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `code` | `string` | Authorization code from OAuth callback |
| `state?` | `string` | State parameter from OAuth callback (for CSRF protection) |

#### Returns

`Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

***

### get2FAStatus()

> **get2FAStatus**(): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`TwoFactorStatusResponse`](/api/sdk/interfaces/twofactorstatusresponse/)\>\>

Check 2FA status for the current user (Supabase-compatible)
Lists all enrolled MFA factors

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`TwoFactorStatusResponse`](/api/sdk/interfaces/twofactorstatusresponse/)\>\>

Promise with all factors and TOTP factors

***

### getAccessToken()

> **getAccessToken**(): `string` \| `null`

Get the current access token

#### Returns

`string` \| `null`

***

### getAuthConfig()

> **getAuthConfig**(): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AuthConfig`](/api/sdk/interfaces/authconfig/)\>\>

Get comprehensive authentication configuration from the server
Returns all public auth settings including signup status, OAuth providers,
SAML providers, password requirements, and CAPTCHA config in a single request.

Use this to:
- Conditionally render signup forms based on signup_enabled
- Display available OAuth/SAML provider buttons
- Show password requirements to users
- Configure CAPTCHA widgets

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AuthConfig`](/api/sdk/interfaces/authconfig/)\>\>

Promise with complete authentication configuration

#### Example

```typescript
const { data, error } = await client.auth.getAuthConfig();
if (data) {
  console.log('Signup enabled:', data.signup_enabled);
  console.log('OAuth providers:', data.oauth_providers);
  console.log('Password min length:', data.password_min_length);
}
```

***

### getCaptchaConfig()

> **getCaptchaConfig**(): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`CaptchaConfig`](/api/sdk/interfaces/captchaconfig/)\>\>

Get CAPTCHA configuration from the server
Use this to determine which CAPTCHA provider to load and configure

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`CaptchaConfig`](/api/sdk/interfaces/captchaconfig/)\>\>

Promise with CAPTCHA configuration (provider, site key, enabled endpoints)

***

### getCurrentUser()

> **getCurrentUser**(): `Promise`\<[`UserResponse`](/api/sdk/type-aliases/userresponse/)\>

Get the current user from the server

#### Returns

`Promise`\<[`UserResponse`](/api/sdk/type-aliases/userresponse/)\>

***

### getOAuthLogoutUrl()

> **getOAuthLogoutUrl**(`provider`, `options?`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`OAuthLogoutResponse`](/api/sdk/interfaces/oauthlogoutresponse/)\>\>

Get OAuth logout URL for a provider
Use this to get the logout URL without automatically redirecting

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `provider` | `string` | OAuth provider name (e.g., 'google', 'github') |
| `options?` | [`OAuthLogoutOptions`](/api/sdk/interfaces/oauthlogoutoptions/) | Optional logout configuration |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`OAuthLogoutResponse`](/api/sdk/interfaces/oauthlogoutresponse/)\>\>

Promise with OAuth logout response including redirect URL if applicable

#### Example

```typescript
const { data, error } = await client.auth.getOAuthLogoutUrl('google')
if (!error && data.redirect_url) {
  // Redirect user to complete logout at provider
  window.location.href = data.redirect_url
}
```

***

### getOAuthProviders()

> **getOAuthProviders**(): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OAuthProvidersResponse`\>\>

Get list of enabled OAuth providers

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OAuthProvidersResponse`\>\>

***

### getOAuthUrl()

> **getOAuthUrl**(`provider`, `options?`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OAuthUrlResponse`\>\>

Get OAuth authorization URL for a provider

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `provider` | `string` | OAuth provider name (e.g., 'google', 'github') |
| `options?` | `OAuthOptions` | Optional OAuth configuration |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OAuthUrlResponse`\>\>

***

### getProviderToken()

> **getProviderToken**(`provider`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`ProviderTokenResponse`](/api/sdk/interfaces/providertokenresponse/)\>\>

Get provider OAuth tokens for calling external APIs

Retrieves the stored OAuth tokens for a provider (e.g., Google, GitHub) that
the user has previously authenticated with. Use these tokens to call provider
APIs directly (e.g., Google Drive API).

The access_token is automatically refreshed if it has expired or is about to expire.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `provider` | `string` | OAuth provider name (e.g., 'google', 'github') |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`ProviderTokenResponse`](/api/sdk/interfaces/providertokenresponse/)\>\>

Promise with provider tokens (access_token, refresh_token, etc.)

#### Examples

```typescript
// Get Google tokens to call Google Drive API
const { data, error } = await client.auth.getProviderToken('google')

if (error) {
  if (error.error_code === 'oauth_token_not_found') {
    // User needs to sign in with Google first
    window.location.href = error.authorize_url
  }
  return
}

// Use the access token to call Google Drive API
const response = await fetch('https://www.googleapis.com/drive/v3/files', {
  headers: {
    'Authorization': `Bearer ${data.access_token}`
  }
})
const files = await response.json()
```

```typescript
// Check token expiry before making API calls
const { data } = await client.auth.getProviderToken('google')

if (data.expires_in < 60) {
  console.warn('Token expires soon, consider caching and refreshing')
}

// Token expiry is also available as ISO timestamp
console.log('Token expires at:', data.token_expiry)
```

***

### getSAMLLoginUrl()

> **getSAMLLoginUrl**(`provider`, `options?`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`SAMLLoginResponse`](/api/sdk/interfaces/samlloginresponse/)\>\>

Get SAML login URL for a specific provider
Use this to redirect the user to the IdP for authentication

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `provider` | `string` | SAML provider name/ID |
| `options?` | [`SAMLLoginOptions`](/api/sdk/interfaces/samlloginoptions/) | Optional login configuration |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`SAMLLoginResponse`](/api/sdk/interfaces/samlloginresponse/)\>\>

Promise with SAML login URL

#### Example

```typescript
const { data, error } = await client.auth.getSAMLLoginUrl('okta')
if (!error) {
  window.location.href = data.url
}
```

***

### getSAMLMetadataUrl()

> **getSAMLMetadataUrl**(`provider`): `string`

Get SAML Service Provider metadata for a specific provider configuration
Use this when configuring your IdP to download the SP metadata XML

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `provider` | `string` | SAML provider name/ID |

#### Returns

`string`

Promise with SP metadata URL

#### Example

```typescript
const metadataUrl = client.auth.getSAMLMetadataUrl('okta')
// Share this URL with your IdP administrator
```

***

### getSAMLProviders()

> **getSAMLProviders**(): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`SAMLProvidersResponse`](/api/sdk/interfaces/samlprovidersresponse/)\>\>

Get list of available SAML SSO providers

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`SAMLProvidersResponse`](/api/sdk/interfaces/samlprovidersresponse/)\>\>

Promise with list of configured SAML providers

#### Example

```typescript
const { data, error } = await client.auth.getSAMLProviders()
if (!error) {
  console.log('Available providers:', data.providers)
}
```

***

### getSession()

> **getSession**(): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<\{ `session`: [`AuthSession`](/api/sdk/interfaces/authsession/) \| `null`; \}\>\>

Get the current session (Supabase-compatible)
Returns the session from the client-side cache without making a network request

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<\{ `session`: [`AuthSession`](/api/sdk/interfaces/authsession/) \| `null`; \}\>\>

***

### getUser()

> **getUser**(): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<\{ `user`: [`User`](/api/sdk/interfaces/user/) \| `null`; \}\>\>

Get the current user (Supabase-compatible)
Returns the user from the client-side session without making a network request
For server-side validation, use getCurrentUser() instead

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<\{ `user`: [`User`](/api/sdk/interfaces/user/) \| `null`; \}\>\>

***

### getUserIdentities()

> **getUserIdentities**(): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`UserIdentitiesResponse`\>\>

Get user identities (linked OAuth providers) - Supabase-compatible
Lists all OAuth identities linked to the current user

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`UserIdentitiesResponse`\>\>

Promise with list of user identities

***

### handleSAMLCallback()

> **handleSAMLCallback**(`samlResponse`, `provider?`): `Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Handle SAML callback after IdP authentication
Call this from your SAML callback page to complete authentication

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `samlResponse` | `string` | Base64-encoded SAML response from the ACS endpoint |
| `provider?` | `string` | SAML provider name (optional, extracted from RelayState) |

#### Returns

`Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Promise with user and session

#### Example

```typescript
// In your SAML callback page
const urlParams = new URLSearchParams(window.location.search)
const samlResponse = urlParams.get('SAMLResponse')

if (samlResponse) {
  const { data, error } = await client.auth.handleSAMLCallback(samlResponse)
  if (!error) {
    console.log('Logged in:', data.user)
  }
}
```

***

### linkIdentity()

> **linkIdentity**(`credentials`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OAuthUrlResponse`\>\>

Link an OAuth identity to current user - Supabase-compatible
Links an additional OAuth provider to the existing account

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `credentials` | `LinkIdentityCredentials` | Provider to link |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OAuthUrlResponse`\>\>

Promise with OAuth URL to complete linking

***

### onAuthStateChange()

> **onAuthStateChange**(`callback`): `object`

Listen to auth state changes (Supabase-compatible)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `callback` | `AuthStateChangeCallback` | Function called when auth state changes |

#### Returns

`object`

Object containing subscription data

| Name | Type |
| ------ | ------ |
| `data` | `object` |
| `data.subscription` | `AuthSubscription` |

#### Example

```typescript
const { data: { subscription } } = client.auth.onAuthStateChange((event, session) => {
  console.log('Auth event:', event, session)
})

// Later, to unsubscribe:
subscription.unsubscribe()
```

***

### reauthenticate()

> **reauthenticate**(): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`ReauthenticateResponse`\>\>

Reauthenticate to get security nonce - Supabase-compatible
Get a security nonce for sensitive operations (password change, etc.)

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`ReauthenticateResponse`\>\>

Promise with nonce for reauthentication

***

### refreshSession()

> **refreshSession**(): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<\{ `session`: [`AuthSession`](/api/sdk/interfaces/authsession/); `user`: [`User`](/api/sdk/interfaces/user/); \}\>\>

Refresh the session (Supabase-compatible)
Returns a new session with refreshed tokens

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<\{ `session`: [`AuthSession`](/api/sdk/interfaces/authsession/); `user`: [`User`](/api/sdk/interfaces/user/); \}\>\>

***

### refreshToken()

> **refreshToken**(): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<\{ `session`: [`AuthSession`](/api/sdk/interfaces/authsession/); `user`: [`User`](/api/sdk/interfaces/user/); \}\>\>

Refresh the session (Supabase-compatible alias)
Alias for refreshSession() to maintain compatibility with Supabase naming
Returns a new session with refreshed tokens

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<\{ `session`: [`AuthSession`](/api/sdk/interfaces/authsession/); `user`: [`User`](/api/sdk/interfaces/user/); \}\>\>

***

### resendOtp()

> **resendOtp**(`params`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OTPResponse`\>\>

Resend OTP (One-Time Password) - Supabase-compatible
Resend OTP code when user doesn't receive it

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `params` | `ResendOtpParams` | Resend parameters including type and email/phone |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OTPResponse`\>\>

Promise with OTP-style response

***

### resetPassword()

> **resetPassword**(`token`, `newPassword`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AuthResponseData`](/api/sdk/type-aliases/authresponsedata/)\>\>

Reset password with token (Supabase-compatible)
Complete the password reset process with a valid token

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `token` | `string` | Password reset token |
| `newPassword` | `string` | New password to set |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AuthResponseData`](/api/sdk/type-aliases/authresponsedata/)\>\>

Promise with user and new session

***

### resetPasswordForEmail()

> **resetPasswordForEmail**(`email`, `options?`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`PasswordResetResponse`\>\>

Supabase-compatible alias for sendPasswordReset()

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `email` | `string` | Email address to send reset link to |
| `options?` | \{ `captchaToken?`: `string`; `redirectTo?`: `string`; \} | Optional redirect and CAPTCHA configuration |
| `options.captchaToken?` | `string` | - |
| `options.redirectTo?` | `string` | - |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`PasswordResetResponse`\>\>

Promise with OTP-style response

***

### sendMagicLink()

> **sendMagicLink**(`email`, `options?`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`MagicLinkResponse`\>\>

Send magic link for passwordless authentication (Supabase-compatible)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `email` | `string` | Email address to send magic link to |
| `options?` | `MagicLinkOptions` | Optional configuration for magic link |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`MagicLinkResponse`\>\>

Promise with OTP-style response

***

### sendPasswordReset()

> **sendPasswordReset**(`email`, `options?`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`PasswordResetResponse`\>\>

Send password reset email (Supabase-compatible)
Sends a password reset link to the provided email address

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `email` | `string` | Email address to send reset link to |
| `options?` | \{ `captchaToken?`: `string`; `redirectTo?`: `string`; \} | Optional configuration including redirect URL and CAPTCHA token |
| `options.captchaToken?` | `string` | - |
| `options.redirectTo?` | `string` | - |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`PasswordResetResponse`\>\>

Promise with OTP-style response

***

### setSession()

> **setSession**(`session`): `Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Set the session manually (Supabase-compatible)
Useful for restoring a session from storage or SSR scenarios

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `session` | \{ `access_token`: `string`; `refresh_token`: `string`; \} | Object containing access_token and refresh_token |
| `session.access_token` | `string` | - |
| `session.refresh_token` | `string` | - |

#### Returns

`Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Promise with session data

***

### setup2FA()

> **setup2FA**(`issuer?`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`TwoFactorSetupResponse`](/api/sdk/interfaces/twofactorsetupresponse/)\>\>

Setup 2FA for the current user (Supabase-compatible)
Enrolls a new MFA factor and returns TOTP details

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `issuer?` | `string` | Optional custom issuer name for the QR code (e.g., "MyApp"). If not provided, uses server default. |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`TwoFactorSetupResponse`](/api/sdk/interfaces/twofactorsetupresponse/)\>\>

Promise with factor id, type, and TOTP setup details

***

### signIn()

> **signIn**(`credentials`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`SignInWith2FAResponse`](/api/sdk/interfaces/signinwith2faresponse/) \| [`AuthResponseData`](/api/sdk/type-aliases/authresponsedata/)\>\>

Sign in with email and password (Supabase-compatible)
Returns { user, session } if successful, or SignInWith2FAResponse if 2FA is required

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `credentials` | [`SignInCredentials`](/api/sdk/interfaces/signincredentials/) |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`SignInWith2FAResponse`](/api/sdk/interfaces/signinwith2faresponse/) \| [`AuthResponseData`](/api/sdk/type-aliases/authresponsedata/)\>\>

***

### signInAnonymously()

> **signInAnonymously**(): `Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Sign in anonymously
Creates a temporary anonymous user session

#### Returns

`Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

***

### signInWithIdToken()

> **signInWithIdToken**(`credentials`): `Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Sign in with ID token (for native mobile apps) - Supabase-compatible
Authenticate using native mobile app ID tokens (Google, Apple)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `credentials` | `SignInWithIdTokenCredentials` | Provider, ID token, and optional nonce |

#### Returns

`Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Promise with user and session

***

### signInWithOAuth()

> **signInWithOAuth**(`provider`, `options?`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<\{ `provider`: `string`; `url`: `string`; \}\>\>

Convenience method to initiate OAuth sign-in
Redirects the user to the OAuth provider's authorization page

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `provider` | `string` | OAuth provider name (e.g., 'google', 'github') |
| `options?` | `OAuthOptions` | Optional OAuth configuration |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<\{ `provider`: `string`; `url`: `string`; \}\>\>

***

### signInWithOtp()

> **signInWithOtp**(`credentials`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OTPResponse`\>\>

Sign in with OTP (One-Time Password) - Supabase-compatible
Sends a one-time password via email or SMS for passwordless authentication

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `credentials` | `SignInWithOtpCredentials` | Email or phone number and optional configuration |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OTPResponse`\>\>

Promise with OTP-style response

***

### signInWithPassword()

> **signInWithPassword**(`credentials`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`SignInWith2FAResponse`](/api/sdk/interfaces/signinwith2faresponse/) \| [`AuthResponseData`](/api/sdk/type-aliases/authresponsedata/)\>\>

Sign in with email and password (Supabase-compatible)
Alias for signIn() to maintain compatibility with common authentication patterns
Returns { user, session } if successful, or SignInWith2FAResponse if 2FA is required

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `credentials` | [`SignInCredentials`](/api/sdk/interfaces/signincredentials/) |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`SignInWith2FAResponse`](/api/sdk/interfaces/signinwith2faresponse/) \| [`AuthResponseData`](/api/sdk/type-aliases/authresponsedata/)\>\>

***

### signInWithSAML()

> **signInWithSAML**(`provider`, `options?`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<\{ `provider`: `string`; `url`: `string`; \}\>\>

Initiate SAML login and redirect to IdP
This is a convenience method that redirects the user to the SAML IdP

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `provider` | `string` | SAML provider name/ID |
| `options?` | [`SAMLLoginOptions`](/api/sdk/interfaces/samlloginoptions/) | Optional login configuration |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<\{ `provider`: `string`; `url`: `string`; \}\>\>

Promise with provider and URL (browser will redirect)

#### Example

```typescript
// In browser, this will redirect to the SAML IdP
await client.auth.signInWithSAML('okta')
```

***

### signOut()

> **signOut**(): `Promise`\<[`VoidResponse`](/api/sdk/type-aliases/voidresponse/)\>

Sign out the current user

#### Returns

`Promise`\<[`VoidResponse`](/api/sdk/type-aliases/voidresponse/)\>

***

### signOutWithOAuth()

> **signOutWithOAuth**(`provider`, `options?`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`OAuthLogoutResponse`](/api/sdk/interfaces/oauthlogoutresponse/)\>\>

Sign out with OAuth provider logout
Revokes tokens at the OAuth provider and optionally redirects for OIDC logout

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `provider` | `string` | OAuth provider name (e.g., 'google', 'github') |
| `options?` | [`OAuthLogoutOptions`](/api/sdk/interfaces/oauthlogoutoptions/) | Optional logout configuration |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`OAuthLogoutResponse`](/api/sdk/interfaces/oauthlogoutresponse/)\>\>

Promise with OAuth logout response

#### Example

```typescript
// This will revoke tokens and redirect to provider's logout page if supported
await client.auth.signOutWithOAuth('google', {
  redirect_url: 'https://myapp.com/logged-out'
})
```

***

### signUp()

> **signUp**(`credentials`): `Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Sign up with email and password (Supabase-compatible)
Returns session when email confirmation is disabled
Returns null session when email confirmation is required

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `credentials` | [`SignUpCredentials`](/api/sdk/interfaces/signupcredentials/) |

#### Returns

`Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

***

### startAutoRefresh()

> **startAutoRefresh**(): `void`

Start the automatic token refresh timer
This is called automatically when autoRefresh is enabled and a session exists
Only works in browser environments

#### Returns

`void`

***

### stopAutoRefresh()

> **stopAutoRefresh**(): `void`

Stop the automatic token refresh timer
Call this when you want to disable auto-refresh without signing out

#### Returns

`void`

***

### unlinkIdentity()

> **unlinkIdentity**(`params`): `Promise`\<[`VoidResponse`](/api/sdk/type-aliases/voidresponse/)\>

Unlink an OAuth identity from current user - Supabase-compatible
Removes a linked OAuth provider from the account

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `params` | `UnlinkIdentityParams` | Identity to unlink |

#### Returns

`Promise`\<[`VoidResponse`](/api/sdk/type-aliases/voidresponse/)\>

Promise with void response

***

### updateUser()

> **updateUser**(`attributes`): `Promise`\<[`UserResponse`](/api/sdk/type-aliases/userresponse/)\>

Update the current user (Supabase-compatible)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `attributes` | [`UpdateUserAttributes`](/api/sdk/interfaces/updateuserattributes/) | User attributes to update (email, password, data for metadata) |

#### Returns

`Promise`\<[`UserResponse`](/api/sdk/type-aliases/userresponse/)\>

***

### verify2FA()

> **verify2FA**(`request`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`TwoFactorLoginResponse`\>\>

Verify 2FA code during login (Supabase-compatible)
Call this after signIn returns requires_2fa: true

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`TwoFactorVerifyRequest`](/api/sdk/interfaces/twofactorverifyrequest/) | User ID and TOTP code |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`TwoFactorLoginResponse`\>\>

Promise with access_token, refresh_token, and user

***

### verifyMagicLink()

> **verifyMagicLink**(`token`): `Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Verify magic link token and sign in

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `token` | `string` | Magic link token from email |

#### Returns

`Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

***

### verifyOtp()

> **verifyOtp**(`params`): `Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Verify OTP (One-Time Password) - Supabase-compatible
Verify OTP tokens for various authentication flows

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `params` | `VerifyOtpParams` | OTP verification parameters including token and type |

#### Returns

`Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Promise with user and session if successful

***

### verifyResetToken()

> **verifyResetToken**(`token`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`VerifyResetTokenResponse`\>\>

Verify password reset token
Check if a password reset token is valid before allowing password reset

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `token` | `string` | Password reset token to verify |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`VerifyResetTokenResponse`\>\>
