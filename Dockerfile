# BUILD
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o producer ./cmd/producer/main.go
RUN go build -o consumer ./cmd/consumer/main.go

# RUN
FROM alpine:latest
COPY --from=builder /app/producer /usr/local/bin/producer
COPY --from=builder /app/consumer /usr/local/bin/consumer
