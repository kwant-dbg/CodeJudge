# Multi-stage Dockerfile optimized for Azure deployment
# Build stage
FROM golang:1.19-alpine AS builder

# Install build dependencies
RUN apk --no-cache add git ca-certificates tzdata

WORKDIR /src

# Copy workspace files first for better layer caching
COPY go.work ./
COPY go.work.sum* ./

# Copy go.mod files for all modules to leverage Docker layer caching
COPY common-go/go.mod common-go/go.sum ./common-go/
COPY monolith-service/go.mod monolith-service/go.sum ./monolith-service/

# Download dependencies
RUN go work sync

# Copy common-go source code
COPY common-go/ ./common-go/

# Copy monolith service source code
COPY monolith-service/ ./monolith-service/

# Build the monolith service
WORKDIR /src/monolith-service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=$(date -u +%Y%m%d-%H%M%S)" \
    -o /out/codejudge-monolith \
    .

# Production stage - Use distroless for security and minimal size
FROM gcr.io/distroless/static-debian11:nonroot

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /out/codejudge-monolith /app/codejudge-monolith

# Set working directory
WORKDIR /app

# Environment variables with Azure-friendly defaults
ENV PORT=8080
ENV GIN_MODE=release

# Expose port (Azure App Service and ACI will override this)
EXPOSE 8080

# Health check for Azure Container Instance monitoring
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
    CMD ["/app/codejudge-monolith", "-health-check"] || exit 1

# Use non-root user for security
USER nonroot:nonroot

# Run the application
ENTRYPOINT ["/app/codejudge-monolith"]