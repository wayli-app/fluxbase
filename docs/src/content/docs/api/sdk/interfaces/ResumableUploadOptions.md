---
editUrl: false
next: false
prev: false
title: "ResumableUploadOptions"
---

Options for resumable chunked uploads

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `cacheControl?` | `string` | Cache-Control header value |
| `chunkSize?` | `number` | Chunk size in bytes for each upload request. **Default** `5242880 (5MB)` |
| `chunkTimeout?` | `number` | Timeout in milliseconds per chunk request. **Default** `60000 (1 minute)` |
| `contentType?` | `string` | MIME type of the file |
| `maxRetries?` | `number` | Number of retry attempts per chunk on failure. **Default** `3` |
| `metadata?` | `Record`\<`string`, `string`\> | Custom metadata to attach to the file |
| `onProgress?` | (`progress`: [`ResumableUploadProgress`](/api/sdk/interfaces/resumableuploadprogress/)) => `void` | Callback for upload progress |
| `resumeSessionId?` | `string` | Existing upload session ID to resume (optional) |
| `retryDelayMs?` | `number` | Base delay in milliseconds for exponential backoff. **Default** `1000` |
| `signal?` | `AbortSignal` | AbortSignal to cancel the upload |
