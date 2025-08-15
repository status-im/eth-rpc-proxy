// Common API utilities and configurations

/**
 * Get the configured proxy URL from environment or default
 * @returns {string} Proxy URL
 */
export const getProxyUrl = () => {
  return process.env.REACT_APP_RPC_PROXY_URL || 'http://localhost:8080';
};

/**
 * Common axios configuration for API requests
 * @param {string} token - Optional JWT token
 * @param {Object} basicAuth - Optional basic auth {username, password}
 * @returns {Object} Axios config object
 */
export const getApiConfig = (token = null, basicAuth = null) => {
  const config = {
    headers: {
      'Content-Type': 'application/json'
    },
    timeout: 10000
  };

  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }

  if (basicAuth && basicAuth.username && basicAuth.password) {
    config.auth = {
      username: basicAuth.username,
      password: basicAuth.password
    };
  }

  return config;
};

/**
 * Standard error handler for API responses
 * @param {Error} error - Axios error object
 * @returns {Object} Formatted error object
 */
export const handleApiError = (error) => {
  return {
    status: error.response?.status || 'Network Error',
    message: error.response?.data?.error?.message || error.response?.data || error.message || 'Unknown error',
    data: error.response?.data
  };
};

/**
 * Available networks configuration
 */
export const AVAILABLE_NETWORKS = [
  { value: 'ethereum/mainnet', label: 'Ethereum Mainnet', chainId: 1 },
  { value: 'ethereum/sepolia', label: 'Ethereum Sepolia', chainId: 11155111 },
  { value: 'optimism/mainnet', label: 'Optimism Mainnet', chainId: 10 },
  { value: 'optimism/sepolia', label: 'Optimism Sepolia', chainId: 11155420 },
  { value: 'arbitrum/mainnet', label: 'Arbitrum One', chainId: 42161 },
  { value: 'arbitrum/sepolia', label: 'Arbitrum Sepolia', chainId: 421614 },
  { value: 'base/mainnet', label: 'Base Mainnet', chainId: 8453 },
  { value: 'base/sepolia', label: 'Base Sepolia', chainId: 84532 },
  { value: 'linea/mainnet', label: 'Linea Mainnet', chainId: 59144 },
  { value: 'linea/sepolia', label: 'Linea Sepolia', chainId: 59141 },
  { value: 'blast/mainnet', label: 'Blast Mainnet', chainId: 81457 },
  { value: 'blast/sepolia', label: 'Blast Sepolia', chainId: 168587773 },
  { value: 'zksync/mainnet', label: 'zkSync Era', chainId: 324 },
  { value: 'zksync/sepolia', label: 'zkSync Sepolia', chainId: 300 },
  { value: 'mantle/mainnet', label: 'Mantle Mainnet', chainId: 5000 },
  { value: 'mantle/sepolia', label: 'Mantle Sepolia', chainId: 5003 },
  { value: 'abstract/mainnet', label: 'Abstract Mainnet', chainId: 2741 },
  { value: 'abstract/testnet', label: 'Abstract Testnet', chainId: 11124 },
  { value: 'unichain/mainnet', label: 'Unichain Mainnet', chainId: 130 },
  { value: 'unichain/sepolia', label: 'Unichain Sepolia', chainId: 1301 },
  { value: 'status/sepolia', label: 'Status Sepolia', chainId: 1660990954 },
  { value: 'bsc/mainnet', label: 'BSC Mainnet', chainId: 56 },
  { value: 'bsc/testnet', label: 'BSC Testnet', chainId: 97 },
  { value: 'polygon/mainnet', label: 'Polygon Mainnet', chainId: 137 },
  { value: 'polygon/amoy', label: 'Polygon Amoy', chainId: 80002 }
];

/**
 * Predefined RPC methods for testing
 */
export const PREDEFINED_RPC_METHODS = [
  { name: 'Get Block Number', method: 'eth_blockNumber', params: [], cacheType: 'short' },
  { name: 'Get Gas Price', method: 'eth_gasPrice', params: [], cacheType: 'minimal' },
  { name: 'Get Chain ID', method: 'eth_chainId', params: [], cacheType: 'permanent' },
  { name: 'Get Latest Block', method: 'eth_getBlockByNumber', params: ['latest', false], cacheType: 'short' },
  { name: 'Get Balance (Vitalik)', method: 'eth_getBalance', params: ['0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045', 'latest'], cacheType: 'short' }
];