# Ethereum RPC Proxy 

The Ethereum RPC Proxy System provides a robust solution for managing and monitoring Ethereum RPC providers. It consists of two main components:
1. **RPC Health Checker**: Monitors and validates RPC provider health
2. **nginx-proxy**: Acts as a reverse proxy with provider failover capabilities

## Running on the local machine

Run the complete system:

1. Create `default_providers.json` and `reference_providers.json` with the providers you want.
2. Create `.htpasswd` with proxy credentials
3. Execute the following commands
```bash
docker-compose up --build
```

The services will be accessible under:
- RPC Health Checker: http://localhost:8080
  - Check the list of validated providers at http://localhost:8081/providers
- nginx-proxy: http://localhost:8081
  - The new RPC endpoint is now available http://localhost:8080/ethereum/mainnet (path is `/chain/network`). 


## Sub projects

- [RPC Health Checker](rpc-health-checker/README.md)
- [nginx-proxy](nginx-proxy/README.md)
