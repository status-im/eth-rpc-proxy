import argparse
import json
from collections import defaultdict

INFURA = "infura"
GROVE = "grove"
NODEFLEET = "nodefleet"

NETWORK_DATA = [
    {
        "chain": "ethereum",
        "network": "mainnet",
        "chainId": 1,
        "providers": {
            INFURA: "https://mainnet.infura.io/v3/",
            GROVE: "https://eth.rpc.grove.city/v1/",
            NODEFLEET: "https://eth-mainnet.alphafleet.io/"
        }
    },
    {
        "chain": "ethereum",
        "network": "sepolia",
        "chainId": 11155111,
        "providers": {
            INFURA: "https://sepolia.infura.io/v3/",
            GROVE: "https://eth-sepolia-testnet.rpc.grove.city/v1/",
            NODEFLEET: "https://eth-sepolia.alphafleet.io/"
        }
    },
    {
        "chain": "optimism",
        "network": "mainnet",
        "chainId": 10,
        "providers": {
            INFURA: "https://optimism-mainnet.infura.io/v3/",
            GROVE: "https://optimism.rpc.grove.city/v1/",
            NODEFLEET: "https://optimism-mainnet.alphafleet.io/"
        }
    },
    {
        "chain": "optimism",
        "network": "sepolia",
        "chainId": 11155420,
        "providers": {
            INFURA: "https://optimism-sepolia.infura.io/v3/",
            GROVE: "https://optimism-sepolia-testnet.rpc.grove.city/v1/",
            NODEFLEET: "https://optimism-sepolia.alphafleet.io/"
        }
    },
    {
        "chain": "arbitrum",
        "network": "mainnet",
        "chainId": 42161,
        "providers": {
            INFURA: "https://arbitrum-mainnet.infura.io/v3/",
            GROVE: "https://arbitrum-one.rpc.grove.city/v1/",
            NODEFLEET: "https://arb-mainnet.alphafleet.io/"
        }
    },
    {
        "chain": "arbitrum",
        "network": "sepolia",
        "chainId": 421614,
        "providers": {
            INFURA: "https://arbitrum-sepolia.infura.io/v3/",
            GROVE: "https://arbitrum-sepolia-testnet.rpc.grove.city/v1/",
            NODEFLEET: "https://arb-sepolia.alphafleet.io/"
        }
    },
    {
        "chain": "base",
        "network": "mainnet",
        "chainId": 8453,
        "providers": {
            INFURA: "https://base-mainnet.infura.io/v3/",
            GROVE: "https://base.rpc.grove.city/v1/",
            NODEFLEET: "https://base-mainnet.alphafleet.io/"
        }
    },
    {
        "chain": "base",
        "network": "sepolia",
        "chainId": 84532,
        "providers": {
            INFURA: "https://base-sepolia.infura.io/v3/",
            GROVE: "https://base-testnet.rpc.grove.city/v1/",
            NODEFLEET: "https://base-sepolia.alphafleet.io/"
        }
    }
]

def parse_provider_spec(provider_spec):
    """Parse provider specification into provider type and auth details."""
    parts = provider_spec.split(":", 2)
    
    if len(parts) < 2:
        raise ValueError(f"Invalid provider format: {provider_spec}. Use provider:token or provider:username:password format")
    
    provider_type = parts[0]
    
    if len(parts) == 2:
        # Token auth format: provider:token
        return {
            "type": provider_type,
            "auth_type": "token-auth",
            "auth_token": parts[1],
            "auth_login": "",
            "auth_password": ""
        }
    else:
        # Basic auth format: provider:username:password
        return {
            "type": provider_type,
            "auth_type": "basic-auth",
            "auth_token": "",
            "auth_login": parts[1],
            "auth_password": parts[2]
        }

def generate_providers(providers, networks, chains, single_provider=False):
    output = {"chains": []}
    
    for network_data in NETWORK_DATA:
        if network_data["chain"] not in chains or network_data["network"] not in networks:
            continue
            
        if single_provider:
            # Use the first provider only
            provider_spec = parse_provider_spec(providers[0])
            p_type = provider_spec["type"]
            
            if p_type not in network_data["providers"]:
                continue
                
            chain_entry = {
                "name": network_data['chain'],
                "network": network_data["network"],
                "chainId": network_data["chainId"],
                "provider": {
                    "name": p_type.capitalize(),
                    "url": network_data["providers"][p_type],
                    "authType": provider_spec["auth_type"],
                    "authToken": provider_spec["auth_token"],
                    "authLogin": provider_spec["auth_login"],
                    "authPassword": provider_spec["auth_password"]
                }
            }
        else:
            # Original multiple providers format
            chain_entry = {
                "name": network_data["chain"],
                "network": network_data["network"],
                "chainId": network_data["chainId"],
                "providers": []
            }
            
            provider_counts = defaultdict(int)
            
            for provider_spec_str in providers:
                provider_spec = parse_provider_spec(provider_spec_str)
                p_type = provider_spec["type"]
                
                if p_type not in network_data["providers"]:
                    continue
                    
                provider_counts[p_type] += 1
                count = provider_counts[p_type]
                
                chain_entry["providers"].append({
                    "name": f"{p_type.capitalize()}{count}",
                    "url": network_data["providers"][p_type],
                    "authType": provider_spec["auth_type"],
                    "authToken": provider_spec["auth_token"],
                    "authLogin": provider_spec["auth_login"],
                    "authPassword": provider_spec["auth_password"]
                })
        
        if (single_provider and "provider" in chain_entry) or (not single_provider and chain_entry["providers"]):
            output["chains"].append(chain_entry)
    
    return output

def main():
    parser = argparse.ArgumentParser(description="Generate providers.json configuration")
    parser.add_argument("--providers", nargs="+", required=True,
                        help="Provider tokens in format provider:token for token auth or provider:username:password for basic auth (e.g. infura:abc123 or grove:user:pass)")
    parser.add_argument("--networks", nargs="+", required=True,
                        choices=["mainnet", "sepolia"],
                        help="Networks to generate configs for")
    parser.add_argument("--chains", nargs="+", required=True,
                        choices=["ethereum", "optimism", "arbitrum", "base"],
                        help="Chains to generate configs for")
    parser.add_argument("--output", "-o", default="generated_providers.json",
                        help="Output file path")
    parser.add_argument("--single-provider", action="store_true",
                        help="Generate config with single provider per chain")
    
    args = parser.parse_args()
    
    config = generate_providers(args.providers, args.networks, args.chains, args.single_provider)
    
    with open(args.output, "w") as f:
        json.dump(config, f, indent=2)
    
    print(f"Successfully generated {args.output}")

if __name__ == "__main__":
    main()
