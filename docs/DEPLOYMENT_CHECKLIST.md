# Deployment Checklist

After adding new chains (see [ADD_NEW_CHAINS.md](ADD_NEW_CHAINS.md)), update the test server:

- [ ] **Enable Robinhood on Alchemy before deploy.** Infura does not support
  this chain. Providers are Alchemy → Robinhood public RPC. Until Alchemy
  accepts the network, 100% of traffic hits Robinhood's rate-limited public
  RPC (no SLA); when that returns 429 there is no further fallback and
  clients get 502.
  - Alchemy: enable `ROBINHOOD_MAINNET` and `ROBINHOOD_TESTNET` on the app
    (dashboard → app → Networks). Verify with `eth_chainId` → `0x1237`
    (4663) / `0xb626` (46630).
  - Reference provider for Robinhood must be Alchemy (not Infura). If the
    reference task lists Infura first and omits Alchemy, Robinhood gets no
    reference entry, the health checker skips it, and the proxy returns 404.
- [ ] Create ticket in [infra-proxy](https://github.com/status-im/infra-proxy)
   - Enable new chains in all affected RPC provider dashboards 
   - Add new chains to the eth-rpc-proxy-setup
   - *Optional: Create PR in infra-proxy with [new chains](@https://github.com/status-im/infra-proxy/blob/643054f1c2359a7ac02202f1f9d3cf6ec9e4af87/ansible/roles/eth-rpc-proxy-setup/tasks/setup.yml#L30)
- [ ] Notify infra-team in #infra-discussion channel
- [ ] Push to `deploy-test` branch to trigger server redeploy
- [ ] *Verify new chains are accessible with CURL