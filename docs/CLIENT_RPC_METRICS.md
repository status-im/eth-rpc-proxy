# Client RPC request counter

Design decisions agreed during review and implemented by the nginx/Lua proxy.

## Agreed semantics

- Count incoming RPC HTTP requests after authorization and successful HTTP body
  reading, including cache hits and requests that subsequently fail processing.
- One HTTP request increments the counter once. A JSON-RPC batch also counts once.
- `rpc_proxy_client_request_bytes_total` and `rpc_proxy_client_response_bytes_total`
  sum HTTP bytes for the same admitted requests and labels. Sizes are counter
  values, never labels. Request bytes include the request line, headers and encoded
  body; response bytes include headers and encoded body sent to the client.
  TCP/TLS overhead and upstream-provider traffic are not included.
- Labels: `chain`, `network`, `requested_provider_type`, `rpc_method`.
- The requested provider type comes from the client route, with `auto` when omitted.
  It does not describe the provider that eventually serves the request.
- For an unconfigured route or unsupported requested provider type, all three route
  labels become `unknown`. Temporary provider health must not redefine a configured
  route as unknown.
- RPC method values use a fixed allowlist independent of cache configuration:
  `eth_getBlockByHash`, `eth_getBlockByNumber`, `eth_getTransactionByHash`,
  `eth_getTransactionReceipt`, `eth_getLogs`, `eth_blockNumber`, `eth_gasPrice`,
  `eth_getBalance`, `eth_getCode`, `eth_getStorageAt`, `eth_getTransactionCount`,
  `eth_call`, `eth_estimateGas`, `eth_sendRawTransaction`, `eth_sendTransaction`,
  `web3_clientVersion`, `web3_sha3`, `net_version`, `net_listening`, `net_peerCount`,
  `eth_chainId`, `eth_syncing`.
- Other single-call methods map to `other`; bodies whose first non-whitespace
  character is `[` map to `batch` without JSON validation, including empty or
  malformed arrays. Other bodies that cannot be classified as a call map to `invalid`.
- Cache counters describe cache lookups and their results. They must not be added
  to this counter to calculate client demand.

## Cardinality estimate

The local `secrets/default_providers.json` contains 37 chain/network routes and
64 configured route/provider-type combinations. Adding `auto` for each route and
one global `unknown` combination gives 102 route label combinations.

22 allowed methods plus `other`, `batch`, and `invalid` give 25 method values:
**102 × 25 = 2,550 possible series per counter per scraped proxy instance**.
Three counters (requests, request bytes, response bytes) give **7,650 series**
for that fixed configuration; adding byte counters does not add label values.
All configured combinations are preinitialized to zero. Deployment target labels distinguish
instances; configuration changes can create additional historical series.

## Collection and lifecycle

The metric is named `rpc_proxy_client_requests_total` and replaces the earlier
outbound-attempt counter and dashboard panel. After authorization, the nginx RPC
location uses `lua_need_request_body on` to read the HTTP body before entering
the content handler. The handler calls `client_metrics.begin_request()` after
`ngx.req.read_body()` and `client_metrics.observe_body()` after decoding.
Request-local state is private to the metrics module. The log phase calls
`finish()` to increment once and clear
that state, using the body's method category or `invalid` when decoding failed.
Requests rejected before admission, including errors, timeouts or disconnects
while reading the HTTP body, remain outside all three counters. A fully received
body rejected during decoding (for example invalid gzip) still counts as `invalid`.
Disconnects after a successful body read remain counted.

The log phase reads nginx's existing `request_length` and `bytes_sent` values.
Byte accounting performs two additional shared-memory increments, without reading,
copying, decompressing or parsing bodies again. Request counter keys are preserved
across nginx reloads; byte counters have separate prefixed keys in the same dict.

The complete local `/app/providers.json` file supplies the route/type allowlist.
`provider_loader.load_configured_routes()` reuses the existing file reader and
`provider_config.configured_routes()` shares decoding and validation with the
routing configuration parser. It returns only chain, network and provider types,
without credentials. Worker 0 passes this ready index to `client_metrics.configure()`
on the reload interval, separately from the health-filtered provider list. The
metrics module preinitializes counters and publishes the allowlist atomically.
Existing counters are preserved. Passing no index after a failed load retains the
previous allowlist. At startup without a readable configuration, only the global
unknown category is initialized and an error is logged.

Worker-shared counters use non-evicting initialization and are preinitialized to
zero. If a worker receives a request before a counter is initialized, it uses
`safe_add` and retries the increment once; another worker winning initialization
does not lose either increment. Allocation failures are logged without failing
client requests. Counters reset when shared memory is lost. Historical label
combinations can remain after configuration changes, so the estimate above
describes a fixed configuration, not a permanent lifetime cap.

Prometheus provides monitoring estimates with `increase`, not an exact durable
request ledger. Restarts between scrapes can lose increments. Existing Prometheus
retention and storage settings are unchanged; deployment infrastructure must
provide sufficient retention for monthly reports.

## Verification

- Lua tests exercise the RPC handler and metric endpoint: cache hits, failover,
  batches, gzip decode errors, exclusion of HTTP body read failures, unknown routes,
  provider health changes, method bounds, zero initialization and allocation errors.
- Status Proxy Admin adds an opt-in `metrics` suite that compares before/after
  `/metrics` snapshots around real requests. Run against one isolated instance
  with no other client traffic for exact delta assertions.
- Admin Stats displays server counter-lifetime totals and exposed series count;
  Grafana's **Client Requests** section provides a primary chain/network table of
  request counts and request/response bytes for the selected period, plus the
  existing requested-provider view. All four labels remain available for queries.
