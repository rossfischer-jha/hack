FROM golang:1.26-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
RUN go build -o fake-launchdarkly-service main.go

FROM alpine:latest
RUN apk --no-cache add curl ca-certificates

WORKDIR /app
COPY --from=builder /build/fake-launchdarkly-service .
COPY config.json .

EXPOSE 4000
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:4000/health || exit 1

CMD ["./fake-launchdarkly-service", "-port", "4000", "-config", "config.json"]
