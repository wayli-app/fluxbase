# ------------------------------------------------------------------------------
# Multi-Stage Dockerfile for Fluxbase (glibc-only, no musl)
# ------------------------------------------------------------------------------
#
# Two image flavors are produced from this single Dockerfile, selected by the
# FLAVOR build argument:
#
#   FLAVOR=full (default):  debian:bookworm-slim runtime. Includes Tesseract
#                           OCR, ffmpeg (for edge-function video transcoding),
#                           and libvips-backed image transformations. Built with
#                           `go build -tags "ocr vips"` and CGO_ENABLED=1.
#                           Tag suffix: none (this is the default image).
#
#   FLAVOR=lite:            distroless/base-debian12 runtime. CGO-enabled glibc
#                           binary (pg_query_go's C parser requires cgo) built
#                           WITHOUT the ocr/vips tags. No OCR, no ffmpeg, no
#                           image transforms (smaller than full). Capabilities
#                           self-disable at runtime and report unavailable via
#                           the API. Tag suffix: -lite.
#
# Usage:
#   Full (default, with admin UI):  docker build -t fluxbase:latest .
#                                   (FLAVOR=full is the default; builds the
#                                    final stage runtime-full.)
#   Lite (smaller, no media deps):  docker build --build-arg FLAVOR=lite \
#                                          --target runtime-lite -t fluxbase:lite .
#   Backend only (for testing):     docker build --target go-builder -t fluxbase:backend .
#
# NOTE: FLAVOR selects the go-builder's CGO/build-tag configuration, while the
# --target flag selects which runtime stage is emitted. They must agree:
#   FLAVOR=full  -> (no --target, or --target runtime-full)
#   FLAVOR=lite  -> --target runtime-lite
#
# ------------------------------------------------------------------------------

ARG FLAVOR=full

FROM denoland/deno:bin-2.6.4 AS deno-bin

# ------------------------------------------------------------------------------
# Stage 1: Build SDKs and Admin UI
# ------------------------------------------------------------------------------
FROM oven/bun:1.3.14-debian AS admin-builder

WORKDIR /build

# Copy all workspace files (excluding node_modules via .dockerignore)
COPY package.json bun.lock ./
COPY sdk/ ./sdk/
COPY sdk-react/ ./sdk-react/
COPY sdk-svelte/ ./sdk-svelte/
COPY sdk-next/ ./sdk-next/
COPY sdk-vue/ ./sdk-vue/
COPY admin/ ./admin/
COPY docs/ ./docs/

# Install dependencies (no-cache to avoid integrity issues)
RUN bun install --no-cache

# Build SDK (run from root to ensure proper binary resolution)
RUN bun run --cwd sdk build

# Generate embedded SDK for job and function runtime
RUN mkdir -p /build/internal/jobs /build/internal/runtime \
    && bun run --cwd sdk generate:embedded-sdk

# Build SDK-React
RUN bun run --cwd sdk-react build

# Refresh dependencies to ensure proper symlinks after SDK builds
# (SDK exports point to dist/ which didn't exist during initial install)
RUN bun install

