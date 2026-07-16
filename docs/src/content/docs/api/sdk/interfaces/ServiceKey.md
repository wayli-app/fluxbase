---
editUrl: false
next: false
prev: false
title: "ServiceKey"
---

Service key for API authentication
Each tenant has their own service_keys table in their database

## Extended by

- [`ServiceKeyWithKey`](/api/sdk/interfaces/servicekeywithkey/)

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="allowed_namespaces"></a> `allowed_namespaces?` | `string`[] | Allowed table namespaces |
| <a id="created_at"></a> `created_at` | `string` | Creation timestamp |
| <a id="created_by"></a> `created_by?` | `string` | User who created the key |
| <a id="deprecated_at"></a> `deprecated_at?` | `string` | Deprecation timestamp (for key rotation) |
| <a id="description"></a> `description?` | `string` | Description |
| <a id="enabled"></a> `enabled` | `boolean` | Whether the key is enabled |
| <a id="expires_at"></a> `expires_at?` | `string` | Expiration timestamp |
| <a id="grace_period_ends_at"></a> `grace_period_ends_at?` | `string` | Grace period end for deprecated keys |
| <a id="id"></a> `id` | `string` | Unique identifier |
| <a id="key_prefix"></a> `key_prefix` | `string` | Key prefix (first 16 chars, for identification) |
| <a id="key_type"></a> `key_type` | `"anon"` \| `"service"` \| `"publishable"` \| `"tenant_service"` \| `"global_service"` | Key type. - `anon` — anonymous access - `publishable` — publishable client key (browser-safe) - `service` — elevated privileges, bypasses RLS (legacy admin-API type) - `tenant_service` — elevated, tenant-scoped (RLS-enforced) - `global_service` — elevated, cross-tenant, bypasses RLS |
| <a id="last_used_at"></a> `last_used_at?` | `string` | Last usage timestamp |
| <a id="name"></a> `name` | `string` | Display name |
| <a id="rate_limit_per_hour"></a> `rate_limit_per_hour?` | `number` | Rate limit per hour |
| <a id="rate_limit_per_minute"></a> `rate_limit_per_minute?` | `number` | Rate limit per minute |
| <a id="replaced_by"></a> `replaced_by?` | `string` | ID of replacement key (after rotation) |
| <a id="revoked_at"></a> `revoked_at?` | `string` | Revocation timestamp |
| <a id="scopes"></a> `scopes` | `string`[] | Permission scopes |
