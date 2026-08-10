---
title: "retry"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.jobs](../index.md)/[FluxbaseJobs](index.md)/[retry](retry.md)

# retry

[jvm]\
suspend fun [retry](retry.md)(jobId: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)): [FluxbaseResponse](../../io.github.nimbleflux.fluxbase/-fluxbase-response/index.md)&lt;[Job](../-job/index.md)&gt;

Retry a failed job. POSTs `/api/v1/jobs/{id}/retry`.
