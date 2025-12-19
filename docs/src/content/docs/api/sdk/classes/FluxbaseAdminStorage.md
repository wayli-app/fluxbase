---
editUrl: false
next: false
prev: false
title: "FluxbaseAdminStorage"
---

Admin storage manager for bucket and object management

## Constructors

### new FluxbaseAdminStorage()

> **new FluxbaseAdminStorage**(`fetch`): [`FluxbaseAdminStorage`](/api/sdk/classes/fluxbaseadminstorage/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |

#### Returns

[`FluxbaseAdminStorage`](/api/sdk/classes/fluxbaseadminstorage/)

## Methods

### createBucket()

> **createBucket**(`name`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`object`\>\>

Create a new storage bucket

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `name` | `string` | Bucket name |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`object`\>\>

Success message

#### Example

```typescript
const { error } = await admin.storage.createBucket('my-bucket');
if (!error) {
  console.log('Bucket created');
}
```

***

### createFolder()

> **createFolder**(`bucket`, `folderPath`): `Promise`\<[`VoidResponse`](/api/sdk/type-aliases/voidresponse/)\>

Create a folder (empty object with directory content type)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucket` | `string` | Bucket name |
| `folderPath` | `string` | Folder path (should end with /) |

#### Returns

`Promise`\<[`VoidResponse`](/api/sdk/type-aliases/voidresponse/)\>

#### Example

```typescript
const { error } = await admin.storage.createFolder('my-bucket', 'new-folder/');
```

***

### deleteBucket()

> **deleteBucket**(`name`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`object`\>\>

Delete a storage bucket

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `name` | `string` | Bucket name |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`object`\>\>

Success message

#### Example

```typescript
const { error } = await admin.storage.deleteBucket('my-bucket');
if (!error) {
  console.log('Bucket deleted');
}
```

***

### deleteObject()

> **deleteObject**(`bucket`, `key`): `Promise`\<[`VoidResponse`](/api/sdk/type-aliases/voidresponse/)\>

Delete an object

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucket` | `string` | Bucket name |
| `key` | `string` | Object key (path) |

#### Returns

`Promise`\<[`VoidResponse`](/api/sdk/type-aliases/voidresponse/)\>

#### Example

```typescript
const { error } = await admin.storage.deleteObject('my-bucket', 'path/to/file.txt');
if (!error) {
  console.log('Object deleted');
}
```

***

### downloadObject()

> **downloadObject**(`bucket`, `key`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`Blob`\>\>

Download an object as a Blob

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucket` | `string` | Bucket name |
| `key` | `string` | Object key (path) |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<`Blob`\>\>

Object data as Blob

#### Example

```typescript
const { data: blob } = await admin.storage.downloadObject('my-bucket', 'file.pdf');
if (blob) {
  // Use the blob
  const url = URL.createObjectURL(blob);
}
```

***

### generateSignedUrl()

> **generateSignedUrl**(`bucket`, `key`, `expiresIn`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`SignedUrlResponse`](/api/sdk/interfaces/signedurlresponse/)\>\>

Generate a signed URL for temporary access

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucket` | `string` | Bucket name |
| `key` | `string` | Object key (path) |
| `expiresIn` | `number` | Expiration time in seconds |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`SignedUrlResponse`](/api/sdk/interfaces/signedurlresponse/)\>\>

Signed URL and expiration info

#### Example

```typescript
const { data } = await admin.storage.generateSignedUrl('my-bucket', 'file.pdf', 3600);
if (data) {
  console.log(`Download at: ${data.url}`);
  console.log(`Expires in: ${data.expires_in} seconds`);
}
```

***

### getObjectMetadata()

> **getObjectMetadata**(`bucket`, `key`): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminStorageObject`](/api/sdk/interfaces/adminstorageobject/)\>\>

Get object metadata

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucket` | `string` | Bucket name |
| `key` | `string` | Object key (path) |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminStorageObject`](/api/sdk/interfaces/adminstorageobject/)\>\>

Object metadata

#### Example

```typescript
const { data } = await admin.storage.getObjectMetadata('my-bucket', 'path/to/file.txt');
if (data) {
  console.log(`File size: ${data.size} bytes`);
}
```

***

### listBuckets()

> **listBuckets**(): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminListBucketsResponse`](/api/sdk/interfaces/adminlistbucketsresponse/)\>\>

List all storage buckets

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminListBucketsResponse`](/api/sdk/interfaces/adminlistbucketsresponse/)\>\>

List of buckets

#### Example

```typescript
const { data, error } = await admin.storage.listBuckets();
if (data) {
  console.log(`Found ${data.buckets.length} buckets`);
}
```

***

### listObjects()

> **listObjects**(`bucket`, `prefix`?, `delimiter`?): `Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminListObjectsResponse`](/api/sdk/interfaces/adminlistobjectsresponse/)\>\>

List objects in a bucket

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucket` | `string` | Bucket name |
| `prefix`? | `string` | Optional path prefix to filter results |
| `delimiter`? | `string` | Optional delimiter for hierarchical listing (usually '/') |

#### Returns

`Promise`\<[`DataResponse`](/api/sdk/type-aliases/dataresponse/)\<[`AdminListObjectsResponse`](/api/sdk/interfaces/adminlistobjectsresponse/)\>\>

List of objects and prefixes (folders)

#### Example

```typescript
// List all objects in bucket
const { data } = await admin.storage.listObjects('my-bucket');

// List objects in a folder
const { data } = await admin.storage.listObjects('my-bucket', 'folder/', '/');
```
