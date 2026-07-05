---
editUrl: false
next: false
prev: false
title: "FluxbaseFetch"
---

## Constructors

### Constructor

> **new FluxbaseFetch**(`baseUrl`, `options?`): `FluxbaseFetch`

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `baseUrl` | `string` |
| `options` | \{ `debug?`: `boolean`; `headers?`: `Record`\<`string`, `string`\>; `timeout?`: `number`; \} |
| `options.debug?` | `boolean` |
| `options.headers?` | `Record`\<`string`, `string`\> |
| `options.timeout?` | `number` |

#### Returns

`FluxbaseFetch`

## Methods

### delete()

> **delete**\<`T`\>(`path`, `options?`): `Promise`\<`T`\>

DELETE request

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `options` | `Omit`\<`FetchOptions`, `"method"`\> |

#### Returns

`Promise`\<`T`\>

***

### get()

> **get**\<`T`\>(`path`, `options?`): `Promise`\<`T`\>

GET request

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `options` | `Omit`\<`FetchOptions`, `"method"`\> |

#### Returns

`Promise`\<`T`\>

***

### getBaseUrl()

> **getBaseUrl**(): `string`

#### Returns

`string`

***

### getBlob()

> **getBlob**(`path`, `options?`): `Promise`\<`Blob`\>

GET request that returns response as Blob (for file downloads)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `options` | `Omit`\<`FetchOptions`, `"method"`\> |

#### Returns

`Promise`\<`Blob`\>

***

### getDefaultHeaders()

> **getDefaultHeaders**(): `Record`\<`string`, `string`\>

#### Returns

`Record`\<`string`, `string`\>

***

### getWithHeaders()

> **getWithHeaders**\<`T`\>(`path`, `options?`): `Promise`\<`FetchResponseWithHeaders`\<`T`\>\>

GET request that returns response with headers (for count queries)

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `options` | `Omit`\<`FetchOptions`, `"method"`\> |

#### Returns

`Promise`\<`FetchResponseWithHeaders`\<`T`\>\>

***

### head()

> **head**(`path`, `options?`): `Promise`\<`Headers`\>

HEAD request

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `options` | `Omit`\<`FetchOptions`, `"method"`\> |

#### Returns

`Promise`\<`Headers`\>

***

### patch()

> **patch**\<`T`\>(`path`, `body?`, `options?`): `Promise`\<`T`\>

PATCH request

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `body?` | `unknown` |
| `options?` | `Omit`\<`FetchOptions`, `"method"` \| `"body"`\> |

#### Returns

`Promise`\<`T`\>

***

### post()

> **post**\<`T`\>(`path`, `body?`, `options?`): `Promise`\<`T`\>

POST request

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `body?` | `unknown` |
| `options?` | `Omit`\<`FetchOptions`, `"method"` \| `"body"`\> |

#### Returns

`Promise`\<`T`\>

***

### postWithHeaders()

> **postWithHeaders**\<`T`\>(`path`, `body?`, `options?`): `Promise`\<`FetchResponseWithHeaders`\<`T`\>\>

POST request that returns response with headers (for POST-based queries with count)

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `body?` | `unknown` |
| `options?` | `Omit`\<`FetchOptions`, `"method"` \| `"body"`\> |

#### Returns

`Promise`\<`FetchResponseWithHeaders`\<`T`\>\>

***

### put()

> **put**\<`T`\>(`path`, `body?`, `options?`): `Promise`\<`T`\>

PUT request

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `body?` | `unknown` |
| `options?` | `Omit`\<`FetchOptions`, `"method"` \| `"body"`\> |

#### Returns

`Promise`\<`T`\>

***

### removeHeader()

> **removeHeader**(`name`): `void`

Remove a custom header

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `name` | `string` |

#### Returns

`void`

***

### request()

> **request**\<`T`\>(`path`, `options`): `Promise`\<`T`\>

Make an HTTP request

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `options` | `FetchOptions` |

#### Returns

`Promise`\<`T`\>

***

### requestWithHeaders()

> **requestWithHeaders**\<`T`\>(`path`, `options`): `Promise`\<`FetchResponseWithHeaders`\<`T`\>\>

Make an HTTP request and return response with headers

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `options` | `FetchOptions` |

#### Returns

`Promise`\<`FetchResponseWithHeaders`\<`T`\>\>

***

### setAnonKey()

> **setAnonKey**(`key`): `void`

Set the anon key for fallback authentication
When setAuthToken(null) is called, the Authorization header will be
restored to use this anon key instead of being deleted

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `key` | `string` |

#### Returns

`void`

***

### setAuthToken()

> **setAuthToken**(`token`): `void`

Update the authorization header
When token is null, restores to anon key if available

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `token` | `string` \| `null` |

#### Returns

`void`

***

### setBeforeRequestCallback()

> **setBeforeRequestCallback**(`callback`): `void`

Register a callback to be called before every request.
The callback receives the headers object and can modify it in place.
This is useful for dynamically injecting headers at request time.

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `callback` | `BeforeRequestCallback` \| `null` |

#### Returns

`void`

***

### setHeader()

> **setHeader**(`name`, `value`): `void`

Set a custom header on all requests

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `name` | `string` |
| `value` | `string` |

#### Returns

`void`

***

### setRefreshTokenCallback()

> **setRefreshTokenCallback**(`callback`): `void`

Register a callback to refresh the token when a 401 error occurs
The callback should return true if refresh was successful, false otherwise

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `callback` | `RefreshTokenCallback` \| `null` |

#### Returns

`void`
