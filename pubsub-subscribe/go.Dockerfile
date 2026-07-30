FROM golang:1.26-alpine

RUN apk add --no-cache ca-certificates curl bash

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY pubsub_listener.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o pubsub_listener .

ENTRYPOINT ["/app/pubsub_listener"]
