# Deployment Checklist

After adding new chains (see [ADD_NEW_CHAINS.md](ADD_NEW_CHAINS.md)), update the test server:

- [ ] Create ticket in [infra-proxy](https://github.com/status-im/infra-proxy)
   - Enable new chains in all affected RPC provider dashboards 
   - Add new chains to the eth-rpc-proxy-setup
   - *Optional: Create PR in infra-proxy with [new chains](@https://github.com/status-im/infra-proxy/blob/643054f1c2359a7ac02202f1f9d3cf6ec9e4af87/ansible/roles/eth-rpc-proxy-setup/tasks/setup.yml#L30)
- [ ] Notify infra-team in #infra-discussion channel
- [ ] Push to `deploy-test` branch to trigger server redeploy
- [ ] *Verify new chains are accessible with CURL 