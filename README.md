# Ethereum RPC Proxy System

The Ethereum RPC Proxy System provides a robust solution for managing and monitoring Ethereum RPC providers. It consists of two main components:

1. **RPC Health Checker**: Monitors and validates RPC provider health
2. **nginx-proxy**: Acts as a reverse proxy with provider failover capabilities

## Running Components Individually

Run the complete system:
```bash
docker-compose up 
```

The services will be available at:
- RPC Health Checker: http://localhost:8080
- nginx-proxy: http://localhost:8081


## Sub projects

- [RPC Health Checker](rpc-health-checker/README.md)
- [nginx-proxy](nginx-proxy/README.md)
