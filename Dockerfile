# ===== Build stage =====
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy mod files dulu supaya caching lebih efisien
COPY go.mod go.sum ./
RUN go mod download

# Copy sisa kode
COPY . .

# Build binary API
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/api .

# ===== Run stage =====
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary dari builder
COPY --from=builder /app/bin/api /app/api

# Copy folder migrations supaya bisa dipakai kalau butuh
COPY --from=builder /app/migrations /app/migrations

EXPOSE 3000

CMD ["/app/api"]