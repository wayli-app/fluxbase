# Multi-Stage Dockerfile for Fluxbase
# Serves both development and production needs
#
# Usage:
#   Production (with admin UI):  docker build -t fluxbase:latest .
#   Backend only (for testing):  docker build --target go-builder -t fluxbase:backend .
#   Development:                 Use docker-compose.yml or make dev

# Stage 1: Build SDKs and Admin UI
FROM node:25-alpine AS admin-builder

WORKDIR /build

# Copy SDK packages first (admin depends on these)
COPY sdk/ ./sdk/
COPY sdk-react/ ./sdk-react/

# Build SDKs
WORKDIR /build/sdk
RUN npm ci && npm run build

# Generate embedded SDK for job and function runtime
# Create the output directories first since they don't exist in this stage
RUN mkdir -p /build/internal/jobs /build/internal/runtime && npm run generate:embedded-sdk

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

# Install build dependencies (including tesseract-dev for gosseract CGO bindings)
# g++ is required because gosseract uses C++ code via CGO
# vips-dev is required for govips image transformation support
RUN apk add --no-cache git make gcc g++ musl-dev tesseract-ocr-dev leptonica-dev vips-dev

WORKDIR /build

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Copy built admin UI from previous stage to the embed location
COPY --from=admin-builder /build/admin/dist ./internal/adminui/dist

# Copy generated embedded SDK for job and function runtime
COPY --from=admin-builder /build/internal/jobs/embedded_sdk.js ./internal/jobs/embedded_sdk.js
COPY --from=admin-builder /build/internal/runtime/embedded_sdk.js ./internal/runtime/embedded_sdk.js

# Build arguments for versioning
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the binary with optimizations and version information
# Include 'ocr' build tag to enable Tesseract OCR support
# Note: We use dynamic linking (no -static) because tesseract/leptonica don't provide static libs on Alpine
# The runtime image installs the required shared libraries (tesseract-ocr, leptonica)
RUN CGO_ENABLED=1 GOOS=linux go build \
    -tags "ocr" \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -o fluxbase-server \
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
# - ca-certificates: For HTTPS connections
# - tzdata: For timezone support
# - tesseract-ocr: For OCR text extraction from image-based PDFs
# - tesseract-ocr-data-eng: English language data for OCR
# - leptonica: Image processing library required by tesseract (dynamic linking)
# - vips: Image processing library for image transformations (dynamic linking)
# - poppler-utils: For PDF to image conversion (pdftoppm)
# - deno: JavaScript/TypeScript runtime for jobs and functions (installed via apk)
# - libc6-compat: Provides glibc compatibility for npm packages like esbuild that ship glibc binaries
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    tesseract-ocr \
    tesseract-ocr-data-eng \
    leptonica \
    vips \
    poppler-utils \
    deno \
    libc6-compat \
    && deno --version

# Create non-root user
RUN addgroup -g 1000 -S fluxbase && \
    adduser -u 1000 -S fluxbase -G fluxbase

WORKDIR /app

# Copy binary to PATH
COPY --from=go-builder /build/fluxbase-server /usr/local/bin/fluxbase-server

# Create necessary directories
RUN mkdir -p /app/storage /app/config /app/data /app/logs && \
    chown -R fluxbase:fluxbase /app

# Switch to non-root user
USER fluxbase

# Expose HTTP port
EXPOSE 8080

# Health check using wget (included in alpine by default)
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

# Environment variables with production defaults
ENV FLUXBASE_SERVER_ADDRESS=:8080 \
    FLUXBASE_DEBUG=false \
    FLUXBASE_LOG_LEVEL=info \
    FLUXBASE_DATABASE_MAX_CONNECTIONS=25 \
    FLUXBASE_DATABASE_MIN_CONNECTIONS=5

# Volume mounts for persistent data
VOLUME ["/app/storage", "/app/config", "/app/logs"]

# Run the application
ENTRYPOINT ["fluxbase-server"]
