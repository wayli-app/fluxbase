package io.github.nimbleflux.fluxbase

import io.github.nimbleflux.fluxbase.auth.FluxbaseAuth
import io.github.nimbleflux.fluxbase.auth.MemoryStorage
import io.github.nimbleflux.fluxbase.auth.StorageAdapter
import io.github.nimbleflux.fluxbase.core.FluxbaseHttpClient
import io.github.nimbleflux.fluxbase.core.HttpTransport
import io.github.nimbleflux.fluxbase.core.KtorHttpTransport

/**
 * Options for constructing a [FluxbaseClient]. Port of `FluxbaseClientOptions`
 * from `sdk/src/types.ts:35`.
 *
 * @param autoRefresh whether to automatically refresh the JWT before expiry (default true).
 * @param persist whether to persist the session to the [storage] adapter (default true).
 * @param storage custom session persistence (default: [MemoryStorage]).
 * @param headers additional default headers on every request.
 * @param timeout request timeout in milliseconds (default 30000).
 * @param debug enable verbose logging (default false).
 */
data class FluxbaseClientOptions(
    val autoRefresh: Boolean = true,
    val persist: Boolean = true,
    val storage: StorageAdapter = MemoryStorage(),
    val headers: Map<String, String> = emptyMap(),
    val timeout: Long = 30_000,
    val debug: Boolean = false,
)

/**
 * The top-level Fluxbase client — port of `FluxbaseClient` from `sdk/src/client.ts`.
 *
 * Wires together the HTTP layer ([http]) and all sub-modules ([auth], and later
 * postgrest, realtime, storage, functions, jobs, etc.). Construct via the
 * companion [create] factory (or the top-level [createFluxbaseClient] function),
 * which resolves the URL and key from arguments or environment variables.
 *
 * Usage:
 * ```
 * val client = FluxbaseClient.create("https://flux.example.com", anonKey)
 * val (session, error) = client.auth.signInWithPassword("user@example.com", "pw")
 * ```
 */
class FluxbaseClient internal constructor(
    /** The shared HTTP client used by all modules. */
    val http: FluxbaseHttpClient,
    /** The auth module. */
    val auth: FluxbaseAuth,
) {
    /**
     * Set the active tenant for multi-tenancy. Sets the `X-FB-Tenant` header.
     * Port of `setTenant()` in `client.ts:567-574`.
     */
    fun setTenant(tenantId: String) {
        http.setHeader("X-FB-Tenant", tenantId)
    }

    companion object {
        /**
         * Create a [FluxbaseClient]. Port of `createClient()` in `client.ts:770-823`.
         *
         * Resolves [url] and [key] from the arguments or environment variables
         * (`FLUXBASE_URL` / `FLUXBASE_ANON_KEY` / `FLUXBASE_SERVICE_TOKEN`),
         * then wires the HTTP client with `apikey` + `Authorization: Bearer` headers
         * and constructs the auth module.
         *
         * @param url the Fluxbase server URL. Falls back to `FLUXBASE_URL` env var.
         * @param key the anon or service-role key. Falls back to env vars.
         * @param options client options (autoRefresh, storage, headers, etc.).
         * @param transport the HTTP transport SPI (default: [KtorHttpTransport]; tests
         *   inject a recording fake).
         */
        fun create(
            url: String? = null,
            key: String? = null,
            options: FluxbaseClientOptions = FluxbaseClientOptions(),
            transport: HttpTransport? = null,
        ): FluxbaseClient {
            val resolvedUrl = url
                ?: System.getenv("FLUXBASE_URL")
                ?: error("Fluxbase URL is required (pass it or set FLUXBASE_URL)")

            val resolvedKey = key
                ?: System.getenv("FLUXBASE_SERVICE_TOKEN")
                ?: System.getenv("FLUXBASE_ANON_KEY")
                ?: error("Fluxbase key is required (pass it or set FLUXBASE_ANON_KEY)")

            val resolvedTransport = transport ?: KtorHttpTransport(resolvedUrl)

            // The HTTP client holds apikey + Authorization headers.
            // In TS, the constructor sets `apikey: key, Authorization: Bearer key`.
            val mergedHeaders = options.headers.toMutableMap().apply {
                put("apikey", resolvedKey)
            }

            val http = FluxbaseHttpClient(
                baseUrl = resolvedUrl,
                transport = resolvedTransport,
            )
            // Seed apikey + anon key as the initial Authorization.
            http.setAnonKey(resolvedKey)
            mergedHeaders.forEach { (k, v) -> http.setHeader(k, v) }

            val auth = FluxbaseAuth(
                http = http,
                autoRefresh = options.autoRefresh,
                storage = if (options.persist) options.storage else MemoryStorage(),
            )

            return FluxbaseClient(http, auth)
        }
    }
}

/**
 * Top-level factory function — Kotlin-idiomatic equivalent of the TS
 * `createClient(url, key, options)`. Delegates to [FluxbaseClient.create].
 */
fun createFluxbaseClient(
    url: String? = null,
    key: String? = null,
    options: FluxbaseClientOptions = FluxbaseClientOptions(),
    transport: HttpTransport? = null,
): FluxbaseClient = FluxbaseClient.create(url, key, options, transport)

/**
 * Start a PostgREST query against [table]. Uses a reified type parameter so the
 * kotlinx.serialization serializer is resolved at compile time.
 *
 * Usage: `client.from<Trip>("trips").select().eq("user_id", uid).execute()`
 *
 * Port of `client.from(table)` in `client.ts:447`.
 */
inline fun <reified T> FluxbaseClient.from(table: String, schema: String? = null): io.github.nimbleflux.fluxbase.postgrest.QueryBuilder<T> {
    val serializer = kotlinx.serialization.serializer<T>()
    return io.github.nimbleflux.fluxbase.postgrest.QueryBuilder(this.http, serializer, table, schema)
}
