---
editUrl: false
next: false
prev: false
title: "FluxbaseFetch"
---

## Constructors

### new FluxbaseFetch()

> **new FluxbaseFetch**(`baseUrl`, `options`): [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `baseUrl` | `string` |
| `options` | `object` |
| `options.debug`? | `boolean` |
| `options.headers`? | `Record`\<`string`, `string`\> |
| `options.timeout`? | `number` |

#### Returns

[`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/)

## Methods

### delete()

> **delete**\<`T`\>(`path`, `options`): `Promise`\<`T`\>

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

> **get**\<`T`\>(`path`, `options`): `Promise`\<`T`\>

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

### getWithHeaders()

> **getWithHeaders**\<`T`\>(`path`, `options`): `Promise`\<`FetchResponseWithHeaders`\<`T`\>\>

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

> **head**(`path`, `options`): `Promise`\<`Headers`\>

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

> **patch**\<`T`\>(`path`, `body`?, `options`?): `Promise`\<`T`\>

PATCH request

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `body`? | `unknown` |
| `options`? | `Omit`\<`FetchOptions`, `"method"` \| `"body"`\> |

#### Returns

`Promise`\<`T`\>

***

### post()

> **post**\<`T`\>(`path`, `body`?, `options`?): `Promise`\<`T`\>

POST request

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `body`? | `unknown` |
| `options`? | `Omit`\<`FetchOptions`, `"method"` \| `"body"`\> |

#### Returns

`Promise`\<`T`\>

***

### put()

> **put**\<`T`\>(`path`, `body`?, `options`?): `Promise`\<`T`\>

PUT request

#### Type Parameters

| Type Parameter | Default type |
| ------ | ------ |
| `T` | `unknown` |

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `body`? | `unknown` |
| `options`? | `Omit`\<`FetchOptions`, `"method"` \| `"body"`\> |

#### Returns

`Promise`\<`T`\>

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
| `token` | `null` \| `string` |

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
| `callback` | `null` \| `RefreshTokenCallback` |

#### Returns

`void`
