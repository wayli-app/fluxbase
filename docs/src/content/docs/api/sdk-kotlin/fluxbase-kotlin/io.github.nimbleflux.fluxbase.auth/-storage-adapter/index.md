---
title: "StorageAdapter"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[StorageAdapter](index.md)

# StorageAdapter

interface [StorageAdapter](index.md)

SPI for persisting the auth session. Port of `StorageAdapter` from `sdk/src/types.ts:25`.

The TS SDK uses `localStorage` in browsers and `MemoryStorage` in Node/SSR. In Kotlin:

- 
   The default [MemoryStorage](../-memory-storage/index.md) keeps the session in memory (JVM/server use).
- 
   On Android, inject an `EncryptedSharedPreferences`-backed implementation.

The storage key is always `"fluxbase.auth.session"` (matching the TS constant `AUTH_STORAGE_KEY` in `auth.ts:61`).

#### Inheritors

| |
|---|
| [MemoryStorage](../-memory-storage/index.md) |

## Functions

| Name | Summary |
|---|---|
| [getItem](get-item.md) | [jvm]<br>abstract fun [getItem](get-item.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? |
| [removeItem](remove-item.md) | [jvm]<br>abstract fun [removeItem](remove-item.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |
| [setItem](set-item.md) | [jvm]<br>abstract fun [setItem](set-item.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |
