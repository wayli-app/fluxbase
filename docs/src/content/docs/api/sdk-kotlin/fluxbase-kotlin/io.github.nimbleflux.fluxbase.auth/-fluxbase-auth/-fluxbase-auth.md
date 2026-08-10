//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.auth](../index.md)/[FluxbaseAuth](index.md)/[FluxbaseAuth](-fluxbase-auth.md)

# FluxbaseAuth

[jvm]\
constructor(http: [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md), autoRefresh: [Boolean](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-boolean/index.html) = true, storage: [StorageAdapter](../-storage-adapter/index.md) = MemoryStorage())

#### Parameters

jvm

| | |
|---|---|
| http | the shared [FluxbaseHttpClient](../../io.github.nimbleflux.fluxbase.core/-fluxbase-http-client/index.md) for making API calls. |
| autoRefresh | whether to automatically refresh the token before expiry (default true; disabled in tests). TS default is true (`auth.ts:55`). |
| storage | the [StorageAdapter](../-storage-adapter/index.md) for session persistence. |
