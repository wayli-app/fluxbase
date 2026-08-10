---
title: "SubmitJobOptions"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.jobs](../index.md)/[SubmitJobOptions](index.md)

# SubmitJobOptions

[jvm]\
data class [SubmitJobOptions](index.md)(val priority: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, val namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val scheduled: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val onBehalfOf: [OnBehalfOf](../-on-behalf-of/index.md)? = null)

Options for [FluxbaseJobs.submit](../-fluxbase-jobs/submit.md).

## Constructors

| | |
|---|---|
| [SubmitJobOptions](-submit-job-options.md) | [jvm]<br>constructor(priority: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, scheduled: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, onBehalfOf: [OnBehalfOf](../-on-behalf-of/index.md)? = null) |

## Properties

| Name | Summary |
|---|---|
| [namespace](namespace.md) | [jvm]<br>val [namespace](namespace.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
| [onBehalfOf](on-behalf-of.md) | [jvm]<br>val [onBehalfOf](on-behalf-of.md): [OnBehalfOf](../-on-behalf-of/index.md)? = null |
| [priority](priority.md) | [jvm]<br>val [priority](priority.md): [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null |
| [scheduled](scheduled.md) | [jvm]<br>val [scheduled](scheduled.md): [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null |
