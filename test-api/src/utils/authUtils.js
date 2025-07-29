import axios from 'axios';
import { argon2id } from 'hash-wasm';
import CryptoJS from 'crypto-js';
import { getProxyUrl, getApiConfig, handleApiError } from './apiUtils';

/**
 * Get a puzzle from the auth service
 * @returns {Promise<Object>} Puzzle object or error
 */
export const getPuzzle = async () => {
  try {
    const proxyUrl = getProxyUrl();
    const response = await axios.get(`${proxyUrl}/auth/puzzle`, getApiConfig());
    return { success: true, data: response.data };
  } catch (error) {
    return { success: false, error: handleApiError(error) };
  }
};

/**
 * Get auth service status
 * @returns {Promise<Object>} Status object or error
 */
export const getAuthStatus = async () => {
  try {
    const proxyUrl = getProxyUrl();
    const response = await axios.get(`${proxyUrl}/auth/status`, getApiConfig());
    return { success: true, data: response.data };
  } catch (error) {
    return { success: false, error: handleApiError(error) };
  }
};

/**
 * Verify a JWT token
 * @param {string} token - JWT token to verify
 * @returns {Promise<Object>} Verification result or error
 */
export const verifyToken = async (token) => {
  try {
    const proxyUrl = getProxyUrl();
    const response = await axios.get(`${proxyUrl}/auth/verify`, getApiConfig(token));
    return { 
      success: true, 
      data: {
        status: response.status,
        headers: response.headers,
        data: response.data || 'Token valid'
      }
    };
  } catch (error) {
    return { 
      success: false, 
      error: {
        status: error.response?.status || 'Network Error',
        headers: error.response?.headers || {},
        data: error.response?.data || error.message
      }
    };
  }
};

/**
 * Submit a puzzle solution to get JWT token
 * @param {Object} solution - Puzzle solution object
 * @returns {Promise<Object>} JWT token result or error
 */
export const submitSolution = async (solution) => {
  try {
    const proxyUrl = getProxyUrl();
    const response = await axios.post(`${proxyUrl}/auth/solve`, solution, getApiConfig());
    return { success: true, token: response.data.token };
  } catch (error) {
    return { success: false, error: handleApiError(error) };
  }
};

/**
 * Check if hash meets difficulty requirement
 * @param {string} hash - Hash to check
 * @param {number} difficulty - Required difficulty (number of leading zeros)
 * @returns {boolean} Whether hash meets difficulty
 */
export const checkDifficulty = (hash, difficulty) => {
  if (hash.length < difficulty) return false;
  for (let i = 0; i < difficulty; i++) {
    if (hash[i] !== '0') return false;
  }
  return true;
};

/**
 * Compute HMAC-SHA256
 * @param {string} data - Data to hash
 * @param {string} secret - Secret key
 * @returns {string} HMAC hex string
 */
export const computeHMAC = (data, secret) => {
  const hmac = CryptoJS.HmacSHA256(data, secret);
  return hmac.toString(CryptoJS.enc.Hex);
};

/**
 * Convert hex string to Uint8Array
 * @param {string} hex - Hex string
 * @returns {Uint8Array} Byte array
 */
export const hexToUint8Array = (hex) => {
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.substr(i, 2), 16);
  }
  return bytes;
};

/**
 * Solve puzzle using Argon2
 * @param {Object} puzzle - Puzzle object from auth service
 * @param {Function} onProgress - Progress callback (optional)
 * @param {number} maxAttempts - Maximum attempts (default: 100000)
 * @returns {Promise<Object>} Solution object or error
 */
export const solvePuzzle = async (puzzle, onProgress = null, maxAttempts = 100000) => {
  const startTime = Date.now();
  
  try {
    const { challenge, salt, difficulty, argon2_params } = puzzle;
    
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
          const endTime = Date.now();
          const solveTime = ((endTime - startTime) / 1000).toFixed(2);
          
          const solution = {
            challenge,
            salt,
            nonce,
            argon_hash: argonHash,
            hmac: puzzle.hmac, // Always use puzzle HMAC
            expires_at: puzzle.expires_at
          };

          return { 
            success: true, 
            solution, 
            solveTime: parseFloat(solveTime),
            attempts: nonce + 1
          };
        }
      } catch (error) {
        console.error('Argon2 computation error:', error);
        continue;
      }

      // Report progress every 1000 attempts
      if (nonce % 1000 === 0 && onProgress) {
        onProgress(nonce, maxAttempts);
        // Allow UI to update
        await new Promise(resolve => setTimeout(resolve, 1));
      }
    }

    const endTime = Date.now();
    const solveTime = ((endTime - startTime) / 1000).toFixed(2);
    return { 
      success: false, 
      error: { 
        message: `Failed to solve puzzle within ${maxAttempts} attempts`, 
        solveTime: parseFloat(solveTime)
      }
    };
  } catch (error) {
    const endTime = Date.now();
    const solveTime = ((endTime - startTime) / 1000).toFixed(2);
    return { 
      success: false, 
      error: { 
        message: error.message, 
        solveTime: parseFloat(solveTime)
      }
    };
  }
};

/**
 * Create solution from debug data
 * @param {Object} puzzle - Puzzle object
 * @returns {Object} Debug solution object
 */
export const createDebugSolution = (puzzle) => {
  if (!puzzle.debug_solution) {
    throw new Error('No debug solution available');
  }

  return {
    challenge: puzzle.challenge,
    salt: puzzle.salt,
    nonce: puzzle.debug_solution.nonce,
    argon_hash: puzzle.debug_solution.argon_hash,
    hmac: puzzle.hmac, // Always use puzzle HMAC
    expires_at: puzzle.expires_at
  };
};

/**
 * Complete puzzle solving and token generation workflow
 * @param {Function} onProgress - Progress callback for puzzle solving
 * @param {Function} onStatusUpdate - Status update callback
 * @returns {Promise<Object>} JWT token result or error
 */
export const generateJwtToken = async (onProgress = null, onStatusUpdate = null) => {
  try {
    // Step 1: Get puzzle
    if (onStatusUpdate) onStatusUpdate('Getting puzzle...');
    const puzzleResult = await getPuzzle();
    if (!puzzleResult.success) {
      return { success: false, error: puzzleResult.error };
    }

    // Step 2: Solve puzzle
    if (onStatusUpdate) onStatusUpdate('Solving puzzle...');
    const solveResult = await solvePuzzle(puzzleResult.data, onProgress);
    if (!solveResult.success) {
      return { success: false, error: solveResult.error };
    }

    // Step 3: Submit solution
    if (onStatusUpdate) onStatusUpdate('Submitting solution...');
    const submitResult = await submitSolution(solveResult.solution);
    if (!submitResult.success) {
      return { success: false, error: submitResult.error };
    }

    if (onStatusUpdate) onStatusUpdate('JWT token generated successfully!');
    return { 
      success: true, 
      token: submitResult.token,
      puzzle: puzzleResult.data,
      solution: solveResult.solution,
      solveTime: solveResult.solveTime,
      attempts: solveResult.attempts
    };
  } catch (error) {
    return { success: false, error: { message: error.message } };
  }
};