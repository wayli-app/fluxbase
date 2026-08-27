---
title: "OnBehalfOf"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.jobs](../)/[OnBehalfOf](./)

# OnBehalfOf

[jvm]\
@Serializable

data class [OnBehalfOf](./)(val userId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val userEmail: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val userRole: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)

Submit a job on behalf of a user (service_role only).

## Constructors

| | |
|---|---|
| [OnBehalfOf](-on-behalf-of/) | [jvm]<br>constructor(userId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), userEmail: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, userRole: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null) |

## Properties

| Name | Summary |
|---|---|
| [userEmail](user-email/) | [jvm]<br>@SerialName(value = &quot;user_email&quot;)<br>val [userEmail](user-email/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [userId](user-id/) | [jvm]<br>@SerialName(value = &quot;user_id&quot;)<br>val [userId](user-id/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) |
| [userRole](user-role/) | [jvm]<br>@SerialName(value = &quot;user_role&quot;)<br>val [userRole](user-role/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
