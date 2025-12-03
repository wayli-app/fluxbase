---
editUrl: false
next: false
prev: false
title: "useStorageUpload"
---

> **useStorageUpload**(`bucket`): `UseMutationResult`\<`null` \| `object`, `Error`, `object`, `unknown`\>

Hook to upload a file to a bucket

Note: You can track upload progress by passing an `onUploadProgress` callback in the options:

## Parameters

| Parameter | Type |
| ------ | ------ |
| `bucket` | `string` |

## Returns

`UseMutationResult`\<`null` \| `object`, `Error`, `object`, `unknown`\>

## Example

```tsx
const upload = useStorageUpload('avatars')

upload.mutate({
  path: 'user.jpg',
  file: file,
  options: {
    onUploadProgress: (progress) => {
      console.log(`${progress.percentage}% uploaded`)
    }
  }
})
```

For automatic progress state management, use `useStorageUploadWithProgress` instead.
