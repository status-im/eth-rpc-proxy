// Main utils export file - convenience imports

// API utilities
export {
  getProxyUrl,
  getApiConfig,
  handleApiError,
  AVAILABLE_NETWORKS,
  PREDEFINED_RPC_METHODS
} from './apiUtils';

// Authentication utilities
export {
  getPuzzle,
  getAuthStatus,
  verifyToken,
  submitSolution,
  checkDifficulty,
  computeHMAC,
  hexToUint8Array,
  solvePuzzle,
  createDebugSolution,
  generateJwtToken
} from './authUtils';

// RPC utilities
export {
  makeRpcRequest,
  makePermanentCacheRequest,
  makeShortCacheRequest,
  makeMinimalCacheRequest,
  makeParallelRpcRequests,
  makeAllCacheTypeRequests,
  testRpcConnectivity,
  formatRpcResult,
  getCacheTypeColor,
  getCacheTypeEmoji
} from './rpcUtils';