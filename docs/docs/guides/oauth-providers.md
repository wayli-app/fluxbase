# OAuth Provider Integration Guide

This guide provides detailed examples for integrating OAuth providers with Fluxbase in popular frameworks.

## Overview

Fluxbase supports OAuth 2.0 authentication with multiple providers, allowing users to sign in with their existing accounts. This guide covers setup and implementation for each supported provider.

## Supported Providers

- Google
- GitHub
- Microsoft / Azure AD
- Apple
- Facebook
- Twitter
- LinkedIn
- GitLab
- Bitbucket
- Custom OAuth 2.0 providers

## Framework-Specific Examples

### React (with React Router)

```typescript
// src/components/OAuth.tsx
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { fluxbase } from '../lib/fluxbase';

export function OAuthButtons() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState<string | null>(null);

  const signInWithProvider = async (provider: string) => {
    try {
      setLoading(provider);

      const { url } = await fluxbase.auth.signInWithOAuth({
        provider,
        options: {
          redirectTo: `${window.location.origin}/auth/callback`,
        },
      });

      // Redirect to OAuth provider
      window.location.href = url;
    } catch (error) {
      console.error('OAuth error:', error);
      setLoading(null);
    }
  };

  return (
    <div className="space-y-2">
      <button
        onClick={() => signInWithProvider('google')}
        disabled={loading !== null}
        className="w-full"
      >
        {loading === 'google' ? 'Loading...' : 'Sign in with Google'}
      </button>

      <button
        onClick={() => signInWithProvider('github')}
        disabled={loading !== null}
        className="w-full"
      >
        {loading === 'github' ? 'Loading...' : 'Sign in with GitHub'}
      </button>
    </div>
  );
}

// src/pages/AuthCallback.tsx
import { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { fluxbase } from '../lib/fluxbase';

export function AuthCallback() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  useEffect(() => {
    const handleCallback = async () => {
      const code = searchParams.get('code');
      const state = searchParams.get('state');
      const error = searchParams.get('error');

      if (error) {
        console.error('OAuth error:', error);
        navigate('/login?error=' + error);
        return;
      }

      if (!code || !state) {
        navigate('/login?error=missing_params');
        return;
      }

      try {
        const { user, session } = await fluxbase.auth.exchangeCodeForSession({
          code,
          state,
        });

        // Store session
        localStorage.setItem('session', JSON.stringify(session));

        // Redirect to dashboard
        navigate('/dashboard');
      } catch (error) {
        console.error('Session exchange error:', error);
        navigate('/login?error=exchange_failed');
      }
    };

    handleCallback();
  }, [searchParams, navigate]);

  return <div>Completing sign in...</div>;
}
```

### Next.js (App Router)

```typescript
// app/auth/oauth/route.ts
import { NextRequest, NextResponse } from 'next/server';
import { createClient } from '@/lib/fluxbase';

export async function GET(request: NextRequest) {
  const searchParams = request.nextUrl.searchParams;
  const provider = searchParams.get('provider');

  if (!provider) {
    return NextResponse.json(
      { error: 'Provider is required' },
      { status: 400 }
    );
  }

  const fluxbase = createClient();

  try {
    const { url } = await fluxbase.auth.signInWithOAuth({
      provider,
      options: {
        redirectTo: `${request.nextUrl.origin}/auth/callback`,
      },
    });

    return NextResponse.redirect(url);
  } catch (error) {
    return NextResponse.json(
      { error: 'Failed to initiate OAuth flow' },
      { status: 500 }
    );
  }
}

// app/auth/callback/route.ts
import { NextRequest, NextResponse } from 'next/server';
import { createClient } from '@/lib/fluxbase';
import { cookies } from 'next/headers';

export async function GET(request: NextRequest) {
  const searchParams = request.nextUrl.searchParams;
  const code = searchParams.get('code');
  const state = searchParams.get('state');
  const error = searchParams.get('error');

  if (error) {
    return NextResponse.redirect(
      `${request.nextUrl.origin}/login?error=${error}`
    );
  }

  if (!code || !state) {
    return NextResponse.redirect(
      `${request.nextUrl.origin}/login?error=missing_params`
    );
  }

  const fluxbase = createClient();

  try {
    const { user, session } = await fluxbase.auth.exchangeCodeForSession({
      code,
      state,
    });

    // Set HTTP-only cookie
    cookies().set('session', JSON.stringify(session), {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      maxAge: 60 * 60 * 24 * 7, // 7 days
    });

    return NextResponse.redirect(`${request.nextUrl.origin}/dashboard`);
  } catch (error) {
    console.error('Session exchange error:', error);
    return NextResponse.redirect(
      `${request.nextUrl.origin}/login?error=exchange_failed`
    );
  }
}

// components/OAuthButtons.tsx
'use client';

export function OAuthButtons() {
  const signInWith = (provider: string) => {
    window.location.href = `/auth/oauth?provider=${provider}`;
  };

  return (
    <div className="space-y-2">
      <button
        onClick={() => signInWith('google')}
        className="w-full"
      >
        Sign in with Google
      </button>

      <button
        onClick={() => signInWith('github')}
        className="w-full"
      >
        Sign in with GitHub
      </button>
    </div>
  );
}
```

