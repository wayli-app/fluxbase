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

Single Logout (SLO) allows users to sign out from both Fluxbase and the IdP simultaneously, terminating sessions across all SSO applications.

### Overview

Fluxbase supports both:

- **SP-initiated SLO** - User logs out from Fluxbase, which notifies the IdP
- **IdP-initiated SLO** - IdP sends logout request to Fluxbase when user logs out elsewhere

### Configuration

To enable SLO, configure SP signing keys for your SAML provider:

```yaml
auth:
  saml_providers:
    - name: okta
      enabled: true
      idp_metadata_url: "https://company.okta.com/app/xxx/sso/saml/metadata"
      # SP signing keys for SLO (PEM-encoded)
      sp_certificate: |
        -----BEGIN CERTIFICATE-----
        MIICpDCCAYwCCQDU+pQ4P1eLvjANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDDAls
        ...
        -----END CERTIFICATE-----
      sp_private_key: |
        -----BEGIN RSA PRIVATE KEY-----
        MIIEowIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGy...
        ...
        -----END RSA PRIVATE KEY-----
```

### Generating SP Signing Keys

Generate a self-signed certificate and key for signing LogoutRequests:

```bash
# Generate private key
openssl genrsa -out sp-key.pem 2048

# Generate certificate (valid for 10 years)
openssl req -new -x509 -key sp-key.pem -out sp-cert.pem -days 3650 \
  -subj "/CN=myapp.example.com"

# View certificate details
openssl x509 -in sp-cert.pem -text -noout
```

### SLO Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/saml/logout/:provider` | GET | Initiate SP-initiated logout |
| `/api/v1/auth/saml/slo` | POST/GET | Handle IdP-initiated logout & SP callback |

### SP-Initiated Logout

When a user logs out from your application:

```bash
# Initiate SAML logout (redirects to IdP)
GET /api/v1/auth/saml/logout/okta?redirect_url=https://myapp.example.com/goodbye
```

The flow:
1. Fluxbase generates a signed LogoutRequest
2. User is redirected to IdP's SLO endpoint
3. IdP terminates the session
4. IdP redirects back to Fluxbase with LogoutResponse
5. User is redirected to `redirect_url`

### IdP-Initiated Logout

When a user logs out from the IdP or another SP:

1. IdP sends LogoutRequest to `/api/v1/auth/saml/slo`
2. Fluxbase finds and terminates the SAML session
3. All JWT tokens for the user are revoked
4. Fluxbase sends LogoutResponse back to IdP

### SDK Usage

```typescript
// Check if user has SAML session with SLO support
const { data } = await client.auth.signOut()

if (data.saml_logout && data.slo_url) {
  // Redirect to IdP for full logout
  window.location.href = data.slo_url
} else {
  // Local logout only
  console.log('Signed out locally')
}
```

### React SDK

```tsx
import { useSignOut, useSession } from '@fluxbase/sdk-react'

function LogoutButton() {
  const signOut = useSignOut()

  const handleLogout = async () => {
    const result = await signOut.mutateAsync()

    if (result.saml_logout && result.slo_url) {
      // Full SAML SLO - redirect to IdP
      window.location.href = result.slo_url
    } else {
      // Local logout complete
      window.location.href = '/login'
    }
  }

  return <button onClick={handleLogout}>Sign Out</button>
}
```

### IdP Configuration for SLO

Configure your IdP to send logout requests to Fluxbase:

**Okta:**
1. In your SAML app, go to **General** → **SAML Settings**
2. Enable **Single Logout**
3. Set **Single Logout URL**: `https://myapp.example.com/api/v1/auth/saml/slo`
4. Upload your SP certificate (from `sp_certificate` config)

**Azure AD:**
1. Go to **Single sign-on** → **SAML**
2. Set **Logout URL**: `https://myapp.example.com/api/v1/auth/saml/slo`

