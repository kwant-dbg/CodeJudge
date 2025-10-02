FROM golang:1.19-alpine AS builder

WORKDIR /src
COPY go.work ./
COPY common-go/go.mod ./common-go/
COPY problems-service-go/go.mod problems-service-go/go.sum ./problems-service-go/
RUN go work sync

COPY common-go ./common-go
COPY problems-service-go ./problems-service-go
RUN CGO_ENABLED=0 GOOS=linux go build -C ./problems-service-go -trimpath -ldflags="-s -w" -o /out/problems_service

FROM alpine:3.16
WORKDIR /app
COPY --from=builder /out/problems_service .
EXPOSE 8000
CMD ["./problems_service"]

