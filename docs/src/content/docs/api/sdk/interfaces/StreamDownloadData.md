---
editUrl: false
next: false
prev: false
title: "StreamDownloadData"
---

Response type for stream downloads, includes file size from Content-Length header

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `size` | `null` \| `number` | File size in bytes from Content-Length header, or null if unknown |
| `stream` | `ReadableStream`\<`Uint8Array`\> | The readable stream for the file content |
