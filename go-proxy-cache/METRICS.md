# Cache Metrics Reference

## Overview
Brief description of all cache metrics with their types and cardinality.

**Note**: All cache metrics are prefixed with `eth_rpc_proxy_` in the actual Prometheus output

## Core Cache Metrics

| Metric Name | Type | Labels | Cardinality | Description |
|-------------|------|--------|-------------|-------------|
| `cache_requests_total` | Counter | `cache_type`, `level`, `network`, `rpc_method` | ~1,035 | Total number of cache requests |
| `cache_hits_total` | Counter | `cache_type`, `level`, `network`, `rpc_method` | ~1,035 | Total number of cache hits |
| `cache_misses_total` | Counter | `cache_type`, `level`, `network`, `rpc_method` | ~1,035 | Total number of cache misses |

**Total Core Metrics Cardinality**: ~3,105 time series

## Operational Metrics

| Metric Name | Type | Labels | Cardinality | Description |
|-------------|------|--------|-------------|-------------|
| `cache_sets_total` | Counter | `level`, `cache_type`, `network` | 45 | Number of cache set operations |
| `cache_evictions_total` | Counter | `level`, `cache_type`, `network` | 45 | Number of cache evictions |
| `cache_errors_total` | Counter | `level`, `kind` | 12 | Cache errors by type |
| `cache_bytes_read_total` | Counter | `level`, `cache_type`, `network` | 45 | Bytes read from cache |
| `cache_bytes_written_total` | Counter | `level`, `cache_type`, `network` | 45 | Bytes written to cache |

**Total Operational Metrics Cardinality**: 192 time series

## Performance Metrics

| Metric Name | Type | Labels | Cardinality | Description |
|-------------|------|--------|-------------|-------------|
| `cache_operation_duration_seconds` | Histogram | `operation`, `level` | 78 | Duration of cache operations (get/set) |
| `cache_item_age_seconds` | Histogram | `level`, `cache_type` | 117 | Age of items at hit time (TTL analysis) |

**Total Performance Metrics Cardinality**: 195 time series

## Capacity Metrics

| Metric Name | Type | Labels | Cardinality | Description |
|-------------|------|--------|-------------|-------------|
| `cache_keys` | Gauge | `level` | 3 | Current number of keys in cache |
| `cache_capacity_bytes` | Gauge | `level` | 1 | L1 cache capacity in bytes |
| `cache_used_bytes` | Gauge | `level` | 1 | L1 cache used space in bytes |

**Total Capacity Metrics Cardinality**: 5 time series

## Label Values

| Label | Values | Count | Description |
|-------|--------|-------|-------------|
| `cache_type` | `permanent`, `short`, `minimal` | 3 | Cache type based on TTL rules |
| `level` | `l1`, `l2`, `origin` | 3 | Cache level or origin server |
| `network` | `ethereum:mainnet`, `polygon:mainnet`, `unknown`, etc. | ~5 | Network identifier |
| `rpc_method` | Whitelisted methods + `other` | 23 | RPC method name (controlled) |
| `operation` | `get`, `set` | 2 | Cache operation type |
| `kind` | `encode`, `decode`, `upstream`, `redis` | 4 | Error type |

## RPC Method Whitelist

| Category | Methods | Count |
|----------|---------|-------|
| **Permanent Data** | `eth_getBlockByHash`, `eth_getBlockByNumber`, `eth_getTransactionByHash`, `eth_getTransactionReceipt`, `eth_getLogs` | 5 |
| **Short-lived Data** | `eth_blockNumber`, `eth_gasPrice`, `eth_getBalance`, `eth_getCode`, `eth_getStorageAt`, `eth_getTransactionCount`, `eth_call`, `eth_estimateGas` | 8 |
| **Minimal Cache** | `eth_sendRawTransaction`, `eth_sendTransaction` | 2 |
| **Web3/Net Methods** | `web3_clientVersion`, `web3_sha3`, `net_version`, `net_listening`, `net_peerCount` | 5 |
| **Other Methods** | All unlisted methods aggregated as `other` | 1 |
| **Total** | | **23** |

## Error Types

| Error Kind | Source | Description |
|------------|--------|-------------|
| `encode` | Serialization | JSON marshaling errors |
| `decode` | Deserialization | JSON unmarshaling errors |
| `upstream` | BigCache | L1 cache operation errors |
| `redis` | KeyDB/Redis | L2 cache connection/operation errors |
