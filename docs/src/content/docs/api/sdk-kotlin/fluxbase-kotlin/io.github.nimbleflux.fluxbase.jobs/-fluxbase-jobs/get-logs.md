---
title: "getLogs"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.jobs](../../)/[FluxbaseJobs](../)/[getLogs](./)

# getLogs

[jvm]\
suspend fun [getLogs](./)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), afterLine: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[List](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-list/index.html)&lt;[ExecutionLog](../../-execution-log/)&gt;&gt;

Get job execution logs. GETs `/api/v1/jobs/{id}/logs`.
