FROM golang:1.19-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY auth-service-go/go.mod auth-service-go/go.sum ./
COPY common-go/ ../common-go/

# Download dependencies
RUN go mod download

# Copy source code
COPY auth-service-go/ ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Expose port
EXPOSE 8003

# Command to run
CMD ["./main"]