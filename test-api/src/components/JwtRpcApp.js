import React, { useState } from 'react';
import PuzzleSolver from './PuzzleSolver';
import RpcTester from './RpcTester';

function JwtRpcApp({ onBackToHome }) {
  const [token, setToken] = useState('');

  return (
    <div className="App">
      <header className="App-header">
        <div className="app-header-nav">
          <button className="button back-button" onClick={onBackToHome}>
            ← Back to Dashboard
          </button>
        </div>
        <h1>🔐 JWT RPC Test Application</h1>
        <p>Test JWT authentication and RPC requests to the proxy</p>
      </header>
      
      <main className="App-main">
        <div className="container">
          <div className="section">
            <PuzzleSolver onTokenGenerated={setToken} />
          </div>
          
          <div className="section">
            <RpcTester token={token} />
          </div>
        </div>
      </main>
    </div>
  );
}

export default JwtRpcApp;