# Frontend build stage
FROM node:20-alpine AS frontend-builder
WORKDIR /app/internal/adapters/primary/telegram/webapp
COPY internal/adapters/primary/telegram/webapp/package*.json ./
RUN npm install
COPY internal/adapters/primary/telegram/webapp/ ./
RUN npm run build

# Backend build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy built frontend assets so go:embed can package them
COPY --from=frontend-builder /app/internal/adapters/primary/telegram/webapp/dist ./internal/adapters/primary/telegram/webapp/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o finance-app ./cmd/finance-app/main.go

# Run stage
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata ledger
WORKDIR /app
RUN adduser -D financeuser
RUN mkdir -p /app/ledger /app/config && \
    chown -R financeuser:financeuser /app
USER financeuser

COPY --from=builder /app/finance-app .
COPY config/templates ./config

# Default Env Vars (can be overridden)
ENV LEDGER_ROOT=/app/ledger
ENV LEDGER_FILE=main.ledger
ENV CONFIG_ROOT=/app/config
ENV HTTP_PORT=8080

EXPOSE 8080

VOLUME ["/app/ledger", "/app/config"]
ENTRYPOINT ["./finance-app"]
