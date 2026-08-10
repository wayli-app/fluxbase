//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[AuthChangeEvent](index.md)

# AuthChangeEvent

[jvm]\
enum [AuthChangeEvent](index.md) : [Enum](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-enum/index.html)&lt;[AuthChangeEvent](index.md)&gt; 

Auth state change events. Port of `AuthChangeEvent` from `sdk/src/types.ts:2374`.

Emitted via `client.auth.authStateChanges` (a Kotlin `Flow`) whenever the session changes. The TS SDK uses `onAuthStateChange(callback)`; in Kotlin we expose the same events as a `Flow` for coroutine-native consumption.

## Entries

| | |
|---|---|
| [SIGNED_IN](-s-i-g-n-e-d_-i-n/index.md) | [jvm]<br>[SIGNED_IN](-s-i-g-n-e-d_-i-n/index.md) |
| [SIGNED_OUT](-s-i-g-n-e-d_-o-u-t/index.md) | [jvm]<br>[SIGNED_OUT](-s-i-g-n-e-d_-o-u-t/index.md) |
| [TOKEN_REFRESHED](-t-o-k-e-n_-r-e-f-r-e-s-h-e-d/index.md) | [jvm]<br>[TOKEN_REFRESHED](-t-o-k-e-n_-r-e-f-r-e-s-h-e-d/index.md) |
| [USER_UPDATED](-u-s-e-r_-u-p-d-a-t-e-d/index.md) | [jvm]<br>[USER_UPDATED](-u-s-e-r_-u-p-d-a-t-e-d/index.md) |
| [PASSWORD_RECOVERY](-p-a-s-s-w-o-r-d_-r-e-c-o-v-e-r-y/index.md) | [jvm]<br>[PASSWORD_RECOVERY](-p-a-s-s-w-o-r-d_-r-e-c-o-v-e-r-y/index.md) |
| [MFA_CHALLENGE_VERIFIED](-m-f-a_-c-h-a-l-l-e-n-g-e_-v-e-r-i-f-i-e-d/index.md) | [jvm]<br>[MFA_CHALLENGE_VERIFIED](-m-f-a_-c-h-a-l-l-e-n-g-e_-v-e-r-i-f-i-e-d/index.md) |

## Properties

| Name | Summary |
|---|---|
| [entries](entries.md) | [jvm]<br>val [entries](entries.md): [EnumEntries](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.enums/-enum-entries/index.html)&lt;[AuthChangeEvent](index.md)&gt;<br>Returns a representation of an immutable list of all enum entries, in the order they're declared. |
| [name](../../io.github.nimbleflux.fluxbase.postgrest/-count-type/-e-s-t-i-m-a-t-e-d/index.md#-372974862%2FProperties%2F-1216412040) | [jvm]<br>val [name](../../io.github.nimbleflux.fluxbase.postgrest/-count-type/-e-s-t-i-m-a-t-e-d/index.md#-372974862%2FProperties%2F-1216412040): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [ordinal](../../io.github.nimbleflux.fluxbase.postgrest/-count-type/-e-s-t-i-m-a-t-e-d/index.md#-739389684%2FProperties%2F-1216412040) | [jvm]<br>val [ordinal](../../io.github.nimbleflux.fluxbase.postgrest/-count-type/-e-s-t-i-m-a-t-e-d/index.md#-739389684%2FProperties%2F-1216412040): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) |

## Functions

| Name | Summary |
|---|---|
| [valueOf](value-of.md) | [jvm]<br>fun [valueOf](value-of.md)(value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [AuthChangeEvent](index.md)<br>Returns the enum constant of this type with the specified name. The string must match exactly an identifier used to declare an enum constant in this type. (Extraneous whitespace characters are not permitted.) |
| [values](values.md) | [jvm]<br>fun [values](values.md)(): [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[AuthChangeEvent](index.md)&gt;<br>Returns an array containing the constants of this enum type, in the order they're declared. |