### Vue 3 (with Vue Router)

```vue
<!-- components/OAuthButtons.vue -->
<template>
  <div class="space-y-2">
    <button
      @click="signInWithProvider('google')"
      :disabled="loading !== null"
      class="w-full"
    >
      {{ loading === 'google' ? 'Loading...' : 'Sign in with Google' }}
    </button>

    <button
      @click="signInWithProvider('github')"
      :disabled="loading !== null"
      class="w-full"
    >
      {{ loading === 'github' ? 'Loading...' : 'Sign in with GitHub' }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { fluxbase } from '@/lib/fluxbase';

const router = useRouter();
const loading = ref<string | null>(null);

const signInWithProvider = async (provider: string) => {
  try {
    loading.value = provider;

    const { url } = await fluxbase.auth.signInWithOAuth({
      provider,
      options: {
        redirectTo: `${window.location.origin}/auth/callback`,
      },
    });

    window.location.href = url;
  } catch (error) {
    console.error('OAuth error:', error);
    loading.value = null;
  }
};
</script>

<!-- pages/AuthCallback.vue -->
<template>
  <div>Completing sign in...</div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { fluxbase } from '@/lib/fluxbase';

const router = useRouter();
const route = useRoute();

onMounted(async () => {
  const { code, state, error } = route.query;

  if (error) {
    console.error('OAuth error:', error);
    router.push({ path: '/login', query: { error: error as string } });
    return;
  }

  if (!code || !state) {
    router.push({ path: '/login', query: { error: 'missing_params' } });
    return;
  }

  try {
    const { user, session } = await fluxbase.auth.exchangeCodeForSession({
      code: code as string,
      state: state as string,
    });

    localStorage.setItem('session', JSON.stringify(session));
    router.push('/dashboard');
  } catch (error) {
    console.error('Session exchange error:', error);
    router.push({ path: '/login', query: { error: 'exchange_failed' } });
  }
});
</script>
```

## Provider-Specific Configuration

### Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing
3. Enable Google+ API
4. Go to **Credentials** → **Create Credentials** → **OAuth 2.0 Client ID**
5. Add authorized redirect URI: `http://your-domain.com/api/v1/auth/callback/google`

**Configuration:**
```yaml
auth:
  oauth:
    google:
      client_id: "YOUR_GOOGLE_CLIENT_ID"
      client_secret: "YOUR_GOOGLE_CLIENT_SECRET"
      redirect_url: "http://localhost:8080/api/v1/auth/callback/google"
      scopes:
        - "openid"
        - "email"
        - "profile"
```

**User Data Mapping:**
- `email` → user.email
- `name` → user.full_name
- `picture` → user.avatar_url
- `sub` → user.provider_id

### GitHub OAuth

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Click **New OAuth App**
3. Fill in application details
4. Set Authorization callback URL: `http://your-domain.com/api/v1/auth/callback/github`

**Configuration:**
```yaml
auth:
  oauth:
    github:
      client_id: "YOUR_GITHUB_CLIENT_ID"
      client_secret: "YOUR_GITHUB_CLIENT_SECRET"
      redirect_url: "http://localhost:8080/api/v1/auth/callback/github"
      scopes:
        - "user:email"
```

**User Data Mapping:**
- `email` → user.email (from /user/emails endpoint)
- `name` → user.full_name
- `avatar_url` → user.avatar_url
- `id` → user.provider_id

### Microsoft / Azure AD

