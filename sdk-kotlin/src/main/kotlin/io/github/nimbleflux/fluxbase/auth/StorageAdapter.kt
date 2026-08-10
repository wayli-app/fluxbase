package io.github.nimbleflux.fluxbase.auth

/**
 * SPI for persisting the auth session. Port of `StorageAdapter` from
 * `sdk/src/types.ts:25`.
 *
 * The TS SDK uses `localStorage` in browsers and `MemoryStorage` in Node/SSR.
 * In Kotlin:
 *   - The default [MemoryStorage] keeps the session in memory (JVM/server use).
 *   - On Android, inject an `EncryptedSharedPreferences`-backed implementation.
 *
 * The storage key is always `"fluxbase.auth.session"` (matching the TS constant
 * `AUTH_STORAGE_KEY` in `auth.ts:61`).
 */
interface StorageAdapter {
    fun getItem(key: String): String?
    fun setItem(key: String, value: String)
    fun removeItem(key: String)
}

/**
 * In-memory [StorageAdapter] — the default for JVM. Session is lost on restart.
 * Port of `MemoryStorage` from `sdk/src/auth.ts:74`.
 */
class MemoryStorage : StorageAdapter {
    private val store = mutableMapOf<String, String>()

    override fun getItem(key: String): String? = store[key]

    override fun setItem(key: String, value: String) {
        store[key] = value
    }

    override fun removeItem(key: String) {
        store.remove(key)
    }
}

/** The storage key under which the session JSON is persisted. */
const val AUTH_STORAGE_KEY = "fluxbase.auth.session"