**Google Workspace:**
1. In your SAML app settings
2. Enable **Signed Response**
3. Set **Logout URL**: `https://myapp.example.com/api/v1/auth/saml/slo`

### Graceful Degradation

If SLO is not available (no IdP SLO URL or missing signing keys), Fluxbase performs local logout only:

- SAML session is deleted
- JWT tokens are revoked
- User is signed out from Fluxbase
- No IdP notification (user remains logged in at IdP)

The signout response indicates this:

```json
{
  "message": "local logout successful",
  "saml_logout": false
}
```

### Troubleshooting SLO

**"SP signing key not configured"**
- Add `sp_certificate` and `sp_private_key` to provider config
- Ensure keys are valid PEM format

**LogoutRequest rejected by IdP**
- Verify SP certificate is registered in IdP
- Check certificate hasn't expired
- Ensure IdP has SLO enabled

**IdP-initiated logout not working**
- Verify SLO URL is configured in IdP
- Check Fluxbase is accessible from IdP
- Review logs for incoming LogoutRequests

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

## Role-Based Access Control (RBAC)

Control which users can authenticate based on their Active Directory groups, Azure AD groups, or other SAML attributes from your identity provider.

### Group-Based Access Control

Restrict SAML authentication based on group membership:

```yaml
auth:
  saml_providers:
    - name: azure-ad
      enabled: true
      idp_metadata_url: "https://login.microsoftonline.com/{tenant}/metadata"
      allow_dashboard_login: true

      # RBAC configuration
      required_groups:
        - "FluxbaseAdmins"      # User must be in at least ONE of these groups
        - "FluxbaseDevelopers"

      required_groups_all:
        - "Verified"            # User must be in ALL of these groups
        - "Active"

      denied_groups:
        - "Contractors"         # Reject users in ANY of these groups
        - "Suspended"

      group_attribute: "groups" # SAML attribute containing groups (default: "groups")
```

### RBAC Rules

Three types of group validation rules:

| Rule | Logic | Example Use Case |
|------|-------|------------------|
| `required_groups` | **OR logic** - User must be in at least ONE group | Allow admins OR editors |
| `required_groups_all` | **AND logic** - User must be in ALL groups | Must be verified AND active |
| `denied_groups` | **DENY** - Reject if user is in ANY of these groups | Block contractors or suspended accounts |

**Execution order**: Denied groups are checked first (highest priority), then required groups.

### Common Group Attributes

Different identity providers use different SAML attribute names for groups:

| Identity Provider | Default Attribute | Alternative Attributes |
|-------------------|-------------------|------------------------|
| Azure AD | `http://schemas.microsoft.com/ws/2008/06/identity/claims/groups` | `groups` (if configured) |
| Okta | `groups` | Custom attribute mapping |
| Google Workspace | `groups` | Custom schema |
| Active Directory | `memberOf` | `http://schemas.xmlsoap.org/claims/Group` |

Configure the attribute name:

```yaml
group_attribute: "memberOf"  # For Active Directory
```

### Example Configurations

**Dashboard admin access (Azure AD)**

Only users in IT or Admin groups can access the dashboard:

```yaml
auth:
  saml_providers:
    - name: azure-ad-dashboard
      enabled: true
      idp_metadata_url: "https://login.microsoftonline.com/{tenant}/metadata"
      allow_dashboard_login: true
      allow_app_login: false
      required_groups:
        - "FluxbaseAdmins"
        - "IT-Team"
```

**Multi-tier access (Okta)**

Admins and editors can access, but contractors are explicitly blocked:

```yaml
auth:
  saml_providers:
    - name: okta-corporate
      enabled: true
      idp_metadata_url: "https://company.okta.com/metadata"
      allow_dashboard_login: true
      required_groups:
        - "Admins"
        - "Editors"
      denied_groups:
        - "Contractors"
        - "Guests"
```

**Strict verification (Active Directory)**

Users must be in Admin group AND have valid employee status:

