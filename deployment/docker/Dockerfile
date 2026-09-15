# -----------------------------
# Builder
# -----------------------------
FROM golang:1.26.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    go build \
    -ldflags="-s -w" \
    -o car_service \
    ./cmd/car_service

# -----------------------------
# Runtime
# -----------------------------
FROM alpine:3.22

LABEL org.opencontainers.image.title="Car Service"
LABEL org.opencontainers.image.description="Cloud-native REST API written in Go"
LABEL org.opencontainers.image.source="https://github.com/InatoInato/car_service"
LABEL org.opencontainers.image.licenses="MIT"

RUN apk add --no-cache ca-certificates wget && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder \
    --chown=appuser:appgroup \
    /app/car_service \
    /app/car_service

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s \
            --timeout=5s \
            --start-period=10s \
            --retries=3 \
    CMD wget --quiet -O /dev/null http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/car_service"]
