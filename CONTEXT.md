# eth-rpc-proxy

Glossary for how this proxy names and relates chains, networks, and RPC providers.

## Language

**Client RPC request**:
One incoming HTTP request to the proxy's RPC interface that has passed authorization, including requests served from cache and requests that later fail processing. A batch is one Client RPC request, independently of the number of calls it contains or attempts to contact providers.
_Avoid_: Upstream request, provider attempt, health-check request

**RPC method category**:
The recognized method of a single RPC call, or `other` for an unrecognized method. A batch has category `batch`; a body that cannot be classified as a single call or batch has category `invalid`.
_Avoid_: HTTP method, first batch method

**Requested provider type**:
The Provider type explicitly selected by the client in the RPC route. When the client leaves the choice to the proxy, the requested type is `auto`, regardless of which Default provider serves the request.
_Avoid_: Actual provider, selected backend, provider name

**Unknown RPC route**:
A client-selected chain/network pair or requested Provider type that is not configured for that route. All such selections form one `unknown` category rather than distinct named routes.
_Avoid_: Unhealthy provider, temporarily unavailable route

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
