//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.jobs](../index.md)/[FluxbaseJobs](index.md)

# FluxbaseJobs

[jvm]\
class [FluxbaseJobs](index.md)(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md))

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
| [FluxbaseJobs](-fluxbase-jobs.md) | [jvm]<br>constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md)) |

## Functions

| Name | Summary |
|---|---|
| [cancel](cancel.md) | [jvm]<br>suspend fun [cancel](cancel.md)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Job](../-job/index.md)&gt;<br>Cancel a running job. POSTs `/api/v1/jobs/{id}/cancel`. |
| [get](get.md) | [jvm]<br>suspend fun [get](get.md)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Job](../-job/index.md)&gt;<br>Get a job by ID. GETs `/api/v1/jobs/{id}`. |
| [getLogs](get-logs.md) | [jvm]<br>suspend fun [getLogs](get-logs.md)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), afterLine: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[ExecutionLog](../-execution-log/index.md)&gt;&gt;<br>Get job execution logs. GETs `/api/v1/jobs/{id}/logs`. |
| [list](list.md) | [jvm]<br>suspend fun [list](list.md)(status: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, namespace: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, limit: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, offset: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[Job](../-job/index.md)&gt;&gt;<br>List jobs. GETs `/api/v1/jobs` with optional filters. |
| [retry](retry.md) | [jvm]<br>suspend fun [retry](retry.md)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Job](../-job/index.md)&gt;<br>Retry a failed job. POSTs `/api/v1/jobs/{id}/retry`. |
| [submit](submit.md) | [jvm]<br>suspend fun [submit](submit.md)(jobName: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), payload: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, options: [SubmitJobOptions](../-submit-job-options/index.md) = SubmitJobOptions()): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Job](../-job/index.md)&gt;<br>Submit a new job for execution. POSTs to `/api/v1/jobs/submit`. Port of `submit()` in `jobs.ts:130`. |
