import React, { useState } from 'react';
import './App.css';
import PuzzleSolver from './components/PuzzleSolver';
import RpcTester from './components/RpcTester';

function App() {
  const [token, setToken] = useState('');

  return (
    <div className="App">
      <header className="App-header">
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

export default App; 