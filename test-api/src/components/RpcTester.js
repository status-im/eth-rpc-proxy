import React, { useState } from 'react';
import axios from 'axios';

const RpcTester = ({ token }) => {
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState('');
  const [response, setResponse] = useState(null);
  const [selectedChain, setSelectedChain] = useState('ethereum/mainnet');
  const [basicAuthUsername, setBasicAuthUsername] = useState('');
  const [basicAuthPassword, setBasicAuthPassword] = useState('');

  const predefinedMethods = [
    { name: 'Get Block Number', method: 'eth_blockNumber', params: [] },
    { name: 'Get Gas Price', method: 'eth_gasPrice', params: [] },
    { name: 'Get Chain ID', method: 'eth_chainId', params: [] },
    { name: 'Get Latest Block', method: 'eth_getBlockByNumber', params: ['latest', false] },
  ];

  const chains = [
    'ethereum/mainnet',
    'ethereum/sepolia', 
    'optimism/mainnet',
    'optimism/sepolia'
  ];

  // Test with predefined method
  const testPredefinedMethod = async (method, params) => {
    if (!token) {
      setStatus('❌ No JWT token available. Generate one first!');
      return;
    }

    setLoading(true);
    setStatus(`Testing ${method}...`);
    
    const rpcRequest = {
      jsonrpc: '2.0',
      method: method,
      params: params,
      id: Date.now()
    };

    try {
      const proxyUrl = process.env.REACT_APP_RPC_PROXY_URL || 'http://localhost:8080';
      const axiosResponse = await axios.post(`${proxyUrl}/${selectedChain}`, rpcRequest, {
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        }
      });
      
      setResponse(axiosResponse.data);
      setStatus('✅ RPC request successful!');
    } catch (error) {
      const errorData = error.response?.data || error.message;
      setResponse(errorData);
      setStatus(`❌ RPC request failed: ${error.response?.status || 'Network Error'}`);
    }
    setLoading(false);
  };

  // Test basic auth as fallback
  const testBasicAuth = async () => {
    setLoading(true);
    setStatus('Testing basic auth fallback...');
    
    const rpcRequest = {
      jsonrpc: '2.0',
      method: 'eth_blockNumber',
      params: [],
      id: Date.now()
    };

    try {
      const proxyUrl = process.env.REACT_APP_RPC_PROXY_URL || 'http://localhost:8080';
      const axiosResponse = await axios.post(`${proxyUrl}/${selectedChain}`, rpcRequest, {
        headers: {
          'Content-Type': 'application/json'
        },
        auth: {
          username: basicAuthUsername,
          password: basicAuthPassword
        }
      });
      
      setResponse(axiosResponse.data);
      setStatus('✅ Basic auth request successful!');
    } catch (error) {
      const errorData = error.response?.data || error.message;
      setResponse(errorData);
      setStatus(`❌ Basic auth request failed: ${error.response?.status || 'Network Error'}`);
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
            {chains.map(chain => (
              <option key={chain} value={chain}>{chain}</option>
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
          {predefinedMethods.map((item, index) => (
            <button
              key={index}
              className="button"
              onClick={() => testPredefinedMethod(item.method, item.params)}
              disabled={loading}
            >
              {item.name}
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