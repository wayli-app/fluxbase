---
editUrl: false
next: false
prev: false
title: "StreamUploadOptions"
---

Options for streaming uploads (memory-efficient for large files)

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `cacheControl?` | `string` | Cache-Control header value |
| `contentType?` | `string` | MIME type of the file |
| `metadata?` | `Record`\<`string`, `string`\> | Custom metadata to attach to the file |
| `onUploadProgress?` | (`progress`: [`UploadProgress`](/api/sdk/interfaces/uploadprogress/)) => `void` | Optional callback to track upload progress |
| `signal?` | `AbortSignal` | AbortSignal to cancel the upload |
| `upsert?` | `boolean` | If true, overwrite existing file at this path |
