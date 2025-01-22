# Create Docker network (if it doesn't exist)
docker network create rpc-network || true

# Build image
docker build -t rpc-proxy .

# Remove existing container (if exists)
docker rm -f rpc-proxy || true

# Run container
docker run -it --rm \
  --name rpc-proxy \
  --network rpc-network \
  -p 8080:8080 \
  -e CONFIG_HEALTH_CHECKER_URL=http://rpc-health-checker:8080/providers \
  rpc-proxy


#!/bin/bash

# Create Docker network (if it doesn't exist)
docker network create rpc-network || true

# Build image
docker build -t rpc-proxy ./rpc-proxy

# Remove existing container (if exists)
docker rm -f rpc-proxy || true

# Run container with environment variable
docker run -d --name rpc-proxy \
  --network rpc-network \
  -p 8080:8080 \

  rpc-proxy
