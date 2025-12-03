---
editUrl: false
next: false
prev: false
title: "ResumableDownloadData"
---

Response type for resumable downloads - stream abstracts chunking

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `size` | `null` \| `number` | File size in bytes from HEAD request, or null if unknown |
| `stream` | `ReadableStream`\<`Uint8Array`\> | The readable stream for the file content (abstracts chunking internally) |
