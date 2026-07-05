---
editUrl: false
next: false
prev: false
title: "FluxbaseTenant"
---

FluxbaseTenant provides multi-tenant management functionality

## Example

```typescript
// List tenants I have access to
const { data } = await client.tenant.listMine()

// Get tenant details
const { data } = await client.tenant.get('tenant-id')

// Create a tenant (instance admin only)
const { data } = await client.tenant.create({
  slug: 'acme-corp',
  name: 'Acme Corporation'
})

// Assign admin to tenant (tenant admin only)
await client.tenant.assignAdmin('tenant-id', {
  user_id: 'user-id'
})
```

## Constructors

### Constructor

> **new FluxbaseTenant**(`fetch`): `FluxbaseTenant`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

`FluxbaseTenant`

## Methods

### assignAdmin()

> **assignAdmin**(`tenantId`, `options`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`TenantAdminAssignment`](/api/sdk/interfaces/tenantadminassignment/)\>\>

Assign an admin to a tenant (tenant admin only)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `tenantId` | `string` | Tenant ID |
| `options` | [`AssignAdminOptions`](/api/sdk/interfaces/assignadminoptions/) | Admin assignment options |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`TenantAdminAssignment`](/api/sdk/interfaces/tenantadminassignment/)\>\>

Promise with created assignment or error

#### Example

```typescript
const { data, error } = await client.tenant.assignAdmin('tenant-id', {
  user_id: 'user-id'
})
```

***

### create()

> **create**(`options`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`Tenant`](/api/sdk/interfaces/tenant/)\>\>

Create a new tenant (instance admin only)

This creates a new isolated database for the tenant.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `options` | [`CreateTenantOptions`](/api/sdk/interfaces/createtenantoptions/) | Tenant creation options |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`Tenant`](/api/sdk/interfaces/tenant/)\>\>

Promise with created tenant or error

#### Example

```typescript
const { data, error } = await client.tenant.create({
  slug: 'acme-corp',
  name: 'Acme Corporation',
  metadata: { plan: 'enterprise' }
})
```

***

### delete()

> **delete**(`id`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<`void`\>\>

Delete a tenant (instance admin only)

This permanently deletes the tenant's database and all its data.
Cannot delete the default tenant.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Tenant ID |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<`void`\>\>

Promise that resolves when deleted

#### Example

```typescript
const { error } = await client.tenant.delete('tenant-id')
```

***

### get()

> **get**(`id`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`Tenant`](/api/sdk/interfaces/tenant/)\>\>

Get a tenant by ID

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Tenant ID |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`Tenant`](/api/sdk/interfaces/tenant/)\>\>

Promise with tenant details or error

#### Example

```typescript
const { data, error } = await client.tenant.get('tenant-id')
```

***

### list()

> **list**(): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`Tenant`](/api/sdk/interfaces/tenant/)[]\>\>

List all tenants (instance admin only)

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`Tenant`](/api/sdk/interfaces/tenant/)[]\>\>

Promise with tenants list or error

#### Example

```typescript
const { data, error } = await client.tenant.list()
```

***

### listAdmins()

> **listAdmins**(`tenantId`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`TenantAdminAssignment`](/api/sdk/interfaces/tenantadminassignment/)[]\>\>

List admins of a tenant

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `tenantId` | `string` | Tenant ID |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`TenantAdminAssignment`](/api/sdk/interfaces/tenantadminassignment/)[]\>\>

Promise with admin list or error

#### Example

```typescript
const { data, error } = await client.tenant.listAdmins('tenant-id')
// data: [{ id: '...', tenant_id: '...', user_id: '...', email: 'admin@example.com' }]
```

***

### listMine()

> **listMine**(): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`TenantWithRole`](/api/sdk/interfaces/tenantwithrole/)[]\>\>

List tenants the current user has access to

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`TenantWithRole`](/api/sdk/interfaces/tenantwithrole/)[]\>\>

Promise with tenants and user's role in each

#### Example

```typescript
const { data, error } = await client.tenant.listMine()
// data: [{ id: '...', slug: 'acme', name: 'Acme', my_role: 'tenant_admin', status: 'active' }]
```

***

### migrate()

> **migrate**(`id`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<\{ `status`: `string`; \}\>\>

Migrate a tenant database to the latest schema (instance admin only)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Tenant ID |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<\{ `status`: `string`; \}\>\>

Promise with migration status or error

#### Example

```typescript
const { data, error } = await client.tenant.migrate('tenant-id')
// data: { status: 'migrated' }
```

***

### removeAdmin()

> **removeAdmin**(`tenantId`, `userId`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<`void`\>\>

Remove an admin from a tenant (tenant admin only)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `tenantId` | `string` | Tenant ID |
| `userId` | `string` | User ID |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<`void`\>\>

Promise that resolves when removed

#### Example

```typescript
const { error } = await client.tenant.removeAdmin('tenant-id', 'user-id')
```

***

### update()

> **update**(`id`, `options`): `Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`Tenant`](/api/sdk/interfaces/tenant/)\>\>

Update a tenant (tenant admin only)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `id` | `string` | Tenant ID |
| `options` | [`UpdateTenantOptions`](/api/sdk/interfaces/updatetenantoptions/) | Update options |

#### Returns

`Promise`\<[`FluxbaseResponse`](/api/sdk/type-aliases/fluxbaseresponse/)\<[`Tenant`](/api/sdk/interfaces/tenant/)\>\>

Promise with updated tenant or error

#### Example

```typescript
const { data, error } = await client.tenant.update('tenant-id', {
  name: 'New Name'
})
```
