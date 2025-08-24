# FlexCore - Multi-stage Dockerfile
# ==================================
# Build stage and production image for FlexCore

# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o flexcore-server ./cmd/server

# Development stage
FROM builder AS development
RUN apk add --no-cache curl
EXPOSE 8080 50051 9090
CMD ["./flexcore-server"]

# Production stage
FROM alpine:3.18 AS production

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/flexcore-server .

# Create non-root user
RUN addgroup -g 1001 flexcore \
    && adduser -D -s /bin/sh -u 1001 -G flexcore flexcore

# Change ownership
RUN chown flexcore:flexcore /app/flexcore-server
USER flexcore

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

EXPOSE 8080 50051 9090

CMD ["./flexcore-server"]