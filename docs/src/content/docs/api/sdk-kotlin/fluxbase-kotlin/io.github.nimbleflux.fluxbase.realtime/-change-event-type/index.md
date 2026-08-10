---
title: "ChangeEventType"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.realtime](../index.md)/[ChangeEventType](index.md)

# ChangeEventType

[jvm]\
enum [ChangeEventType](index.md) : [Enum](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-enum/index.html)&lt;[ChangeEventType](index.md)&gt; 

The type of a postgres_changes event. Port of the TS `eventType` values.

## Entries

| | |
|---|---|
| [INSERT](-i-n-s-e-r-t/index.md) | [jvm]<br>[INSERT](-i-n-s-e-r-t/index.md) |
| [UPDATE](-u-p-d-a-t-e/index.md) | [jvm]<br>[UPDATE](-u-p-d-a-t-e/index.md) |
| [DELETE](-d-e-l-e-t-e/index.md) | [jvm]<br>[DELETE](-d-e-l-e-t-e/index.md) |

## Properties

| Name | Summary |
|---|---|
| [entries](entries.md) | [jvm]<br>val [entries](entries.md): [EnumEntries](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.enums/-enum-entries/index.html)&lt;[ChangeEventType](index.md)&gt;<br>Returns a representation of an immutable list of all enum entries, in the order they're declared. |
| [name](../../io.github.nimbleflux.fluxbase.vector/-vector-search-metric/-i-n-n-e-r_-p-r-o-d-u-c-t/index.md#-372974862%2FProperties%2F-1216412040) | [jvm]<br>val [name](../../io.github.nimbleflux.fluxbase.vector/-vector-search-metric/-i-n-n-e-r_-p-r-o-d-u-c-t/index.md#-372974862%2FProperties%2F-1216412040): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [ordinal](../../io.github.nimbleflux.fluxbase.vector/-vector-search-metric/-i-n-n-e-r_-p-r-o-d-u-c-t/index.md#-739389684%2FProperties%2F-1216412040) | [jvm]<br>val [ordinal](../../io.github.nimbleflux.fluxbase.vector/-vector-search-metric/-i-n-n-e-r_-p-r-o-d-u-c-t/index.md#-739389684%2FProperties%2F-1216412040): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) |
| [wildcard](wildcard.md) | [jvm]<br>val [wildcard](wildcard.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |

## Functions

| Name | Summary |
|---|---|
| [valueOf](value-of.md) | [jvm]<br>fun [valueOf](value-of.md)(value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [ChangeEventType](index.md)<br>Returns the enum constant of this type with the specified name. The string must match exactly an identifier used to declare an enum constant in this type. (Extraneous whitespace characters are not permitted.) |
| [values](values.md) | [jvm]<br>fun [values](values.md)(): [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[ChangeEventType](index.md)&gt;<br>Returns an array containing the constants of this enum type, in the order they're declared. |
