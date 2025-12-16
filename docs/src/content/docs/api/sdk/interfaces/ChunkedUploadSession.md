---
editUrl: false
next: false
prev: false
title: "ChunkedUploadSession"
---

Chunked upload session information

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `bucket` | `string` | Target bucket |
| `chunkSize` | `number` | Chunk size used |
| `completedChunks` | `number`[] | Array of completed chunk indices (0-indexed) |
| `createdAt` | `string` | Session creation time |
| `expiresAt` | `string` | Session expiration time |
| `path` | `string` | Target file path |
| `sessionId` | `string` | Unique session identifier for resume |
| `status` | `"active"` \| `"completing"` \| `"completed"` \| `"aborted"` \| `"expired"` | Session status |
| `totalChunks` | `number` | Total number of chunks |
| `totalSize` | `number` | Total file size |
