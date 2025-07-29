# Ethereum RPC Proxy 

[![Tests](https://github.com/status-im/eth-rpc-proxy/actions/workflows/test.yml/badge.svg)](https://github.com/status-im/eth-rpc-proxy/actions/workflows/test.yml)

The Ethereum RPC Proxy System provides a robust solution for managing and monitoring Ethereum RPC providers. It consists of four main components:
1. **RPC Health Checker**: Monitors and validates RPC provider health
2. **nginx-proxy**: Acts as a reverse proxy with provider failover and three-level caching
3. **KeyDB**: Persistent L3 cache service for distributed caching across instances
4. **Auth Service**: Handles JWT-based authentication with puzzle solving

## Running on the local machine

Run the complete system:

1. Generate provider configuration files and store them in the secrets folder:
   ```bash
   # Create secrets directory if it doesn't exist
   mkdir -p secrets
   
   # Generate default_providers.json with different authentication methods
   python3 rpc-health-checker/generate_providers.py \
     --providers infura:YOUR_INFURA_TOKEN grove:YOUR_GROVE_TOKEN alchemy:YOUR_ALCHEMY_TOKEN status_network \
     --networks mainnet sepolia \
     --chains ethereum optimism arbitrum base status \
     --output secrets/default_providers.json
   
   # Or use mix of token, basic auth, and no-auth providers
   python3 rpc-health-checker/generate_providers.py \
     --providers infura:YOUR_INFURA_TOKEN grove:username:password alchemy:YOUR_ALCHEMY_TOKEN status_network \
     --networks mainnet sepolia \
     --chains ethereum optimism arbitrum base status \
     --output secrets/default_providers.json
     
   # Generate reference_providers.json with single provider per chain
   python3 rpc-health-checker/generate_providers.py \
     --single-provider \
     --providers infura:YOUR_INFURA_TOKEN_REFERENCE alchemy:YOUR_ALCHEMY_TOKEN status_network \
     --networks mainnet sepolia \
     --chains ethereum optimism arbitrum base status \
     --output secrets/reference_providers.json
    ``` 
   Please replace:
   - `YOUR_INFURA_TOKEN` and `YOUR_INFURA_TOKEN_REFERENCE` with your Infura API tokens
   - `YOUR_GROVE_TOKEN` with your Grove API token, or use `username:password` for basic auth
   - `YOUR_ALCHEMY_TOKEN` with your Alchemy API token
   - Other provider credentials as needed

   **Note**: `--providers` accepts multiple providers with different authentication methods:
   - No auth format: just provider name (e.g., `status_network`)
   - Token auth format: `provider:token` (e.g., `infura:abc123`, `alchemy:xyz789`)
   - Basic auth format: `provider:username:password` (e.g., `grove:user:pass`)
   - Example: `--providers infura:TOKEN1 grove:user:pass status_network nodefleet:TOKEN2 alchemy:TOKEN3`

2. Create .htpasswd file for nginx proxy authentication:
   ```bash
   htpasswd -c secrets/.htpasswd dev
   ```
3. Execute the following commands to start the system:
    ```bash
    docker-compose -f docker-compose-local.yml up -d --build
    ```
4. Run test requests:
    ```bash
    # For Ethereum mainnet using any available provider (with automatic failover between all providers)
    curl -u dev:<password> -X POST http://localhost:8080/ethereum/mainnet \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

    # For Status Network Sepolia using specific provider type
    curl -u dev:<password> -X POST http://localhost:8080/status/sepolia/status_network \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

    # For Ethereum mainnet using Infura providers (with automatic failover between all Infura instances)
    curl -u dev:<password> -X POST http://localhost:8080/ethereum/mainnet/infura \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
    ```

The services will be accessible under:
- RPC Health Checker: http://localhost:8081
  - Check the list of validated providers at http://localhost:8081/providers
- nginx-proxy: http://localhost:8080
  - The RPC endpoint is available in two formats:
    - `http://localhost:8080/{chain}/{network}` - uses any available provider with failover across all providers
    - `http://localhost:8080/{chain}/{network}/{provider_type}` - uses providers of the specified type with failover between instances of that type
  - Features **three-level cache architecture** (L1/L2/L3) for optimized performance
  - Example paths: 
    - `/ethereum/mainnet` - tries all available providers until success
    - `/ethereum/mainnet/infura` - tries all available Infura providers until success
    - `/status/sepolia/status_network` - tries all available Status Network providers until success
- KeyDB: Persistent L3 cache service (Redis-compatible)
  - Provides distributed caching across proxy instances
- Prometheus: http://localhost:9090
  - Metrics and monitoring interface
- Grafana: http://localhost:3000 (default credentials: admin/admin)
  - Visualization dashboards for RPC metrics and health status

![grafana.png](grafana.png)

## Sub projects

- [RPC Health Checker](rpc-health-checker/README.md)
- [nginx-proxy](nginx-proxy/README.md)
