---
title: "submit"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../)/[io.github.nimbleflux.fluxbase.jobs](../../)/[FluxbaseJobs](../)/[submit](./)

# submit

[jvm]\
suspend fun [submit](./)(jobName: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), payload: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, options: [SubmitJobOptions](../../-submit-job-options/) = SubmitJobOptions()): [FluxbaseResponse](../../../iogithubnimblefluxfluxbase/-fluxbase-response/)&lt;[Job](../../-job/)&gt;

Submit a new job for execution. POSTs to `/api/v1/jobs/submit`. Port of `submit()` in `jobs.ts:130`.
