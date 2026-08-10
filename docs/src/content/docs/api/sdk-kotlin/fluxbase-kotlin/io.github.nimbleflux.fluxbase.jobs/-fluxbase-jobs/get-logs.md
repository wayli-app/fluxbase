//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.jobs](../index.md)/[FluxbaseJobs](index.md)/[getLogs](get-logs.md)

# getLogs

[jvm]\
suspend fun [getLogs](get-logs.md)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), afterLine: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[ExecutionLog](../-execution-log/index.md)&gt;&gt;

Get job execution logs. GETs `/api/v1/jobs/{id}/logs`.
