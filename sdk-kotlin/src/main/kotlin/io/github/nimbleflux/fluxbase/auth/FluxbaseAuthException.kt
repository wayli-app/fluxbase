package io.github.nimbleflux.fluxbase.auth

/** Thrown for client-side OAuth flow errors (e.g. missing pending provider). */
class FluxbaseAuthException(message: String) : Exception(message)
