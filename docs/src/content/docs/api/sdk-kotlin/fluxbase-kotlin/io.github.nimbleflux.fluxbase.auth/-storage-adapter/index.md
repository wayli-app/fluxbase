---
title: "StorageAdapter"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.auth](../)/[StorageAdapter](./)

# StorageAdapter

interface [StorageAdapter](./)

SPI for persisting the auth session. Port of `StorageAdapter` from `sdk/src/types.ts:25`.

The TS SDK uses `localStorage` in browsers and `MemoryStorage` in Node/SSR. In Kotlin:

- 
   The default [MemoryStorage](../-memory-storage/) keeps the session in memory (JVM/server use).
- 
   On Android, inject an `EncryptedSharedPreferences`-backed implementation.

The storage key is always `"fluxbase.auth.session"` (matching the TS constant `AUTH_STORAGE_KEY` in `auth.ts:61`).

#### Inheritors

| |
|---|
| [MemoryStorage](../-memory-storage/) |

## Functions

| Name | Summary |
|---|---|
| [getItem](get-item/) | [jvm]<br>abstract fun [getItem](get-item/)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? |
| [removeItem](remove-item/) | [jvm]<br>abstract fun [removeItem](remove-item/)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |
| [setItem](set-item/) | [jvm]<br>abstract fun [setItem](set-item/)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |
