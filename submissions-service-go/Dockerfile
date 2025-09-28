FROM golang:1.19-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o submissions_service .

FROM alpine:3.16
WORKDIR /app
COPY --from=builder /app/submissions_service .
EXPOSE 8001
CMD ["./submissions_service"]

