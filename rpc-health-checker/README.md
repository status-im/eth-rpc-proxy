# RPC Health Checker

The RPC Health Checker is a Go-based service that periodically monitors and validates the health of Ethereum RPC providers. It compares responses from multiple providers against reference providers to ensure consistency and reliability.

## Features
- Periodic health checks of RPC providers
- Support for multiple chains and networks
- JSON-based configuration
- REST API for status validated providers
- Docker container support

## Local Development

### Running Locally

1. Clone the repository
2. Install dependencies:
```bash
go mod download
```

3. Run the service with the following command line arguments:

```bash
go run main.go [arguments]
```

#### Command Line Arguments

| Argument               | Description                                      | Default Value         |
|------------------------|--------------------------------------------------|-----------------------|
| `--checker-config`      | Path to checker configuration file              | `checker_config.json` |
| `--default-providers`   | Path to default providers JSON file             |                       |
| `--reference-providers` | Path to reference providers JSON file           |                       |

Example:
```bash
go run main.go \
  --checker-config checker_config.json \
  --default-providers default_providers.json \
  --reference-providers reference_providers.json
```

4. The service will be available at `http://localhost:8080`

### Using Docker

Build and run the container:
```bash
./build_docker_locally_run.sh
```

The container will be available at `http://localhost:8081`

## Deployment

### Docker Deployment

1. Build the Docker image:
```bash
docker build -t rpc-health-checker .
```

2. Run the container:

Example: Store configuration files with sensitive data in a `secrets` directory:
- `default_providers.json`
- `reference_providers.json`
```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/secrets:/config \
  -e DEFAULT_PROVIDERS_PATH=/config/default_providers.json \
  -e REFERENCE_PROVIDERS_PATH=/config/reference_providers.json \
  rpc-health-checker
```

## Configuration

The service uses several JSON configuration files:

### checker_config.json
```json
{
  "interval_seconds": 30,
  "default_providers_path": "default_providers.json",
  "reference_providers_path": "reference_providers.json",
  "output_providers_path": "providers.json",
  "tests_config_path": "test_methods.json",
  "logs_path": "logs"
}
```

### default_providers.json
Contains the list of providers to monitor:
```json
{
  "chains": [
    {
      "name": "ethereum",
      "network": "mainnet",
      "chainId": 1,
      "providers": [
        {
          "name": "infura",
          "url": "https://mainnet.infura.io/v3",
          "authType": "token-auth",
          "authToken": "111"
        }
      ]
    }
  ]
}
```

### reference_providers.json
Contains reference providers used for comparison:
```json
{
  "chains": [
    {
      "name": "ethereum",
      "network": "mainnet",
      "chainId": 1,
      "provider": {
          "name": "infura",
          "url": "https://mainnet.infura.io/v3",
          "authType": "token-auth",
          "authToken": "test"
      }
    }
  ]
}
```

### test_methods.json
Defines the RPC methods to test:
```json
[
  {
    "method": "eth_blockNumber",
    "params": [],
    "maxDifference": "0"
  }
]
```

## Request Flow

1. The `scheduler.Start()` triggers periodic validation based on configured interval
2. Validation cycle:
   - `runner.Run()` executes the validation prcess:
     - `validateChains()` processes each chain
       - For each chain/network:
         - For each method (`TestMultipleEVMMethods()`)
         - Run tests for each provider (`TestEVMMethodWithCaller()`)
         - Compare results against reference provider (`ValidateMultipleEVMMethods`)
       - Filter valid providers (`getValidProviders()`)
     - Write valid providers to output file (`writeValidChains()`)
3. Results are available via:
   - REST API endpoint `/providers`

## API Endpoints

- `GET /health` - Service health status
- `GET /providers` - Current provider statuses

## Environment Variables

- `PORT`: HTTP server port (default: 8080)
