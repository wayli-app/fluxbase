---
editUrl: false
next: false
prev: false
title: "ServiceKeyWithKey"
---

Service key with the full key value (only returned on creation)

## Extends

- [`ServiceKey`](/api/sdk/interfaces/servicekey/)

## Properties

| Property | Type | Description | Inherited from |
| ------ | ------ | ------ | ------ |
| <a id="allowed_namespaces"></a> `allowed_namespaces?` | `string`[] | Allowed table namespaces | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`allowed_namespaces`](/api/sdk/interfaces/servicekey/#allowed_namespaces) |
| <a id="created_at"></a> `created_at` | `string` | Creation timestamp | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`created_at`](/api/sdk/interfaces/servicekey/#created_at) |
| <a id="created_by"></a> `created_by?` | `string` | User who created the key | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`created_by`](/api/sdk/interfaces/servicekey/#created_by) |
| <a id="deprecated_at"></a> `deprecated_at?` | `string` | Deprecation timestamp (for key rotation) | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`deprecated_at`](/api/sdk/interfaces/servicekey/#deprecated_at) |
| <a id="description"></a> `description?` | `string` | Description | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`description`](/api/sdk/interfaces/servicekey/#description) |
| <a id="enabled"></a> `enabled` | `boolean` | Whether the key is enabled | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`enabled`](/api/sdk/interfaces/servicekey/#enabled) |
| <a id="expires_at"></a> `expires_at?` | `string` | Expiration timestamp | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`expires_at`](/api/sdk/interfaces/servicekey/#expires_at) |
| <a id="grace_period_ends_at"></a> `grace_period_ends_at?` | `string` | Grace period end for deprecated keys | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`grace_period_ends_at`](/api/sdk/interfaces/servicekey/#grace_period_ends_at) |
| <a id="id"></a> `id` | `string` | Unique identifier | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`id`](/api/sdk/interfaces/servicekey/#id) |
| <a id="key"></a> `key` | `string` | The full key value - only shown once at creation | - |
| <a id="key_prefix"></a> `key_prefix` | `string` | Key prefix (first 16 chars, for identification) | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`key_prefix`](/api/sdk/interfaces/servicekey/#key_prefix) |
| <a id="key_type"></a> `key_type` | `"anon"` \| `"service"` \| `"publishable"` \| `"tenant_service"` \| `"global_service"` | Key type. - `anon` — anonymous access - `publishable` — publishable client key (browser-safe) - `service` — elevated privileges, bypasses RLS (legacy admin-API type) - `tenant_service` — elevated, tenant-scoped (RLS-enforced) - `global_service` — elevated, cross-tenant, bypasses RLS | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`key_type`](/api/sdk/interfaces/servicekey/#key_type) |
| <a id="last_used_at"></a> `last_used_at?` | `string` | Last usage timestamp | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`last_used_at`](/api/sdk/interfaces/servicekey/#last_used_at) |
| <a id="name"></a> `name` | `string` | Display name | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`name`](/api/sdk/interfaces/servicekey/#name) |
| <a id="rate_limit_per_hour"></a> `rate_limit_per_hour?` | `number` | Rate limit per hour | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`rate_limit_per_hour`](/api/sdk/interfaces/servicekey/#rate_limit_per_hour) |
| <a id="rate_limit_per_minute"></a> `rate_limit_per_minute?` | `number` | Rate limit per minute | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`rate_limit_per_minute`](/api/sdk/interfaces/servicekey/#rate_limit_per_minute) |
| <a id="replaced_by"></a> `replaced_by?` | `string` | ID of replacement key (after rotation) | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`replaced_by`](/api/sdk/interfaces/servicekey/#replaced_by) |
| <a id="revoked_at"></a> `revoked_at?` | `string` | Revocation timestamp | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`revoked_at`](/api/sdk/interfaces/servicekey/#revoked_at) |
| <a id="scopes"></a> `scopes` | `string`[] | Permission scopes | [`ServiceKey`](/api/sdk/interfaces/servicekey/).[`scopes`](/api/sdk/interfaces/servicekey/#scopes) |
