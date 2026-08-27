---
title: "TwoFactorSetupResponse"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.auth](../)/[TwoFactorSetupResponse](./)

# TwoFactorSetupResponse

[jvm]\
@Serializable

data class [TwoFactorSetupResponse](./)(val id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val type: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;totp&quot;, val totp: [TotpSetup](../-totp-setup/))

## Constructors

| | |
|---|---|
| [TwoFactorSetupResponse](-two-factor-setup-response/) | [jvm]<br>constructor(id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), type: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;totp&quot;, totp: [TotpSetup](../-totp-setup/)) |

## Properties

| Name | Summary |
|---|---|
| [id](id/) | [jvm]<br>val [id](id/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [totp](totp/) | [jvm]<br>val [totp](totp/): [TotpSetup](../-totp-setup/) |
| [type](type/) | [jvm]<br>val [type](type/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
