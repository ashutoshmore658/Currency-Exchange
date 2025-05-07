# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o exchange-rate-service ./cmd/currencyexchangeserver/main.go

# Stage 2: Minimal image
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/exchange-rate-service .
COPY --from=builder /app/cmd/currencyexchangeserver/banner.txt ./cmd/currencyexchangeserver/banner.txt

EXPOSE 8080

ENV SERVER_PORT=8080
ENV REDIS_ADDR=redis:6379
ENV REDIS_PASSWORD=
ENV REDIS_DB=0
ENV LATEST_RATE_CACHE_TTL=55m
ENV HISTORICAL_CACHE_TTL=24h
ENV REFRESH_INTERVAL=1h
ENV HISTORY_DAYS_LIMIT=90

ENTRYPOINT ["./exchange-rate-service"]
