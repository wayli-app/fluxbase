---
title: Kotlin SDK — Getting Started
description: Native Kotlin/JVM SDK for Fluxbase, for Android and server-side Kotlin applications.
---

# Kotlin SDK — Getting Started

The Fluxbase Kotlin SDK (`fluxbase-kotlin`) is a native Kotlin port of the
[TypeScript SDK](/sdk/getting-started/). It speaks Fluxbase's wire protocol
directly — you cannot use the Supabase Kotlin SDK against a Fluxbase instance,
because Fluxbase's URL paths, realtime protocol, 2FA API, and feature surface
(jobs, secrets, knowledge bases, multi-tenancy) differ from Supabase.

> **Status:** early development. The core HTTP layer and auth module are shipping
> first; PostgREST, realtime, storage, functions, jobs, and the remaining surface
> follow. See the [API reference](/api/sdk-kotlin/) for what's available today.

## Installation

### Gradle (Kotlin DSL)

```kotlin
repositories {
    mavenCentral()
    // Once published to GitHub Packages:
    maven("https://maven.pkg.github.com/nimbleflux/fluxbase")
}

dependencies {
    implementation("io.github.nimbleflux:fluxbase-kotlin:2026.8.8")
}
```

### Dev (composite build, from source)

If you're developing the SDK alongside an app (e.g. Wayli), consume it directly
without publishing:

```kotlin
// settings.gradle.kts
includeBuild("../dev/fluxbase/sdk-kotlin")

// build.gradle.kts
dependencies {
    implementation("io.github.nimbleflux:fluxbase-kotlin")
}
```

## Quick start

```kotlin
import io.github.nimbleflux.fluxbase.FluxbaseClient

val client = FluxbaseClient(
    url = "https://flux.example.com",
    key = anonKey,
)

// Sign in with email/password
val (data, error) = client.auth.signInWithPassword("user@example.com", "password")
if (error != null) {
    // handle error
    return
}

// 2FA challenge (if enabled)
if (data == null) {
    // client.auth.verify2fa(userId, code)
}
```

## Why a separate SDK?

Fluxbase deliberately mimics Supabase's *SDK ergonomics* (method names,
`{data, error}` result tuples) but its **HTTP transport is incompatible**:

| Layer | Supabase | Fluxbase |
|---|---|---|
| Auth | `/auth/v1/...` | `/api/v1/auth/...` |
| DB (PostgREST) | `/rest/v1/:table` | `/api/v1/tables/:schema/:table` |
| Realtime | `/realtime/v1/websocket` (Phoenix) | `/realtime?token=` (custom JSON) |
| Storage | `/storage/v1/...` | `/api/v1/storage/...` |
| Functions | `/functions/v1/:name` | `/api/v1/functions/:name/invoke` |
| 2FA | Factor-based MFA | `setup2FA` / `enable2FA` / `verify2FA` |

Plus Fluxbase has features Supabase lacks: background jobs (+cron), encrypted
secrets, app/system settings, namespaced RPC, AI/knowledge bases, and
multi-tenancy. This SDK covers all of them natively.

## Architecture

The SDK mirrors the TypeScript SDK's module structure 1:1:

- `core` — HTTP transport, types, error handling
- `auth` — sessions, email/password, OAuth, TOTP 2FA, magic links
- `postgrest` — chainable query builder (filters, PostGIS, pgvector)
- `realtime` — WebSocket subscriptions (postgres_changes, broadcast, presence)
- `storage` — file upload/download (simple, stream, chunked/resumable)
- `functions` — edge function invocation
- `jobs` — background job submission and tracking
- `rpc` — namespaced stored procedure calls
- `secrets` — encrypted secret management
- `settings` — app/system/user settings

All HTTP I/O goes through a single `HttpTransport` SPI, which makes every module
testable with recording fakes — no mock libraries or live server needed for unit
tests.

## Next steps

- [API Reference](/api/sdk-kotlin/) — auto-generated from source (Dokka)
- [TypeScript SDK guide](/sdk/getting-started/) — the API is intentionally similar
- [HTTP API reference](/api/http/) — for direct HTTP calls without an SDK
