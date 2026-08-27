---
title: "SubmitJobOptions"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.jobs](../)/[SubmitJobOptions](./)

# SubmitJobOptions

[jvm]\
data class [SubmitJobOptions](./)(val priority: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, val namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val scheduled: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val onBehalfOf: [OnBehalfOf](../-on-behalf-of/)? = null)

Options for [FluxbaseJobs.submit](../-fluxbase-jobs/submit/).

## Constructors

| | |
|---|---|
| [SubmitJobOptions](-submit-job-options/) | [jvm]<br>constructor(priority: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, scheduled: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, onBehalfOf: [OnBehalfOf](../-on-behalf-of/)? = null) |

## Properties

| Name | Summary |
|---|---|
| [namespace](namespace/) | [jvm]<br>val [namespace](namespace/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [onBehalfOf](on-behalf-of/) | [jvm]<br>val [onBehalfOf](on-behalf-of/): [OnBehalfOf](../-on-behalf-of/)? = null |
| [priority](priority/) | [jvm]<br>val [priority](priority/): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null |
| [scheduled](scheduled/) | [jvm]<br>val [scheduled](scheduled/): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
