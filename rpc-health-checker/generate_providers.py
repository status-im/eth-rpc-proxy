import argparse
import json
from collections import defaultdict
from network_data import NETWORK_DATA

def parse_provider_spec(provider_spec):
    """Parse provider specification into provider type and auth details."""
    parts = provider_spec.split(":", 2)

    if len(parts) == 1:
        return {
            "type": provider_spec,
            "auth_type": "no-auth",
            "auth_token": "",
            "auth_login": "",
            "auth_password": ""
        }

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

def create_provider_entry(p_type, provider_spec, network_data, count=None):
    """Create a provider entry with consistent format."""
    base_name = f"{p_type.capitalize()}-{count}" if count is not None else p_type.capitalize()
    chain_name = network_data["chain"].capitalize()
    name = f"{base_name} {chain_name}"
    return {
        "type": p_type,
        "name": name,
        "url": network_data["providers"][p_type],
        "authType": provider_spec["auth_type"],
        "authToken": provider_spec["auth_token"],
        "authLogin": provider_spec["auth_login"],
        "authPassword": provider_spec["auth_password"],
        "chainId": network_data["chainId"]
    }

def create_chain_entry(network_data):
    """Create a base chain entry with consistent format."""
    return {
        "name": network_data["chain"],
        "network": network_data["network"],
        "chainId": network_data["chainId"]
    }

def create_single_provider_entry(providers, network_data):
    """Find and create a single provider entry for the network."""
    for provider_spec_str in providers:
        provider_spec = parse_provider_spec(provider_spec_str)
        p_type = provider_spec["type"]
        
        if p_type not in network_data["providers"]:
            continue
        
        chain_entry = create_chain_entry(network_data)
        chain_entry["provider"] = create_provider_entry(p_type, provider_spec, network_data)
        return chain_entry
    
    return None

def create_multi_provider_entry(providers, network_data):
    """Create a chain entry with multiple providers."""
    chain_entry = create_chain_entry(network_data)
    chain_entry["providers"] = []
    
    provider_counts = defaultdict(int)
    
    for provider_spec_str in providers:
        provider_spec = parse_provider_spec(provider_spec_str)
        p_type = provider_spec["type"]
        
        if p_type not in network_data["providers"]:
            continue
            
        provider_counts[p_type] += 1
        count = provider_counts[p_type]
        
        chain_entry["providers"].append(
            create_provider_entry(p_type, provider_spec, network_data, count)
        )
    
    return chain_entry

def is_valid_chain_entry(chain_entry, single_provider):
    """Check if chain entry has valid providers."""
    if single_provider:
        return "provider" in chain_entry
    return bool(chain_entry.get("providers", []))

def generate_single_provider_config(providers, networks, chains):
    """Generate configuration with single provider per chain."""
    output = {"chains": []}
    
    for network_data in NETWORK_DATA:
        if network_data["chain"] not in chains or network_data["network"] not in networks:
            continue
            
        chain_entry = create_single_provider_entry(providers, network_data)
        if chain_entry and is_valid_chain_entry(chain_entry, single_provider=True):
            output["chains"].append(chain_entry)
    
    return output

def generate_multi_provider_config(providers, networks, chains):
    """Generate configuration with multiple providers per chain."""
    output = {"chains": []}
    
    for network_data in NETWORK_DATA:
        if network_data["chain"] not in chains or network_data["network"] not in networks:
            continue
            
        chain_entry = create_multi_provider_entry(providers, network_data)
        if chain_entry and is_valid_chain_entry(chain_entry, single_provider=False):
            output["chains"].append(chain_entry)
    
    return output

def main():
    # Automatically generate the list of supported chains and networks from NETWORK_DATA
    supported_chains = sorted(list(set(network["chain"] for network in NETWORK_DATA)))
    supported_networks = sorted(list(set(network["network"] for network in NETWORK_DATA)))
    
    # Get all unique provider types used in NETWORK_DATA
    all_provider_types = set()
    for network in NETWORK_DATA:
        all_provider_types.update(network["providers"].keys())
    supported_providers = sorted(list(all_provider_types))
    
    parser = argparse.ArgumentParser(description="Generate providers.json configuration")
    parser.add_argument("--providers", nargs="+", required=True,
                        help=f"Provider specification formats: provider (no auth), provider:token (token auth), or provider:username:password (basic auth). Supported providers: {', '.join(supported_providers)}")
    parser.add_argument("--networks", nargs="+", required=True,
                        choices=supported_networks,
                        help="Networks to generate configs for")
    parser.add_argument("--chains", nargs="+", required=True,
                        choices=supported_chains,
                        help="Chains to generate configs for")
    parser.add_argument("--output", "-o", default="generated_providers.json",
                        help="Output file path")
    parser.add_argument("--single-provider", action="store_true",
                        help="Generate config with single provider per chain")
    
    args = parser.parse_args()

    # Choose the appropriate generator function based on the single_provider flag
    generator_func = generate_single_provider_config if args.single_provider else generate_multi_provider_config
    config = generator_func(args.providers, args.networks, args.chains)
    
    with open(args.output, "w") as f:
        json.dump(config, f, indent=2)
    
    print(f"Successfully generated {args.output}")

if __name__ == "__main__":
    main()
