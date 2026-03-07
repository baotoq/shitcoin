# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY internal/ internal/

# Build static binary with stripped debug symbols
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app cmd/shitcoin/main.go

# Runtime stage
FROM alpine:3.21

# Create non-root user
RUN addgroup -g 1001 -S appgroup && adduser -S appuser -u 1001 -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=appuser:appgroup /app ./app

# Copy config from build context
COPY --chown=appuser:appgroup etc/shitcoin.yaml etc/shitcoin.yaml

# Create data directory for runtime BoltDB storage
RUN mkdir -p /app/data && chown appuser:appgroup /app/data

USER appuser

EXPOSE 8080

CMD ["./app", "-f", "etc/shitcoin.yaml", "startnode"]
