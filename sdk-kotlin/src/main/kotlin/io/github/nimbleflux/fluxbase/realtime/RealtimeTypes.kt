package io.github.nimbleflux.fluxbase.realtime

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

/**
 * The type of a postgres_changes event. Port of the TS `eventType` values.
 */
enum class ChangeEventType {
    INSERT, UPDATE, DELETE;

    val wildcard: String get() = "*"
}

/**
 * A normalized postgres_changes payload — the Supabase-compatible shape the TS SDK
 * produces by converting the server's `new_record`/`old_record` to `new`/`old`.
 * Port of `RealtimePostgresChangesPayload` from `sdk/src/types.ts:408`.
 */
@Serializable
data class PostgresChangesPayload(
    val eventType: String,
    val schema: String? = null,
    val table: String? = null,
    @SerialName("commit_timestamp") val commitTimestamp: String? = null,
    val newRecord: JsonElement? = null,
    val oldRecord: JsonElement? = null,
    val errors: String? = null,
) {
    /**
     * Convenience accessor — the TS SDK exposes `new` and `old`; in Kotlin
     * those are keywords, so we use [newRecord] / [oldRecord].
     */
    val new: JsonElement? get() = newRecord
    val old: JsonElement? get() = oldRecord
}

/**
 * Callback type for postgres_changes events.
 */
typealias ChangeEventCallback = (PostgresChangesPayload) -> Unit

/**
 * Callback type for broadcast events.
 */
typealias BroadcastCallback = (JsonElement) -> Unit

/**
 * Subscription config for a postgres_changes channel.
 * Port of `PostgresChangesConfig` from `sdk/src/types.ts:397`.
 */
data class PostgresChangesConfig(
    val event: String = "*",
    val schema: String = "public",
    val table: String? = null,
    val filter: String? = null,
)
