---
editUrl: false
next: false
prev: false
title: "useStorageUploadWithProgress"
---

> **useStorageUploadWithProgress**(`bucket`): `object`

Hook to upload a file to a bucket with built-in progress tracking

## Parameters

| Parameter | Type |
| ------ | ------ |
| `bucket` | `string` |

## Returns

`object`

| Name | Type | Default value |
| ------ | ------ | ------ |
| `progress` | `null` \| `UploadProgress` | - |
| `reset` | () => `void` | - |
| `upload` | `UseMutationResult`\<`null` \| `object`, `Error`, `object`, `unknown`\> | mutation |

## Example

```tsx
const { upload, progress, reset } = useStorageUploadWithProgress('avatars')

// Upload with automatic progress tracking
upload.mutate({
  path: 'user.jpg',
  file: file
})

// Display progress
console.log(progress) // { loaded: 1024, total: 2048, percentage: 50 }
```