1. Go to [Azure Portal](https://portal.azure.com/)
2. Navigate to **Azure Active Directory** → **App registrations**
3. Click **New registration**
4. Add redirect URI: `http://your-domain.com/api/v1/auth/callback/microsoft`

**Configuration:**
```yaml
auth:
  oauth:
    microsoft:
      client_id: "YOUR_MICROSOFT_CLIENT_ID"
      client_secret: "YOUR_MICROSOFT_CLIENT_SECRET"
      redirect_url: "http://localhost:8080/api/v1/auth/callback/microsoft"
      scopes:
        - "openid"
        - "email"
        - "profile"
```

### Custom OAuth Provider

For providers not natively supported, you can configure a custom OAuth 2.0 provider:

```yaml
auth:
  oauth:
    custom:
      my_provider:
        name: "My Custom Provider"
        authorization_url: "https://provider.com/oauth/authorize"
        token_url: "https://provider.com/oauth/token"
        userinfo_url: "https://provider.com/oauth/userinfo"
        client_id: "YOUR_CLIENT_ID"
        client_secret: "YOUR_CLIENT_SECRET"
        redirect_url: "http://localhost:8080/api/v1/auth/callback/my_provider"
        scopes:
          - "openid"
          - "email"
          - "profile"
```

**Usage:**
```typescript
const { url } = await fluxbase.auth.signInWithOAuth({
  provider: "my_provider",
});
```

## Best Practices

### 1. Handle OAuth Errors Gracefully

```typescript
try {
  const { url } = await fluxbase.auth.signInWithOAuth({
    provider: 'google',
  });
  window.location.href = url;
} catch (error) {
  if (error.code === 'provider_not_configured') {
    showError('Google sign-in is not available');
  } else if (error.code === 'network_error') {
    showError('Network error. Please try again.');
  } else {
    showError('An error occurred. Please try again.');
  }
}
```

### 2. Store State Securely

The `state` parameter prevents CSRF attacks. Fluxbase generates and validates it automatically, but you can provide your own:

```typescript
const { url } = await fluxbase.auth.signInWithOAuth({
  provider: 'google',
  options: {
    state: generateSecureRandomString(), // Optional: custom state
  },
});
```

### 3. Handle Multiple Providers

```typescript
const providers = [
  { id: 'google', name: 'Google', icon: GoogleIcon },
  { id: 'github', name: 'GitHub', icon: GitHubIcon },
  { id: 'microsoft', name: 'Microsoft', icon: MicrosoftIcon },
];

function SignInOptions() {
  return (
    <div className="space-y-2">
      {providers.map((provider) => (
        <button
          key={provider.id}
          onClick={() => signInWith(provider.id)}
          className="flex items-center gap-2 w-full"
        >
          <provider.icon className="w-5 h-5" />
          Sign in with {provider.name}
        </button>
      ))}
    </div>
  );
}
```

### 4. Link Existing Account

Allow users to link OAuth providers to existing accounts:

```typescript
// User is already authenticated
const { user } = await fluxbase.auth.getUser();

if (user) {
  // Link OAuth provider to existing account
  const { url } = await fluxbase.auth.linkIdentity({
    provider: 'google',
  });

  window.location.href = url;
}
```

### 5. Unlink OAuth Provider

```typescript
await fluxbase.auth.unlinkIdentity({
  provider: 'google',
});
```

## Troubleshooting

### "Redirect URI mismatch" Error

**Problem:** The OAuth provider rejects the redirect URI.

**Solution:**
1. Ensure the redirect URI in your Fluxbase config matches exactly what's registered with the provider
2. Include the protocol (http/https)
3. Don't include trailing slashes unless the provider requires them
4. For localhost development, some providers require explicit registration of localhost URIs

### "Invalid State Parameter" Error

**Problem:** State validation fails during callback.

**Solution:**
1. Ensure cookies are enabled (state is stored in a cookie)
2. Check that your domain allows cross-site cookies if needed
3. Verify the `state` parameter is passed correctly from OAuth provider

### Users Can't Sign In

**Problem:** OAuth flow completes but user isn't authenticated.

**Solution:**
1. Check that the OAuth provider returns required fields (email is usually required)
2. Verify scopes include necessary permissions
3. Check Fluxbase logs for specific errors
4. Ensure the OAuth provider's credentials are correct

### Development vs Production URLs

Use environment variables for redirect URLs:

```typescript
const redirectUrl =
  process.env.NODE_ENV === 'production'
    ? 'https://yourdomain.com/auth/callback'
    : 'http://localhost:3000/auth/callback';

const { url } = await fluxbase.auth.signInWithOAuth({
  provider: 'google',
  options: {
    redirectTo: redirectUrl,
  },
});
```

## Security Considerations

1. **Always use HTTPS in production** - OAuth requires secure connections
2. **Validate state parameter** - Fluxbase does this automatically
3. **Don't expose client secrets** - Keep them server-side only
4. **Implement PKCE** - Fluxbase uses PKCE for enhanced security
5. **Limit scopes** - Only request necessary permissions
6. **Monitor OAuth usage** - Watch for unusual patterns

## Learn More

- [Authentication Guide](/docs/guides/authentication)
- [REST API Reference](/docs/api/authentication)
- [SDK Documentation](/docs/api/sdk)
