# fluxbase-kotlin

A full-parity Kotlin port of [`@nimbleflux/fluxbase-sdk`](../sdk), the TypeScript
SDK for [Fluxbase](https://fluxbase.eu) — a self-hostable, Supabase-compatible
Backend-as-a-Service.

> **Status:** early development. The core HTTP layer and auth module are in
> progress (S1). PostgREST, realtime, storage, functions, jobs, rpc, secrets,
> settings, AI, and the remaining TS-parity surface follow in subsequent
> milestones — see the plan in `docs/plans/` (Wayli repo).

## Why a native Kotlin SDK?

Fluxbase mimics Supabase's *SDK ergonomics* (method names, `{data, error}` tuples)
but its **wire protocol is entirely different** — different URL paths, a custom
WebSocket realtime protocol (not Phoenix), a custom 2FA API, and large
Fluxbase-only surfaces (jobs, secrets, settings, knowledge bases, multi-tenancy).
You cannot point Supabase-Kotlin at a Fluxbase URL. This SDK speaks Fluxbase's
wire protocol natively, from Kotlin/JVM and (planned) Android.

## Quick start

```kotlin
val client = FluxbaseClient(
    url = "https://flux.example.com",
    key = anonKey,
)

// Sign in
val (session, error) = client.auth.signInWithPassword("user@example.com", "password")

// Query
val trips = client.from("trips").select().eq("user_id", uid).execute()
```

## Development

### Prerequisites
- **JDK 21** (Temurin recommended).
- A running Fluxbase instance for integration tests (see CONTRIBUTING.md).

### Build & test
```bash
./gradlew test                 # unit tests (pure fakes, ~1s)
./gradlew integrationTest      # contract tests vs localhost:8080
./gradlew check                # lint + unit + integration + coverage
```

### Architecture
The SDK mirrors the TS SDK's module structure 1:1 (`core`, `auth`, `postgrest`,
`realtime`, `storage`, `functions`, `jobs`, `rpc`, `secrets`, `settings`, `ai`, …).
All HTTP I/O goes through a single `HttpTransport` SPI, which makes every module
trivially testable with a recording fake — no mock libraries, no live server needed
for unit tests. See `src/test/kotlin/.../core/test/RecordingHttp.kt`.

### Consuming from the Wayli app (dev)
```kotlin
// settings.gradle.kts
includeBuild("../dev/fluxbase/sdk-kotlin")

// build.gradle.kts
dependencies { implementation("io.github.nimbleflux:fluxbase-kotlin") }
```

### Consuming a published version (release)
```kotlin
repositories { maven("https://maven.pkg.github.com/nimbleflux/fluxbase") }
dependencies { implementation("io.github.nimbleflux:fluxbase-kotlin:2026.8.8") }
```

See CONTRIBUTING.md for the full dev environment setup and the TDD workflow.
