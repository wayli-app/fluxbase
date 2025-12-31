---
editUrl: false
next: false
prev: false
title: "ClientKeysManager"
---

Client Keys management client

Provides methods for managing client keys for service-to-service authentication.
Client keys allow external services to authenticate without user credentials.

## Example

```typescript
const client = createClient({ url: 'http://localhost:8080' })
await client.auth.login({ email: 'user@example.com', password: 'password' })

// Create a client key
const { client_key, key } = await client.management.clientKeys.create({
  name: 'Production Service',
  scopes: ['read:users', 'write:users'],
  rate_limit_per_minute: 100
})

// List client keys
const { client_keys } = await client.management.clientKeys.list()
```

## Constructors

### new ClientKeysManager()

> **new ClientKeysManager**(`fetch`): [`ClientKeysManager`](/api/sdk/classes/clientkeysmanager/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`ClientKeysManager`](/api/sdk/classes/clientkeysmanager/)

## Methods

### create()

> **create**(`request`): `Promise`\<[`CreateClientKeyResponse`](/api/sdk/interfaces/createclientkeyresponse/)\>

Create a new client key

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`CreateClientKeyRequest`](/api/sdk/interfaces/createclientkeyrequest/) | Client key configuration |

#### Returns

`Promise`\<[`CreateClientKeyResponse`](/api/sdk/interfaces/createclientkeyresponse/)\>

Created client key with the full key value (only shown once)

#### Example

```typescript
const { client_key, key } = await client.management.clientKeys.create({
  name: 'Production Service',
  description: 'Client key for production service',
  scopes: ['read:users', 'write:users'],
  rate_limit_per_minute: 100,
  expires_at: '2025-12-31T23:59:59Z'
})

// Store the key securely - it won't be shown again
console.log('Client Key:', key)
```

***

### delete()

> **delete**(`keyId`): `Promise`\<[`DeleteClientKeyResponse`](/api/sdk/interfaces/deleteclientkeyresponse/)\>

Delete a client key

Permanently removes the client key from the system.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `keyId` | `string` | Client key ID |

#### Returns

`Promise`\<[`DeleteClientKeyResponse`](/api/sdk/interfaces/deleteclientkeyresponse/)\>

Deletion confirmation

#### Example

```typescript
await client.management.clientKeys.delete('key-uuid')
console.log('Client key deleted')
```

***

### get()

> **get**(`keyId`): `Promise`\<[`ClientKey`](/api/sdk/interfaces/clientkey/)\>

Get a specific client key by ID

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `keyId` | `string` | Client key ID |

#### Returns

`Promise`\<[`ClientKey`](/api/sdk/interfaces/clientkey/)\>

Client key details

#### Example

```typescript
const clientKey = await client.management.clientKeys.get('key-uuid')
console.log('Last used:', clientKey.last_used_at)
```

***

### list()

> **list**(): `Promise`\<[`ListClientKeysResponse`](/api/sdk/interfaces/listclientkeysresponse/)\>

List all client keys for the authenticated user

#### Returns

`Promise`\<[`ListClientKeysResponse`](/api/sdk/interfaces/listclientkeysresponse/)\>

List of client keys (without full key values)

#### Example

```typescript
const { client_keys, total } = await client.management.clientKeys.list()

client_keys.forEach(key => {
  console.log(`${key.name}: ${key.key_prefix}... (expires: ${key.expires_at})`)
})
```

***

### revoke()

> **revoke**(`keyId`): `Promise`\<[`RevokeClientKeyResponse`](/api/sdk/interfaces/revokeclientkeyresponse/)\>

Revoke a client key

Revoked keys can no longer be used but remain in the system for audit purposes.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `keyId` | `string` | Client key ID |

#### Returns

`Promise`\<[`RevokeClientKeyResponse`](/api/sdk/interfaces/revokeclientkeyresponse/)\>

Revocation confirmation

#### Example

```typescript
await client.management.clientKeys.revoke('key-uuid')
console.log('Client key revoked')
```

***

### update()

> **update**(`keyId`, `updates`): `Promise`\<[`ClientKey`](/api/sdk/interfaces/clientkey/)\>

Update a client key

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `keyId` | `string` | Client key ID |
| `updates` | [`UpdateClientKeyRequest`](/api/sdk/interfaces/updateclientkeyrequest/) | Fields to update |

#### Returns

`Promise`\<[`ClientKey`](/api/sdk/interfaces/clientkey/)\>

Updated client key

#### Example

```typescript
const updated = await client.management.clientKeys.update('key-uuid', {
  name: 'Updated Name',
  rate_limit_per_minute: 200
})
```
