# Build stage
FROM golang:1.24-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o cnap \
    ./cmd/cnap

# Runtime stage
FROM alpine:3.19

# Install PostgreSQL and other dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    postgresql \
    postgresql-contrib \
    bash \
    su-exec

# Create directories
RUN mkdir -p /var/lib/postgresql/data /app /run/postgresql && \
    chown -R postgres:postgres /var/lib/postgresql /run/postgresql

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/cnap .
COPY --from=builder /build/configs ./configs

# Copy startup script
COPY docker/start.sh /start.sh
RUN chmod +x /start.sh

# Expose PostgreSQL port
EXPOSE 5432

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/app/cnap", "health"]

# Run the startup script
ENTRYPOINT ["/start.sh"]
