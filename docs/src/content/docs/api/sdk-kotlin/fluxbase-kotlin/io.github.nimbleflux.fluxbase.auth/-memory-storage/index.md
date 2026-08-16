---
title: "MemoryStorage"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[MemoryStorage](index.md)

# MemoryStorage

[jvm]\
class [MemoryStorage](index.md) : [StorageAdapter](../-storage-adapter/index.md)

In-memory [StorageAdapter](../-storage-adapter/index.md) — the default for JVM. Session is lost on restart. Port of `MemoryStorage` from `sdk/src/auth.ts:74`.

## Constructors

| | |
|---|---|
| [MemoryStorage](-memory-storage.md) | [jvm]<br>constructor() |

## Functions

| Name | Summary |
|---|---|
| [getItem](get-item.md) | [jvm]<br>open override fun [getItem](get-item.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? |
| [removeItem](remove-item.md) | [jvm]<br>open override fun [removeItem](remove-item.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |
| [setItem](set-item.md) | [jvm]<br>open override fun [setItem](set-item.md)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |
