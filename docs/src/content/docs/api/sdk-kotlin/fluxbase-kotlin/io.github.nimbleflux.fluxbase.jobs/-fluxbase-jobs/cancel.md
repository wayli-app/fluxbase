---
title: "cancel"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.jobs](../../)/[FluxbaseJobs](../)/[cancel](./)

# cancel

[jvm]\
suspend fun [cancel](./)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Job](../../-job/)&gt;

Cancel a running job. POSTs `/api/v1/jobs/{id}/cancel`.
