//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.jobs](../index.md)/[FluxbaseJobs](index.md)/[submit](submit.md)

# submit

[jvm]\
suspend fun [submit](submit.md)(jobName: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), payload: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, options: [SubmitJobOptions](../-submit-job-options/index.md) = SubmitJobOptions()): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Job](../-job/index.md)&gt;

Submit a new job for execution. POSTs to `/api/v1/jobs/submit`. Port of `submit()` in `jobs.ts:130`.
