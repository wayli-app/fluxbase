//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[HttpMethod](index.md)

# HttpMethod

[jvm]\
enum [HttpMethod](index.md) : [Enum](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-enum/index.html)&lt;[HttpMethod](index.md)&gt; 

HTTP method enum used by the SDK's client layer. (Distinct from Ktor's `HttpMethod` to avoid naming clashes in the transport.)

## Entries

| | |
|---|---|
| [GET](-g-e-t/index.md) | [jvm]<br>[GET](-g-e-t/index.md) |
| [POST](-p-o-s-t/index.md) | [jvm]<br>[POST](-p-o-s-t/index.md) |
| [PUT](-p-u-t/index.md) | [jvm]<br>[PUT](-p-u-t/index.md) |
| [PATCH](-p-a-t-c-h/index.md) | [jvm]<br>[PATCH](-p-a-t-c-h/index.md) |
| [DELETE](-d-e-l-e-t-e/index.md) | [jvm]<br>[DELETE](-d-e-l-e-t-e/index.md) |
| [HEAD](-h-e-a-d/index.md) | [jvm]<br>[HEAD](-h-e-a-d/index.md) |

## Properties

| Name | Summary |
|---|---|
| [entries](entries.md) | [jvm]<br>val [entries](entries.md): [EnumEntries](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.enums/-enum-entries/index.html)&lt;[HttpMethod](index.md)&gt;<br>Returns a representation of an immutable list of all enum entries, in the order they're declared. |
| [name](-h-e-a-d/index.md#-372974862%2FProperties%2F-1216412040) | [jvm]<br>val [name](-h-e-a-d/index.md#-372974862%2FProperties%2F-1216412040): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [ordinal](-h-e-a-d/index.md#-739389684%2FProperties%2F-1216412040) | [jvm]<br>val [ordinal](-h-e-a-d/index.md#-739389684%2FProperties%2F-1216412040): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) |

## Functions

| Name | Summary |
|---|---|
| [valueOf](value-of.md) | [jvm]<br>fun [valueOf](value-of.md)(value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [HttpMethod](index.md)<br>Returns the enum constant of this type with the specified name. The string must match exactly an identifier used to declare an enum constant in this type. (Extraneous whitespace characters are not permitted.) |
| [values](values.md) | [jvm]<br>fun [values](values.md)(): [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[HttpMethod](index.md)&gt;<br>Returns an array containing the constants of this enum type, in the order they're declared. |
