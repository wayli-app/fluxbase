---
title: "create"
editUrl: false
next: false
prev: false
---

//[fluxbase-kotlin](../../../../../)/[io.github.nimbleflux.fluxbase](../../../)/[FluxbaseClient](../../)/[Companion](../)/[create](./)

# create

[jvm]\
fun [create](./)(url: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, options: [FluxbaseClientOptions](../../../-fluxbase-client-options/) = FluxbaseClientOptions(), transport: [HttpTransport](../../../../iogithubnimblefluxfluxbasecore/-http-transport/)? = null): [FluxbaseClient](../../)

Create a [FluxbaseClient](../../). Port of `createClient()` in `client.ts:770-823`.

Resolves [url](./) and [key](./) from the arguments or environment variables (`FLUXBASE_URL` / `FLUXBASE_ANON_KEY` / `FLUXBASE_SERVICE_TOKEN`), then wires the HTTP client with `apikey` + `Authorization: Bearer` headers and constructs the auth module.

#### Parameters

jvm

| | |
|---|---|
| url | the Fluxbase server URL. Falls back to `FLUXBASE_URL` env var. |
| key | the anon or service-role key. Falls back to env vars. |
| options | client options (autoRefresh, storage, headers, etc.). |
| transport | the HTTP transport SPI (default: [KtorHttpTransport](../../../../iogithubnimblefluxfluxbasecore/-ktor-http-transport/); tests inject a recording fake). |
