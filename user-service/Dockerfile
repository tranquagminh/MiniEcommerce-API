# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build with security flags
ARG VERSION=dev
ARG BUILD_DATE
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildDate=${BUILD_DATE}" \
    -a -installsuffix cgo \
    -o user-service \
    ./cmd/main.go

# Final stage - use distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy migrations with proper permissions
COPY --from=builder --chown=nonroot:nonroot /app/migrations /migrations

# Copy binary with proper permissions
COPY --from=builder --chown=nonroot:nonroot /app/user-service /user-service

# Use non-root user (defined in distroless:nonroot)
USER nonroot:nonroot

# Expose port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/user-service", "-healthcheck"]

# Set environment
ENV ENVIRONMENT=production \
    MIGRATIONS_PATH=/migrations

# Run the binary
ENTRYPOINT ["/user-service"]