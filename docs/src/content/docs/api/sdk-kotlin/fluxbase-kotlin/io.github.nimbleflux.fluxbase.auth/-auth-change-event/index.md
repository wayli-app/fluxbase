---
title: "AuthChangeEvent"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.auth](../)/[AuthChangeEvent](./)

# AuthChangeEvent

[jvm]\
enum [AuthChangeEvent](./) : [Enum](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-enum/index.html)&lt;[AuthChangeEvent](./)&gt; 

Auth state change events. Port of `AuthChangeEvent` from `sdk/src/types.ts:2374`.

Emitted via `client.auth.authStateChanges` (a Kotlin `Flow`) whenever the session changes. The TS SDK uses `onAuthStateChange(callback)`; in Kotlin we expose the same events as a `Flow` for coroutine-native consumption.

## Entries

| | |
|---|---|
| [SIGNED_IN](-s-i-g-n-e-d_-i-n/) | [jvm]<br>[SIGNED_IN](-s-i-g-n-e-d_-i-n/) |
| [SIGNED_OUT](-s-i-g-n-e-d_-o-u-t/) | [jvm]<br>[SIGNED_OUT](-s-i-g-n-e-d_-o-u-t/) |
| [TOKEN_REFRESHED](-t-o-k-e-n_-r-e-f-r-e-s-h-e-d/) | [jvm]<br>[TOKEN_REFRESHED](-t-o-k-e-n_-r-e-f-r-e-s-h-e-d/) |
| [USER_UPDATED](-u-s-e-r_-u-p-d-a-t-e-d/) | [jvm]<br>[USER_UPDATED](-u-s-e-r_-u-p-d-a-t-e-d/) |
| [PASSWORD_RECOVERY](-p-a-s-s-w-o-r-d_-r-e-c-o-v-e-r-y/) | [jvm]<br>[PASSWORD_RECOVERY](-p-a-s-s-w-o-r-d_-r-e-c-o-v-e-r-y/) |
| [MFA_CHALLENGE_VERIFIED](-m-f-a_-c-h-a-l-l-e-n-g-e_-v-e-r-i-f-i-e-d/) | [jvm]<br>[MFA_CHALLENGE_VERIFIED](-m-f-a_-c-h-a-l-l-e-n-g-e_-v-e-r-i-f-i-e-d/) |

## Properties

| Name | Summary |
|---|---|
| [entries](entries/) | [jvm]<br>val [entries](entries/): [EnumEntries](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.enums/-enum-entries/index.html)&lt;[AuthChangeEvent](./)&gt;<br>Returns a representation of an immutable list of all enum entries, in the order they're declared. |
| [name](../../iogithubnimblefluxfluxbasevector/-vector-search-metric/-i-n-n-e-r_-p-r-o-d-u-c-t/#-372974862%2FProperties%2F-1216412040) | [jvm]<br>val [name](../../iogithubnimblefluxfluxbasevector/-vector-search-metric/-i-n-n-e-r_-p-r-o-d-u-c-t/#-372974862%2FProperties%2F-1216412040): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [ordinal](../../iogithubnimblefluxfluxbasevector/-vector-search-metric/-i-n-n-e-r_-p-r-o-d-u-c-t/#-739389684%2FProperties%2F-1216412040) | [jvm]<br>val [ordinal](../../iogithubnimblefluxfluxbasevector/-vector-search-metric/-i-n-n-e-r_-p-r-o-d-u-c-t/#-739389684%2FProperties%2F-1216412040): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) |

## Functions

| Name | Summary |
|---|---|
| [valueOf](value-of/) | [jvm]<br>fun [valueOf](value-of/)(value: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [AuthChangeEvent](./)<br>Returns the enum constant of this type with the specified name. The string must match exactly an identifier used to declare an enum constant in this type. (Extraneous whitespace characters are not permitted.) |
| [values](values/) | [jvm]<br>fun [values](values/)(): [Array](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-array/index.html)&lt;[AuthChangeEvent](./)&gt;<br>Returns an array containing the constants of this enum type, in the order they're declared. |
