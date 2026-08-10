---
title: "Package-level declarations"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../index.md)/[io.github.nimbleflux.fluxbase.jobs](index.md)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [ExecutionLog](-execution-log/index.md) | [jvm]<br>@Serializable<br>data class [ExecutionLog](-execution-log/index.md)(val line: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val timestamp: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val level: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)<br>Log entry from a job execution. |
| [FluxbaseJobs](-fluxbase-jobs/index.md) | [jvm]<br>class [FluxbaseJobs](-fluxbase-jobs/index.md)(http: [FluxbaseHttpClient](../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))<br>Background Jobs module — port of `FluxbaseJobs` from `sdk/src/jobs.ts`. |
| [Job](-job/index.md) | [jvm]<br>@Serializable<br>data class [Job](-job/index.md)(val id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val jobName: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val status: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;pending&quot;, val payload: JsonElement? = null, val result: JsonElement? = null, val error: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val priority: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 5, val progressPercent: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, val progressMessage: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val retryCount: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 0, val createdBy: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val createdAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;, val updatedAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;)<br>A background job. Port of `Job` from `sdk/src/types.ts:2675`. |
| [OnBehalfOf](-on-behalf-of/index.md) | [jvm]<br>@Serializable<br>data class [OnBehalfOf](-on-behalf-of/index.md)(val userId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val userEmail: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val userRole: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)<br>Submit a job on behalf of a user (service_role only). |
| [SubmitJobOptions](-submit-job-options/index.md) | [jvm]<br>data class [SubmitJobOptions](-submit-job-options/index.md)(val priority: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, val namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val scheduled: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val onBehalfOf: [OnBehalfOf](-on-behalf-of/index.md)? = null)<br>Options for [FluxbaseJobs.submit](-fluxbase-jobs/submit.md). |
