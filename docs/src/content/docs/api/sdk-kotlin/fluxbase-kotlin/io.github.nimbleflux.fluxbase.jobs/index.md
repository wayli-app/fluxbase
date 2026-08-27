---
title: "Package-level declarations"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../)/[io.github.nimbleflux.fluxbase.jobs](./)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [ExecutionLog](-execution-log/) | [jvm]<br>@Serializable<br>data class [ExecutionLog](-execution-log/)(val line: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val timestamp: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val level: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)<br>Log entry from a job execution. |
| [FluxbaseJobs](-fluxbase-jobs/) | [jvm]<br>class [FluxbaseJobs](-fluxbase-jobs/)(http: [FluxbaseHttpClient](../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/))<br>Background Jobs module — port of `FluxbaseJobs` from `sdk/src/jobs.ts`. |
| [Job](-job/) | [jvm]<br>@Serializable<br>data class [Job](-job/)(val id: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val jobName: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val status: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;pending&quot;, val payload: JsonElement? = null, val result: JsonElement? = null, val error: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val priority: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 5, val progressPercent: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, val progressMessage: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val retryCount: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html) = 0, val createdBy: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val createdAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;, val updatedAt: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html) = &quot;&quot;)<br>A background job. Port of `Job` from `sdk/src/types.ts:2675`. |
| [OnBehalfOf](-on-behalf-of/) | [jvm]<br>@Serializable<br>data class [OnBehalfOf](-on-behalf-of/)(val userId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), val userEmail: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val userRole: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null)<br>Submit a job on behalf of a user (service_role only). |
| [SubmitJobOptions](-submit-job-options/) | [jvm]<br>data class [SubmitJobOptions](-submit-job-options/)(val priority: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, val namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val scheduled: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val onBehalfOf: [OnBehalfOf](-on-behalf-of/)? = null)<br>Options for [FluxbaseJobs.submit](-fluxbase-jobs/submit/). |
