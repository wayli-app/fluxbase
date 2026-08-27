---
title: "TwoFactorStatusResponse"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.auth](../)/[TwoFactorStatusResponse](./)

# TwoFactorStatusResponse

[jvm]\
@Serializable

data class [TwoFactorStatusResponse](./)(val all: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Factor](../-factor/)&gt; = emptyList(), val totp: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Factor](../-factor/)&gt; = emptyList())

## Constructors

| | |
|---|---|
| [TwoFactorStatusResponse](-two-factor-status-response/) | [jvm]<br>constructor(all: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Factor](../-factor/)&gt; = emptyList(), totp: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Factor](../-factor/)&gt; = emptyList()) |

## Properties

| Name | Summary |
|---|---|
| [all](all/) | [jvm]<br>val [all](all/): [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Factor](../-factor/)&gt; |
| [totp](totp/) | [jvm]<br>val [totp](totp/): [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Factor](../-factor/)&gt; |
