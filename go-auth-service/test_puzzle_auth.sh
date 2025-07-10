#!/usr/bin/env bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

if [ $# -eq 0 ]; then
    echo -e "${YELLOW}=== Go Puzzle Auth Tester - Build & Run ===${NC}"
    echo "Usage: $0 <proxy_server_url>"
    echo "Example: $0 https://your-proxy-server.com"
    echo "Example: $0 http://localhost:8081"
    exit 1
fi

PROXY_URL="$1"

echo -e "${YELLOW}=== Building test-puzzle-auth command ===${NC}"

# Build the command
go build -o test-puzzle-auth ./cmd/test-puzzle-auth

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ test-puzzle-auth built successfully${NC}"
echo ""

# Run the command with provided URL
echo -e "${YELLOW}=== Running test-puzzle-auth against: $PROXY_URL ===${NC}"
echo ""

./test-puzzle-auth "$PROXY_URL"

exit_code=$?

echo ""
if [ $exit_code -eq 0 ]; then
    echo -e "${GREEN}=== Test completed successfully! ===${NC}"
else
    echo -e "${RED}=== Test failed with exit code: $exit_code ===${NC}"
fi

exit $exit_code 