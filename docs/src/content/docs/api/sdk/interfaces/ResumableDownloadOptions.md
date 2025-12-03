---
editUrl: false
next: false
prev: false
title: "ResumableDownloadOptions"
---

Options for resumable chunked downloads

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `chunkSize?` | `number` | Chunk size in bytes for each download request. **Default** `5242880 (5MB)` |
| `chunkTimeout?` | `number` | Timeout in milliseconds per chunk request. **Default** `30000` |
| `maxRetries?` | `number` | Number of retry attempts per chunk on failure. **Default** `3` |
| `onProgress?` | (`progress`: [`DownloadProgress`](/api/sdk/interfaces/downloadprogress/)) => `void` | Callback for download progress |
| `retryDelayMs?` | `number` | Base delay in milliseconds for exponential backoff. **Default** `1000` |
| `signal?` | `AbortSignal` | AbortSignal to cancel the download |
