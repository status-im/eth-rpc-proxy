import axios from 'axios';
import { getProxyUrl, getApiConfig, handleApiError } from './apiUtils';

/**
 * Make a JSON-RPC request to the proxy
 * @param {string} method - RPC method name
 * @param {Array} params - RPC method parameters
 * @param {string} network - Network path (e.g., 'ethereum/mainnet')
 * @param {string} token - Optional JWT token
 * @param {Object} basicAuth - Optional basic auth {username, password}
 * @returns {Promise<Object>} RPC response or error
 */
export const makeRpcRequest = async (method, params = [], network = 'ethereum/mainnet', token = null, basicAuth = null) => {
  try {
    const proxyUrl = getProxyUrl();
    const rpcPayload = {
      jsonrpc: '2.0',
      method: method,
      params: params,
      id: 1  // Use constant ID for better caching since we handle requests synchronously
    };

    console.log(`Making ${method} request to ${network}:`, rpcPayload);
    
    const response = await axios.post(
      `${proxyUrl}/${network}`, 
      rpcPayload, 
      getApiConfig(token, basicAuth)
    );

    console.log(`${method} response:`, response.data);

    return {
      success: true,
      data: response.data,
      method: method,
      network: network,
      timestamp: new Date().toLocaleString()
    };
  } catch (error) {
    console.error(`Error making ${method} request:`, error);
    const errorInfo = handleApiError(error);
    
    return {
      success: false,
      error: errorInfo,
      method: method,
      network: network,
      timestamp: new Date().toLocaleString()
    };
  }
};

/**
 * Make a permanent cache RPC request (eth_chainId)
 * @param {string} network - Network path
 * @param {string} token - Optional JWT token
 * @param {Object} basicAuth - Optional basic auth
 * @returns {Promise<Object>} RPC response or error
 */
export const makePermanentCacheRequest = async (network = 'ethereum/mainnet', token = null, basicAuth = null) => {
  const result = await makeRpcRequest('eth_chainId', [], network, token, basicAuth);
  return { ...result, cacheType: 'permanent' };
};

/**
 * Make a short cache RPC request (eth_blockNumber)
 * @param {string} network - Network path
 * @param {string} token - Optional JWT token
 * @param {Object} basicAuth - Optional basic auth
 * @returns {Promise<Object>} RPC response or error
 */
export const makeShortCacheRequest = async (network = 'ethereum/mainnet', token = null, basicAuth = null) => {
  const result = await makeRpcRequest('eth_blockNumber', [], network, token, basicAuth);
  return { ...result, cacheType: 'short' };
};

/**
 * Make a minimal cache RPC request (eth_gasPrice)
 * @param {string} network - Network path
 * @param {string} token - Optional JWT token
 * @param {Object} basicAuth - Optional basic auth
 * @returns {Promise<Object>} RPC response or error
 */
export const makeMinimalCacheRequest = async (network = 'ethereum/mainnet', token = null, basicAuth = null) => {
  const result = await makeRpcRequest('eth_gasPrice', [], network, token, basicAuth);
  return { ...result, cacheType: 'minimal' };
};

/**
 * Make multiple RPC requests in parallel
 * @param {Array} requests - Array of request objects {method, params, cacheType}
 * @param {string} network - Network path
 * @param {string} token - Optional JWT token
 * @param {Object} basicAuth - Optional basic auth
 * @returns {Promise<Array>} Array of RPC responses
 */
export const makeParallelRpcRequests = async (requests, network = 'ethereum/mainnet', token = null, basicAuth = null) => {
  const promises = requests.map(request => {
    const result = makeRpcRequest(request.method, request.params || [], network, token, basicAuth);
    if (request.cacheType) {
      return result.then(r => ({ ...r, cacheType: request.cacheType }));
    }
    return result;
  });

  try {
    return await Promise.all(promises);
  } catch (error) {
    console.error('Error in parallel RPC requests:', error);
    throw error;
  }
};

/**
 * Make all three cache type requests (permanent, short, minimal)
 * @param {string} network - Network path
 * @param {string} token - Optional JWT token
 * @param {Object} basicAuth - Optional basic auth
 * @returns {Promise<Object>} Object with results for each cache type
 */
export const makeAllCacheTypeRequests = async (network = 'ethereum/mainnet', token = null, basicAuth = null) => {
  try {
    const requests = [
      { method: 'eth_chainId', params: [], cacheType: 'permanent' },
      { method: 'eth_blockNumber', params: [], cacheType: 'short' },
      { method: 'eth_gasPrice', params: [], cacheType: 'minimal' }
    ];

    const results = await makeParallelRpcRequests(requests, network, token, basicAuth);
    
    return {
      permanent: results[0],
      short: results[1],
      minimal: results[2]
    };
  } catch (error) {
    console.error('Error making all cache type requests:', error);
    throw error;
  }
};

/**
 * Test RPC connectivity with basic method
 * @param {string} network - Network path
 * @param {string} token - Optional JWT token
 * @param {Object} basicAuth - Optional basic auth
 * @returns {Promise<Object>} Connection test result
 */
export const testRpcConnectivity = async (network = 'ethereum/mainnet', token = null, basicAuth = null) => {
  const startTime = Date.now();
  
  try {
    const result = await makeRpcRequest('eth_blockNumber', [], network, token, basicAuth);
    const responseTime = Date.now() - startTime;
    
    return {
      success: result.success,
      network: network,
      responseTime: responseTime,
      blockNumber: result.success ? result.data.result : null,
      error: result.success ? null : result.error,
      timestamp: new Date().toLocaleString()
    };
  } catch (error) {
    const responseTime = Date.now() - startTime;
    return {
      success: false,
      network: network,
      responseTime: responseTime,
      blockNumber: null,
      error: handleApiError(error),
      timestamp: new Date().toLocaleString()
    };
  }
};

/**
 * Format RPC result for display
 * @param {Object} result - RPC result object
 * @returns {Object} Formatted result for UI display
 */
export const formatRpcResult = (result) => {
  if (!result) return null;

  return {
    method: result.method,
    network: result.network,
    cacheType: result.cacheType || 'unknown',
    timestamp: result.timestamp,
    success: result.success,
    responseData: result.success ? result.data : null,
    error: result.success ? null : result.error,
    displayValue: result.success ? result.data.result : 'Error'
  };
};

/**
 * Get display color for cache type
 * @param {string} cacheType - Cache type (permanent, short, minimal)
 * @returns {string} CSS color value
 */
export const getCacheTypeColor = (cacheType) => {
  switch (cacheType) {
    case 'permanent':
      return '#4CAF50';
    case 'short':
      return '#FF9800';
    case 'minimal':
      return '#2196F3';
    default:
      return '#666';
  }
};

/**
 * Get display emoji for cache type
 * @param {string} cacheType - Cache type (permanent, short, minimal)
 * @returns {string} Emoji character
 */
export const getCacheTypeEmoji = (cacheType) => {
  switch (cacheType) {
    case 'permanent':
      return '🔒';
    case 'short':
      return '⏱️';
    case 'minimal':
      return '⚡';
    default:
      return '❓';
  }
};