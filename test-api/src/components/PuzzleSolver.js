import React, { useState } from 'react';
import axios from 'axios';
import { argon2id } from 'hash-wasm';
import CryptoJS from 'crypto-js';

const PuzzleSolver = ({ onTokenGenerated }) => {
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState('');
  const [puzzle, setPuzzle] = useState(null);
  const [solution, setSolution] = useState(null);
  const [token, setToken] = useState('');
  const [solving, setSolving] = useState(false);
  const [verifyResponse, setVerifyResponse] = useState(null);
  const [authStatus, setAuthStatus] = useState(null);

  // Get puzzle from proxy
  const getPuzzle = async () => {
    setLoading(true);
    setStatus('Getting puzzle...');
    try {
      const proxyUrl = process.env.REACT_APP_RPC_PROXY_URL || 'http://localhost:8080';
      const response = await axios.get(`${proxyUrl}/auth/puzzle`);
      setPuzzle(response.data);
      setStatus('✅ Puzzle received');
    } catch (error) {
      setStatus(`❌ Error getting puzzle: ${error.response?.data || error.message}`);
    }
    setLoading(false);
  };

  // Test token verification
  const testVerify = async () => {
    if (!token) {
      setStatus('❌ No JWT token available. Generate one first!');
      return;
    }

    setLoading(true);
    setStatus('Testing token verification...');
    try {
      const proxyUrl = process.env.REACT_APP_RPC_PROXY_URL || 'http://localhost:8080';
      const response = await axios.get(`${proxyUrl}/auth/verify`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });
      
      setVerifyResponse({
        status: response.status,
        headers: response.headers,
        data: response.data || 'Token valid'
      });
      setStatus('✅ Token verification successful!');
    } catch (error) {
      setVerifyResponse({
        status: error.response?.status || 'Network Error',
        headers: error.response?.headers || {},
        data: error.response?.data || error.message
      });
      setStatus(`❌ Token verification failed: ${error.response?.status || 'Network Error'}`);
    }
    setLoading(false);
  };

  // Get auth service status
  const getAuthStatus = async () => {
    setLoading(true);
    setStatus('Getting auth service status...');
    try {
      const proxyUrl = process.env.REACT_APP_RPC_PROXY_URL || 'http://localhost:8080';
      const response = await axios.get(`${proxyUrl}/auth/status`);
      setAuthStatus(response.data);
      setStatus('✅ Auth status received');
    } catch (error) {
      setAuthStatus({
        error: error.response?.data || error.message,
        status: error.response?.status || 'Network Error'
      });
      setStatus(`❌ Error getting auth status: ${error.response?.data || error.message}`);
    }
    setLoading(false);
  };

  // Check if hash meets difficulty requirement
  const checkDifficulty = (hash, difficulty) => {
    if (hash.length < difficulty) return false;
    for (let i = 0; i < difficulty; i++) {
      if (hash[i] !== '0') return false;
    }
    return true;
  };

  // Compute HMAC-SHA256
  const computeHMAC = (data, secret) => {
    const hmac = CryptoJS.HmacSHA256(data, secret);
    return hmac.toString(CryptoJS.enc.Hex);
  };

  // Convert hex string to Uint8Array
  const hexToUint8Array = (hex) => {
    const bytes = new Uint8Array(hex.length / 2);
    for (let i = 0; i < hex.length; i += 2) {
      bytes[i / 2] = parseInt(hex.substr(i, 2), 16);
    }
    return bytes;
  };

  // Solve puzzle using Argon2
  const solvePuzzle = async () => {
    if (!puzzle) {
      setStatus('❌ No puzzle to solve');
      return;
    }

    setSolving(true);
    setStatus('🔍 Solving puzzle...');
    const startTime = Date.now();

    try {
      const { challenge, salt, difficulty, argon2_params, hmac } = puzzle;
      const maxAttempts = 100000;
      
      // Convert salt from hex to Uint8Array
      const saltBytes = hexToUint8Array(salt);

      for (let nonce = 0; nonce < maxAttempts; nonce++) {
        // Create input: challenge + salt + nonce
        const input = `${challenge}${salt}${nonce}`;
        
        try {
          // Compute Argon2id hash using hash-wasm
          const argonHash = await argon2id({
            password: input,
            salt: saltBytes,
            parallelism: argon2_params.threads,
            iterations: argon2_params.time,
            memorySize: argon2_params.memory_kb,
            hashLength: argon2_params.key_len,
            outputType: 'hex'
          });

          // Check if this hash meets the difficulty requirement
          if (checkDifficulty(argonHash, difficulty)) {
            // We found a valid nonce! Use puzzle HMAC
            const endTime = Date.now();
            const solveTime = ((endTime - startTime) / 1000).toFixed(2);
            
            const solutionData = {
              challenge,
              salt,
              nonce,
              argon_hash: argonHash,
              hmac: puzzle.hmac, // Always use puzzle HMAC
              expires_at: puzzle.expires_at
            };

            // Compare with debug solution if available
            let debugInfo = '';
            if (puzzle.debug_solution) {
              const debugMatches = argonHash === puzzle.debug_solution.argon_hash;
              debugInfo = debugMatches ? 
                ' (✅ Matches debug solution)' : 
                ` (❌ Debug: nonce=${puzzle.debug_solution.nonce}, hash=${puzzle.debug_solution.argon_hash})`;
            }

            setSolution(solutionData);
            setStatus(`✅ Puzzle solved in ${solveTime}s! Found valid nonce: ${nonce}${debugInfo}`);
            console.log('Generated solution:', JSON.stringify(solutionData, null, 2));
            setSolving(false);
            return;
          }
        } catch (error) {
          console.error('Argon2 computation error:', error);
          continue;
        }

        // Update status every 1000 attempts
        if (nonce % 1000 === 0) {
          setStatus(`🔍 Solving puzzle... Attempt ${nonce}/${maxAttempts}`);
          // Allow UI to update
          await new Promise(resolve => setTimeout(resolve, 1));
        }
      }

      const endTime = Date.now();
      const solveTime = ((endTime - startTime) / 1000).toFixed(2);
      setStatus(`❌ Failed to solve puzzle within attempt limit (${solveTime}s)`);
    } catch (error) {
      const endTime = Date.now();
      const solveTime = ((endTime - startTime) / 1000).toFixed(2);
      setStatus(`❌ Error solving puzzle after ${solveTime}s: ${error.message}`);
    }
    setSolving(false);
  };

  // Submit solution
  const submitSolution = async () => {
    if (!solution) {
      setStatus('❌ No solution to submit');
      return;
    }

    setLoading(true);
    setStatus('Submitting solution...');
    try {
      const proxyUrl = process.env.REACT_APP_RPC_PROXY_URL || 'http://localhost:8080';
      const response = await axios.post(`${proxyUrl}/auth/solve`, solution, {
        headers: {
          'Content-Type': 'application/json'
        }
      });
      
      const jwtToken = response.data.token;
      setToken(jwtToken);
      onTokenGenerated(jwtToken);
      setStatus('✅ JWT token generated successfully!');
    } catch (error) {
      setStatus(`❌ Error submitting solution: ${error.response?.data || error.message}`);
    }
    setLoading(false);
  };

  // Reset everything
  const reset = () => {
    setPuzzle(null);
    setSolution(null);
    setToken('');
    setStatus('');
    setVerifyResponse(null);
    setAuthStatus(null);
    onTokenGenerated('');
  };

  return (
    <div>
      <h2>🧩 Puzzle Solver & Auth Testing</h2>
      <p>Solves puzzles automatically using Argon2 in the browser</p>
      
      <div className="card">
        <h3>🔧 Auth Service Controls</h3>
        <div style={{display: 'grid', gap: '0.5rem', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))'}}>
          <button 
            className="button" 
            onClick={getPuzzle} 
            disabled={loading || solving}
          >
            Get Puzzle
          </button>
          <button 
            className="button" 
            onClick={testVerify} 
            disabled={loading || solving || !token}
          >
            Test Token Verify
          </button>
          <button 
            className="button" 
            onClick={getAuthStatus} 
            disabled={loading || solving}
          >
            Get Auth Status
          </button>
        </div>
      </div>

      {puzzle && (
        <div className="card">
          <h3>📋 Puzzle Details</h3>
          <div className="json-display">
            {JSON.stringify(puzzle, null, 2)}
          </div>
          <button 
            className="button" 
            onClick={solvePuzzle} 
            disabled={loading || solving}
          >
            {solving ? '🔍 Solving...' : '🚀 Solve Puzzle'}
          </button>
          {puzzle.debug_solution && (
            <div style={{marginTop: '10px'}}>
              <button 
                className="button" 
                onClick={() => {
                  const debugSolution = {
                    challenge: puzzle.challenge,
                    salt: puzzle.salt,
                    nonce: puzzle.debug_solution.nonce,
                    argon_hash: puzzle.debug_solution.argon_hash,
                    hmac: puzzle.hmac, // Always use puzzle HMAC
                    expires_at: puzzle.expires_at
                  };
                  setSolution(debugSolution);
                  setStatus('✅ Used debug solution with puzzle HMAC');
                  console.log('Debug solution:', JSON.stringify(debugSolution, null, 2));
                }}
                disabled={loading || solving}
                style={{backgroundColor: '#e74c3c', marginLeft: '10px'}}
              >
                🐛 Use Debug Solution
              </button>
            </div>
          )}
        </div>
      )}

      {solution && (
        <div className="card">
          <h3>🔑 Solution</h3>
          <div className="json-display">
            {JSON.stringify(solution, null, 2)}
          </div>
          
          <button 
            className="button" 
            onClick={submitSolution} 
            disabled={loading}
          >
            Submit Solution & Get Token
          </button>
        </div>
      )}



      {status && (
        <div className="card">
          <h3>📊 Status</h3>
          <p className={status.includes('✅') ? 'success' : status.includes('❌') ? 'error' : 'loading'}>
            {status}
          </p>
        </div>
      )}

      {verifyResponse && (
        <div className="card">
          <h3>🔍 Token Verification Result</h3>
          <div className="json-display">
            <p><strong>Status:</strong> {verifyResponse.status}</p>
            <p><strong>Rate Limit:</strong> {verifyResponse.headers['x-ratelimit-remaining'] || 'N/A'} / {verifyResponse.headers['x-ratelimit-limit'] || 'N/A'}</p>
            <pre>{JSON.stringify(verifyResponse.data, null, 2)}</pre>
          </div>
        </div>
      )}

      {authStatus && (
        <div className="card">
          <h3>📊 Auth Service Status</h3>
          <div className="json-display">
            {JSON.stringify(authStatus, null, 2)}
          </div>
        </div>
      )}

      <button className="button" onClick={reset} disabled={loading || solving}>
        🔄 Reset
      </button>
    </div>
  );
};

export default PuzzleSolver; 