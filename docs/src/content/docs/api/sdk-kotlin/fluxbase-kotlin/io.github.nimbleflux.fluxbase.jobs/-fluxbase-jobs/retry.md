---
title: "retry"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.jobs](../../)/[FluxbaseJobs](../)/[retry](./)

# retry

[jvm]\
suspend fun [retry](./)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Job](../../-job/)&gt;

Retry a failed job. POSTs `/api/v1/jobs/{id}/retry`.
