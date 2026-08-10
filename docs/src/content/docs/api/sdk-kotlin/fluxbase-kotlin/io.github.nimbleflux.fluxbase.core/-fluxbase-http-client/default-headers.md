//[fluxbase-kotlin](../../../index.md)/[io.github.nimbleflux.fluxbase.core](../index.md)/[FluxbaseHttpClient](index.md)/[defaultHeaders](default-headers.md)

# defaultHeaders

[jvm]\
val [defaultHeaders](default-headers.md): [MutableMap](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin.collections/-mutable-map/index.html)&lt;[String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html), [String](https://kotlinlang.org/api/latest/jvm/stdlib/kotlin-stdlib/kotlin/-string/index.html)&gt;

Default headers applied to every request. `Content-Type: application/json` is always present; `Authorization` is managed via [setAuthToken](set-auth-token.md).
