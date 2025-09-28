FROM golang:1.19-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o plagiarism_service .

FROM alpine:3.16
WORKDIR /app
COPY --from=builder /app/plagiarism_service .
EXPOSE 8002
CMD ["./plagiarism_service"]
