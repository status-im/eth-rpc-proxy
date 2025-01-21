# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -o /health-checker

# Runtime stage  
FROM alpine:3.18

WORKDIR /app
COPY --from=builder /health-checker /health-checker
COPY checker_config.json /app/checker_config.json
COPY test_methods.json /app/test_methods.json
COPY default_providers.json /app/default_providers.json
COPY reference_providers.json /app/reference_providers.json

EXPOSE 8080

CMD ["sh", "-c", "/health-checker --checker-config /app/checker_config.json --default-providers ${DEFAULT_PROVIDERS_PATH:-/app/default_providers.json} --reference-providers ${REFERENCE_PROVIDERS_PATH:-/app/reference_providers.json}"]
