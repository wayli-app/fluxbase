---
title: "Dashboard Auth"
description: "The /dashboard/auth/* surface authenticates Fluxbase operators (platform admins) — distinct from /api/v1/auth/* which backs your application's end-users."
---

Fluxbase has **two separate authentication surfaces**. Understanding which one you're dealing with avoids a common source of confusion:

| Surface | Path prefix | Authenticates | User table | Roles |
|---------|-------------|---------------|------------|-------|
| **Application auth** | `/api/v1/auth/*` | End-users of the apps you build | `auth.users` | `authenticated`, `anon`, … |
| **Dashboard auth** | `/dashboard/auth/*` | **Operators of Fluxbase itself** | `platform.users` | `instance_admin`, `tenant_admin`, `dashboard_user` |

The admin UI (`/admin/login`) uses **dashboard auth**. Dashboard users are the people who administer tenants, manage service keys, and configure providers — they are not the users of your customer applications.

Both surfaces share the same JWT signing key and issuer (`fluxbase`); the distinction is enforced by the `role` claim and the middleware that guards each surface.

## Signup

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/dashboard/auth/signup` | Public | First-user self-registration only. Refuses with `403` once any `platform.users` exist; afterwards accounts are invite/SSO provisioned. |

## Login, refresh, and 2FA

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/dashboard/auth/login` | Public | Email/password login. Returns tokens, or `{ requires_2fa, user_id }` if TOTP is enabled. |
| `POST` | `/dashboard/auth/refresh` | Public | Exchange a refresh token for a new access token. |
| `POST` | `/dashboard/auth/2fa/verify` | Public | Complete login for a 2FA-pending user (`user_id` + TOTP `code`). |

## Password reset

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/dashboard/auth/password/reset` | Public | Trigger a reset email (always `200` to prevent enumeration). |
| `POST` | `/dashboard/auth/password/reset/verify` | Public | Check a reset token's validity before showing the form. |
| `POST` | `/dashboard/auth/password/reset/confirm` | Public | Set a new password with a valid token. |

## SSO (OAuth & SAML)

Providers must have `allow_dashboard_login = true` to appear here.

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/dashboard/auth/sso/providers` | Public | List enabled SSO providers + `password_login_disabled` flag. |
| `GET` | `/dashboard/auth/sso/oauth/:provider` | Public | Returns `{ url }` to start an OAuth flow. |
| `GET` | `/dashboard/auth/sso/oauth/:provider/callback` | Public | OAuth callback; redirects to the admin UI with tokens in the URL fragment. |
| `GET` | `/dashboard/auth/sso/saml/:provider` | Public | Redirect to the SAML IdP. |
| `POST` | `/dashboard/auth/sso/saml/acs` | Public | Consume the SAML response; creates a session and redirects with tokens. |

## Profile & account (require dashboard auth)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/dashboard/auth/me` | Required | Current dashboard user. |
| `PUT` | `/dashboard/auth/profile` | Required | Update `full_name` / `avatar_url`. |
| `POST` | `/dashboard/auth/password/change` | Required | Change password (requires `current_password`). |
| `DELETE` | `/dashboard/auth/account` | Required | Soft-delete the account (requires `password` confirmation). |

## 2FA management (require dashboard auth)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/dashboard/auth/2fa/setup` | Required | Generate a TOTP secret + QR URL. |
| `POST` | `/dashboard/auth/2fa/enable` | Required | Verify the first code; enables TOTP and returns backup codes. |
| `POST` | `/dashboard/auth/2fa/disable` | Required | Disable TOTP (requires `password`). |

## Auth model

- **Middleware:** `RequireDashboardAuth` validates the Bearer access token and **rejects unless `role == "instance_admin"`** for protected routes.
- **Tokens:** issued by the same `JWTManager` as application auth; the `role` claim (`instance_admin` / `tenant_admin`) and tenant context (`tenant_id`, `tenant_role`, `is_instance_admin`) are baked into the JWT.
- **Sessions:** one active session per user in `platform.sessions`; old sessions are replaced on login.

:::note[Refresh-surface quirk]
The admin UI's axios refresh interceptor actually calls the sibling `/api/v1/admin/refresh` endpoint, not `/dashboard/auth/refresh`. Both refresh paths work; the SPA uses the admin one. There is **no logout endpoint** on `/dashboard/auth/*` — logout lives at `/api/v1/admin/logout`.
:::

## What's intentionally absent

This surface deliberately omits application-only features: magic links, OTP sign-in, anonymous auth, impersonation, identity link/unlink, and open signup. For those, use `/api/v1/auth/*` ([Authentication](/guides/authentication/)).

## Learn More

- [Authentication](/guides/authentication/) — application end-user auth
- [OAuth Providers](/guides/oauth-providers/) / [SAML SSO](/guides/saml-sso/) — enabling `allow_dashboard_login`
- [Multi-Tenancy](/guides/multi-tenancy/) — tenant admin assignments
