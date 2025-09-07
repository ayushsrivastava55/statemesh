# Multi-stage Dockerfile for Cosmos State Mesh

# Build stage
FROM golang:1.21-alpine AS builder

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

# Build the application
RUN make build

# API server stage
FROM alpine:3.18 AS api

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl

# Create non-root user
RUN adduser -D -s /bin/sh statemesh

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/state-mesh /app/state-mesh

# Copy configuration template
COPY --from=builder /app/config /app/config

# Change ownership
RUN chown -R statemesh:statemesh /app

# Switch to non-root user
USER statemesh

# Expose ports
EXPOSE 8080 8081 8082

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8082/health || exit 1

# Default command
ENTRYPOINT ["/app/state-mesh"]
CMD ["serve"]

# Ingester stage
FROM alpine:3.18 AS ingester

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl

# Create non-root user
RUN adduser -D -s /bin/sh statemesh

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/state-mesh /app/state-mesh

# Copy configuration template
COPY --from=builder /app/config /app/config

# Change ownership
RUN chown -R statemesh:statemesh /app

# Switch to non-root user
USER statemesh

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8082/health || exit 1

# Default command
ENTRYPOINT ["/app/state-mesh"]
CMD ["ingest"]
