import React, { useState, useEffect } from 'react';
import axios from 'axios';

const CacheMetrics = ({ onBackToHome }) => {
  const [loading, setLoading] = useState(false);
  const [metrics, setMetrics] = useState(null);
  const [error, setError] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [refreshInterval, setRefreshInterval] = useState(5); // seconds

  const proxyUrl = process.env.REACT_APP_RPC_PROXY_URL || 'http://localhost:8080';

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