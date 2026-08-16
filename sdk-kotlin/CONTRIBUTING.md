# Contributing to fluxbase-kotlin

This SDK is a 1:1 Kotlin port of the TypeScript SDK (`../sdk`). The TS SDK is the
authoritative spec for behavior — every URL, header, and message shape here is
ported from it.

## Prerequisites

- **JDK 21** (Temurin recommended). Install via [sdkman](https://sdkman.io):
  ```bash
  curl -s "https://get.sdkman.io" | bash
  sdk install java 21.0.12-tem
  ```
- **Docker** (for the integration-test Fluxbase instance).
- No system Gradle needed — use the wrapper: `./gradlew`.

## Dev environment

### Start a local Fluxbase for integration tests

The fastest iteration loop is a native Go server + one Postgres container.

```bash
# 1. Start Postgres (keep running)
docker run -d --name fluxbase-pg -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=fluxbase -p 5432:5432 ghcr.io/nimbleflux/fluxbase-postgres:18

# 2. Generate dev secrets + anon/service-role JWTs
cd ~/dev/fluxbase
./deploy/generate-keys.sh --stdout > .env.dev
# → POSTGRES_PASSWORD, FLUXBASE_AUTH_JWT_SECRET, FLUXBASE_ENCRYPTION_KEY,
#   FLUXBASE_SECURITY_SETUP_TOKEN, FLUXBASE_ANON_KEY, FLUXBASE_SERVICE_ROLE_KEY

# 3. Run the server (native Go, rebuilds on change; Deno on PATH for edge functions)
export $(grep -v '^#' .env.dev | xargs)
export FLUXBASE_DATABASE_HOST=localhost FLUXBASE_DATABASE_PORT=5432 \
       FLUXBASE_DATABASE_USER=postgres FLUXBASE_DATABASE_PASSWORD=$POSTGRES_PASSWORD \
       FLUXBASE_DATABASE_DATABASE=fluxbase FLUXBASE_DATABASE_SSL_MODE=disable \
       FLUXBASE_STORAGE_PROVIDER=local FLUXBASE_STORAGE_LOCAL_PATH=./storage
./run-server.sh
# Health: http://localhost:8080/health  →  {"status":"ok","database":"connected"}
```

Integration tests read the connection details from environment variables:
- `FLUXBASE_TEST_URL` (default `http://localhost:8080`)
- `FLUXBASE_TEST_ANON_KEY` (from `.env.dev`)
- `FLUXBASE_TEST_SERVICE_KEY` (from `.env.dev`, for admin operations)

### Reset between sessions
```bash
make db-reset-full   # nukes all non-system schemas; re-applies on next server start
```

### Regenerate the OpenAPI spec (for codegen bootstrapping)
The spec is served live and is **not** checked in (it's in `.gitignore`):
```bash
curl -s -H "X-Service-Key: $FLUXBASE_SERVICE_ROLE_KEY" \
  http://localhost:8080/openapi.json > sdk-kotlin/openapi.json
```
The admin/service-role key yields the full spec (incl. table schemas); anon
returns a near-empty spec.

## TDD workflow

We use **test-driven development**. The TS SDK's own test suite is the oracle.

### The loop (Red → Green → Refactor)
1. **Red** — write a failing Kotlin test that mirrors a specific TS test. Example
   (mirrors `auth.test.ts` signIn):
   ```kotlin
   @Test
   fun `signIn posts to auth signin and stores session`() = runTest {
       val recording = RecordingHttp(mockResponseBody = signInResponseJson)
       val client = FluxbaseClient("http://localhost:8080", recording)

       val (data, error) = client.auth.signInWithPassword("user@example.com", "pw")

       assertEquals("/api/v1/auth/signin", recording.lastPath)
       assertEquals("POST", recording.lastMethod)
       assertNull(error)
       assertEquals("access-token", data?.session?.accessToken)
   }
   ```
   Run `./gradlew test` — it fails (the auth module isn't implemented yet).
2. **Green** — implement the minimum code to pass.
3. **Refactor** — clean up; tests stay green.

### Two test tiers

The SDK has two source sets mirroring the TS SDK's own split:

1. **Unit tests** (`src/test/kotlin`) — fast (~1s), no server. Uses
   [RecordingHttp](src/test/kotlin/io/github/nimbleflux/fluxbase/core/test/RecordingHttp.kt),
   a recording fake that asserts on exact URL/query/body shapes. This is where
   TDD lives. Run: `./gradlew test`.

2. **Contract/integration tests** (`src/integrationTest/kotlin`) — against a live
   Fluxbase on `localhost:8080`. Proves wire compatibility end-to-end: signup →
   signin → refresh; CRUD + PostgREST filters; RPC; functions; jobs; storage;
   realtime. This is the authoritative "are we doing it right?" signal. Run:
   `./gradlew integrationTest`.

### Coverage
Kover reports are generated on `check`. Target ≥80% on `core`, `auth`,
`postgrest`, `realtime` (the TS SDK threshold is 50%; we aim higher).

### Linting
```bash
./gradlew detekt   # static analysis (config: detekt.yml)
```

## Parity tracking

Every TS `*.test.ts` behavior should have a Kotlin counterpart. The
`parity-check` task (S5) will enforce this automatically; until then, track
parity manually: when porting a module, open the corresponding `sdk/src/<module>.test.ts`
and ensure each `it(...)` block has a matching Kotlin `@Test`.

## Publishing

The Kotlin artifact is published to GitHub Packages (`io.github.nimbleflux:fluxbase-kotlin`)
as part of the **main Fluxbase release** — there is no separate Kotlin tag scheme.

- The release workflow (`.github/workflows/release.yml`) fires on the main `v*` tag.
- The `publish_kotlin_sdk` input toggle (default on) runs the `publish-kotlin-sdk`
  job, which builds and uploads the artifact with `-Pversion=<main-tag-version>`.
- To skip Kotlin publishing on a release where it didn't change, set
  `publish_kotlin_sdk=false` when triggering the release.
- Dev consumption is via composite build (no publish needed).

## Known issues to fix upstream

- `sdk/src/jobs.ts:64` calls `/rest/v1/user_profiles` (unregistered path) —
  should be `/api/v1/tables/user_profiles`. Fix when porting the jobs module.
