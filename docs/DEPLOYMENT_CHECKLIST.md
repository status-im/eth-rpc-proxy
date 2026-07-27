# Deployment Checklist

After adding new chains (see [ADD_NEW_CHAINS.md](ADD_NEW_CHAINS.md)), update the test server:

- [ ] **Enable the new networks in Infura and Alchemy dashboards before deploy.**
  Until Infura and Alchemy accept the chain, the proxy falls through to any
  public/no-auth backup provider. For Robinhood that means 100% of traffic
  hits Robinhood's rate-limited public RPC (no SLA); when that returns 429
  there is no further fallback and clients get 502.
  - Infura: enable Robinhood mainnet (`robinhood-mainnet.infura.io`) and
    testnet (`robinhood-testnet.infura.io`) on the project. Verify with
    `eth_chainId` → `0x1237` (4663) / `0xb626` (46630).
  - Alchemy: enable `ROBINHOOD_MAINNET` and `ROBINHOOD_TESTNET` on the app
    (dashboard → app → Networks). Same `eth_chainId` checks as above.
- [ ] Create ticket in [infra-proxy](https://github.com/status-im/infra-proxy)
   - Enable new chains in all affected RPC provider dashboards 
   - Add new chains to the eth-rpc-proxy-setup
   - *Optional: Create PR in infra-proxy with [new chains](@https://github.com/status-im/infra-proxy/blob/643054f1c2359a7ac02202f1f9d3cf6ec9e4af87/ansible/roles/eth-rpc-proxy-setup/tasks/setup.yml#L30)
- [ ] Notify infra-team in #infra-discussion channel
- [ ] Push to `deploy-test` branch to trigger server redeploy
- [ ] *Verify new chains are accessible with CURL