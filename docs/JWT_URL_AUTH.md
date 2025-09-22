# JWT Token Authentication via URL Query Parameters

This document describes the enhanced JWT authentication functionality that allows passing JWT tokens via URL query parameters in addition to the traditional Authorization header method.

## Overview

The proxy now supports multiple methods for JWT token authentication:

1. **Authorization Header** (existing method)
2. **URL Query Parameters** (new method)

Both methods can be used interchangeably and provide the same level of security and functionality.

## Supported Authentication Methods

### 1. Authorization Header (Existing)

```bash
curl -X POST "http://proxy-url/ethereum/mainnet" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

### 2. URL Query Parameters (New)

The following query parameter names are supported for JWT token authentication:

#### Using `token` parameter:
```bash
curl -X POST "http://proxy-url/ethereum/mainnet?token=YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

#### Using `jwt` parameter:
```bash
curl -X POST "http://proxy-url/ethereum/mainnet?jwt=YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

#### Using `access_token` parameter:
```bash
curl -X POST "http://proxy-url/ethereum/mainnet?access_token=YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

## Security Features

### Automatic Parameter Sanitization

When JWT tokens are passed via query parameters, the proxy automatically:

1. **Extracts** the JWT token from the query parameters
2. **Validates** the token using the same validation logic as header-based authentication
3. **Removes all query parameters** from the request before forwarding to blockchain providers (since no other query parameters are expected for RPC calls)

Example:
```bash
# Original request
GET /ethereum/mainnet?token=jwt123

# Forwarded to provider (all query params removed)
GET /ethereum/mainnet
```

### Token Priority

If both Authorization header and query parameters are present, the system follows this priority:

1. **Authorization Header** - checked first
2. **Query Parameters** - checked if no valid Authorization header found
   - `token` parameter
   - `jwt` parameter  
   - `access_token` parameter

## Getting JWT Tokens

To obtain JWT tokens for authentication, use the provided script:

```bash
./go-auth-service/get_proxy_token.sh [environment]
```

Available environments:
- `local` (default) - http://localhost:8081
- `test` - https://test.eth-rpc.status.im  
- `prod` - https://prod.eth-rpc.status.im

Example:
```bash
# Get token for local development
./go-auth-service/get_proxy_token.sh local

# Get token for test environment
./go-auth-service/get_proxy_token.sh test
```

## Rate Limiting

Rate limiting works identically for both authentication methods:
- Same token = same rate limit counter
- Rate limits are applied per token, regardless of how the token was provided
- Headers `X-RateLimit-Limit` and `X-RateLimit-Remaining` are set for both methods

## Caching

JWT token validation caching works the same for both methods:
- Tokens are cached after successful validation
- Cache keys are based on the token value, not the source method
- `X-Cache-Status` header indicates cache hit/miss status

## Testing

Use the provided test script to verify functionality:

```bash
./test_jwt_url_auth.sh
```

The test script verifies:
- Authorization header method still works
- All query parameter methods work
- Parameter sanitization works correctly
- Requests without authentication are properly rejected

## Error Handling

Authentication errors are handled identically for both methods:
- `401 Unauthorized` for invalid or missing tokens
- `429 Too Many Requests` for rate limit violations
- Same error response format and status codes
