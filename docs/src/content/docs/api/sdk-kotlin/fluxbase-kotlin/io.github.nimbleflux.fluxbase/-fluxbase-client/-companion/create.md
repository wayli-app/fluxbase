//[fluxbase-kotlin](../../../../index.md)/[io.github.nimbleflux.fluxbase](../../index.md)/[FluxbaseClient](../index.md)/[Companion](index.md)/[create](create.md)

# create

[jvm]\
fun [create](create.md)(url: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, options: [FluxbaseClientOptions](../../-fluxbase-client-options/index.md) = FluxbaseClientOptions(), transport: [HttpTransport](../../../io.github.nimbleflux.fluxbase.core/-http-transport/index.md)? = null): [FluxbaseClient](../index.md)

Create a [FluxbaseClient](../index.md). Port of `createClient()` in `client.ts:770-823`.

Resolves [url](create.md) and [key](create.md) from the arguments or environment variables (`FLUXBASE_URL` / `FLUXBASE_ANON_KEY` / `FLUXBASE_SERVICE_TOKEN`), then wires the HTTP client with `apikey` + `Authorization: Bearer` headers and constructs the auth module.

#### Parameters

jvm

| | |
|---|---|
| url | the Fluxbase server URL. Falls back to `FLUXBASE_URL` env var. |
| key | the anon or service-role key. Falls back to env vars. |
| options | client options (autoRefresh, storage, headers, etc.). |
| transport | the HTTP transport SPI (default: [KtorHttpTransport](../../../io.github.nimbleflux.fluxbase.core/-ktor-http-transport/index.md); tests inject a recording fake). |
