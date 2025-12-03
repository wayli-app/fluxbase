---
editUrl: false
next: false
prev: false
title: "APIKeysManager"
---

API Keys management client

Provides methods for managing API keys for service-to-service authentication.
API keys allow external services to authenticate without user credentials.

## Example

```typescript
const client = createClient({ url: 'http://localhost:8080' })
await client.auth.login({ email: 'user@example.com', password: 'password' })

// Create an API key
const { api_key, key } = await client.management.apiKeys.create({
  name: 'Production Service',
  scopes: ['read:users', 'write:users'],
  rate_limit_per_minute: 100
})

// List API keys
const { api_keys } = await client.management.apiKeys.list()
```

## Constructors

### new APIKeysManager()

> **new APIKeysManager**(`fetch`): [`APIKeysManager`](/api/sdk/classes/apikeysmanager/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`APIKeysManager`](/api/sdk/classes/apikeysmanager/)

## Methods

### create()

> **create**(`request`): `Promise`\<[`CreateAPIKeyResponse`](/api/sdk/interfaces/createapikeyresponse/)\>

Create a new API key

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `request` | [`CreateAPIKeyRequest`](/api/sdk/interfaces/createapikeyrequest/) | API key configuration |

#### Returns

`Promise`\<[`CreateAPIKeyResponse`](/api/sdk/interfaces/createapikeyresponse/)\>

Created API key with the full key value (only shown once)

#### Example

```typescript
const { api_key, key } = await client.management.apiKeys.create({
  name: 'Production Service',
  description: 'API key for production service',
  scopes: ['read:users', 'write:users'],
  rate_limit_per_minute: 100,
  expires_at: '2025-12-31T23:59:59Z'
})

// Store the key securely - it won't be shown again
console.log('API Key:', key)
```

***

### delete()

> **delete**(`keyId`): `Promise`\<[`DeleteAPIKeyResponse`](/api/sdk/interfaces/deleteapikeyresponse/)\>

Delete an API key

Permanently removes the API key from the system.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `keyId` | `string` | API key ID |

#### Returns

`Promise`\<[`DeleteAPIKeyResponse`](/api/sdk/interfaces/deleteapikeyresponse/)\>

Deletion confirmation

#### Example

```typescript
await client.management.apiKeys.delete('key-uuid')
console.log('API key deleted')
```

***

### get()

> **get**(`keyId`): `Promise`\<[`APIKey`](/api/sdk/interfaces/apikey/)\>

Get a specific API key by ID

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `keyId` | `string` | API key ID |

#### Returns

`Promise`\<[`APIKey`](/api/sdk/interfaces/apikey/)\>

API key details

#### Example

```typescript
const apiKey = await client.management.apiKeys.get('key-uuid')
console.log('Last used:', apiKey.last_used_at)
```

***

### list()

> **list**(): `Promise`\<[`ListAPIKeysResponse`](/api/sdk/interfaces/listapikeysresponse/)\>

List all API keys for the authenticated user

#### Returns

`Promise`\<[`ListAPIKeysResponse`](/api/sdk/interfaces/listapikeysresponse/)\>

List of API keys (without full key values)

#### Example

```typescript
const { api_keys, total } = await client.management.apiKeys.list()

api_keys.forEach(key => {
  console.log(`${key.name}: ${key.key_prefix}... (expires: ${key.expires_at})`)
})
```

***

### revoke()

> **revoke**(`keyId`): `Promise`\<[`RevokeAPIKeyResponse`](/api/sdk/interfaces/revokeapikeyresponse/)\>

Revoke an API key

Revoked keys can no longer be used but remain in the system for audit purposes.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `keyId` | `string` | API key ID |

#### Returns

`Promise`\<[`RevokeAPIKeyResponse`](/api/sdk/interfaces/revokeapikeyresponse/)\>

Revocation confirmation

#### Example

```typescript
await client.management.apiKeys.revoke('key-uuid')
console.log('API key revoked')
```

***

### update()

> **update**(`keyId`, `updates`): `Promise`\<[`APIKey`](/api/sdk/interfaces/apikey/)\>

Update an API key

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `keyId` | `string` | API key ID |
| `updates` | [`UpdateAPIKeyRequest`](/api/sdk/interfaces/updateapikeyrequest/) | Fields to update |

#### Returns

`Promise`\<[`APIKey`](/api/sdk/interfaces/apikey/)\>

Updated API key

#### Example

```typescript
const updated = await client.management.apiKeys.update('key-uuid', {
  name: 'Updated Name',
  rate_limit_per_minute: 200
})
```
