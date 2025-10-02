FROM golang:1.19-alpine AS builder

WORKDIR /src
COPY go.work ./
COPY common-go/go.mod ./common-go/
COPY plagiarism-service-go/go.mod plagiarism-service-go/go.sum ./plagiarism-service-go/
RUN go work sync

COPY common-go ./common-go
COPY plagiarism-service-go ./plagiarism-service-go
RUN CGO_ENABLED=0 GOOS=linux go build -C ./plagiarism-service-go -trimpath -ldflags="-s -w" -o /out/plagiarism_service

FROM alpine:3.16
WORKDIR /app
COPY --from=builder /out/plagiarism_service .
EXPOSE 8002
CMD ["./plagiarism_service"]
