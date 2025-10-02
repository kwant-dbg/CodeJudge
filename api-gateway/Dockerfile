FROM golang:1.19-alpine AS builder

WORKDIR /src
COPY go.work ./
COPY common-go/go.mod ./common-go/
COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/
RUN go work sync

COPY common-go ./common-go
COPY api-gateway ./api-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -C ./api-gateway -trimpath -ldflags="-s -w" -o /out/api_gateway

FROM alpine:3.16
WORKDIR /app
COPY --from=builder /out/api_gateway .
EXPOSE 8080
CMD ["./api_gateway"]

