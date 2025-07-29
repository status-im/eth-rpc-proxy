import React, { useState } from 'react';
import { 
  makeRpcRequest, 
  AVAILABLE_NETWORKS, 
  PREDEFINED_RPC_METHODS,
  formatRpcResult 
} from '../utils';

const RpcTester = ({ token }) => {
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState('');
  const [response, setResponse] = useState(null);
  const [selectedChain, setSelectedChain] = useState('ethereum/mainnet');
  const [basicAuthUsername, setBasicAuthUsername] = useState('');
  const [basicAuthPassword, setBasicAuthPassword] = useState('');

  // Test with predefined method
  const testPredefinedMethod = async (method, params) => {
    if (!token) {
      setStatus('❌ No JWT token available. Generate one first!');
      return;
    }

    setLoading(true);
    setStatus(`Testing ${method}...`);
    
    try {
      const result = await makeRpcRequest(method, params, selectedChain, token);
      
      if (result.success) {
        setResponse(result.data);
        setStatus('✅ RPC request successful!');
      } else {
        setResponse(result.error);
        setStatus(`❌ RPC request failed: ${result.error.status}`);
      }
    } catch (error) {
      setResponse(error.message);
      setStatus(`❌ RPC request failed: ${error.message}`);
    }
    setLoading(false);
  };

  // Test basic auth as fallback
  const testBasicAuth = async () => {
    setLoading(true);
    setStatus('Testing basic auth fallback...');
    
    try {
      const basicAuth = {
        username: basicAuthUsername,
        password: basicAuthPassword
      };

      const result = await makeRpcRequest('eth_blockNumber', [], selectedChain, null, basicAuth);
      
      if (result.success) {
        setResponse(result.data);
        setStatus('✅ Basic auth request successful!');
      } else {
        setResponse(result.error);
        setStatus(`❌ Basic auth request failed: ${result.error.status}`);
      }
    } catch (error) {
      setResponse(error.message);
      setStatus(`❌ Basic auth request failed: ${error.message}`);
    }
    setLoading(false);
  };

  const clearResponse = () => {
    setResponse(null);
    setStatus('');
  };

  return (
    <div>
      <h2>🚀 RPC Request Tester</h2>
      
      <div className="card">
        <h3>⚙️ Configuration</h3>
        <label>
          <strong>Chain:</strong>
          <select 
            value={selectedChain} 
            onChange={(e) => setSelectedChain(e.target.value)}
            className="input"
          >
            {AVAILABLE_NETWORKS.map(network => (
              <option key={network.value} value={network.value}>
                {network.label}
              </option>
            ))}
          </select>
        </label>
        
        <p><strong>Token Status:</strong> {token ? '✅ Available' : '❌ No token'}</p>
        {token && (
          <div className="token-display" style={{fontSize: '0.8rem', marginTop: '1rem'}}>
            <strong>Current Token:</strong><br/>
            <div style={{display: 'flex', alignItems: 'center', gap: '10px', marginTop: '5px'}}>
              <textarea 
                readOnly 
                value={token} 
                style={{
                  width: '100%', 
                  minHeight: '60px', 
                  fontSize: '0.7rem', 
                  fontFamily: 'monospace',
                  resize: 'vertical'
                }}
              />
              <button 
                className="button" 
                onClick={() => {
                  navigator.clipboard.writeText(token);
                  setStatus('✅ Token copied to clipboard!');
                }}
                style={{minWidth: '80px', height: 'fit-content'}}
              >
                📋
              </button>
            </div>
          </div>
        )}
      </div>

      <div className="card">
        <h3>📋 Predefined Methods</h3>
        <div style={{display: 'grid', gap: '0.5rem'}}>
          {PREDEFINED_RPC_METHODS.map((item, index) => (
            <button
              key={index}
              className="button"
              onClick={() => testPredefinedMethod(item.method, item.params)}
              disabled={loading}
              style={{
                borderLeft: `4px solid ${item.cacheType === 'permanent' ? '#4CAF50' : 
                                         item.cacheType === 'short' ? '#FF9800' : '#2196F3'}`
              }}
            >
              {item.name} ({item.cacheType})
            </button>
          ))}
        </div>
      </div>

      <div className="card">
        <h3>🔓 Test Basic Auth Fallback</h3>
        <p>Test without JWT token using basic authentication</p>
        <div style={{display: 'grid', gap: '10px', marginBottom: '15px'}}>
          <div>
            <label>
              <strong>Username:</strong>
              <input
                type="text"
                placeholder="Username"
                value={basicAuthUsername}
                onChange={(e) => setBasicAuthUsername(e.target.value)}
                className="input"
              />
            </label>
          </div>
          <div>
            <label>
              <strong>Password:</strong>
              <input
                type="password"
                placeholder="Password"
                value={basicAuthPassword}
                onChange={(e) => setBasicAuthPassword(e.target.value)}
                className="input"
              />
            </label>
          </div>
        </div>
        <button
          className="button"
          onClick={testBasicAuth}
          disabled={loading || !basicAuthUsername || !basicAuthPassword}
        >
          Test Basic Auth
        </button>
      </div>

      {status && (
        <div className="card">
          <h3>📊 Status</h3>
          <p className={status.includes('✅') ? 'success' : status.includes('❌') ? 'error' : 'loading'}>
            {status}
          </p>
        </div>
      )}

      {response && (
        <div className="card">
          <h3>📄 Response</h3>
          <div className="json-display">
            {JSON.stringify(response, null, 2)}
          </div>
          <button className="button" onClick={clearResponse}>
            Clear Response
          </button>
        </div>
      )}
    </div>
  );
};

export default RpcTester; 