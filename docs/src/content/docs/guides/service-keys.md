---
title: "Service Keys"
description: "Service keys are admin-managed, tenant-scoped API keys with elevated privileges for server-to-server access. Create, rotate, revoke, and audit them."
---

Service keys are **admin-managed** API keys used for server-to-server and privileged access. Unlike client keys (per-user, browser-safe) and JWTs (per-session), a service key grants elevated, often tenant-wide access and is meant to stay secret on your backend.

## When to use a service key

- A backend service or cron job calling the Fluxbase API as a privileged caller
- A CI/CD pipeline running migrations or syncing functions
- Any context where you need to bypass Row-Level Security or act on behalf of a tenant rather than a user

:::warning
Service keys (`service` / `global_service`) bypass RLS. Treat them like database credentials — store in a secret manager, never ship them to a browser, and rotate them regularly.
:::

## Key types

| Type | Prefix | Privilege | RLS |
|------|--------|-----------|-----|
| `anon` | `pk_live_` / `fb_anon_` | Anonymous access | Enforced (`anon` role) |
| `publishable` | `fb_pk_` | Browser-safe publishable key | Enforced (`authenticated`) |
| `service` | `sk_live_` | Elevated, legacy admin-API type | **Bypassed** (`service_role`) |
| `tenant_service` | `fb_tsk_` | Elevated, **tenant-scoped** | Enforced for the tenant (`tenant_service`) |
| `global_service` | `fb_gsk_` | Elevated, cross-tenant | **Bypassed** (`service_role`) |

The admin create endpoint only emits `anon` and `service`; the wider set (`publishable` / `tenant_service` / `global_service`) is produced by tenant bootstrap and config. See the SDK's `ServiceKey` type for the full union.

## Authentication

Send the key in any of these forms:

```bash
# Preferred
curl -H "X-Service-Key: sk_live_..." https://api.example.com/api/v1/...

# Or
curl -H "Authorization: ServiceKey sk_live_..." https://api.example.com/api/v1/...

# A service-role JWT also works
curl -H "Authorization: Bearer eyJ..." https://api.example.com/api/v1/...
```

Expired, disabled, or revoked keys are rejected at auth time. See [Client Keys](/guides/client-keys/) for the client-vs-service-vs-JWT comparison.

## Management API

All endpoints are under `/api/v1/admin/service-keys` and require the `admin`, `instance_admin`, or `tenant_admin` role (`tenant_admin` is tenant-scoped via RLS).

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/` | List service keys |
| `POST` | `/` | Create a key (full plaintext returned once) |
| `GET` | `/{id}` | Get a key |
| `PUT` | `/{id}` | Update name/description/scopes/namespaces/enabled/limits |
| `DELETE` | `/{id}` | Hard-delete the key |
| `POST` | `/{id}/disable` | Soft-disable (reversible) |
| `POST` | `/{id}/enable` | Re-enable |
| `POST` | `/{id}/revoke` | Emergency revoke (immediate, permanent) |
| `POST` | `/{id}/deprecate` | Mark deprecated with a grace period |
| `POST` | `/{id}/rotate` | Create a replacement and deprecate the old key |
| `GET` | `/{id}/revocations` | Revocation audit history |

Create returns the full key **once** — store it immediately.

```bash
curl -X POST https://api.example.com/api/v1/admin/service-keys \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"CI pipeline","key_type":"service","scopes":["*"]}'
```

## SDK

```typescript
const { data, error } = await client.admin.serviceKeys.create({
  name: "CI pipeline",
  key_type: "service",
  scopes: ["*"],
});
// data.key is the full key — save it now, it's not retrievable later

// Rotate: returns a new full key; the old key is deprecated
const { data: rotated } = await client.admin.serviceKeys.rotate(id);

// Revoke immediately
await client.admin.serviceKeys.revoke(id, { reason: "compromised" });

// Audit history (multi-event)
const { data: history } = await client.admin.serviceKeys.getRevocationHistory(id);
```

Requires an admin token: `client.admin.setToken(...)`. See the [SDK admin reference](/sdk/admin/) for all methods.

## Lifecycle: what actually invalidates a key

This is the important part — **rotate and deprecate do not stop a key from working on their own**.

| Operation | DB effect | Stops authenticating? | Reversible |
|-----------|-----------|----------------------|------------|
| **disable** | `enabled=false` | **Immediately** | Yes (`enable`) |
| **enable** | `enabled=true` | — | — |
| **revoke** | `revoked_at=now`, `enabled=false` + audit row (`emergency`) | **Immediately, permanently** | No |
| **deprecate** | `deprecated_at=now`, `grace_period_ends_at` set | **No** — the key keeps working past the grace period until you disable/revoke/delete it | Clear the fields via SQL |
| **rotate** | New key created; old key `deprecated_at` + `replaced_by` set + audit row (`rotation`) | **No** — the old key keeps working until you disable/revoke/delete it | — |
| **delete** | Row removed (audit rows cascade-delete) | **Immediately** | No |

:::note[Practical rotation workflow]
1. `rotate` → deploy the new key.
2. Confirm all callers use the new key.
3. `revoke` (or `disable` first if you want a reversible step) the old key.
4. `delete` the old key once you're confident.

The grace period from `deprecate`/`rotate` is a **tracking date**, not a kill switch.
:::

## Revocation audit

Every `revoke` and `rotate` writes a row to the `auth.service_key_revocations` audit table (`revocation_type` = `emergency` / `rotation`). Query it via `GET /{id}/revocations` or `client.admin.serviceKeys.getRevocationHistory(id)`. Note: deleting a key cascade-deletes its audit rows.

## CLI

```bash
fluxbase servicekeys list
fluxbase servicekeys create --name "CI" --type service
fluxbase servicekeys rotate <id>
fluxbase servicekeys revoke <id> --reason "compromised"
fluxbase servicekeys revocations <id>
```

See [CLI: Service Keys](/cli/commands/#service-key-commands).

## Learn More

- [Client Keys](/guides/client-keys/) — the per-user, browser-safe counterpart
- [Authentication](/guides/authentication/) — JWTs and the auth model
- [Multi-Tenancy](/guides/multi-tenancy/) — tenant-scoped key types
- [Endpoint Protection](/security/endpoint-protection/) — per-key rate limits and threat model
