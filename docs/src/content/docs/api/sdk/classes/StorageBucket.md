---
editUrl: false
next: false
prev: false
title: "StorageBucket"
---

## Constructors

### new StorageBucket()

> **new StorageBucket**(`fetch`, `bucketName`): [`StorageBucket`](/api/sdk/classes/storagebucket/)

#### Parameters

| Parameter | Type |
| ------ | ------ |
| `fetch` | [`FluxbaseFetch`](/api/sdk/classes/fluxbasefetch/) |
| `bucketName` | `string` |

#### Returns

[`StorageBucket`](/api/sdk/classes/storagebucket/)

## Methods

### abortResumableUpload()

> **abortResumableUpload**(`sessionId`): `Promise`\<`object`\>

Abort an in-progress resumable upload

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `sessionId` | `string` | The upload session ID to abort |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `error` | `null` \| `Error` |

***

### copy()

> **copy**(`fromPath`, `toPath`): `Promise`\<`object`\>

Copy a file to a new location

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `fromPath` | `string` | Source file path |
| `toPath` | `string` | Destination file path |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object` |
| `error` | `null` \| `Error` |

***

### createSignedUrl()

> **createSignedUrl**(`path`, `options`?): `Promise`\<`object`\>

Create a signed URL for temporary access to a file

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `path` | `string` | The file path |
| `options`? | [`SignedUrlOptions`](/api/sdk/interfaces/signedurloptions/) | Signed URL options |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object` |
| `error` | `null` \| `Error` |

***

### download()

#### download(path)

> **download**(`path`): `Promise`\<`object`\>

Download a file from the bucket

##### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `path` | `string` | The path/key of the file |

##### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `Blob` |
| `error` | `null` \| `Error` |

##### Example

```typescript
// Default: returns Blob
const { data: blob } = await storage.from('bucket').download('file.pdf');

// Streaming: returns { stream, size } for progress tracking
const { data } = await storage.from('bucket').download('large.json', { stream: true });
console.log(`File size: ${data.size} bytes`);
// Process data.stream...
```

#### download(path, options)

> **download**(`path`, `options`): `Promise`\<`object`\>

##### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `options` | `object` |
| `options.signal`? | `AbortSignal` |
| `options.stream` | `true` |
| `options.timeout`? | `number` |

##### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`StreamDownloadData`](/api/sdk/interfaces/streamdownloaddata/) |
| `error` | `null` \| `Error` |

#### download(path, options)

> **download**(`path`, `options`): `Promise`\<`object`\>

##### Parameters

| Parameter | Type |
| ------ | ------ |
| `path` | `string` |
| `options` | `object` |
| `options.signal`? | `AbortSignal` |
| `options.stream`? | `false` |
| `options.timeout`? | `number` |

##### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `Blob` |
| `error` | `null` \| `Error` |

***

### downloadResumable()

> **downloadResumable**(`path`, `options`?): `Promise`\<`object`\>

Download a file with resumable chunked downloads for large files.
Returns a ReadableStream that abstracts the chunking internally.

Features:
- Downloads file in chunks using HTTP Range headers
- Automatically retries failed chunks with exponential backoff
- Reports progress via callback
- Falls back to regular streaming if Range not supported

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `path` | `string` | The file path within the bucket |
| `options`? | [`ResumableDownloadOptions`](/api/sdk/interfaces/resumabledownloadoptions/) | Download options including chunk size, retries, and progress callback |

#### Returns

`Promise`\<`object`\>

A ReadableStream and file size (consumer doesn't need to know about chunking)

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`ResumableDownloadData`](/api/sdk/interfaces/resumabledownloaddata/) |
| `error` | `null` \| `Error` |

#### Example

```typescript
const { data, error } = await storage.from('bucket').downloadResumable('large.json', {
  chunkSize: 5 * 1024 * 1024, // 5MB chunks
  maxRetries: 3,
  onProgress: (progress) => console.log(`${progress.percentage}% complete`)
});
if (data) {
  console.log(`File size: ${data.size} bytes`);
  // Process data.stream...
}
```

***

### getPublicUrl()

> **getPublicUrl**(`path`): `object`

Get a public URL for a file

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `path` | `string` | The file path |

#### Returns

`object`

| Name | Type |
| ------ | ------ |
| `data` | `object` |
| `data.publicUrl` | `string` |

***

### getResumableUploadStatus()

> **getResumableUploadStatus**(`sessionId`): `Promise`\<`object`\>

Get the status of a resumable upload session

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `sessionId` | `string` | The upload session ID to check |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`ChunkedUploadSession`](/api/sdk/interfaces/chunkeduploadsession/) |
| `error` | `null` \| `Error` |

***

### list()

> **list**(`pathOrOptions`?, `maybeOptions`?): `Promise`\<`object`\>

List files in the bucket
Supports both Supabase-style list(path, options) and Fluxbase-style list(options)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `pathOrOptions`? | `string` \| [`ListOptions`](/api/sdk/interfaces/listoptions/) | The folder path or list options |
| `maybeOptions`? | [`ListOptions`](/api/sdk/interfaces/listoptions/) | List options when first param is a path |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`FileObject`](/api/sdk/interfaces/fileobject/)[] |
| `error` | `null` \| `Error` |

***

### listShares()

> **listShares**(`path`): `Promise`\<`object`\>