```yaml
auth:
  saml_providers:
    - name: ad-sso
      enabled: true
      idp_metadata_url: "https://adfs.company.com/metadata"
      allow_dashboard_login: true
      required_groups:
        - "Domain Admins"
      required_groups_all:
        - "CN=Employees,OU=Groups,DC=company,DC=com"
        - "CN=Active,OU=Status,DC=company,DC=com"
      group_attribute: "memberOf"
```

### Error Messages

When group validation fails, users see clear error messages:

- `"Access denied: user is member of restricted group 'Contractors'"` - User in denied group
- `"Access denied: missing required group 'FluxbaseAdmins'"` - User missing a required group from `required_groups_all`
- `"Access denied: user must be member of one of: [Admins, Editors]"` - User doesn't have any of the `required_groups`

### Troubleshooting RBAC

**"Groups not being extracted"**

1. Check the group attribute name:
   ```bash
   # View SAML assertion to see actual attribute names
   # Use browser developer tools or SAML Tracer extension
   ```

2. Configure the correct attribute:
   ```yaml
   group_attribute: "memberOf"  # Or the correct attribute name
   ```

**"User has group but still rejected"**

- Group names are **case-sensitive**: `"Admins"` ≠ `"admins"`
- Check for whitespace in group names
- Verify exact group name from IdP

**"Azure AD not sending group names"**

Azure AD can be configured to send either:
- Group UUIDs (default): `["abc123...", "def456..."]`
- Group names: `["FluxbaseAdmins", "Developers"]`

To send group names instead of UUIDs:
1. In Azure Portal, go to **Enterprise Applications** → Your App → **Single sign-on**
2. Edit **Attributes & Claims**
3. Edit the `groups` claim
4. Set **Source attribute** to `group.displayname` (instead of `group.objectid`)

**"How do I find my group names/IDs?"**

Use SAML Tracer browser extension to inspect the actual SAML assertion and see the groups being sent by your IdP.

## Dashboard SSO (Admin Login)

SAML and OAuth providers can be used for dashboard admin authentication, enabling SSO-only mode where password login is disabled.

### Enable Dashboard SSO

When creating or editing an SSO provider, enable "Allow dashboard login":

```yaml
auth:
  saml_providers:
    - name: corporate-sso
      enabled: true
      idp_metadata_url: "https://company.okta.com/metadata"
      allow_dashboard_login: true   # Enable for admin login
      allow_app_login: true         # Also allow for app users
```

### Disable Password Login

Once SSO is configured for dashboard login, you can disable password authentication:

1. Go to **Authentication** → **Auth Settings** in the dashboard
2. Enable **Disable Password Login** under "Dashboard Login"
3. Save settings

When enabled:
- The login page shows only SSO buttons
- Password form is hidden
- Backend rejects password login attempts

### CLI SSO Login

With password login disabled, use the `--sso` flag for CLI authentication:

```bash
# SSO login (opens browser)
fluxbase auth login --server https://api.example.com --sso

# Or use an API token
fluxbase auth login --server https://api.example.com --token your-api-token
```

The CLI automatically detects when password login is disabled and initiates SSO flow.

### Emergency Recovery

If you're locked out due to misconfigured SSO, set the environment variable to bypass the setting:

```bash
FLUXBASE_DASHBOARD_FORCE_PASSWORD_LOGIN=true
```

This temporarily re-enables password login regardless of the database setting.

### Security Considerations

1. **Require at least one SSO provider** - Cannot disable password login without configured SSO
2. **Test SSO login first** - Verify SSO works before disabling passwords
3. **Document recovery procedures** - Ensure administrators know about the env var override
4. **Audit SSO provider changes** - Monitor for accidental removal of SSO providers

## Next Steps

- [Authentication](/docs/guides/authentication) - Authentication overview
- [OAuth Providers](/docs/guides/oauth-providers) - Social login configuration
- [Row-Level Security](/docs/guides/row-level-security) - Data access control
