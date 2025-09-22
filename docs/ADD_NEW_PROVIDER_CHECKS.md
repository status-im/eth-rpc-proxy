# Adding New RPC Provider Checks

This guide explains how to add new RPC method checks to the `test_methods.json` file.

## Overview

The `test_methods.json` file contains a list of RPC methods that will be tested against different providers. Each method entry includes:
- The RPC method name
- Parameters for the method call
- Maximum allowed difference in results between providers

The validation process works as follows:
1. For each RPC method, the system makes parallel requests to:
   - The provider being tested (from `default_providers.json`)
   - A reference provider (specified in `reference_providers.json`)
2. The responses from both providers are compared
3. If the difference between results exceeds the specified `maxDifference`:
   - The provider is temporarily excluded from the list of valid providers
   - The provider will not be used by the nginx proxy until the next successful validation
4. The list of valid providers is continuously updated and used by the nginx proxy for routing requests

This ensures that only reliable and consistent providers are used for serving RPC requests.

## File Structure

```json
[
  {
    "method": "eth_blockNumber",
    "params": [],
    "maxDifference": "4"
  },
  {
    "method": "eth_getBalance",
    "params": [
      "0x9B27B66D4de4e839326b98108d978526a18E95a3",
      "latest"
    ],
    "maxDifference": "0"
  }
]
```

## Adding a New Check

1. Add a new entry to the JSON array with the following structure:
```json
{
  "method": "eth_yourMethod",
  "params": [
    // Your method parameters here
  ],
  "maxDifference": "0"
}
```

2. Required fields:
   - `method`: The RPC method name (e.g., "eth_getBalance")
   - `params`: Array of parameters for the method call
   - `maxDifference`: Maximum allowed difference in results (as a string)
