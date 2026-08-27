---
title: "ValidateInvitationResponse"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.management](../)/[ValidateInvitationResponse](./)

# ValidateInvitationResponse

[jvm]\
@Serializable

data class [ValidateInvitationResponse](./)(val valid: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html), val invitation: [Invitation](../-invitation/)? = null, val error: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)

## Constructors

| | |
|---|---|
| [ValidateInvitationResponse](-validate-invitation-response/) | [jvm]<br>constructor(valid: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html), invitation: [Invitation](../-invitation/)? = null, error: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [error](error/) | [jvm]<br>val [error](error/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [invitation](invitation/) | [jvm]<br>val [invitation](invitation/): [Invitation](../-invitation/)? = null |
| [valid](valid/) | [jvm]<br>val [valid](valid/): [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) |
