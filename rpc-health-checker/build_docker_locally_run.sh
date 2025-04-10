#!/bin/bash

# Create Docker network (if it doesn't exist)
docker network create rpc-network || true

# Build image
docker build -t rpc-health-checker .

# Remove existing container (if present)
docker rm -f rpc-health-checker || true

# Run container in network
docker run -it --name rpc-health-checker --network rpc-network -p 8081:8080 \
  -v $(pwd)/secrets:/config \
  -e DEFAULT_PROVIDERS_PATH=/config/default_providers.json \
  -e REFERENCE_PROVIDERS_PATH=/config/reference_providers.json \
  rpc-health-checker
