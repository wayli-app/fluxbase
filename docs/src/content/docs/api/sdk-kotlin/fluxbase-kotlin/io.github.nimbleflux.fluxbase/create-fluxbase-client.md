//[fluxbase-kotlin](../../index.md)/[io.github.nimbleflux.fluxbase](index.md)/[createFluxbaseClient](create-fluxbase-client.md)

# createFluxbaseClient

[jvm]\
fun [createFluxbaseClient](create-fluxbase-client.md)(url: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, key: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, options: [FluxbaseClientOptions](-fluxbase-client-options/index.md) = FluxbaseClientOptions(), transport: [HttpTransport](../io.github.nimbleflux.fluxbase.core/-http-transport/index.md)? = null): [FluxbaseClient](-fluxbase-client/index.md)

Top-level factory function — Kotlin-idiomatic equivalent of the TS `createClient(url, key, options)`. Delegates to [FluxbaseClient.create](-fluxbase-client/-companion/create.md).
