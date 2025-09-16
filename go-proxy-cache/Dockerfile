# Build stage
FROM golang:1.23-alpine AS builder

# Install dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rpc-cache ./cmd/rpc-cache

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/rpc-cache .

# Create config directory
RUN mkdir -p /app

# Copy config files
COPY cache_config.yaml cache_rules.yaml /app/

# Expose port
EXPOSE 8080

# Environment variables with defaults
ENV CACHE_CONFIG_FILE=/app/cache_config.yaml
ENV CACHE_RULES_FILE=/app/cache_rules.yaml
ENV CACHE_SERVER_ADDR=:8080

# Run the binary
CMD ["./rpc-cache"]

