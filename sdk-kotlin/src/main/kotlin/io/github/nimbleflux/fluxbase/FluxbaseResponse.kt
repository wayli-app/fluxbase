package io.github.nimbleflux.fluxbase

/**
 * The result type for all Fluxbase SDK operations — the Kotlin equivalent of the
 * TS SDK's `{ data: T; error: null } | { data: null; error: Error }` union
 * (`sdk/src/types.ts:3904`).
 *
 * Instead of throwing, SDK methods return a [FluxbaseResponse] so callers can
 * destructure the result:
 * ```
 * val (session, error) = client.auth.signInWithPassword(...)
 * if (error != null) { ... }
 * ```
 *
 * This matches the TS pattern `const { data, error } = await client.auth.signIn(...)`.
 */
sealed interface FluxbaseResponse<out T> {
    /**
     * The data payload on success, or null on error.
     * In the TS SDK this is the `data` field.
     */
    val data: T?

    /**
     * The error on failure. Null on success.
     */
    val error: FluxbaseError?

    /**
     * Destructuring support: `val (data, error) = result`.
     * Matches the TS `const { data, error } = await ...` pattern.
     * These must be members (not extensions) for Kotlin to recognize them in
     * destructuring declarations.
     */
    operator fun component1(): T? = data
    operator fun component2(): FluxbaseError? = error

    /** Successful result. */
    data class Success<T>(override val data: T) : FluxbaseResponse<T> {
        override val error: FluxbaseError? get() = null
    }

    /** Failed result. */
    data class Error(override val error: FluxbaseError) : FluxbaseResponse<Nothing> {
        override val data: Nothing? get() = null
    }
}

/**
 * An error from a Fluxbase API call. Port of `FluxbaseError` from
 * `sdk/src/types.ts:235`.
 */
data class FluxbaseError(
    val status: Int? = null,
    val code: String? = null,
    val details: Any? = null,
    override val message: String,
) : RuntimeException(message)

/**
 * Returns the data on success, or null on error.
 * Equivalent to `result.data` in the TS SDK.
 */
fun <T> FluxbaseResponse<T>.getOrNull(): T? = data

/**
 * Returns the data on success, or throws the error on failure.
 */
fun <T> FluxbaseResponse<T>.getOrThrow(): T =
    when (this) {
        is FluxbaseResponse.Success -> data
        is FluxbaseResponse.Error -> throw error
    }

/**
 * Wraps a suspending block, catching exceptions and converting them to
 * [FluxbaseResponse.Error]. The TS SDK uses `wrapAsync` for the same purpose
 * (`sdk/src/utils/error-handling.ts`).
 */
suspend fun <T> fluxbaseResponse(block: suspend () -> T): FluxbaseResponse<T> =
    try {
        FluxbaseResponse.Success(block())
    } catch (e: FluxbaseError) {
        FluxbaseResponse.Error(e)
    } catch (e: io.github.nimbleflux.fluxbase.core.FluxbaseException) {
        FluxbaseResponse.Error(
            FluxbaseError(
                status = e.status,
                code = e.code,
                details = e.details,
                message = e.message ?: "Unknown error",
            ),
        )
    } catch (e: Exception) {
        FluxbaseResponse.Error(FluxbaseError(message = e.message ?: "Unknown error"))
    }
