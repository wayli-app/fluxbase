---
title: "FluxbaseJobs"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../)/[io.github.nimbleflux.fluxbase.jobs](../)/[FluxbaseJobs](./)

# FluxbaseJobs

[jvm]\
class [FluxbaseJobs](./)(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/))

Background Jobs module — port of `FluxbaseJobs` from `sdk/src/jobs.ts`.

Submit, track, and cancel background jobs. Fluxbase's jobs surface has no Supabase equivalent — this is a Fluxbase-only feature.

NOTE: The TS SDK has a bug on `jobs.ts:64` — it calls the unregistered path `/rest/v1/user_profiles` instead of `/api/v1/tables/user_profiles`. The `getCurrentUserRole()` auto-population of `onBehalfOf` for service-role clients is not ported here (it requires app-specific user_profiles knowledge); callers pass `onBehalfOf` explicitly if needed.

Usage:

```kotlin
val (job, error) = client.jobs.submit("trip-detection", mapOf("date" to "2024-01-01"), SubmitJobOptions(namespace = "wayli"))
val (status, _) = client.jobs.get(job!!.id)
```

## Constructors

| | |
|---|---|
| [FluxbaseJobs](-fluxbase-jobs/) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../iogithubnimblefluxfluxbasecore/-fluxbase-http-client/)) |

## Functions

| Name | Summary |
|---|---|
| [cancel](cancel/) | [jvm]<br>suspend fun [cancel](cancel/)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Job](../-job/)&gt;<br>Cancel a running job. POSTs `/api/v1/jobs/{id}/cancel`. |
| [get](get/) | [jvm]<br>suspend fun [get](get/)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Job](../-job/)&gt;<br>Get a job by ID. GETs `/api/v1/jobs/{id}`. |
| [getLogs](get-logs/) | [jvm]<br>suspend fun [getLogs](get-logs/)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), afterLine: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[ExecutionLog](../-execution-log/)&gt;&gt;<br>Get job execution logs. GETs `/api/v1/jobs/{id}/logs`. |
| [list](list/) | [jvm]<br>suspend fun [list](list/)(status: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, limit: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, offset: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Job](../-job/)&gt;&gt;<br>List jobs. GETs `/api/v1/jobs` with optional filters. |
| [retry](retry/) | [jvm]<br>suspend fun [retry](retry/)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Job](../-job/)&gt;<br>Retry a failed job. POSTs `/api/v1/jobs/{id}/retry`. |
| [submit](submit/) | [jvm]<br>suspend fun [submit](submit/)(jobName: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), payload: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, options: [SubmitJobOptions](../-submit-job-options/) = SubmitJobOptions()): [FluxbaseResponse](../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Job](../-job/)&gt;<br>Submit a new job for execution. POSTs to `/api/v1/jobs/submit`. Port of `submit()` in `jobs.ts:130`. |
