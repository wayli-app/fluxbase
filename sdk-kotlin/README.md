# fluxbase-kotlin

A native Kotlin port of [`@nimbleflux/fluxbase-sdk`](../sdk), the TypeScript SDK
for [Fluxbase](https://fluxbase.eu) — a self-hostable, Supabase-compatible
Backend-as-a-Service.

> **Status:** v1-ready. 16 modules shipped covering the public client surface
> (auth, PostgREST, realtime, storage, functions, jobs, rpc, secrets, settings,
> GraphQL, vector, branching, tenant, management). The admin sub-clients, AI
> chat, knowledge-base, and storage chunked/resumable upload are intentionally
> deferred (see [Not yet ported](#not-yet-ported)).

## Why a native Kotlin SDK?

Fluxbase mimics Supabase's *SDK ergonomics* (method names, `{data, error}` tuples)
but its **wire protocol is entirely different** — different URL paths, a custom
WebSocket realtime protocol (not Phoenix), a custom 2FA API, and large
Fluxbase-only surfaces (jobs, secrets, settings, knowledge bases, multi-tenancy).
You cannot point Supabase-Kotlin at a Fluxbase URL. This SDK speaks Fluxbase's
wire protocol natively, from Kotlin/JVM (Android support planned).

## Quick start

```kotlin
val client = FluxbaseClient.create(
    url = "https://flux.example.com",
    key = anonKey,
)

// Auth (returns {data, error} tuples, like the TS SDK)
val (session, error) = client.auth.signInWithPassword("user@example.com", "password")

// PostgREST query builder
val trips = client.from<Trip>("trips").select().eq("user_id", uid).execute()
```

## Modules

| Module | Example |
|---|---|
| **auth** | `client.auth.signInWithPassword(email, pw)`, `setup2FA()`, `refreshSession()` |
| **postgrest** | `client.from<T>("trips").select().eq(...).order("id").execute()` |
| **realtime** | `client.channel("public:trips").on("INSERT") { payload -> ... }.subscribe()` |
| **storage** | `client.storage.from("imgs").upload("a.jpg", bytes, contentType = "image/jpeg")` |
| **functions** | `client.functions.invoke<Result>("hello", mapOf("name" to "kotlin"))` |
| **jobs** | `client.jobs.submit("cleanup", payload, SubmitJobOptions(namespace = "myapp"))` |
| **rpc** | `client.rpc.invoke<Result>("compute", mapOf("x" to 1))` |
| **secrets** | `client.secrets.set("api_key", value)` (encrypted, server-side) |
| **settings** | `client.settings.get("app.name")`, `client.settings.setSetting("theme", mapOf("mode" to "dark"))` |
| **graphql** | `client.graphql.query<MyData>("query { trips { id } }")` |
| **vector** | `client.vector.embed(...)`, `client.vector.search(...)` |
| **branching** | `client.branching.list()` |
| **tenant** | `client.tenant.listMine()` |
| **management** | `client.management.clientKeys.list()` |

Sessions auto-refresh: any request that returns 401 triggers a single
(deduped) `refreshSession()` and one retry with the new token.

## Not yet ported

Deferred until needed (tracked against `sdk/src/`):

- **Admin aggregate** + 8 admin sub-clients (`admin-functions`, `admin-jobs`,
  `admin-rpc`, `admin-migrations`, `admin-storage`, `admin-realtime`,
  `admin-ai`, `admin-service-keys`).
- **AI chat** WebSocket (`/ai/ws`) + **knowledge-base** module.
- **Storage chunked / resumable upload** (simple upload works today).
- Smaller surfaces: DDL manager, OAuth provider CRUD, impersonation, bundling,
  schema query builder.

These are absent, not stubbed — calling them is a compile error, not a runtime
surprise.

## Development

### Prerequisites
- **JDK 21** (Temurin recommended; matches CI).
- A running Fluxbase instance for integration tests (see CONTRIBUTING.md).

### Build & test
```bash
./gradlew test                 # unit tests (pure fakes, ~seconds)
./gradlew detekt               # static analysis (config: detekt.yml)
./gradlew integrationTest      # contract tests vs localhost:8080 (live server)
./gradlew build -x test        # assemble JARs (sources + javadoc)
./gradlew dokkaGfm             # regenerate API docs into ../docs/.../api/sdk-kotlin
```

### Architecture
All HTTP I/O flows through a single `HttpTransport` SPI with two paths: a text
path (`request`) for JSON APIs and a binary path (`requestBytes`) for raw bytes
(storage downloads). A Ktor/OkHttp implementation backs production; tests inject
a recording fake (`RecordingHttp`) — no mock libraries, no live server needed for
unit tests. Realtime has the same shape: a `WebSocketTransport` SPI with a Ktor
implementation and a fake for protocol tests.

## Consuming

### Dev (composite build)
```kotlin
// settings.gradle.kts
includeBuild("../dev/fluxbase/sdk-kotlin")

// build.gradle.kts
dependencies { implementation("io.github.nimbleflux:fluxbase-kotlin") }
```

### Release (GitHub Packages)
```kotlin
repositories { maven("https://maven.pkg.github.com/nimbleflux/fluxbase") }
dependencies { implementation("io.github.nimbleflux:fluxbase-kotlin:2026.8.8-rc.1") }
```

The artifact version is derived from the main Fluxbase release tag and published
via the `publish_kotlin_sdk` toggle in `.github/workflows/release.yml`. See
CONTRIBUTING.md for details.
