---
editUrl: false
next: false
prev: false
title: "ResumableUploadProgress"
---

Upload progress information for resumable uploads

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `bytesPerSecond` | `number` | Transfer rate in bytes per second |
| `currentChunk` | `number` | Current chunk being uploaded (1-indexed) |
| `loaded` | `number` | Number of bytes uploaded so far |
| `percentage` | `number` | Upload percentage (0-100) |
| `sessionId` | `string` | Upload session ID (for resume capability) |
| `total` | `number` | Total file size in bytes |
| `totalChunks` | `number` | Total number of chunks |
