//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.jobs](../index.md)/[FluxbaseJobs](index.md)/[cancel](cancel.md)

# cancel

[jvm]\
suspend fun [cancel](cancel.md)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Job](../-job/index.md)&gt;

Cancel a running job. POSTs `/api/v1/jobs/{id}/cancel`.
