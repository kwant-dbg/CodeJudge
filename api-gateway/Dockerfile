FROM golang:1.19-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o api_gateway .

FROM alpine:3.16
WORKDIR /app
COPY --from=builder /app/api_gateway .
EXPOSE 8080
CMD ["./api_gateway"]

