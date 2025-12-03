---
editUrl: false
next: false
prev: false
title: "FluxbaseStorage"
---

## Constructors

### new FluxbaseStorage()

> **new FluxbaseStorage**(`fetch`): [`FluxbaseStorage`](/api/sdk/classes/fluxbasestorage/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseStorage`](/api/sdk/classes/fluxbasestorage/)

## Methods

### createBucket()

> **createBucket**(`bucketName`): `Promise`\<`object`\>

Create a new bucket

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucketName` | `string` | The name of the bucket to create |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object` |
| `error` | `null` \| `Error` |

***

### deleteBucket()

> **deleteBucket**(`bucketName`): `Promise`\<`object`\>

Delete a bucket

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucketName` | `string` | The name of the bucket to delete |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object` |
| `error` | `null` \| `Error` |

***

### emptyBucket()

> **emptyBucket**(`bucketName`): `Promise`\<`object`\>

Empty a bucket (delete all files)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucketName` | `string` | The name of the bucket to empty |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object` |
| `error` | `null` \| `Error` |

***

### from()

> **from**(`bucketName`): [`StorageBucket`](/api/sdk/classes/storagebucket/)

Get a reference to a storage bucket

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucketName` | `string` | The name of the bucket |

#### Returns

[`StorageBucket`](/api/sdk/classes/storagebucket/)

***

### getBucket()

> **getBucket**(`bucketName`): `Promise`\<`object`\>

Get bucket details

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucketName` | `string` | The name of the bucket |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `Bucket` |
| `error` | `null` \| `Error` |

***

### listBuckets()

> **listBuckets**(): `Promise`\<`object`\>

List all buckets

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object`[] |
| `error` | `null` \| `Error` |

***

### updateBucketSettings()

> **updateBucketSettings**(`bucketName`, `settings`): `Promise`\<`object`\>

Update bucket settings (RLS - requires admin or service key)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucketName` | `string` | The name of the bucket |
| `settings` | `BucketSettings` | Bucket settings to update |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |
