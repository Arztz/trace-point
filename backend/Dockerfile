# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy Go source code (cmd, internal, configs)
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY configs/ ./configs/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server/main.go

# Runtime stage
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN adduser -D -u 1000 appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .
COPY --from=builder /app/configs ./configs

# Create data directory
RUN mkdir -p /app/data && chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# Run the application
CMD ["./server"]