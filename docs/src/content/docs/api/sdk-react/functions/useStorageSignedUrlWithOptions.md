---
editUrl: false
next: false
prev: false
title: "useStorageSignedUrlWithOptions"
---

> **useStorageSignedUrlWithOptions**(`bucket`, `path`, `options`?): `UseQueryResult`\<`null` \| `string`, `Error`\>

Hook to create a signed URL with full options including image transformations

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucket` | `string` | The storage bucket name |
| `path` | `null` \| `string` | The file path (or null to disable) |
| `options`? | [`SignedUrlOptions`](/api/sdk-react/interfaces/signedurloptions/) | Signed URL options including expiration and transforms |

## Returns

`UseQueryResult`\<`null` \| `string`, `Error`\>

## Example

```tsx
function SecureThumbnail({ path }: { path: string }) {
  const { data: url } = useStorageSignedUrlWithOptions('images', path, {
    expiresIn: 3600,
    transform: {
      width: 400,
      height: 300,
      format: 'webp',
      quality: 85,
      fit: 'cover'
    }
  });

  return <img src={url || ''} alt="Secure Thumbnail" />;
}
```
