# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o finance-app ./cmd/finance-app/main.go

# Run stage
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
RUN adduser -D financeuser
USER financeuser

COPY --from=builder /app/finance-app .
COPY config ./config

# Default Env Vars (can be overridden)
ENV LEDGER_ROOT=/app/ledger
ENV LEDGER_FILE=main.ledger
ENV CONFIG_ROOT=/app/config

VOLUME ["/app/data", "/app/config"]
ENTRYPOINT ["./finance-app"]
