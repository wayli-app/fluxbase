# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o fluxbase cmd/fluxbase/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 -S fluxbase && \
    adduser -u 1000 -S fluxbase -G fluxbase

# Set working directory
WORKDIR /home/fluxbase

# Copy binary from builder
COPY --from=builder /app/fluxbase /usr/local/bin/fluxbase

# Copy migrations
COPY --from=builder /app/internal/database/migrations ./migrations

# Create directories for storage and config
RUN mkdir -p storage config && \
    chown -R fluxbase:fluxbase storage config

# Switch to non-root user
USER fluxbase

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["fluxbase"]