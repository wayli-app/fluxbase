# Multi-Stage Dockerfile for Fluxbase
# Serves both development and production needs
#
# Usage:
#   Production (with admin UI):  docker build -t fluxbase:latest .
#   Backend only (for testing):  docker build --target go-builder -t fluxbase:backend .
#   Development:                 Use docker-compose.yml or make dev

# Stage 1: Build SDKs and Admin UI
FROM node:20-alpine AS admin-builder

WORKDIR /build

# Copy SDK packages first (admin depends on these)
COPY sdk/ ./sdk/
COPY sdk-react/ ./sdk-react/

# Build SDKs
WORKDIR /build/sdk
RUN npm ci && npm run build

WORKDIR /build/sdk-react
RUN npm ci && npm run build

# Now build admin UI
WORKDIR /build/admin

# Copy package files
COPY admin/package*.json ./

# Install dependencies (will use local SDK packages)
RUN npm ci

# Copy admin source
COPY admin/ ./

# Build admin UI
RUN npm run build

# Stage 2: Build Go Binary
FROM golang:1.25-alpine AS go-builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

WORKDIR /build

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Copy built admin UI from previous stage to the embed location
COPY --from=admin-builder /build/admin/dist ./internal/adminui/dist

# Build arguments for versioning
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the binary with optimizations and version information
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -extldflags '-static' -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -a -installsuffix cgo \
    -o fluxbase \
    ./cmd/fluxbase

# Stage 3: Production Runtime Image
FROM alpine:3.19

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

LABEL maintainer="Fluxbase Team" \
      description="Fluxbase - Production-Ready Backend-as-a-Service" \
      version="${VERSION}" \
      commit="${COMMIT}" \
      build-date="${BUILD_DATE}"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    postgresql-client \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 -S fluxbase && \
    adduser -u 1000 -S fluxbase -G fluxbase

WORKDIR /app

# Copy binary from builder
COPY --from=go-builder /build/fluxbase /app/fluxbase

# Copy migrations (embedded in binary but also available as files)
COPY --from=go-builder /build/internal/database/migrations /app/migrations

# Copy example configuration from builder stage
COPY --from=go-builder /build/fluxbase.yaml.example /app/fluxbase.yaml.example

# Create necessary directories
RUN mkdir -p /app/storage /app/config /app/data /app/logs && \
    chown -R fluxbase:fluxbase /app

# Switch to non-root user
USER fluxbase

# Expose HTTP port
EXPOSE 8080

# Health check using the /health endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Environment variables with production defaults
ENV FLUXBASE_SERVER_ADDRESS=:8080 \
    FLUXBASE_DEBUG=false \
    FLUXBASE_LOG_LEVEL=info \
    FLUXBASE_DATABASE_MAX_CONNECTIONS=25 \
    FLUXBASE_DATABASE_MIN_CONNECTIONS=5

# Volume mounts for persistent data
VOLUME ["/app/storage", "/app/config", "/app/logs"]

# Run the application
ENTRYPOINT ["/app/fluxbase"]
