//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[TwoFactorSetupResponse](index.md)

# TwoFactorSetupResponse

[jvm]\
@Serializable

data class [TwoFactorSetupResponse](index.md)(val id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val type: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;totp&quot;, val totp: [TotpSetup](../-totp-setup/index.md))

## Constructors

| | |
|---|---|
| [TwoFactorSetupResponse](-two-factor-setup-response.md) | [jvm]<br>constructor(id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), type: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;totp&quot;, totp: [TotpSetup](../-totp-setup/index.md)) |

## Properties

| Name | Summary |
|---|---|
| [id](id.md) | [jvm]<br>val [id](id.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [totp](totp.md) | [jvm]<br>val [totp](totp.md): [TotpSetup](../-totp-setup/index.md) |
| [type](type.md) | [jvm]<br>val [type](type.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
