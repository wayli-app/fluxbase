---
title: "MemoryStorage"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.auth](../)/[MemoryStorage](./)

# MemoryStorage

[jvm]\
class [MemoryStorage](./) : [StorageAdapter](../-storage-adapter/)

In-memory [StorageAdapter](../-storage-adapter/) — the default for JVM. Session is lost on restart. Port of `MemoryStorage` from `sdk/src/auth.ts:74`.

## Constructors

| | |
|---|---|
| [MemoryStorage](-memory-storage/) | [jvm]<br>constructor() |

## Functions

| Name | Summary |
|---|---|
| [getItem](get-item/) | [jvm]<br>open override fun [getItem](get-item/)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? |
| [removeItem](remove-item/) | [jvm]<br>open override fun [removeItem](remove-item/)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |
| [setItem](set-item/) | [jvm]<br>open override fun [setItem](set-item/)(key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) |
