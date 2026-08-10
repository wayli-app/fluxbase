//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.postgrest](../index.md)/[VectorMetric](index.md)

# VectorMetric

[jvm]\
enum [VectorMetric](index.md) : [Enum](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-enum/index.html)&lt;[VectorMetric](index.md)&gt; 

Vector similarity metric for pgvector queries. Port of `VectorMetric` from `sdk/src/query-builder.ts:507`.

## Entries

| | |
|---|---|
| [L2](-l2/index.md) | [jvm]<br>[L2](-l2/index.md) |
| [COSINE](-c-o-s-i-n-e/index.md) | [jvm]<br>[COSINE](-c-o-s-i-n-e/index.md) |
| [INNER_PRODUCT](-i-n-n-e-r_-p-r-o-d-u-c-t/index.md) | [jvm]<br>[INNER_PRODUCT](-i-n-n-e-r_-p-r-o-d-u-c-t/index.md) |

## Properties

| Name | Summary |
|---|---|
| [entries](entries.md) | [jvm]<br>val [entries](entries.md): [EnumEntries](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.enums/-enum-entries/index.html)&lt;[VectorMetric](index.md)&gt;<br>Returns a representation of an immutable list of all enum entries, in the order they're declared. |
| [name](../../io.github.nimbleflux.fluxbase.realtime/-change-event-type/-d-e-l-e-t-e/index.md#-372974862%2FProperties%2F-1216412040) | [jvm]<br>val [name](../../io.github.nimbleflux.fluxbase.realtime/-change-event-type/-d-e-l-e-t-e/index.md#-372974862%2FProperties%2F-1216412040): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [operator](operator.md) | [jvm]<br>val [operator](operator.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [ordinal](../../io.github.nimbleflux.fluxbase.realtime/-change-event-type/-d-e-l-e-t-e/index.md#-739389684%2FProperties%2F-1216412040) | [jvm]<br>val [ordinal](../../io.github.nimbleflux.fluxbase.realtime/-change-event-type/-d-e-l-e-t-e/index.md#-739389684%2FProperties%2F-1216412040): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) |

## Functions

| Name | Summary |
|---|---|
| [valueOf](value-of.md) | [jvm]<br>fun [valueOf](value-of.md)(value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [VectorMetric](index.md)<br>Returns the enum constant of this type with the specified name. The string must match exactly an identifier used to declare an enum constant in this type. (Extraneous whitespace characters are not permitted.) |
| [values](values.md) | [jvm]<br>fun [values](values.md)(): [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[VectorMetric](index.md)&gt;<br>Returns an array containing the constants of this enum type, in the order they're declared. |
