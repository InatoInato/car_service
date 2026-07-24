# Builder
FROM golang:1.26.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Removed hardcoded GOARCH so Docker handles target platform automatically
# Added -ldflags="-s -w" to strip debug symbols and shrink binary size
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o car_service ./cmd/car_service

# Runtime
FROM alpine:3.22

# Install ca-certificates and create a non-root user/group
RUN apk add --no-cache ca-certificates && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/car_service .

# Run as non-root user for security
USER appuser

# Replaced missing curl with built-in wget
HEALTHCHECK --interval=30s \
            --timeout=5s \
            --start-period=10s \
            --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./car_service"]