INFURA = "infura"
GROVE = "grove"
NODEFLEET = "nodefleet"
STATUS_NETWORK = "status_network"
ALCHEMY = "alchemy"

NETWORK_DATA = [
    {
        "chain": "ethereum",
        "network": "mainnet",
        "chainId": 1,
        "providers": {
            INFURA: "https://mainnet.infura.io/v3/",
            GROVE: "https://eth.rpc.grove.city/v1/",
            NODEFLEET: "https://eth-mainnet.alphafleet.io/",
            ALCHEMY: "https://eth-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "ethereum",
        "network": "sepolia",
        "chainId": 11155111,
        "providers": {
            INFURA: "https://sepolia.infura.io/v3/",
            GROVE: "https://eth-sepolia-testnet.rpc.grove.city/v1/",
            NODEFLEET: "https://eth-sepolia.alphafleet.io/",
            ALCHEMY: "https://eth-sepolia.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "optimism",
        "network": "mainnet",
        "chainId": 10,
        "providers": {
            INFURA: "https://optimism-mainnet.infura.io/v3/",
            GROVE: "https://optimism.rpc.grove.city/v1/",
            NODEFLEET: "https://optimism-mainnet.alphafleet.io/",
            ALCHEMY: "https://opt-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "optimism",
        "network": "sepolia",
        "chainId": 11155420,
        "providers": {
            INFURA: "https://optimism-sepolia.infura.io/v3/",
            GROVE: "https://optimism-sepolia-testnet.rpc.grove.city/v1/",
            NODEFLEET: "https://optimism-sepolia.alphafleet.io/",
            ALCHEMY: "https://opt-sepolia.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "arbitrum",
        "network": "mainnet",
        "chainId": 42161,
        "providers": {
            INFURA: "https://arbitrum-mainnet.infura.io/v3/",
            GROVE: "https://arbitrum-one.rpc.grove.city/v1/",
            NODEFLEET: "https://arb-mainnet.alphafleet.io/",
            ALCHEMY: "https://arb-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "arbitrum",
        "network": "sepolia",
        "chainId": 421614,
        "providers": {
            INFURA: "https://arbitrum-sepolia.infura.io/v3/",
            GROVE: "https://arbitrum-sepolia-testnet.rpc.grove.city/v1/",
            NODEFLEET: "https://arb-sepolia.alphafleet.io/",
            ALCHEMY: "https://arb-sepolia.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "base",
        "network": "mainnet",
        "chainId": 8453,
        "providers": {
            INFURA: "https://base-mainnet.infura.io/v3/",
            GROVE: "https://base.rpc.grove.city/v1/",
            NODEFLEET: "https://base-mainnet.alphafleet.io/",
            ALCHEMY: "https://base-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "base",
        "network": "sepolia",
        "chainId": 84532,
        "providers": {
            INFURA: "https://base-sepolia.infura.io/v3/",
            GROVE: "https://base-testnet.rpc.grove.city/v1/",
            NODEFLEET: "https://base-sepolia.alphafleet.io/",
            ALCHEMY: "https://base-sepolia.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "linea",
        "network": "mainnet",
        "chainId": 59144,
        "providers": {
            INFURA: "https://linea-mainnet.infura.io/v3/",
            GROVE: "https://linea.rpc.grove.city/v1/",
            NODEFLEET: "https://linea-mainnet.alphafleet.io/",
            ALCHEMY: "https://linea-mainnet.g.alchemy.com/v2/",
        },
    },
    {
        "chain": "linea",
        "network": "sepolia",
        "chainId": 59141,
        "providers": {
            INFURA: "https://linea-sepolia.infura.io/v3/",
            ALCHEMY: "https://linea-sepolia.g.alchemy.com/v2/",
        }
    },
    {
        "chain": "blast",
        "network": "mainnet",
        "chainId": 81457,
        "providers": {
            INFURA: "https://blast-mainnet.infura.io/v3/",
            GROVE: "https://blast.rpc.grove.city/v1/",
            NODEFLEET: "https://blast-mainnet.alphafleet.io/",
            ALCHEMY: "https://blast-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "blast",
        "network": "sepolia",
        "chainId": 168587773,
        "providers": {
            INFURA: "https://blast-sepolia.infura.io/v3/",
            ALCHEMY: "https://blast-sepolia.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "zksync",
        "network": "mainnet",
        "chainId": 324,
        "providers": {
            INFURA: "https://zksync-mainnet.infura.io/v3/",
            GROVE: "https://zksync-era.rpc.grove.city/v1/",
            ALCHEMY: "https://zksync-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "zksync",
        "network": "sepolia",
        "chainId": 300,
        "providers": {
            INFURA: "https://zksync-sepolia.infura.io/v3/",
            ALCHEMY: "https://zksync-sepolia.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "mantle",
        "network": "mainnet",
        "chainId": 5000,
        "providers": {
            INFURA: "https://mantle-mainnet.infura.io/v3/",
            GROVE: "https://mantle.rpc.grove.city/v1/",
            ALCHEMY: "https://mantle-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "mantle",
        "network": "sepolia",
        "chainId": 5003,
        "providers": {
            INFURA: "https://mantle-sepolia.infura.io/v3/",
            ALCHEMY: "https://mantle-sepolia.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "abstract",
        "network": "mainnet",
        "chainId": 2741,
        "providers": {
            ALCHEMY: "https://abstract-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "abstract",
        "network": "testnet",
        "chainId": 11124,
        "providers": {
            ALCHEMY: "https://abstract-testnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "unichain",
        "network": "mainnet",
        "chainId": 130,
        "providers": {
            INFURA: "https://unichain-mainnet.infura.io/v3/",
            ALCHEMY: "https://unichain-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "unichain",
        "network": "sepolia",
        "chainId": 1301,
        "providers": {
            INFURA: "https://unichain-sepolia.infura.io/v3/",
            ALCHEMY: "https://unichain-sepolia.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "status",
        "network": "sepolia",
        "chainId": 1660990954,
        "providers": {
            STATUS_NETWORK: "https://public.sepolia.rpc.status.network/"
        }
    },
    {
        "chain": "bsc",
        "network": "mainnet",
        "chainId": 56,
        "providers": {
            INFURA: "https://bsc-mainnet.infura.io/v3/",
            GROVE: "https://bsc.rpc.grove.city/v1/",
            NODEFLEET: "https://bsc-mainnet.alphafleet.io/",
            ALCHEMY: "https://bnb-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "bsc",
        "network": "testnet",
        "chainId": 97,
        "providers": {
            INFURA: "https://bsc-testnet.infura.io/v3/",
            ALCHEMY: "https://bnb-testnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "polygon",
        "network": "mainnet",
        "chainId": 137,
        "providers": {
            INFURA: "https://polygon-mainnet.infura.io/v3/",
            GROVE: "https://polygon.rpc.grove.city/v1/",
            NODEFLEET: "https://polygon-mainnet.alphafleet.io/",
            ALCHEMY: "https://polygon-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "polygon",
        "network": "amoy",
        "chainId": 80002,
        "providers": {
            INFURA: "https://polygon-amoy.infura.io/v3/",
            GROVE: "https://polygon-amoy-testnet.rpc.grove.city/v1/",
            ALCHEMY: "https://polygon-amoy.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "polygon-zkevm",
        "network": "mainnet",
        "chainId": 1101,
        "providers": {
            ALCHEMY: "https://polygonzkevm-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "polygon-zkevm",
        "network": "cardona",
        "chainId": 2442,
        "providers": {
            ALCHEMY: "https://polygonzkevm-cardona.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "ink",
        "network": "mainnet",
        "chainId": 57073,
        "providers": {
            ALCHEMY: "https://ink-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "ink",
        "network": "sepolia",
        "chainId": 763373,
        "providers": {
            ALCHEMY: "https://ink-sepolia.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "soneium",
        "network": "mainnet",
        "chainId": 1868,
        "providers": {
            ALCHEMY: "https://soneium-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "soneium",
        "network": "minato",
        "chainId": 1946,
        "providers": {
            ALCHEMY: "https://soneium-minato.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "scroll",
        "network": "mainnet",
        "chainId": 534352,
        "providers": {
            INFURA: "https://scroll-mainnet.infura.io/v3/",
            ALCHEMY: "https://scroll-mainnet.g.alchemy.com/v2/"
        }
    },
    {
        "chain": "scroll",
        "network": "sepolia",
        "chainId": 534351,
        "providers": {
            INFURA: "https://scroll-sepolia.infura.io/v3/",
            ALCHEMY: "https://scroll-sepolia.g.alchemy.com/v2/"
        }
    }
]