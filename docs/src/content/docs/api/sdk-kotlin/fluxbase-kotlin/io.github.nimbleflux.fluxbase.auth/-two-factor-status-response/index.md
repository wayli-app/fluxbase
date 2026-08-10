//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[TwoFactorStatusResponse](index.md)

# TwoFactorStatusResponse

[jvm]\
@Serializable

data class [TwoFactorStatusResponse](index.md)(val all: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Factor](../-factor/index.md)&gt; = emptyList(), val totp: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Factor](../-factor/index.md)&gt; = emptyList())

## Constructors

| | |
|---|---|
| [TwoFactorStatusResponse](-two-factor-status-response.md) | [jvm]<br>constructor(all: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Factor](../-factor/index.md)&gt; = emptyList(), totp: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Factor](../-factor/index.md)&gt; = emptyList()) |

## Properties

| Name | Summary |
|---|---|
| [all](all.md) | [jvm]<br>val [all](all.md): [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Factor](../-factor/index.md)&gt; |
| [totp](totp.md) | [jvm]<br>val [totp](totp.md): [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Factor](../-factor/index.md)&gt; |
