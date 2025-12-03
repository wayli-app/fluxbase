---
editUrl: false
next: false
prev: false
title: "DownloadProgress"
---

Download progress information

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `bytesPerSecond` | `number` | Transfer rate in bytes per second |
| `currentChunk` | `number` | Current chunk being downloaded (1-indexed) |
| `loaded` | `number` | Number of bytes downloaded so far |
| `percentage` | `null` \| `number` | Download percentage (0-100), or null if total is unknown |
| `total` | `null` \| `number` | Total file size in bytes, or null if unknown |
| `totalChunks` | `null` \| `number` | Total number of chunks, or null if total size unknown |
