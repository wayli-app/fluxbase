//[fluxbase-kotlin](../../index.md)/[io.github.nimbleflux.fluxbase](index.md)/[fluxbaseResponse](fluxbase-response.md)

# fluxbaseResponse

[jvm]\
suspend fun &lt;[T](fluxbase-response.md)&gt; [fluxbaseResponse](fluxbase-response.md)(block: suspend () -&gt; [T](fluxbase-response.md)): [FluxbaseResponse](-fluxbase-response/index.md)&lt;[T](fluxbase-response.md)&gt;

Wraps a suspending block, catching exceptions and converting them to [FluxbaseResponse.Error](-fluxbase-response/-error/index.md). The TS SDK uses `wrapAsync` for the same purpose (`sdk/src/utils/error-handling.ts`).
