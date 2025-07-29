import React, { useState } from 'react';
import { 
  getPuzzle,
  getAuthStatus,
  verifyToken,
  solvePuzzle,
  submitSolution,
  createDebugSolution,
  generateJwtToken
} from '../utils';

const PuzzleSolver = ({ onTokenGenerated }) => {
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState('');
  const [puzzle, setPuzzle] = useState(null);
  const [solution, setSolution] = useState(null);
  const [token, setToken] = useState('');
  const [solving, setSolving] = useState(false);
  const [verifyResponse, setVerifyResponse] = useState(null);
  const [authStatus, setAuthStatus] = useState(null);

  // Get puzzle from proxy using utils
  const handleGetPuzzle = async () => {
    setLoading(true);
    setStatus('Getting puzzle...');
    
    try {
      const result = await getPuzzle();
      if (result.success) {
        setPuzzle(result.data);
        setStatus('✅ Puzzle received');
      } else {
        setStatus(`❌ Error getting puzzle: ${result.error.message}`);
      }
    } catch (error) {
      setStatus(`❌ Error getting puzzle: ${error.message}`);
    }
    setLoading(false);
  };

  // Test token verification using utils
  const testVerify = async () => {
    if (!token) {
      setStatus('❌ No JWT token available. Generate one first!');
      return;
    }

    setLoading(true);
    setStatus('Testing token verification...');
    
    try {
      const result = await verifyToken(token);
      if (result.success) {
        setVerifyResponse(result.data);
        setStatus('✅ Token verification successful!');
      } else {
        setVerifyResponse(result.error);
        setStatus(`❌ Token verification failed: ${result.error.status}`);
      }
    } catch (error) {
      setStatus(`❌ Token verification error: ${error.message}`);
    }
    setLoading(false);
  };

  // Get auth service status using utils
  const handleGetAuthStatus = async () => {
    setLoading(true);
    setStatus('Getting auth service status...');
    
    try {
      const result = await getAuthStatus();
      if (result.success) {
        setAuthStatus(result.data);
        setStatus('✅ Auth status received');
      } else {
        setAuthStatus({
          error: result.error.message,
          status: result.error.status
        });
        setStatus(`❌ Error getting auth status: ${result.error.message}`);
      }
    } catch (error) {
      setStatus(`❌ Error getting auth status: ${error.message}`);
    }
    setLoading(false);
  };

  // Solve puzzle using utils
  const handleSolvePuzzle = async () => {
    if (!puzzle) {
      setStatus('❌ No puzzle to solve');
      return;
    }

    setSolving(true);
    setStatus('🔍 Solving puzzle...');

    try {
      const onProgress = (attempts, maxAttempts) => {
        setStatus(`🔍 Solving puzzle... Attempt ${attempts}/${maxAttempts}`);
      };

      const result = await solvePuzzle(puzzle, onProgress);
      
      if (result.success) {
        setSolution(result.solution);
        let debugInfo = '';
        if (puzzle.debug_solution) {
          const debugMatches = result.solution.argon_hash === puzzle.debug_solution.argon_hash;
          debugInfo = debugMatches ? 
            ' (✅ Matches debug solution)' : 
            ` (❌ Debug: nonce=${puzzle.debug_solution.nonce}, hash=${puzzle.debug_solution.argon_hash})`;
        }
        setStatus(`✅ Puzzle solved in ${result.solveTime}s! Found valid nonce: ${result.solution.nonce}${debugInfo}`);
        console.log('Generated solution:', JSON.stringify(result.solution, null, 2));
      } else {
        setStatus(`❌ ${result.error.message} (${result.error.solveTime}s)`);
      }
    } catch (error) {
      setStatus(`❌ Error solving puzzle: ${error.message}`);
    }
    setSolving(false);
  };

  // Submit solution using utils
  const handleSubmitSolution = async () => {
    if (!solution) {
      setStatus('❌ No solution to submit');
      return;
    }

    setLoading(true);
    setStatus('Submitting solution...');
    
    try {
      const result = await submitSolution(solution);
      if (result.success) {
        const jwtToken = result.token;
        setToken(jwtToken);
        onTokenGenerated(jwtToken);
        setStatus('✅ JWT token generated successfully!');
      } else {
        setStatus(`❌ Error submitting solution: ${result.error.message}`);
      }
    } catch (error) {
      setStatus(`❌ Error submitting solution: ${error.message}`);
    }
    setLoading(false);
  };

  // Use debug solution
  const handleUseDebugSolution = () => {
    try {
      const debugSolution = createDebugSolution(puzzle);
      setSolution(debugSolution);
      setStatus('✅ Used debug solution with puzzle HMAC');
      console.log('Debug solution:', JSON.stringify(debugSolution, null, 2));
    } catch (error) {
      setStatus(`❌ Error creating debug solution: ${error.message}`);
    }
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
            onClick={handleGetPuzzle} 
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
            onClick={handleGetAuthStatus} 
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
            onClick={handleSolvePuzzle} 
            disabled={loading || solving}
          >
            {solving ? '🔍 Solving...' : '🚀 Solve Puzzle'}
          </button>
          {puzzle.debug_solution && (
            <div style={{marginTop: '10px'}}>
              <button 
                className="button" 
                onClick={handleUseDebugSolution}
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
            onClick={handleSubmitSolution} 
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