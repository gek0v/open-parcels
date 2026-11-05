# syntax=docker/dockerfile:1

# -----------------------------------------------------------------------------
# Stage 1: Builder
# -----------------------------------------------------------------------------
FROM golang:1.23-alpine AS builder

WORKDIR /src

# Install build dependencies & certificates
RUN apk add --no-cache git ca-certificates tzdata

# Cache Go modules dependencies
COPY go.mod ./
RUN go mod download || true

# Copy application source code
COPY main.go ./
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/

# Compile statically linked binary with stripped debug symbols
ARG VERSION=v1.0.0
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X open-parcels/cmd.Version=${VERSION}" \
    -o /bin/open-parcels .

# -----------------------------------------------------------------------------
# Stage 2: Final Minimal Runtime Image
# -----------------------------------------------------------------------------
FROM alpine:3.20

# Install runtime dependencies for TLS and timezone resolution
RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

# Copy compiled binary from builder stage
COPY --from=builder /bin/open-parcels /usr/local/bin/open-parcels

# Switch to unprivileged security context
USER 10001:10001
WORKDIR /home/appuser

# Expose HTTP REST API and MCP streaming endpoint port
EXPOSE 8080

# Health check against server endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Default execution: run REST API daemon and MCP server
ENTRYPOINT ["/usr/local/bin/open-parcels"]
CMD ["serve", "--port", "8080"]
