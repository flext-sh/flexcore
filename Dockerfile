# FlexCore Development Dockerfile
# For production, use: deployments/docker/production/Dockerfile.production
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o flexcore ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates curl

WORKDIR /app

COPY --from=builder /app/flexcore .

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./flexcore"]