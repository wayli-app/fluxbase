# ------------------------------------------------------------------------------
# Multi-Stage Dockerfile for Fluxbase (glibc-only, no musl)
# ------------------------------------------------------------------------------
#
# Usage:
#   Production (with admin UI):  docker build -t fluxbase:latest .
#   Backend only (for testing):  docker build --target go-builder -t fluxbase:backend .
#
# ------------------------------------------------------------------------------

FROM denoland/deno:bin-2.6.4 AS deno-bin

# ------------------------------------------------------------------------------
# Stage 1: Build SDKs and Admin UI
# ------------------------------------------------------------------------------
FROM node:25-bookworm AS admin-builder

WORKDIR /build

# Copy SDK packages first (admin depends on these)
COPY sdk/ ./sdk/
COPY sdk-react/ ./sdk-react/

# Build SDKs
WORKDIR /build/sdk
RUN npm ci && npm run build

# Generate embedded SDK for job and function runtime
RUN mkdir -p /build/internal/jobs /build/internal/runtime \
    && npm run generate:embedded-sdk

WORKDIR /build/sdk-react
RUN npm ci && npm run build

# Build admin UI
WORKDIR /build/admin
COPY admin/package*.json ./
RUN npm ci
COPY admin/ ./
RUN npm run build


# ------------------------------------------------------------------------------
# Stage 2: Build Go Binary (glibc, CGO-enabled)
# ------------------------------------------------------------------------------
FROM golang:1.25-bookworm AS go-builder

# Install build dependencies for CGO-based libraries
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    make \
    gcc \
    g++ \
    pkg-config \
    libtesseract-dev \
    libleptonica-dev \
    libvips-dev \
    poppler-utils \
    ca-certificates \
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

# Build the Go binary
# - CGO enabled
# - glibc-native
# - OCR build tag enabled
RUN CGO_ENABLED=1 GOOS=linux go build \
    -tags "ocr" \
    -ldflags="-w -s \
        -X main.Version=${VERSION} \
        -X main.Commit=${COMMIT} \
        -X main.BuildDate=${BUILD_DATE}" \
    -o fluxbase-server \
    ./cmd/fluxbase


# ------------------------------------------------------------------------------
# Stage 3: Production Runtime Image (glibc)
# ------------------------------------------------------------------------------
FROM debian:bookworm-slim

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

LABEL maintainer="Fluxbase Team" \
      description="Fluxbase - Production-Ready Backend-as-a-Service" \
      version="${VERSION}" \
      commit="${COMMIT}" \
      build-date="${BUILD_DATE}"

# Install runtime dependencies
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
    libstdc++6 \
    wget \
    unzip \
    && rm -rf /var/lib/apt/lists/*

COPY --from=deno-bin /deno /usr/local/bin/deno

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
    FLUXBASE_LOG_LEVEL=info \
    FLUXBASE_DATABASE_MAX_CONNECTIONS=25 \
    FLUXBASE_DATABASE_MIN_CONNECTIONS=5

VOLUME ["/app/storage", "/app/config", "/app/logs"]

ENTRYPOINT ["fluxbase-server"]
