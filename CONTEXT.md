# eth-rpc-proxy

Glossary for how this proxy names and relates chains, networks, and RPC providers.

## Language

**Chain**:
The blockchain product slug used in the public URL path `/{chain}/{network}` (e.g. `ethereum`, `robinhood`).
_Avoid_: Network (when meaning the product), chain name, brand

**Network**:
The deployment environment of a Chain (e.g. `mainnet`, `sepolia`, `testnet`, `hoodi`).
_Avoid_: Chain (when meaning the environment), testnet/mainnet as a product name

**Chain/network pair**:
A concrete routable target identified by `chain` + `network` and a numeric `chainId` (e.g. `robinhood/mainnet` = 4663).
_Avoid_: Chain alone, network alone, chain ID alone when referring to a route

**Provider type**:
A named class of RPC backend (e.g. `infura`, `alchemy`, `robinhood`) with a shared auth model and URL template.
_Avoid_: Provider (when meaning the type rather than an instance), vendor, backend

**Default provider**:
A candidate RPC endpoint listed for a chain/network pair that the proxy may use after health validation.
_Avoid_: Provider list entry, upstream

**Reference provider**:
The ground-truth endpoint for a chain/network pair against which other Default providers are compared during health checks.
_Avoid_: Primary provider, oracle, source of truth (as a synonym for the config role)

**Auto-approve**:
When a Default provider has the same Provider type as the Reference provider, it is treated as valid without cross-checking responses.
_Avoid_: Skip validation, trust same type
