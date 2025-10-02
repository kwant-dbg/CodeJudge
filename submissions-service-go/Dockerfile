FROM golang:1.19-alpine AS builder

WORKDIR /src
COPY go.work ./
COPY common-go/go.mod ./common-go/
COPY submissions-service-go/go.mod submissions-service-go/go.sum ./submissions-service-go/
RUN go work sync

COPY common-go ./common-go
COPY submissions-service-go ./submissions-service-go
RUN CGO_ENABLED=0 GOOS=linux go build -C ./submissions-service-go -trimpath -ldflags="-s -w" -o /out/submissions_service

FROM alpine:3.16
WORKDIR /app
COPY --from=builder /out/submissions_service .
EXPOSE 8001
CMD ["./submissions_service"]

