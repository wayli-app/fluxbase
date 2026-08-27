---
title: "ChangeEventType"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.realtime](../)/[ChangeEventType](./)

# ChangeEventType

[jvm]\
enum [ChangeEventType](./) : [Enum](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-enum/index.html)&lt;[ChangeEventType](./)&gt; 

The type of a postgres_changes event. Port of the TS `eventType` values.

## Entries

| | |
|---|---|
| [INSERT](-i-n-s-e-r-t/) | [jvm]<br>[INSERT](-i-n-s-e-r-t/) |
| [UPDATE](-u-p-d-a-t-e/) | [jvm]<br>[UPDATE](-u-p-d-a-t-e/) |
| [DELETE](-d-e-l-e-t-e/) | [jvm]<br>[DELETE](-d-e-l-e-t-e/) |

## Properties

| Name | Summary |
|---|---|
| [entries](entries/) | [jvm]<br>val [entries](entries/): [EnumEntries](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.enums/-enum-entries/index.html)&lt;[ChangeEventType](./)&gt;<br>Returns a representation of an immutable list of all enum entries, in the order they're declared. |
| [name](../../iogithubnimblefluxfluxbasevector/-vector-search-metric/-i-n-n-e-r_-p-r-o-d-u-c-t/#-372974862%2FProperties%2F-1216412040) | [jvm]<br>val [name](../../iogithubnimblefluxfluxbasevector/-vector-search-metric/-i-n-n-e-r_-p-r-o-d-u-c-t/#-372974862%2FProperties%2F-1216412040): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [ordinal](../../iogithubnimblefluxfluxbasevector/-vector-search-metric/-i-n-n-e-r_-p-r-o-d-u-c-t/#-739389684%2FProperties%2F-1216412040) | [jvm]<br>val [ordinal](../../iogithubnimblefluxfluxbasevector/-vector-search-metric/-i-n-n-e-r_-p-r-o-d-u-c-t/#-739389684%2FProperties%2F-1216412040): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) |
| [wildcard](wildcard/) | [jvm]<br>val [wildcard](wildcard/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |

## Functions

| Name | Summary |
|---|---|
| [valueOf](value-of/) | [jvm]<br>fun [valueOf](value-of/)(value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [ChangeEventType](./)<br>Returns the enum constant of this type with the specified name. The string must match exactly an identifier used to declare an enum constant in this type. (Extraneous whitespace characters are not permitted.) |
| [values](values/) | [jvm]<br>fun [values](values/)(): [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[ChangeEventType](./)&gt;<br>Returns an array containing the constants of this enum type, in the order they're declared. |
