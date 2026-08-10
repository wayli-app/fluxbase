//[fluxbase-kotlin](../../index.md)/[io.github.nimbleflux.fluxbase](index.md)

# Package-level declarations

## Types

| Name | Summary |
|---|---|
| [FluxbaseError](-fluxbase-error/index.md) | [jvm]<br>data class [FluxbaseError](-fluxbase-error/index.md)(val status: [Int](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-int/index.html)? = null, val code: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)? = null, val details: [Any](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-any/index.html)? = null, val message: [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)) : [RuntimeException](https://docs.oracle.com/javase/8/docs/api/java/lang/RuntimeException.html)<br>An error from a Fluxbase API call. Port of `FluxbaseError` from `sdk/src/types.ts:235`. |
| [FluxbaseResponse](-fluxbase-response/index.md) | [jvm]<br>sealed interface [FluxbaseResponse](-fluxbase-response/index.md)&lt;out [T](-fluxbase-response/index.md)&gt;<br>The result type for all Fluxbase SDK operations — the Kotlin equivalent of the TS SDK's `{ data: T; error: null } | { data: null; error: Error }` union (`sdk/src/types.ts:3904`). |

## Functions

| Name | Summary |
|---|---|
| [fluxbaseResponse](fluxbase-response.md) | [jvm]<br>suspend fun &lt;[T](fluxbase-response.md)&gt; [fluxbaseResponse](fluxbase-response.md)(block: suspend () -&gt; [T](fluxbase-response.md)): [FluxbaseResponse](-fluxbase-response/index.md)&lt;[T](fluxbase-response.md)&gt;<br>Wraps a suspending block, catching exceptions and converting them to [FluxbaseResponse.Error](-fluxbase-response/-error/index.md). The TS SDK uses `wrapAsync` for the same purpose (`sdk/src/utils/error-handling.ts`). |
| [getOrNull](get-or-null.md) | [jvm]<br>fun &lt;[T](get-or-null.md)&gt; [FluxbaseResponse](-fluxbase-response/index.md)&lt;[T](get-or-null.md)&gt;.[getOrNull](get-or-null.md)(): [T](get-or-null.md)?<br>Returns the data on success, or null on error. Equivalent to `result.data` in the TS SDK. |
| [getOrThrow](get-or-throw.md) | [jvm]<br>fun &lt;[T](get-or-throw.md)&gt; [FluxbaseResponse](-fluxbase-response/index.md)&lt;[T](get-or-throw.md)&gt;.[getOrThrow](get-or-throw.md)(): [T](get-or-throw.md)<br>Returns the data on success, or throws the error on failure. |
