---
editUrl: false
next: false
prev: false
title: "useStorageTransformUrl"
---

> **useStorageTransformUrl**(`bucket`, `path`, `transform`): `string` \| `null`

Hook to get a public URL for an image with transformations applied

Only works for image files (JPEG, PNG, WebP, GIF, AVIF, etc.)

## Parameters

| Parameter | Type | Description |
| ------ | ------ | ------ |
| `bucket` | `string` | The storage bucket name |
| `path` | `null` \| `string` | The file path (or null to disable) |
| `transform` | [`TransformOptions`](/api/sdk-react/interfaces/transformoptions/) | Transformation options (width, height, format, quality, fit) |

## Returns

`string` \| `null`

## Example

```tsx
function ImageThumbnail({ path }: { path: string }) {
  const url = useStorageTransformUrl('images', path, {
    width: 300,
    height: 200,
    format: 'webp',
    quality: 85,
    fit: 'cover'
  });

  return <img src={url || ''} alt="Thumbnail" />;
}
```
