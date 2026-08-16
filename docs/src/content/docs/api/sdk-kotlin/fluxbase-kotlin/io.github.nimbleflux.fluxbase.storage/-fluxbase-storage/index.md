---
title: "FluxbaseStorage"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.storage](../index.md)/[FluxbaseStorage](index.md)

# FluxbaseStorage

[jvm]\
class [FluxbaseStorage](index.md)(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))

Storage module — port of `FluxbaseStorage` from `sdk/src/storage.ts`.

File upload/download/list against Fluxbase's `/api/v1/storage/{bucket}/...`.

Usage:

```kotlin
client.storage.from("trip-images").upload("photo.jpg", bytes, contentType = "image/jpeg")
val (files, _) = client.storage.from("trip-images").list()
val bytes = client.storage.from("trip-images").download("photo.jpg")
```

NOTE: The TS SDK's chunked/resumable upload (custom init/upload/complete protocol) is not yet ported — simple upload covers the Wayli app's needs (trip media). Resumable upload will be added when the Wayli app's media pipeline needs it.

## Constructors

| | |
|---|---|
| [FluxbaseStorage](-fluxbase-storage.md) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)) |

## Functions

| Name | Summary |
|---|---|
| [from](from.md) | [jvm]<br>fun [from](from.md)(bucket: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [StorageBucket](../-storage-bucket/index.md)<br>Start operating on a bucket. Port of `from(bucket)` in `storage.ts`. |
