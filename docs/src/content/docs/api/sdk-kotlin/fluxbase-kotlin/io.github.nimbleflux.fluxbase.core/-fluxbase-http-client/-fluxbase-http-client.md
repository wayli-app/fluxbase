//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[FluxbaseHttpClient](index.md)/[FluxbaseHttpClient](-fluxbase-http-client.md)

# FluxbaseHttpClient

[jvm]\
constructor(baseUrl: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), transport: [HttpTransport](../-http-transport/index.md), json: Json = defaultJson)

#### Parameters

jvm

| | |
|---|---|
| baseUrl | the Fluxbase server URL (trailing slash stripped). |
| transport | the I/O SPI. If null, a Ktor-backed transport is used at runtime; tests inject an io.github.nimbleflux.fluxbase.core.test.RecordingHttp. |
| json | the JSON decoder used for typed [get](get.md)/[post](post.md) responses. |
