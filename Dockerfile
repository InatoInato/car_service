# Builder
FROM golang:1.26.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o car_service ./cmd/car_service

# Runtime
FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /root/

COPY --from=builder /app/car_service .

CMD ["./car_service"]