List users a file is shared with (RLS)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `path` | `string` | The file path |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `FileShare`[] |
| `error` | `null` \| `Error` |

***

### move()

> **move**(`fromPath`, `toPath`): `Promise`\<`object`\>

Move a file to a new location

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `fromPath` | `string` | Current file path |
| `toPath` | `string` | New file path |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object` |
| `error` | `null` \| `Error` |

***

### remove()

> **remove**(`paths`): `Promise`\<`object`\>

Remove files from the bucket

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `paths` | `string`[] | Array of file paths to remove |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| [`FileObject`](/api/sdk/interfaces/fileobject/)[] |
| `error` | `null` \| `Error` |

***

### revokeShare()

> **revokeShare**(`path`, `userId`): `Promise`\<`object`\>

Revoke file access from a user (RLS)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `path` | `string` | The file path |
| `userId` | `string` | The user ID to revoke access from |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |

***

### share()

> **share**(`path`, `options`): `Promise`\<`object`\>

Share a file with another user (RLS)

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `path` | `string` | The file path |
| `options` | `ShareFileOptions` | Share options (userId and permission) |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` |
| `error` | `null` \| `Error` |

***

### upload()

> **upload**(`path`, `file`, `options`?): `Promise`\<`object`\>

Upload a file to the bucket

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `path` | `string` | The path/key for the file |
| `file` | `Blob` \| `ArrayBufferView` \| `ArrayBuffer` \| `File` | The file to upload (File, Blob, ArrayBuffer, or ArrayBufferView like Uint8Array) |
| `options`? | [`UploadOptions`](/api/sdk/interfaces/uploadoptions/) | Upload options |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object` |
| `error` | `null` \| `Error` |

***

### uploadLargeFile()

> **uploadLargeFile**(`path`, `file`, `options`?): `Promise`\<`object`\>

Upload a large file using streaming for reduced memory usage.
This is a convenience method that converts a File or Blob to a stream.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `path` | `string` | The path/key for the file |
| `file` | `Blob` \| `File` | The File or Blob to upload |
| `options`? | [`StreamUploadOptions`](/api/sdk/interfaces/streamuploadoptions/) | Upload options |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object` |
| `error` | `null` \| `Error` |

#### Example

```typescript
const file = new File([...], 'large-video.mp4');
const { data, error } = await storage
  .from('videos')
  .uploadLargeFile('video.mp4', file, {
    contentType: 'video/mp4',
    onUploadProgress: (p) => console.log(`${p.percentage}% complete`),
  });
```

***

### uploadResumable()

> **uploadResumable**(`path`, `file`, `options`?): `Promise`\<`object`\>

Upload a large file with resumable chunked uploads.

Features:
- Uploads file in chunks for reliability
- Automatically retries failed chunks with exponential backoff
- Reports progress via callback with chunk-level granularity
- Can resume interrupted uploads using session ID

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `path` | `string` | The file path within the bucket |
| `file` | `Blob` \| `File` | The File or Blob to upload |
| `options`? | [`ResumableUploadOptions`](/api/sdk/interfaces/resumableuploadoptions/) | Upload options including chunk size, retries, and progress callback |

#### Returns

`Promise`\<`object`\>

Upload result with file info

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object` |
| `error` | `null` \| `Error` |

#### Example

```ts
const { data, error } = await storage.from('uploads').uploadResumable('large.zip', file, {
  chunkSize: 5 * 1024 * 1024, // 5MB chunks
  maxRetries: 3,
  onProgress: (p) => {
    console.log(`${p.percentage}% (chunk ${p.currentChunk}/${p.totalChunks})`);
    console.log(`Speed: ${(p.bytesPerSecond / 1024 / 1024).toFixed(2)} MB/s`);
    console.log(`Session ID (for resume): ${p.sessionId}`);
  }
});

// To resume an interrupted upload:
const { data, error } = await storage.from('uploads').uploadResumable('large.zip', file, {
  resumeSessionId: 'previous-session-id',
});
```

***

### uploadStream()

> **uploadStream**(`path`, `stream`, `size`, `options`?): `Promise`\<`object`\>

Upload a file using streaming for reduced memory usage.
This method bypasses FormData buffering and streams data directly to the server.
Ideal for large files where memory efficiency is important.

#### Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `path` | `string` | The path/key for the file |
| `stream` | `ReadableStream`\<`Uint8Array`\> | ReadableStream of the file data |
| `size` | `number` | The size of the file in bytes (required for Content-Length header) |
| `options`? | [`StreamUploadOptions`](/api/sdk/interfaces/streamuploadoptions/) | Upload options |

#### Returns

`Promise`\<`object`\>

| Name | Type |
| ------ | ------ |
| `data` | `null` \| `object` |
| `error` | `null` \| `Error` |

#### Example

```typescript
// Upload from a File's stream
const file = new File([...], 'large-video.mp4');
const { data, error } = await storage
  .from('videos')
  .uploadStream('video.mp4', file.stream(), file.size, {
    contentType: 'video/mp4',
  });

// Upload from a fetch response stream
const response = await fetch('https://example.com/data.zip');
const size = parseInt(response.headers.get('content-length') || '0');
const { data, error } = await storage
  .from('files')
  .uploadStream('data.zip', response.body!, size, {
    contentType: 'application/zip',
  });
```
