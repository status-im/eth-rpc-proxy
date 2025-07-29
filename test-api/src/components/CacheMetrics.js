import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { 
  getProxyUrl, 
  AVAILABLE_NETWORKS,
  makePermanentCacheRequest,
  makeShortCacheRequest,
  makeMinimalCacheRequest,
  formatRpcResult,
  getCacheTypeColor,
  generateJwtToken
} from '../utils';

const CacheMetrics = ({ onBackToHome }) => {
  const [loading, setLoading] = useState(false);
  const [metrics, setMetrics] = useState(null);
  const [error, setError] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [refreshInterval, setRefreshInterval] = useState(5); // seconds

  // JWT Token state
  const [token, setToken] = useState('');
  const [tokenLoading, setTokenLoading] = useState(false);
  const [tokenError, setTokenError] = useState('');
  const [tokenStatus, setTokenStatus] = useState('');

  // RPC Testing state
  const [selectedNetwork, setSelectedNetwork] = useState('ethereum/mainnet');
  const [rpcLoading, setRpcLoading] = useState(false);
  const [rpcResults, setRpcResults] = useState({});
  const [rpcError, setRpcError] = useState('');

  const proxyUrl = getProxyUrl();

  // Auto-generate token on component mount (optional)
  useEffect(() => {
    // Uncomment the line below to automatically generate token when component loads
    // handleGenerateToken();
  }, []);

  // Parsing Prometheus metrics
  const parsePrometheusMetrics = (text) => {
    const lines = text.split('\n').filter(line => line.trim() && !line.startsWith('#'));
    const metrics = {};
    
    lines.forEach(line => {
      const match = line.match(/^([^{]+)(?:\{([^}]*)\})?\s+(.+)$/);
      if (match) {
        const [, metricName, labels, value] = match;
        const labelObj = {};
        
        if (labels) {
          const labelMatches = labels.match(/(\w+)="([^"]*)"/g);
          if (labelMatches) {
            labelMatches.forEach(labelMatch => {
              const [, key, val] = labelMatch.match(/(\w+)="([^"]*)"/);
              labelObj[key] = val;
            });
          }
        }
        
        if (!metrics[metricName]) {
          metrics[metricName] = [];
        }
        
        metrics[metricName].push({
          labels: labelObj,
          value: parseFloat(value)
        });
      }
    });
    
    return metrics;
  };

  // Fetching metrics
  const fetchMetrics = async () => {
    setLoading(true);
    setError('');
    
    try {
      const response = await axios.get(`${proxyUrl}/metrics/cache`, {
        timeout: 10000
      });
      
      const parsedMetrics = parsePrometheusMetrics(response.data);
      setMetrics(parsedMetrics);
    } catch (err) {
      console.error('Error fetching metrics:', err);
      setError(`Error fetching metrics: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  // Formatting byte size
  const formatBytes = (bytes) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  // JWT Token generation using authUtils
  // This function automatically solves the puzzle and gets a valid JWT token
  // Returns the token directly to avoid React state race conditions
  const handleGenerateToken = async () => {
    setTokenLoading(true);
    setTokenError('');
    setTokenStatus('');
    
    try {
      const onProgress = (attempts, maxAttempts) => {
        setTokenStatus(`🔍 Solving puzzle... ${attempts}/${maxAttempts} attempts`);
      };

      const onStatusUpdate = (status) => {
        setTokenStatus(status);
      };

      const result = await generateJwtToken(onProgress, onStatusUpdate);
      
      if (result.success) {
        setToken(result.token);
        setTokenStatus(`✅ JWT token generated successfully! Solved in ${result.solveTime}s with ${result.attempts} attempts`);
        console.log('JWT Token generated for CacheMetrics:', result.token.substring(0, 50) + '...');
        return result.token; // Return the token directly
      } else {
        setTokenError(`Failed to generate token: ${result.error.message}`);
        setTokenStatus('');
        throw new Error(`Failed to generate token: ${result.error.message}`);
      }
    } catch (err) {
      console.error('Error generating JWT token:', err);
      setTokenError(`Error generating token: ${err.message}`);
      setTokenStatus('');
      throw err; // Re-throw to handle in ensureToken
    } finally {
      setTokenLoading(false);
    }
  };

  // Ensure we have a token before making RPC requests
  // Returns the token directly to avoid state synchronization issues
  // Handles concurrent token generation requests properly
  const ensureToken = async () => {
    if (!token && !tokenLoading) {
      console.log('No JWT token available, generating one...');
      return await handleGenerateToken();
    }
    
    // Wait for token generation to complete if it's in progress
    if (tokenLoading) {
      return new Promise((resolve, reject) => {
        const startTime = Date.now();
        const timeout = 30000; // 30 second timeout
        
        const checkToken = () => {
          if (!tokenLoading) {
            if (token) {
              console.log('Token generation completed, using existing token:', token.substring(0, 20) + '...');
              resolve(token);
            } else {
              reject(new Error('Token generation completed but no token available'));
            }
          } else if (Date.now() - startTime > timeout) {
            reject(new Error('Token generation timeout'));
          } else {
            setTimeout(checkToken, 100);
          }
        };
        checkToken();
      });
    }
    
    console.log('Using existing token:', token.substring(0, 20) + '...');
    return token;
  };

  // RPC request functions using utils with JWT token
  const makePermanentRequest = async () => {
    setRpcLoading(true);
    setRpcError('');
    
    try {
      const currentToken = await ensureToken();
      if (!currentToken) {
        throw new Error('Failed to generate JWT token');
      }

      console.log('Making permanent cache request with token:', currentToken.substring(0, 20) + '...');
      const result = await makePermanentCacheRequest(selectedNetwork, currentToken);
      
      // If request failed due to auth issues, try regenerating token once
      if (!result.success && result.error && 
          (result.error.status === 401 || result.error.status === 403)) {
        console.log('Auth error detected, regenerating token and retrying...');
        const newToken = await handleGenerateToken();
        const retryResult = await makePermanentCacheRequest(selectedNetwork, newToken);
        const formatted = formatRpcResult(retryResult);
        
        setRpcResults(prev => ({
          ...prev,
          'eth_chainId': formatted
        }));
      } else {
        const formatted = formatRpcResult(result);
        
        setRpcResults(prev => ({
          ...prev,
          'eth_chainId': formatted
        }));
      }
    } catch (err) {
      console.error('Error making permanent cache request:', err);
      setRpcError(`Error making eth_chainId request: ${err.message}`);
    } finally {
      setRpcLoading(false);
    }
  };

  const makeShortRequest = async () => {
    setRpcLoading(true);
    setRpcError('');
    
    try {
      const currentToken = await ensureToken();
      if (!currentToken) {
        throw new Error('Failed to generate JWT token');
      }

      console.log('Making short cache request with token:', currentToken.substring(0, 20) + '...');
      const result = await makeShortCacheRequest(selectedNetwork, currentToken);
      
      // If request failed due to auth issues, try regenerating token once
      if (!result.success && result.error && 
          (result.error.status === 401 || result.error.status === 403)) {
        console.log('Auth error detected, regenerating token and retrying...');
        const newToken = await handleGenerateToken();
        const retryResult = await makeShortCacheRequest(selectedNetwork, newToken);
        const formatted = formatRpcResult(retryResult);
        
        setRpcResults(prev => ({
          ...prev,
          'eth_blockNumber': formatted
        }));
      } else {
        const formatted = formatRpcResult(result);
        
        setRpcResults(prev => ({
          ...prev,
          'eth_blockNumber': formatted
        }));
      }
    } catch (err) {
      console.error('Error making short cache request:', err);
      setRpcError(`Error making eth_blockNumber request: ${err.message}`);
    } finally {
      setRpcLoading(false);
    }
  };

  const makeMinimalRequest = async () => {
    setRpcLoading(true);
    setRpcError('');
    
    try {
      const currentToken = await ensureToken();
      if (!currentToken) {
        throw new Error('Failed to generate JWT token');
      }

      console.log('Making minimal cache request with token:', currentToken.substring(0, 20) + '...');
      const result = await makeMinimalCacheRequest(selectedNetwork, currentToken);
      
      // If request failed due to auth issues, try regenerating token once
      if (!result.success && result.error && 
          (result.error.status === 401 || result.error.status === 403)) {
        console.log('Auth error detected, regenerating token and retrying...');
        const newToken = await handleGenerateToken();
        const retryResult = await makeMinimalCacheRequest(selectedNetwork, newToken);
        const formatted = formatRpcResult(retryResult);
        
        setRpcResults(prev => ({
          ...prev,
          'eth_gasPrice': formatted
        }));
      } else {
        const formatted = formatRpcResult(result);
        
        setRpcResults(prev => ({
          ...prev,
          'eth_gasPrice': formatted
        }));
      }
    } catch (err) {
      console.error('Error making minimal cache request:', err);
      setRpcError(`Error making eth_gasPrice request: ${err.message}`);
    } finally {
      setRpcLoading(false);
    }
  };

  // Auto-refresh
  useEffect(() => {
    let interval;
    if (autoRefresh) {
      interval = setInterval(fetchMetrics, refreshInterval * 1000);
    }
    return () => {
      if (interval) clearInterval(interval);
    };
  }, [autoRefresh, refreshInterval, proxyUrl]);

  // Getting metric value by cache type
  const getMetricValue = (metricName, cacheType) => {
    if (!metrics || !metrics[metricName]) return 0;
    const metric = metrics[metricName].find(m => m.labels.cache_type === cacheType);
    return metric ? metric.value : 0;
  };

  // Getting total metric without labels
  const getTotalMetricValue = (metricName) => {
    if (!metrics || !metrics[metricName]) return 0;
    const metric = metrics[metricName].find(m => Object.keys(m.labels).length === 0);
    return metric ? metric.value : 0;
  };

  const cacheTypes = ['permanent', 'short', 'minimal', 'providers', 'jwt_tokens'];

  return (
    <div className="App">
      <header className="App-header">
        <h1>📊 Cache Metrics Dashboard</h1>
        <p>Real-time cache performance and usage metrics</p>
        <div style={{ marginTop: '10px' }}>
          <button className="button secondary" onClick={onBackToHome}>
            ← Back to Home
          </button>
          <button 
            className="button primary" 
            onClick={fetchMetrics} 
            disabled={loading}
            style={{ marginLeft: '10px' }}
          >
            {loading ? '🔄 Loading...' : '🔄 Refresh Metrics'}
          </button>
        </div>
      </header>

      <main className="App-main">
        <div style={{ maxWidth: '1200px', margin: '0 auto', padding: '20px' }}>
          
          {/* Auto-refresh controls */}
          <div style={{ 
            background: '#2a2a2a', 
            padding: '15px', 
            borderRadius: '8px', 
            marginBottom: '20px',
            display: 'flex',
            alignItems: 'center',
            gap: '15px'
          }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <input
                type="checkbox"
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.target.checked)}
              />
              Auto-refresh
            </label>
            <label style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              Interval:
              <select
                value={refreshInterval}
                onChange={(e) => setRefreshInterval(parseInt(e.target.value))}
                disabled={!autoRefresh}
                style={{ 
                  padding: '4px 8px', 
                  borderRadius: '4px', 
                  border: '1px solid #444',
                  background: '#1a1a1a',
                  color: 'white'
                }}
              >
                <option value={1}>1s</option>
                <option value={5}>5s</option>
                <option value={10}>10s</option>
                <option value={30}>30s</option>
              </select>
            </label>
            {autoRefresh && (
              <span style={{ color: '#4CAF50', fontSize: '14px' }}>
                🟢 Auto-refreshing every {refreshInterval}s
              </span>
            )}
          </div>

          {/* JWT Token Status */}
          <div style={{ 
            background: '#2a2a2a', 
            padding: '20px', 
            borderRadius: '8px', 
            marginBottom: '20px',
            border: `1px solid ${token ? '#4CAF50' : '#444'}`
          }}>
            <h2 style={{ margin: '0 0 15px 0' }}>🔐 JWT Authentication</h2>
            <p style={{ margin: '0 0 15px 0', color: '#ccc', fontSize: '14px' }}>
              JWT token is automatically generated for authenticated RPC requests
            </p>

            <div style={{ 
              display: 'flex', 
              alignItems: 'center', 
              gap: '15px',
              marginBottom: '15px'
            }}>
              <div style={{ 
                padding: '8px 12px', 
                borderRadius: '20px',
                background: token ? '#1b4332' : '#444',
                border: `1px solid ${token ? '#4CAF50' : '#666'}`,
                fontSize: '14px'
              }}>
                {token ? '✅ Token Active' : '⚪ No Token'}
              </div>
              
              <button 
                onClick={async () => {
                  try {
                    await handleGenerateToken();
                  } catch (err) {
                    console.error('Manual token generation failed:', err);
                  }
                }}
                disabled={tokenLoading || rpcLoading}
                style={{
                  background: token ? '#666' : '#4CAF50',
                  color: 'white',
                  border: 'none',
                  padding: '8px 16px',
                  borderRadius: '6px',
                  cursor: (tokenLoading || rpcLoading) ? 'not-allowed' : 'pointer',
                  opacity: (tokenLoading || rpcLoading) ? 0.6 : 1,
                  fontSize: '14px'
                }}
              >
                {tokenLoading ? '⏳ Generating...' : token ? '🔄 Regenerate Token' : '🚀 Generate Token'}
              </button>

              {token && (
                <button 
                  onClick={() => {
                    navigator.clipboard.writeText(token);
                    setTokenStatus('✅ Token copied to clipboard!');
                  }}
                  style={{
                    background: '#666',
                    color: 'white',
                    border: 'none',
                    padding: '8px 16px',
                    borderRadius: '6px',
                    cursor: 'pointer',
                    fontSize: '14px'
                  }}
                >
                  📋 Copy Token
                </button>
              )}
            </div>

            {/* Token Status/Progress */}
            {tokenStatus && (
              <div style={{
                background: tokenStatus.includes('✅') ? '#1b4332' : '#1a1a1a',
                color: tokenStatus.includes('✅') ? '#4CAF50' : '#FF9800',
                padding: '10px',
                borderRadius: '4px',
                marginBottom: '10px',
                border: `1px solid ${tokenStatus.includes('✅') ? '#4CAF50' : '#FF9800'}`,
                fontSize: '14px'
              }}>
                {tokenStatus}
              </div>
            )}

            {/* Token Error */}
            {tokenError && (
              <div style={{
                background: '#4a1414',
                color: '#f44336',
                padding: '10px',
                borderRadius: '4px',
                marginBottom: '10px',
                border: '1px solid #f44336',
                fontSize: '14px'
              }}>
                ❌ {tokenError}
              </div>
            )}

            {/* Token Preview */}
            {token && (
              <div style={{
                background: '#1a1a1a',
                padding: '10px',
                borderRadius: '4px',
                marginTop: '10px',
                border: '1px solid #333'
              }}>
                <div style={{ fontSize: '12px', color: '#888', marginBottom: '5px' }}>
                  Token Preview:
                </div>
                <div style={{
                  fontFamily: 'monospace',
                  fontSize: '11px',
                  color: '#4CAF50',
                  wordBreak: 'break-all',
                  lineHeight: '1.4'
                }}>
                  {token.substring(0, 100)}...
                </div>
              </div>
            )}
          </div>

          {/* RPC Testing Controls */}
          <div style={{ 
            background: '#2a2a2a', 
            padding: '20px', 
            borderRadius: '8px', 
            marginBottom: '20px',
            border: '1px solid #444'
          }}>
            <h2 style={{ margin: '0 0 15px 0' }}>🚀 RPC Cache Testing</h2>
            <p style={{ margin: '0 0 15px 0', color: '#ccc', fontSize: '14px' }}>
              Test different cache types by making authenticated RPC requests to specific networks
            </p>
            
            {/* Network Selection */}
            <div style={{ marginBottom: '15px' }}>
              <label style={{ display: 'block', marginBottom: '8px', fontWeight: 'bold' }}>
                Select Network:
              </label>
              <select
                value={selectedNetwork}
                onChange={(e) => setSelectedNetwork(e.target.value)}
                style={{ 
                  padding: '8px 12px', 
                  borderRadius: '4px', 
                  border: '1px solid #444',
                  background: '#1a1a1a',
                  color: 'white',
                  minWidth: '250px'
                }}
              >
                {AVAILABLE_NETWORKS.map(network => (
                  <option key={network.value} value={network.value}>
                    {network.label} (Chain ID: {network.chainId})
                  </option>
                ))}
              </select>
            </div>

            {/* Info about automatic authentication */}
            <div style={{
              background: '#1a1a1a',
              padding: '10px',
              borderRadius: '4px',
              marginBottom: '15px',
              fontSize: '13px',
              color: '#ccc',
              border: '1px solid #333'
            }}>
              ℹ️ <strong>Automatic Authentication:</strong> JWT token will be generated automatically if needed before making RPC requests.
              {token && (
                <span style={{ color: '#4CAF50' }}> Current token is ready for use!</span>
              )}
            </div>

            {/* RPC Request Buttons */}
            <div style={{ 
              display: 'flex', 
              gap: '10px', 
              flexWrap: 'wrap',
              marginBottom: '15px'
            }}>
              <button 
                className="button primary" 
                onClick={makePermanentRequest}
                disabled={rpcLoading || tokenLoading}
                style={{ 
                  background: '#4CAF50',
                  border: 'none',
                  padding: '10px 15px',
                  borderRadius: '5px',
                  cursor: (rpcLoading || tokenLoading) ? 'not-allowed' : 'pointer',
                  opacity: (rpcLoading || tokenLoading) ? 0.6 : 1,
                  position: 'relative'
                }}
              >
                {rpcLoading || tokenLoading ? '⏳' : '🔒'} Permanent Cache (eth_chainId)
                {!token && !tokenLoading && (
                  <span style={{ 
                    fontSize: '10px', 
                    display: 'block', 
                    opacity: 0.8 
                  }}>
                    (will auto-generate JWT)
                  </span>
                )}
              </button>
              <button 
                className="button primary" 
                onClick={makeShortRequest}
                disabled={rpcLoading || tokenLoading}
                style={{ 
                  background: '#FF9800',
                  border: 'none',
                  padding: '10px 15px',
                  borderRadius: '5px',
                  cursor: (rpcLoading || tokenLoading) ? 'not-allowed' : 'pointer',
                  opacity: (rpcLoading || tokenLoading) ? 0.6 : 1
                }}
              >
                {rpcLoading || tokenLoading ? '⏳' : '⏱️'} Short Cache (eth_blockNumber)
                {!token && !tokenLoading && (
                  <span style={{ 
                    fontSize: '10px', 
                    display: 'block', 
                    opacity: 0.8 
                  }}>
                    (will auto-generate JWT)
                  </span>
                )}
              </button>
              <button 
                className="button primary" 
                onClick={makeMinimalRequest}
                disabled={rpcLoading || tokenLoading}
                style={{ 
                  background: '#2196F3',
                  border: 'none',
                  padding: '10px 15px',
                  borderRadius: '5px',
                  cursor: (rpcLoading || tokenLoading) ? 'not-allowed' : 'pointer',
                  opacity: (rpcLoading || tokenLoading) ? 0.6 : 1
                }}
              >
                {rpcLoading || tokenLoading ? '⏳' : '⚡'} Minimal Cache (eth_gasPrice)
                {!token && !tokenLoading && (
                  <span style={{ 
                    fontSize: '10px', 
                    display: 'block', 
                    opacity: 0.8 
                  }}>
                    (will auto-generate JWT)
                  </span>
                )}
              </button>
            </div>

            {/* Test All Button */}
            <div style={{ marginBottom: '15px' }}>
              <button 
                onClick={async () => {
                  setRpcLoading(true);
                  setRpcError('');
                  
                  try {
                    const currentToken = await ensureToken();
                    if (!currentToken) {
                      throw new Error('Failed to generate JWT token');
                    }

                    console.log('Making all cache requests with token:', currentToken.substring(0, 20) + '...');
                    
                    // Make all three requests in parallel
                    const [permanentResult, shortResult, minimalResult] = await Promise.all([
                      makePermanentCacheRequest(selectedNetwork, currentToken),
                      makeShortCacheRequest(selectedNetwork, currentToken),
                      makeMinimalCacheRequest(selectedNetwork, currentToken)
                    ]);

                    setRpcResults({
                      'eth_chainId': formatRpcResult(permanentResult),
                      'eth_blockNumber': formatRpcResult(shortResult),
                      'eth_gasPrice': formatRpcResult(minimalResult)
                    });
                  } catch (err) {
                    console.error('Error testing all cache types:', err);
                    setRpcError(`Error testing all cache types: ${err.message}`);
                  } finally {
                    setRpcLoading(false);
                  }
                }}
                disabled={rpcLoading || tokenLoading}
                style={{
                  background: 'linear-gradient(45deg, #4CAF50, #FF9800, #2196F3)',
                  color: 'white',
                  border: 'none',
                  padding: '12px 20px',
                  borderRadius: '6px',
                  cursor: (rpcLoading || tokenLoading) ? 'not-allowed' : 'pointer',
                  opacity: (rpcLoading || tokenLoading) ? 0.6 : 1,
                  fontWeight: 'bold',
                  marginRight: '10px'
                }}
              >
                {rpcLoading || tokenLoading ? '⏳ Testing...' : '🚀 Test All Cache Types'}
              </button>
              
              {Object.keys(rpcResults).length > 0 && (
                <button 
                  onClick={() => setRpcResults({})}
                  style={{
                    background: '#666',
                    color: 'white',
                    border: 'none',
                    padding: '12px 20px',
                    borderRadius: '6px',
                    cursor: 'pointer'
                  }}
                >
                  🗑️ Clear Results
                </button>
              )}
            </div>

            {/* Current Status Display */}
            <div style={{
              background: '#1a1a1a',
              padding: '10px',
              borderRadius: '4px',
              marginBottom: '15px',
              fontSize: '13px',
              border: '1px solid #333'
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <span style={{ color: '#888' }}>Status:</span>
                {tokenLoading && (
                  <span style={{ color: '#FF9800' }}>⏳ Generating JWT token...</span>
                )}
                {rpcLoading && (
                  <span style={{ color: '#2196F3' }}>🌐 Making RPC request...</span>
                )}
                {!tokenLoading && !rpcLoading && token && (
                  <span style={{ color: '#4CAF50' }}>✅ Ready for requests</span>
                )}
                {!tokenLoading && !rpcLoading && !token && (
                  <span style={{ color: '#888' }}>⚪ Click any RPC button to start</span>
                )}
              </div>
            </div>

            {/* RPC Error Display */}
            {rpcError && (
              <div style={{
                background: '#ffebee',
                color: '#c62828',
                padding: '10px',
                borderRadius: '4px',
                marginBottom: '15px',
                border: '1px solid #e57373'
              }}>
                ❌ {rpcError}
              </div>
            )}

            {/* RPC Results Display */}
            {Object.keys(rpcResults).length > 0 && (
              <div style={{ marginTop: '15px' }}>
                <h3 style={{ margin: '0 0 10px 0' }}>📊 RPC Results</h3>
                <div style={{ 
                  display: 'grid', 
                  gap: '10px',
                  gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))'
                }}>
                  {Object.entries(rpcResults).map(([method, result]) => (
                    <div key={method} style={{ 
                      background: '#1a1a1a', 
                      padding: '15px', 
                      borderRadius: '6px',
                      border: '1px solid #333'
                    }}>
                      <div style={{ 
                        display: 'flex', 
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        marginBottom: '10px'
                      }}>
                        <strong style={{ 
                          color: getCacheTypeColor(result.cacheType)
                        }}>
                          {result.method || method}
                        </strong>
                        <span style={{ 
                          background: getCacheTypeColor(result.cacheType),
                          color: 'white',
                          padding: '2px 8px',
                          borderRadius: '12px',
                          fontSize: '12px'
                        }}>
                          {result.cacheType}
                        </span>
                      </div>
                      <div style={{ fontSize: '14px', color: '#ccc', marginBottom: '8px' }}>
                        Network: {result.network}
                      </div>
                      <div style={{ fontSize: '14px', color: '#ccc', marginBottom: '8px' }}>
                        Time: {result.timestamp}
                      </div>
                      <div style={{ 
                        fontSize: '12px', 
                        color: '#4CAF50', 
                        marginBottom: '8px',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '5px'
                      }}>
                        🔐 Authenticated with JWT token
                      </div>
                      <div style={{ 
                        background: '#0a0a0a', 
                        padding: '10px', 
                        borderRadius: '4px',
                        fontFamily: 'monospace',
                        fontSize: '12px',
                        whiteSpace: 'pre-wrap',
                        overflow: 'auto'
                      }}>
                        {result.error ? (
                          <span style={{ color: '#f44336' }}>
                            Error: {JSON.stringify(result.error, null, 2)}
                          </span>
                        ) : (
                          <span style={{ color: '#4CAF50' }}>
                            Result: {JSON.stringify(result.responseData?.result || result.displayValue, null, 2)}
                          </span>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          {error && (
            <div style={{
              background: '#ffebee',
              color: '#c62828',
              padding: '15px',
              borderRadius: '8px',
              marginBottom: '20px',
              border: '1px solid #e57373'
            }}>
              ❌ {error}
            </div>
          )}

          {metrics && (
            <>
              {/* General metrics */}
              <div style={{ marginBottom: '30px' }}>
                <h2>📈 Total Cache Statistics</h2>
                <div style={{ 
                  display: 'grid', 
                  gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', 
                  gap: '15px',
                  marginBottom: '20px'
                }}>
                  <div style={{ background: '#2a2a2a', padding: '20px', borderRadius: '8px' }}>
                    <h3 style={{ margin: '0 0 10px 0', color: '#4CAF50' }}>Total Capacity</h3>
                    <p style={{ margin: 0, fontSize: '24px', fontWeight: 'bold' }}>
                      {formatBytes(getTotalMetricValue('nginx_cache_total_capacity_bytes'))}
                    </p>
                  </div>
                  <div style={{ background: '#2a2a2a', padding: '20px', borderRadius: '8px' }}>
                    <h3 style={{ margin: '0 0 10px 0', color: '#FF9800' }}>Total Used</h3>
                    <p style={{ margin: 0, fontSize: '24px', fontWeight: 'bold' }}>
                      {formatBytes(getTotalMetricValue('nginx_cache_total_used_bytes'))}
                    </p>
                  </div>
                </div>
              </div>

              {/* Metrics by cache types */}
              <div style={{ marginBottom: '30px' }}>
                <h2>🗂️ Cache Types Breakdown</h2>
                <div style={{ 
                  display: 'grid', 
                  gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', 
                  gap: '20px' 
                }}>
                  {cacheTypes.map(cacheType => (
                    <div key={cacheType} style={{ 
                      background: '#2a2a2a', 
                      padding: '20px', 
                      borderRadius: '8px',
                      border: '1px solid #444'
                    }}>
                      <h3 style={{ margin: '0 0 15px 0', textTransform: 'capitalize' }}>
                        {cacheType === 'jwt_tokens' ? 'JWT Tokens' : cacheType} Cache
                      </h3>
                      
                      <div style={{ marginBottom: '10px' }}>
                        <strong>Capacity:</strong> {formatBytes(getMetricValue('nginx_cache_capacity_bytes', cacheType))}
                      </div>
                      <div style={{ marginBottom: '10px' }}>
                        <strong>Used:</strong> {formatBytes(getMetricValue('nginx_cache_used_bytes', cacheType))}
                      </div>
                      <div style={{ marginBottom: '10px' }}>
                        <strong>Free:</strong> {formatBytes(getMetricValue('nginx_cache_free_bytes', cacheType))}
                      </div>
                      <div style={{ marginBottom: '15px' }}>
                        <strong>Usage:</strong> {getMetricValue('nginx_cache_usage_percent', cacheType).toFixed(2)}%
                      </div>
                      
                      {/* Progress bar */}
                      <div style={{ 
                        background: '#1a1a1a', 
                        borderRadius: '10px', 
                        height: '10px', 
                        overflow: 'hidden',
                        marginBottom: '15px'
                      }}>
                        <div style={{ 
                          background: getMetricValue('nginx_cache_usage_percent', cacheType) > 80 ? '#f44336' : 
                                    getMetricValue('nginx_cache_usage_percent', cacheType) > 60 ? '#ff9800' : '#4caf50',
                          height: '100%',
                          width: `${getMetricValue('nginx_cache_usage_percent', cacheType)}%`,
                          transition: 'width 0.3s ease'
                        }} />
                      </div>

                      {/* Hit/miss statistics for RPC caches */}
                      {['permanent', 'short', 'minimal', 'all'].includes(cacheType) && (
                        <div style={{ fontSize: '14px', color: '#ccc' }}>
                          <div>Hits: {getMetricValue('nginx_cache_hits_total', cacheType)}</div>
                          <div>Misses: {getMetricValue('nginx_cache_misses_total', cacheType)}</div>
                          <div>Requests: {getMetricValue('nginx_cache_requests_total', cacheType)}</div>
                          <div>Hit Rate: {getMetricValue('nginx_cache_hit_rate', cacheType).toFixed(2)}%</div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>

              {/* Hit/Miss statistics */}
              <div style={{ marginBottom: '30px' }}>
                <h2>🎯 Cache Performance</h2>
                <div style={{ 
                  display: 'grid', 
                  gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))', 
                  gap: '15px' 
                }}>
                  {['permanent', 'short', 'minimal', 'all'].map(cacheType => (
                    <div key={cacheType} style={{ 
                      background: '#2a2a2a', 
                      padding: '15px', 
                      borderRadius: '8px' 
                    }}>
                      <h4 style={{ margin: '0 0 10px 0', textTransform: 'capitalize' }}>
                        {cacheType} Cache
                      </h4>
                      <div style={{ fontSize: '18px', fontWeight: 'bold', marginBottom: '5px' }}>
                        Hit Rate: {getMetricValue('nginx_cache_hit_rate', cacheType).toFixed(2)}%
                      </div>
                      <div style={{ fontSize: '14px', color: '#ccc' }}>
                        {getMetricValue('nginx_cache_hits_total', cacheType)} hits / {getMetricValue('nginx_cache_requests_total', cacheType)} total
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Multi-Level Cache Statistics */}
              <div style={{ marginBottom: '30px' }}>
                <h2>🏗️ Multi-Level Cache Performance</h2>
                <p style={{ color: '#ccc', marginBottom: '20px', fontSize: '14px' }}>
                  L1: In-memory LRU cache • L2: Shared memory cache • L3: KeyDB persistent cache
                </p>
                
                {/* L1 & L2 Cache by Type */}
                <div style={{ 
                  display: 'grid', 
                  gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', 
                  gap: '20px',
                  marginBottom: '20px'
                }}>
                  {['permanent', 'short', 'minimal'].map(cacheType => (
                    <div key={cacheType} style={{ 
                      background: '#2a2a2a', 
                      padding: '20px', 
                      borderRadius: '8px',
                      border: '1px solid #444'
                    }}>
                      <h3 style={{ margin: '0 0 15px 0', textTransform: 'capitalize' }}>
                        {cacheType} Cache Levels
                      </h3>
                      
                      {/* L1 Cache */}
                      <div style={{ 
                        background: '#1a1a1a', 
                        padding: '12px', 
                        borderRadius: '6px',
                        marginBottom: '10px',
                        border: '1px solid #4CAF50'
                      }}>
                        <div style={{ 
                          display: 'flex', 
                          justifyContent: 'space-between', 
                          alignItems: 'center',
                          marginBottom: '8px'
                        }}>
                          <span style={{ fontWeight: 'bold', color: '#4CAF50' }}>L1 (Memory)</span>
                          <span style={{ 
                            background: '#4CAF50', 
                            color: 'white', 
                            padding: '2px 8px', 
                            borderRadius: '10px', 
                            fontSize: '12px' 
                          }}>
                            {getMetricValue('nginx_l1_cache_hit_rate', cacheType).toFixed(1)}%
                          </span>
                        </div>
                        <div style={{ fontSize: '14px', color: '#ccc' }}>
                          Hits: {getMetricValue('nginx_l1_cache_hits_total', cacheType)}
                        </div>
                      </div>
                      
                      {/* L2 Cache */}
                      <div style={{ 
                        background: '#1a1a1a', 
                        padding: '12px', 
                        borderRadius: '6px',
                        marginBottom: '10px',
                        border: '1px solid #FF9800'
                      }}>
                        <div style={{ 
                          display: 'flex', 
                          justifyContent: 'space-between', 
                          alignItems: 'center',
                          marginBottom: '8px'
                        }}>
                          <span style={{ fontWeight: 'bold', color: '#FF9800' }}>L2 (Shared)</span>
                          <span style={{ 
                            background: '#FF9800', 
                            color: 'white', 
                            padding: '2px 8px', 
                            borderRadius: '10px', 
                            fontSize: '12px' 
                          }}>
                            {getMetricValue('nginx_l2_cache_hit_rate', cacheType).toFixed(1)}%
                          </span>
                        </div>
                        <div style={{ fontSize: '14px', color: '#ccc' }}>
                          Hits: {getMetricValue('nginx_l2_cache_hits_total', cacheType)}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
                
                {/* L3 Cache (Global) */}
                <div style={{ 
                  background: '#2a2a2a', 
                  padding: '20px', 
                  borderRadius: '8px',
                  border: '1px solid #2196F3'
                }}>
                  <h3 style={{ margin: '0 0 15px 0', color: '#2196F3' }}>
                    L3 Cache (KeyDB) - Global Statistics
                  </h3>
                  <div style={{ 
                    display: 'grid', 
                    gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', 
                    gap: '15px' 
                  }}>
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: '24px', fontWeight: 'bold', color: '#2196F3' }}>
                        {getTotalMetricValue('nginx_l3_cache_hit_rate').toFixed(1)}%
                      </div>
                      <div style={{ fontSize: '14px', color: '#ccc' }}>Hit Rate</div>
                    </div>
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: '24px', fontWeight: 'bold', color: '#4CAF50' }}>
                        {getTotalMetricValue('nginx_l3_cache_hits_total')}
                      </div>
                      <div style={{ fontSize: '14px', color: '#ccc' }}>Hits</div>
                    </div>
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: '24px', fontWeight: 'bold', color: '#f44336' }}>
                        {getTotalMetricValue('nginx_l3_cache_misses_total')}
                      </div>
                      <div style={{ fontSize: '14px', color: '#ccc' }}>Misses</div>
                    </div>
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: '24px', fontWeight: 'bold', color: '#666' }}>
                        {getTotalMetricValue('nginx_l3_cache_hits_total') + getTotalMetricValue('nginx_l3_cache_misses_total')}
                      </div>
                      <div style={{ fontSize: '14px', color: '#ccc' }}>Total Requests</div>
                    </div>
                  </div>
                </div>
              </div>

              {/* Last update time */}
              <div style={{ 
                textAlign: 'center', 
                color: '#888', 
                fontSize: '14px',
                marginTop: '20px'
              }}>
                Last updated: {new Date().toLocaleString()}
              </div>
            </>
          )}

          {!metrics && !loading && !error && (
            <div style={{
              textAlign: 'center',
              padding: '40px',
              color: '#666'
            }}>
              Click "Refresh Metrics" to load cache statistics
            </div>
          )}
        </div>
      </main>
    </div>
  );
};

export default CacheMetrics;