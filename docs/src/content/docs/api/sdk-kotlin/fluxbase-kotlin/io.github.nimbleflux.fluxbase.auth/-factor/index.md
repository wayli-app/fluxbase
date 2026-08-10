//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[Factor](index.md)

# Factor

[jvm]\
@Serializable

data class [Factor](index.md)(val id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val type: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;totp&quot;, val status: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;verified&quot;, val createdAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;, val updatedAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;, val friendlyName: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)

## Constructors

| | |
|---|---|
| [Factor](-factor.md) | [jvm]<br>constructor(id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), type: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;totp&quot;, status: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;verified&quot;, createdAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;, updatedAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;, friendlyName: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [createdAt](created-at.md) | [jvm]<br>@SerialName(value = &quot;created_at&quot;)<br>val [createdAt](created-at.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [friendlyName](friendly-name.md) | [jvm]<br>@SerialName(value = &quot;friendly_name&quot;)<br>val [friendlyName](friendly-name.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [id](id.md) | [jvm]<br>val [id](id.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [status](status.md) | [jvm]<br>val [status](status.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [type](type.md) | [jvm]<br>val [type](type.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [updatedAt](updated-at.md) | [jvm]<br>@SerialName(value = &quot;updated_at&quot;)<br>val [updatedAt](updated-at.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
