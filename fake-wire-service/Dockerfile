FROM golang:1.23-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
RUN go build -o fake-wire-service main.go

FROM alpine:latest
RUN apk --no-cache add curl ca-certificates

WORKDIR /app
COPY --from=builder /build/fake-wire-service .
COPY config.json .

EXPOSE 8080
CMD ["./fake-wire-service"]