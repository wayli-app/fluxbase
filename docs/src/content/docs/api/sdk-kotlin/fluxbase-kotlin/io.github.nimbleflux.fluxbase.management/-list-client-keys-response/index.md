---
title: "ListClientKeysResponse"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.management](../)/[ListClientKeysResponse](./)

# ListClientKeysResponse

[jvm]\
@Serializable

data class [ListClientKeysResponse](./)(val clientKeys: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[ClientKey](../-client-key/)&gt; = emptyList(), val total: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 0)

## Constructors

| | |
|---|---|
| [ListClientKeysResponse](-list-client-keys-response/) | [jvm]<br>constructor(clientKeys: [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[ClientKey](../-client-key/)&gt; = emptyList(), total: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 0) |

## Properties

| Name | Summary |
|---|---|
| [clientKeys](client-keys/) | [jvm]<br>@SerialName(value = &quot;client_keys&quot;)<br>val [clientKeys](client-keys/): [List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[ClientKey](../-client-key/)&gt; |
| [total](total/) | [jvm]<br>val [total](total/): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 0 |