# Install Node.js for vite build (bun has compatibility issues with vite 7)
RUN apt-get update && apt-get install -y --no-install-recommends nodejs npm && rm -rf /var/lib/apt/lists/*

# Build admin UI (use npx with node for better vite compatibility)
RUN cd /build/admin && bunx tsc -b && npx vite build


# ------------------------------------------------------------------------------
# Stage 2: Build Go Binary
# ------------------------------------------------------------------------------
FROM golang:1.26.1-bookworm AS go-builder

ARG FLAVOR=full

# Install build dependencies. CGO is required for BOTH flavors because
# pg_query_go (used for SQL validation) compiles a C parser via cgo. The
# Tesseract/Leptonica/libvips dev packages are only needed by the full flavor
# (they back the OCR + image-transform code paths); the lite flavor omits them
# and compiles without the `ocr vips` tags, so those code paths use their stubs.
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    make \
    gcc \
    g++ \
    pkg-config \
    ca-certificates \
    && case "$FLAVOR" in \
         full) apt-get install -y --no-install-recommends \
                 libtesseract-dev \
                 libleptonica-dev \
                 libvips-dev \
                 poppler-utils;; \
         lite) ;; \
         *) echo "Unsupported FLAVOR: $FLAVOR (expected 'full' or 'lite')" && exit 1;; \
       esac \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Copy built admin UI
COPY --from=admin-builder /build/admin/dist ./internal/adminui/dist

# Copy generated embedded SDKs
COPY --from=admin-builder /build/internal/jobs/embedded_sdk.js ./internal/jobs/embedded_sdk.js
COPY --from=admin-builder /build/internal/runtime/embedded_sdk.js ./internal/runtime/embedded_sdk.js

# Build arguments for versioning
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the Go binary.
#   full: CGO enabled, glibc-native, OCR + vips build tags (Tesseract + libvips).
#   lite: CGO enabled (pg_query_go needs the C parser), no OCR/vips tags so the
#         stubs compile in. The lite binary is glibc-linked (NOT pure-static),
#         so it runs on distroless/base-debian12 (glibc) but not on scratch.
RUN set -eux; \
    if [ "$FLAVOR" = "full" ]; then \
      BUILD_TAGS="ocr vips"; \
    else \
      BUILD_TAGS=""; \
    fi; \
    CGO_ENABLED=1 GOOS=linux go build \
      -tags "$BUILD_TAGS" \
      -ldflags="-w -s \
        -X main.Version=${VERSION} \
        -X main.Commit=${COMMIT} \
        -X main.BuildDate=${BUILD_DATE}" \
      -o fluxbase-server \
      ./cmd/fluxbase


# ------------------------------------------------------------------------------
# Stage 3a: Fetch pgschema (shared by both runtime images)
# ------------------------------------------------------------------------------
FROM debian:bookworm-slim AS pgschema-fetcher

# Install pgschema for declarative schema management
# Bumped from 1.7.4 to 1.12.1: fixes plan validation for LANGUAGE sql functions
# with SET search_path that use extension operators (e.g. pgvector <=>), plus
# schema-qualified function-body references (pgplex/pgschema#399).
ARG PGSCHEMA_VERSION=1.12.1
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/* \
    && ARCH=$(dpkg --print-architecture) \
    && if [ "$ARCH" = "amd64" ]; then PGSCHEMA_ARCH="linux-amd64"; \
    elif [ "$ARCH" = "arm64" ]; then PGSCHEMA_ARCH="linux-arm64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi \
    && curl -fsSL "https://github.com/pgplex/pgschema/releases/download/v${PGSCHEMA_VERSION}/pgschema-${PGSCHEMA_VERSION}-${PGSCHEMA_ARCH}" -o /usr/local/bin/pgschema \
    && chmod +x /usr/local/bin/pgschema


# ------------------------------------------------------------------------------
# Stage 3b: Prepare writable runtime directories for the distroless lite image.
#
# base-debian12:nonroot runs as uid/gid 65532, but distroless has no shell, so
# we cannot `mkdir`/`chown` inside the lite stage itself. We create the runtime
# dirs here with the right owner and copy them into the lite stage with
# --chown. This mirrors what runtime-full does via `mkdir + chown`. Without it,
# a named or anonymous volume mounted on /app/storage is root-owned, the
# LocalStorage.Health probe cannot write .health_check, and validateStorageHealth
# aborts startup (see cmd/fluxbase/main.go).
# ------------------------------------------------------------------------------
FROM debian:bookworm-slim AS lite-dirs
RUN mkdir -p /appdir/storage /appdir/config /appdir/data /appdir/logs \
    && chown -R 65532:65532 /appdir


# ------------------------------------------------------------------------------
# Stage 3c: Lite Production Runtime Image (distroless, glibc CGO binary)
# ------------------------------------------------------------------------------
FROM gcr.io/distroless/base-debian12:nonroot AS runtime-lite

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

LABEL maintainer="Fluxbase Team" \
      description="Fluxbase - Production-Ready Backend-as-a-Service (lite, no OCR/ffmpeg/vips)" \
      version="${VERSION}" \
      commit="${COMMIT}" \
      build-date="${BUILD_DATE}" \
      org.opencontainers.image.variant="lite"

WORKDIR /app

# Copy the Go binary and helper binaries. The binary is glibc-linked (cgo is
# required for pg_query_go), and distroless/base-debian12 ships glibc, so it
# runs here. The deno binary is also glibc-linked and runs unmodified.
COPY --from=go-builder /build/fluxbase-server /usr/local/bin/fluxbase-server
COPY --from=deno-bin /deno /usr/local/bin/deno
COPY --from=pgschema-fetcher /usr/local/bin/pgschema /usr/local/bin/pgschema

# Seed writable runtime directories owned by uid/gid 65532 (the :nonroot user).
# distroless has no shell, so we cannot mkdir/chown at runtime; COPY --chown
# from the lite-dirs stage bakes in the correct ownership. This makes
# /app/storage writable when an anonymous volume is mounted (Docker copies
# existing image-owned content into a new anonymous volume with ownership
# preserved), matching what runtime-full sets up via mkdir + chown. Required so
# LocalStorage.Health can write its .health_check probe on first boot.
COPY --from=lite-dirs --chown=65532:65532 /appdir/storage /app/storage
COPY --from=lite-dirs --chown=65532:65532 /appdir/config  /app/config
COPY --from=lite-dirs --chown=65532:65532 /appdir/data    /app/data
COPY --from=lite-dirs --chown=65532:65532 /appdir/logs    /app/logs

# distroless :nonroot runs as uid 65532. The runtime writes /tmp/deno for edge
# functions; base-debian12 provides a writable /tmp by default.
ENV FLUXBASE_SERVER_ADDRESS=:8080 \
    FLUXBASE_DEBUG=false \
    FLUXBASE_LOGGING_CONSOLE_LEVEL=info \
    FLUXBASE_DATABASE_MAX_CONNECTIONS=25 \
    FLUXBASE_DATABASE_MIN_CONNECTIONS=5 \
    TMPDIR=/tmp

EXPOSE 8080
# No HEALTHCHECK: distroless has no shell/wget. Rely on external probes
# (docker compose healthcheck / k8s readiness) hitting GET /health.

USER nonroot:nonroot

VOLUME ["/app/storage", "/app/config", "/app/logs"]

ENTRYPOINT ["fluxbase-server"]


# ------------------------------------------------------------------------------
# Stage 3d: Full Production Runtime Image (glibc, with OCR/ffmpeg/vips)
#
# This is the FINAL stage (and therefore the default image produced by
# `docker build` with no --target), preserving the historical default behavior.
# The lite image is produced with `docker build --target runtime-lite`.
# ------------------------------------------------------------------------------
FROM debian:bookworm-slim AS runtime-full

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

LABEL maintainer="Fluxbase Team" \
      description="Fluxbase - Production-Ready Backend-as-a-Service (full)" \
      version="${VERSION}" \
      commit="${COMMIT}" \
      build-date="${BUILD_DATE}" \
      org.opencontainers.image.variant="full"

# Install runtime dependencies (OCR, media, image transforms)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    curl \
    gnupg \
    tesseract-ocr \
    tesseract-ocr-eng \
    libtesseract5 \
    libleptonica-dev \
    libvips \
    poppler-utils \
    ffmpeg \
    libstdc++6 \
    wget \
    unzip \
    && rm -rf /var/lib/apt/lists/*

COPY --from=deno-bin /deno /usr/local/bin/deno
COPY --from=pgschema-fetcher /usr/local/bin/pgschema /usr/local/bin/pgschema

# Create non-root user
RUN groupadd -g 1000 fluxbase \
    && useradd -u 1000 -g fluxbase -s /usr/sbin/nologin fluxbase

WORKDIR /app

# Copy binary
COPY --from=go-builder /build/fluxbase-server /usr/local/bin/fluxbase-server

# Create directories
RUN mkdir -p /app/storage /app/config /app/data /app/logs \
    && chown -R fluxbase:fluxbase /app

USER fluxbase

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

ENV FLUXBASE_SERVER_ADDRESS=:8080 \
    FLUXBASE_DEBUG=false \
    FLUXBASE_LOGGING_CONSOLE_LEVEL=info \
    FLUXBASE_DATABASE_MAX_CONNECTIONS=25 \
    FLUXBASE_DATABASE_MIN_CONNECTIONS=5

VOLUME ["/app/storage", "/app/config", "/app/logs"]

ENTRYPOINT ["fluxbase-server"]
