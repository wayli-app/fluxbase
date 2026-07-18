---
editUrl: false
next: false
prev: false
title: "useAuthConfig"
---

> **useAuthConfig**(): `UseQueryResult`\<`NoInfer`\<`AuthConfig`\>, `Error`\>

Hook to get the complete authentication configuration from the server

Returns all public auth settings in a single request including:
- Signup enabled status
- Email verification requirements
- Magic link availability
- MFA availability
- Password requirements (length, complexity)
- Available OAuth providers (Google, GitHub, etc.)
- Available SAML providers (enterprise SSO)
- CAPTCHA configuration

Use this to conditionally render UI elements based on server configuration,
such as hiding signup forms when signup is disabled or displaying available
OAuth provider buttons.

## Returns

`UseQueryResult`\<`NoInfer`\<`AuthConfig`\>, `Error`\>

Query result with authentication configuration

## Examples

```tsx
function AuthPage() {
  const { data: config, isLoading } = useAuthConfig();

  if (isLoading) return <Loading />;

  return (
    <div>
      {config?.signup_enabled && (
        <SignupForm passwordMinLength={config.password_min_length} />
      )}

      {config?.oauth_providers.map(provider => (
        <OAuthButton
          key={provider.provider}
          provider={provider.provider}
          displayName={provider.display_name}
          authorizeUrl={provider.authorize_url}
        />
      ))}

      {config?.saml_providers.map(provider => (
        <SAMLButton
          key={provider.provider}
          provider={provider.provider}
          displayName={provider.display_name}
        />
      ))}
    </div>
  );
}
```

**Showing password requirements**

```tsx
function PasswordInput() {
  const { data: config } = useAuthConfig();

  return (
    <div>
      <input type="password" minLength={config?.password_min_length} />
      <ul>
        <li>Minimum {config?.password_min_length || 8} characters</li>
        {config?.password_require_uppercase && <li>One uppercase letter</li>}
        {config?.password_require_lowercase && <li>One lowercase letter</li>}
        {config?.password_require_number && <li>One number</li>}
        {config?.password_require_special && <li>One special character</li>}
      </ul>
    </div>
  );
}
```
