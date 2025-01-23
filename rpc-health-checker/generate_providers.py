import argparse
import json
from collections import defaultdict

INFURA = "infura"
GROVE = "grove"

NETWORK_DATA = [
    {
        "chain": "ethereum",
        "network": "mainnet",
        "chainId": 1,
        "providers": {
            INFURA: "https://mainnet.infura.io/v3/",
            GROVE: "https://eth.rpc.grove.city/v1/"
        }
    },
    {
        "chain": "ethereum",
        "network": "sepolia",
        "chainId": 11155111,
        "providers": {
            INFURA: "https://sepolia.infura.io/v3/",
            GROVE: "https://eth-sepolia-testnet.rpc.grove.city/v1/"
        }
    },
    {
        "chain": "optimism",
        "network": "mainnet",
        "chainId": 10,
        "providers": {
            INFURA: "https://optimism-mainnet.infura.io/v3/",
            GROVE: "https://optimism.rpc.grove.city/v1/"
        }
    },
    {
        "chain": "optimism",
        "network": "sepolia",
        "chainId": 11155420,
        "providers": {
            INFURA: "https://optimism-sepolia.infura.io/v3/",
            GROVE: "https://optimism-sepolia-testnet.rpc.grove.city/v1/"
        }
    },
    {
        "chain": "arbitrum",
        "network": "mainnet",
        "chainId": 42161,
        "providers": {
            INFURA: "https://arbitrum-mainnet.infura.io/v3/",
            GROVE: "https://arbitrum-one.rpc.grove.city/v1/"
        }
    },
    {
        "chain": "arbitrum",
        "network": "sepolia",
        "chainId": 421614,
        "providers": {
            INFURA: "https://arbitrum-sepolia.infura.io/v3/",
            GROVE: "https://arbitrum-sepolia-testnet.rpc.grove.city/v1/"
        }
    },
    {
        "chain": "base",
        "network": "mainnet",
        "chainId": 8453,
        "providers": {
            INFURA: "https://base-mainnet.infura.io/v3/",
            GROVE: "https://base.rpc.grove.city/v1/"
        }
    },
    {
        "chain": "base",
        "network": "sepolia",
        "chainId": 84532,
        "providers": {
            INFURA: "https://base-sepolia.infura.io/v3/",
            GROVE: "https://base-testnet.rpc.grove.city/v1/"
        }
    }
]

def generate_providers(providers, networks, chains):
    output = {"chains": []}
    
    for network_data in NETWORK_DATA:
        if network_data["chain"] not in chains or network_data["network"] not in networks:
            continue
            
        chain_entry = {
            "name": network_data["chain"].capitalize(),
            "network": network_data["network"],
            "chainId": network_data["chainId"],
            "providers": []
        }
        
        provider_counts = defaultdict(int)
        
        for provider_spec in providers:
            p_type, _, p_token = provider_spec.partition(":")
            
            # Get the appropriate URL from providers map
            if p_type not in network_data["providers"]:
                continue
                
            provider_counts[p_type] += 1
            count = provider_counts[p_type]
            
            chain_entry["providers"].append({
                "name": f"{p_type.capitalize()}{count}",
                "url": network_data["providers"][p_type],
                "authType": "token-auth",
                "authToken": p_token
            })
        
        if chain_entry["providers"]:
            output["chains"].append(chain_entry)
    
    return output

def main():
    parser = argparse.ArgumentParser(description="Generate providers.json configuration")
    parser.add_argument("--providers", nargs="+", required=True,
                        help="Provider tokens in format provider:token (e.g. infura:abc123)")
    parser.add_argument("--networks", nargs="+", required=True,
                        choices=["mainnet", "sepolia"],
                        help="Networks to generate configs for")
    parser.add_argument("--chains", nargs="+", required=True,
                        choices=["ethereum", "optimism", "arbitrum", "base"],
                        help="Chains to generate configs for")
    parser.add_argument("--output", "-o", default="generated_providers.json",
                        help="Output file path")
    
    args = parser.parse_args()
    
    # Validate provider format
    for provider in args.providers:
        if ":" not in provider:
            raise ValueError(f"Invalid provider format: {provider}. Use provider:token format")
    
    config = generate_providers(args.providers, args.networks, args.chains)
    
    with open(args.output, "w") as f:
        json.dump(config, f, indent=2)
    
    print(f"Successfully generated {args.output}")

if __name__ == "__main__":
    main()
