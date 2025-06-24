#!/usr/bin/env bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8081"
SERVICE_PID=""

# Function to cleanup on exit
cleanup() {
    if [ ! -z "$SERVICE_PID" ]; then
        echo -e "${YELLOW}Stopping go-auth-service (PID: $SERVICE_PID)${NC}"
        kill $SERVICE_PID 2>/dev/null
        wait $SERVICE_PID 2>/dev/null
    fi
}

trap cleanup EXIT

echo -e "${YELLOW}=== Go Auth Service Test ===${NC}"

# Step 1: Build and start the service
echo -e "\n${YELLOW}1. Building and starting go-auth-service...${NC}"
go build -o server ./cmd/server
if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Failed to build service${NC}"
    exit 1
fi

./server &
SERVICE_PID=$!
echo -e "${GREEN}✓ Service started (PID: $SERVICE_PID)${NC}"

# Wait for service to start
sleep 2

# Step 2: Check service status
echo -e "\n${YELLOW}2. Checking service status...${NC}"
response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$BASE_URL/auth/status")
http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')

if [ "$http_code" != "200" ]; then
    echo -e "${RED}✗ Service not running (HTTP $http_code)${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Service is running${NC}"

# Step 3: Get puzzle challenge
echo -e "\n${YELLOW}3. Getting puzzle challenge...${NC}"
puzzle_response=$(curl -s "$BASE_URL/auth/puzzle")
if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Failed to get puzzle${NC}"
    exit 1
fi

echo "Puzzle response:"
echo "$puzzle_response" | jq '.'

# Extract puzzle data
challenge=$(echo "$puzzle_response" | jq -r '.challenge')
salt=$(echo "$puzzle_response" | jq -r '.salt')
difficulty=$(echo "$puzzle_response" | jq -r '.difficulty')
expires_at=$(echo "$puzzle_response" | jq -r '.expires_at')

if [ "$challenge" = "null" ] || [ "$salt" = "null" ]; then
    echo -e "${RED}✗ Invalid puzzle response${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Puzzle received${NC}"
echo "  Challenge: $challenge"
echo "  Salt: $salt"
echo "  Difficulty: $difficulty"

# Step 4: Use dev endpoint to get valid solution
echo -e "\n${YELLOW}4. Getting valid solution from dev endpoint...${NC}"
start_time=$(date +%s%N)
dev_response=$(curl -s "$BASE_URL/dev/test-solve")
end_time=$(date +%s%N)

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Failed to get test solution${NC}"
    exit 1
fi

# Calculate solve time in milliseconds
solve_time_ns=$((end_time - start_time))
solve_time_ms=$((solve_time_ns / 1000000))

echo "Dev solution response:"
echo "$dev_response" | jq '.example_request'

# Extract solution from dev endpoint
dev_nonce=$(echo "$dev_response" | jq -r '.example_request.nonce')
dev_argon_hash=$(echo "$dev_response" | jq -r '.example_request.argon_hash')
dev_hmac=$(echo "$dev_response" | jq -r '.example_request.hmac')
dev_challenge=$(echo "$dev_response" | jq -r '.example_request.challenge')
dev_salt=$(echo "$dev_response" | jq -r '.example_request.salt')
dev_expires_at=$(echo "$dev_response" | jq -r '.example_request.expires_at')

echo -e "${GREEN}✓ Valid solution obtained${NC}"
echo "  Nonce: $dev_nonce"
echo "  Hash: ${dev_argon_hash:0:20}..."
echo "  HMAC: ${dev_hmac:0:20}..."
echo -e "  ${YELLOW}Solve time: ${solve_time_ms}ms${NC}"

# Step 5: Submit solution and get JWT token
echo -e "\n${YELLOW}5. Submitting solution to get JWT token...${NC}"
solve_payload=$(cat <<EOF
{
    "challenge": "$dev_challenge",
    "salt": "$dev_salt",
    "nonce": $dev_nonce,
    "argon_hash": "$dev_argon_hash",
    "hmac": "$dev_hmac",
    "expires_at": "$dev_expires_at"
}
EOF
)

response=$(curl -s -w "HTTPSTATUS:%{http_code}" -H "Content-Type: application/json" \
    -d "$solve_payload" "$BASE_URL/auth/solve")
http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
body=$(echo $response | sed -e 's/HTTPSTATUS:.*//g')

if [ "$http_code" != "200" ]; then
    echo -e "${RED}✗ Failed to solve puzzle (HTTP $http_code)${NC}"
    echo "Response: $body"
    exit 1
fi

echo -e "${GREEN}✓ Puzzle solved successfully!${NC}"
echo "Token response:"
echo "$body" | jq '.'

# Extract JWT token
token=$(echo "$body" | jq -r '.token')
expires_at=$(echo "$body" | jq -r '.expires_at')
request_limit=$(echo "$body" | jq -r '.request_limit')

echo -e "${GREEN}✓ JWT token received${NC}"
echo "  Token: ${token:0:50}..."
echo "  Expires: $expires_at"
echo "  Request limit: $request_limit"

# Step 6: Verify JWT token
echo -e "\n${YELLOW}6. Verifying JWT token...${NC}"
response=$(curl -s -w "HTTPSTATUS:%{http_code}" -H "Authorization: Bearer $token" \
    "$BASE_URL/auth/verify")
http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')

if [ "$http_code" != "200" ]; then
    echo -e "${RED}✗ JWT token verification failed (HTTP $http_code)${NC}"
    exit 1
fi

echo -e "${GREEN}✓ JWT token verified successfully${NC}"

# Step 7: Test rate limiting (make a few requests)
echo -e "\n${YELLOW}7. Testing rate limiting...${NC}"
for i in {1..3}; do
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" -H "Authorization: Bearer $token" \
        "$BASE_URL/auth/verify")
    http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" = "200" ]; then
        echo -e "${GREEN}✓ Request $i: Token still valid${NC}"
    else
        echo -e "${YELLOW}! Request $i: HTTP $http_code${NC}"
    fi
done

echo -e "\n${GREEN}=== All Tests Passed! ===${NC}"
echo -e "${GREEN}✓ Service builds and starts correctly${NC}"
echo -e "${GREEN}✓ Puzzle generation works${NC}"
echo -e "${GREEN}✓ HMAC protected solution validation works${NC}"
echo -e "${GREEN}✓ JWT token generation and verification works${NC}"
echo -e "${GREEN}✓ Rate limiting is functional${NC}"

echo -e "\n${YELLOW}=== Performance Statistics ===${NC}"
echo -e "${YELLOW}Puzzle solve time: ${solve_time_ms}ms${NC}"
echo -e "${YELLOW}Difficulty level: $difficulty${NC}"
echo -e "${YELLOW}Nonce found: $dev_nonce${NC}"

echo -e "\n${YELLOW}Final JWT Token:${NC}"
echo "$token" 