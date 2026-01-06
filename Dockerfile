# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/indexer ./cmd/indexer
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/api ./cmd/api

# Indexer image
FROM alpine:3.19 AS indexer

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/indexer /usr/local/bin/indexer

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/indexer"]

# API image
FROM alpine:3.19 AS api

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/api /usr/local/bin/api

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/api"]
