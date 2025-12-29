---
title: "SAML SSO"
---

Fluxbase supports SAML 2.0 Single Sign-On (SSO) for enterprise authentication. Integrate with identity providers like Okta, Azure AD, OneLogin, Google Workspace, and any SAML 2.0 compliant IdP.

## Overview

SAML SSO enables:

- **Enterprise authentication** - Users sign in with corporate credentials
- **Centralized identity** - Manage users from your IdP
- **Security compliance** - Meet enterprise security requirements
- **Automatic provisioning** - Create users on first login (optional)

## Supported Identity Providers

Fluxbase works with any SAML 2.0 compliant identity provider:

| Provider | Documentation |
|----------|---------------|
| Okta | [Okta SAML Setup](https://developer.okta.com/docs/guides/build-sso-integration/saml2/main/) |
| Azure AD | [Azure AD SAML](https://docs.microsoft.com/en-us/azure/active-directory/fundamentals/auth-saml) |
| Google Workspace | [Google SAML Apps](https://support.google.com/a/answer/6087519) |
| OneLogin | [OneLogin SAML](https://developers.onelogin.com/saml) |
| Auth0 | [Auth0 SAML](https://auth0.com/docs/protocols/saml) |
| Keycloak | [Keycloak SAML](https://www.keycloak.org/docs/latest/server_admin/#_saml) |
| PingIdentity | [PingIdentity SAML](https://docs.pingidentity.com/) |
| JumpCloud | [JumpCloud SSO](https://jumpcloud.com/support/saml-sso-integration) |

## Configuration

### YAML Configuration

```yaml
auth:
  saml_providers:
    - name: okta
      enabled: true
      idp_metadata_url: "https://company.okta.com/app/xxx/sso/saml/metadata"
      entity_id: "https://myapp.example.com/auth/saml"
      acs_url: "https://myapp.example.com/api/v1/auth/saml/acs"
      attribute_mapping:
        email: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
        name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name"
      auto_create_users: true
      default_role: "authenticated"
```

### Configuration Options

| Option | Description | Required |
|--------|-------------|----------|
| `name` | Provider identifier (used in URLs) | Yes |
| `enabled` | Enable this provider | Yes |
| `idp_metadata_url` | URL to IdP metadata XML | One of metadata_url or metadata_xml |
| `idp_metadata_xml` | Inline IdP metadata XML | One of metadata_url or metadata_xml |
| `entity_id` | SP Entity ID (unique identifier) | No (auto-generated) |
| `acs_url` | Assertion Consumer Service URL | No (auto-generated) |
| `attribute_mapping` | Map SAML attributes to user fields | No (defaults provided) |
| `auto_create_users` | Create user on first login | No (default: true) |
| `default_role` | Role for new users | No (default: authenticated) |

### Environment Variables

For a single provider, use environment variables:

```bash
FLUXBASE_AUTH_SAML_PROVIDERS_0_NAME=okta
FLUXBASE_AUTH_SAML_PROVIDERS_0_ENABLED=true
FLUXBASE_AUTH_SAML_PROVIDERS_0_IDP_METADATA_URL=https://company.okta.com/app/xxx/sso/saml/metadata
FLUXBASE_AUTH_SAML_PROVIDERS_0_AUTO_CREATE_USERS=true
```

## Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/saml/metadata/:provider` | GET | SP metadata XML for IdP registration |
| `/api/v1/auth/saml/login/:provider` | GET | Initiate SAML login (redirects to IdP) |
| `/api/v1/auth/saml/acs` | POST | Assertion Consumer Service (IdP callback) |
| `/api/v1/auth/saml/providers` | GET | List available SAML providers |

## Setup Guide

### Step 1: Configure Fluxbase

Add your SAML provider configuration:

```yaml
auth:
  saml_providers:
    - name: corporate-sso
      enabled: true
      idp_metadata_url: "https://idp.example.com/metadata"
      auto_create_users: true
```

### Step 2: Get SP Metadata

Fetch Fluxbase's Service Provider metadata:

```bash
curl https://myapp.example.com/api/v1/auth/saml/metadata/corporate-sso
```

This returns XML containing:
- Entity ID
- ACS URL
- Certificate (if configured)

### Step 3: Configure Your IdP

Register Fluxbase as a Service Provider in your IdP:

1. **Entity ID**: `https://myapp.example.com/auth/saml`
2. **ACS URL**: `https://myapp.example.com/api/v1/auth/saml/acs`
3. **Name ID Format**: `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress`

### Step 4: Configure Attribute Mapping

Map your IdP's attribute names to Fluxbase fields:

```yaml
attribute_mapping:
  # Standard SAML claims
  email: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
  name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name"

  # Or custom attribute names from your IdP
  email: "user.email"
  name: "user.displayName"
```

### Step 5: Test Login

```bash
# Redirect browser to initiate SAML login
open "https://myapp.example.com/api/v1/auth/saml/login/corporate-sso?redirect_to=https://myapp.example.com/dashboard"
```

## IdP-Specific Setup

### Okta

1. In Okta Admin Console, go to **Applications** → **Create App Integration**
2. Select **SAML 2.0**
3. Configure:
   - **Single sign-on URL**: `https://myapp.example.com/api/v1/auth/saml/acs`
   - **Audience URI (SP Entity ID)**: `https://myapp.example.com/auth/saml`
   - **Name ID format**: EmailAddress
4. Download the IdP metadata URL from **Sign On** tab
5. Configure Fluxbase:

```yaml
auth:
  saml_providers:
    - name: okta
      enabled: true
      idp_metadata_url: "https://your-org.okta.com/app/xxx/sso/saml/metadata"
```

### Azure AD

1. In Azure Portal, go to **Enterprise Applications** → **New application**
2. Select **Create your own application** → **Non-gallery**
3. Go to **Single sign-on** → **SAML**
4. Configure:
   - **Identifier (Entity ID)**: `https://myapp.example.com/auth/saml`
   - **Reply URL (ACS URL)**: `https://myapp.example.com/api/v1/auth/saml/acs`
5. Download **Federation Metadata XML**
6. Configure Fluxbase:

```yaml
auth:
  saml_providers:
    - name: azure-ad
      enabled: true
      idp_metadata_url: "https://login.microsoftonline.com/{tenant-id}/federationmetadata/2007-06/federationmetadata.xml"
      attribute_mapping:
        email: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
        name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/displayname"
```

### Google Workspace

1. In Google Admin Console, go to **Apps** → **Web and mobile apps**
2. Click **Add App** → **Add custom SAML app**
3. Download IdP metadata
4. Configure SP details:
   - **ACS URL**: `https://myapp.example.com/api/v1/auth/saml/acs`
   - **Entity ID**: `https://myapp.example.com/auth/saml`
5. Configure attribute mapping (email, name)
6. Configure Fluxbase:

```yaml
auth:
  saml_providers:
    - name: google
      enabled: true
      idp_metadata_xml: |
        <?xml version="1.0" encoding="UTF-8"?>
        <EntityDescriptor ...>
          <!-- Paste metadata XML here -->
        </EntityDescriptor>
```

## SDK Usage

### TypeScript SDK

```typescript
import { FluxbaseClient } from '@fluxbase/sdk'

const client = new FluxbaseClient({ url: 'http://localhost:8080' })

// List available SAML providers
const { data: providers } = await client.auth.getSAMLProviders()
// [{ name: 'okta', sso_url: '...', enabled: true }]

// Get SAML login URL
const { data: loginUrl } = await client.auth.getSAMLLoginUrl('okta', {
  redirectTo: 'https://myapp.example.com/dashboard'
})

// Redirect user to IdP
window.location.href = loginUrl

// After IdP callback, user is authenticated
// Session is automatically established
const { data: session } = await client.auth.getSession()
```

### React SDK

```tsx
import {
  useSAMLProviders,
  useInitiateSAMLLogin
} from '@fluxbase/sdk-react'

function SSOLoginButtons() {
  const { data: providers, isLoading } = useSAMLProviders()
  const initiateSAML = useInitiateSAMLLogin()

  if (isLoading) return <div>Loading...</div>

  return (
    <div>
      {providers?.map(provider => (
        <button
          key={provider.name}
          onClick={() => initiateSAML.mutate({
            provider: provider.name,
            redirectTo: '/dashboard'
          })}
        >
          Sign in with {provider.name}
        </button>
      ))}
    </div>
  )
}
```

### Handling Callback

After SAML authentication, users are redirected to your specified URL with a session established:

```tsx
// pages/auth/callback.tsx
import { useEffect } from 'react'
import { useSession } from '@fluxbase/sdk-react'
import { useNavigate } from 'react-router-dom'

function SAMLCallback() {
  const { data: session, isLoading } = useSession()
  const navigate = useNavigate()

  useEffect(() => {
    if (!isLoading && session) {
      // User is authenticated, redirect to dashboard
      navigate('/dashboard')
    }
  }, [session, isLoading])

  return <div>Completing sign in...</div>
}
```

## REST API

### List SAML Providers

```bash
GET /api/v1/auth/saml/providers
```

Response:
```json
{
  "providers": [
    {
      "name": "okta",
      "enabled": true,
      "sso_url": "https://company.okta.com/app/xxx/sso/saml"
    }
  ]
}
```

### Initiate SAML Login

```bash
GET /api/v1/auth/saml/login/:provider?redirect_to=https://myapp.example.com/dashboard
```

Redirects to IdP with SAML AuthnRequest.

### Get SP Metadata

```bash
GET /api/v1/auth/saml/metadata/:provider
```

Returns SP metadata XML for IdP registration.

## Attribute Mapping

Map SAML assertion attributes to user fields:

### Default Mapping

```yaml
attribute_mapping:
  email: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
  name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name"
```

### Common Attribute URIs

| Attribute | Standard URI |
|-----------|--------------|
| Email | `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress` |
| Name | `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name` |
| Given Name | `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname` |
| Surname | `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname` |
| UPN | `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/upn` |

### Custom Attributes

If your IdP uses custom attribute names:

```yaml
attribute_mapping:
  email: "userEmail"
  name: "displayName"
  # Additional attributes stored in user_metadata
  department: "department"
  title: "jobTitle"
```

## User Provisioning

### Auto-Create Users

When `auto_create_users: true`:

1. User authenticates via SAML
2. If user doesn't exist, account is created
3. Email and name populated from SAML attributes
4. User assigned `default_role`

### Disable Auto-Creation

Set `auto_create_users: false` to require pre-existing accounts:

```yaml
auth:
  saml_providers:
    - name: corporate-sso
      auto_create_users: false  # User must exist in database
```

Users must be created via admin API or invite before SAML login works.

### Linking Existing Accounts

SAML identity is linked to existing account if emails match:

```sql
-- SAML identity stored in auth.identities
SELECT * FROM auth.identities
WHERE provider = 'saml'
AND provider_id = 'okta:user@example.com';
```

## Security

### Best Practices

1. **Use HTTPS** - All SAML communication must use HTTPS
2. **Validate signatures** - Fluxbase validates IdP signatures automatically
3. **Time validation** - Assertions are rejected if expired
4. **Replay prevention** - Assertion IDs are tracked to prevent reuse
5. **Audience validation** - Assertions must be intended for your SP

### Certificate Validation

IdP certificates are extracted from metadata and used to validate assertion signatures:

```yaml
# IdP certificate is automatically extracted from metadata
# No manual configuration required for signature validation
```

### Session Security

SAML sessions are tracked:

```sql
-- View active SAML sessions
SELECT * FROM auth.saml_sessions
WHERE user_id = 'xxx'
ORDER BY created_at DESC;
```

## Troubleshooting

### Common Issues

**"SAML provider not found"**
- Verify provider name matches configuration
- Check provider is enabled

**"Failed to fetch IdP metadata"**
- Verify `idp_metadata_url` is accessible
- Check network connectivity
- Try using `idp_metadata_xml` directly

**"Invalid SAML assertion"**
- Check clock synchronization between servers
- Verify ACS URL matches IdP configuration
- Ensure Entity ID matches in IdP and Fluxbase

**"Email attribute not found"**
- Check attribute mapping configuration
- Verify IdP is sending email attribute
- Check SAML assertion for actual attribute names

### Debug Logging

Enable debug logging to troubleshoot SAML:

```yaml
debug: true
```

Check logs for:
- SAML request/response details
- Attribute extraction
- Signature validation

### Testing with SAML Tracer

Use browser extensions like [SAML Tracer](https://addons.mozilla.org/en-US/firefox/addon/saml-tracer/) to inspect:

- AuthnRequest sent to IdP
- SAML Response from IdP
- Assertion attributes

## Single Logout (SLO)

Single Logout allows signing out from all SSO applications:

```bash
# Initiate logout (redirects to IdP for SLO)
POST /api/v1/auth/saml/logout/:provider
```

Note: SLO support depends on your IdP configuration.

## Multiple Providers

Configure multiple SAML providers for different user groups:

```yaml
auth:
  saml_providers:
    - name: corporate
      enabled: true
      idp_metadata_url: "https://corporate.okta.com/metadata"
      auto_create_users: true

    - name: partner
      enabled: true
      idp_metadata_url: "https://partner.auth0.com/metadata"
      auto_create_users: false  # Partners must be pre-created
      default_role: "partner"
```

Frontend shows all providers:

```tsx
{providers.map(p => (
  <button onClick={() => loginWithSAML(p.name)}>
    Sign in with {p.name}
  </button>
))}
```

## Next Steps

- [Authentication](/docs/guides/authentication) - Authentication overview
- [OAuth Providers](/docs/guides/oauth-providers) - Social login configuration
- [Row-Level Security](/docs/guides/row-level-security) - Data access control
