---
editUrl: false
next: false
prev: false
title: "FluxbaseAuth"
---

## Constructors

### new FluxbaseAuth()

> **new FluxbaseAuth**(`fetch`, `autoRefresh`, `persist`): [`FluxbaseAuth`](/api/sdk/classes/fluxbaseauth/)

#### Parameters

| Parameter | Type | Default value |
| ------ | ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) | `undefined` |
| `autoRefresh` | `boolean` | `true` |
| `persist` | `boolean` | `true` |

#### Returns

[`FluxbaseAuth`](/api/sdk/classes/fluxbaseauth/)

## Methods

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

> **exchangeCodeForSession**(`code`): `Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Exchange OAuth authorization code for session
This is typically called in your OAuth callback handler

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `code` | `string` | Authorization code from OAuth callback |

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

> **getAccessToken**(): `null` \| `string`

Get the current access token

#### Returns

`null` \| `string`

***

### getCurrentUser()

> **getCurrentUser**(): `Promise`\<[`UserResponse`](/api/sdk/type-aliases/userresponse/)\>

Get the current user from the server

#### Returns

`Promise`\<[`UserResponse`](/api/sdk/type-aliases/userresponse/)\>

***

### getOAuthProviders()

> **getOAuthProviders**(): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OAuthProvidersResponse`\>\>

Get list of enabled OAuth providers

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OAuthProvidersResponse`\>\>

***

### getOAuthUrl()

> **getOAuthUrl**(`provider`, `options`?): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OAuthUrlResponse`\>\>

Get OAuth authorization URL for a provider

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `provider` | `string` | OAuth provider name (e.g., 'google', 'github') |
| `options`? | `OAuthOptions` | Optional OAuth configuration |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`OAuthUrlResponse`\>\>

***

### getSession()

> **getSession**(): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<`object`\>\>

Get the current session (Supabase-compatible)
Returns the session from the client-side cache without making a network request

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<`object`\>\>

***

### getUser()

> **getUser**(): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<`object`\>\>

Get the current user (Supabase-compatible)
Returns the user from the client-side session without making a network request
For server-side validation, use getCurrentUser() instead

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<`object`\>\>

***

### getUserIdentities()

> **getUserIdentities**(): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`UserIdentitiesResponse`\>\>

Get user identities (linked OAuth providers) - Supabase-compatible
Lists all OAuth identities linked to the current user

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`UserIdentitiesResponse`\>\>

Promise with list of user identities

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

> **refreshSession**(): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<`object`\>\>

Refresh the session (Supabase-compatible)
Returns a new session with refreshed tokens

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<`object`\>\>

***

### refreshToken()

> **refreshToken**(): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<`object`\>\>

Refresh the session (Supabase-compatible alias)
Alias for refreshSession() to maintain compatibility with Supabase naming
Returns a new session with refreshed tokens

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<`object`\>\>

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

> **resetPasswordForEmail**(`email`, `_options`?): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`PasswordResetResponse`\>\>

Supabase-compatible alias for sendPasswordReset()

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `email` | `string` | Email address to send reset link to |
| `_options`? | `object` | Optional redirect configuration (currently not used) |
| `_options.redirectTo`? | `string` | - |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`PasswordResetResponse`\>\>

Promise with OTP-style response

***

### sendMagicLink()

> **sendMagicLink**(`email`, `options`?): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`MagicLinkResponse`\>\>

Send magic link for passwordless authentication (Supabase-compatible)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `email` | `string` | Email address to send magic link to |
| `options`? | `MagicLinkOptions` | Optional configuration for magic link |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`MagicLinkResponse`\>\>

Promise with OTP-style response

***

### sendPasswordReset()

> **sendPasswordReset**(`email`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`PasswordResetResponse`\>\>

Send password reset email (Supabase-compatible)
Sends a password reset link to the provided email address

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `email` | `string` | Email address to send reset link to |

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
| `session` | `object` | Object containing access_token and refresh_token |
| `session.access_token` | `string` | - |
| `session.refresh_token` | `string` | - |

#### Returns

`Promise`\<[`FluxbaseAuthResponse`](/api/sdk/type-aliases/fluxbaseauthresponse/)\>

Promise with session data

***

### setup2FA()

> **setup2FA**(`issuer`?): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`TwoFactorSetupResponse`](/api/sdk/interfaces/twofactorsetupresponse/)\>\>

Setup 2FA for the current user (Supabase-compatible)
Enrolls a new MFA factor and returns TOTP details

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `issuer`? | `string` | Optional custom issuer name for the QR code (e.g., "MyApp"). If not provided, uses server default. |

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

> **signInWithOAuth**(`provider`, `options`?): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`object`\>\>

Convenience method to initiate OAuth sign-in
Redirects the user to the OAuth provider's authorization page

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `provider` | `string` | OAuth provider name (e.g., 'google', 'github') |
| `options`? | `OAuthOptions` | Optional OAuth configuration |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`object`\>\>

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

### signOut()

> **signOut**(): `Promise`\<[`VoidResponse`](/api/sdk/type-aliases/voidresponse/)\>

Sign out the current user

#### Returns

`Promise`\<[`VoidResponse`](/api/sdk/type-aliases/voidresponse/)\>

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
