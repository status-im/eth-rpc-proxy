#!/usr/bin/env bash

BASE_URL=""

# Parse URL/alias
case ${1:-local} in
    local)
        BASE_URL="http://localhost:8081"
        ;;
    test)
        BASE_URL="https://test.eth-rpc.status.im"
        ;;
    prod)
        BASE_URL="https://prod.eth-rpc.status.im"
        ;;
    *)
        BASE_URL="$1"
        ;;
esac

# Use persistent venv directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="$SCRIPT_DIR/.venv_get_proxy_token"
TEMP_DIR=$(mktemp -d)

# Cleanup function
cleanup() {
    if [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
}

trap cleanup EXIT

# Setup venv
if [ ! -d "$VENV_DIR" ]; then
    python3 -m venv "$VENV_DIR" >/dev/null 2>&1
    source "$VENV_DIR/bin/activate"
    pip install --quiet argon2-cffi requests >/dev/null 2>&1
else
    source "$VENV_DIR/bin/activate"
fi

# Create Python solver
cat > "$TEMP_DIR/solver.py" << 'EOF'
import sys
import requests
import time
from argon2.low_level import hash_secret_raw, Type

BASE_URL = sys.argv[1]

def compute_argon2_hash(challenge, salt, nonce, params):
    input_str = f"{challenge}{salt}{nonce}"
    salt_bytes = bytes.fromhex(salt)
    
    hash_result = hash_secret_raw(
        secret=input_str.encode(),
        salt=salt_bytes,
        time_cost=params['time'],
        memory_cost=params['memory_kb'],
        parallelism=params['threads'],
        hash_len=params['key_len'],
        type=Type.ID
    )
    
    return hash_result.hex()

def solve_puzzle(puzzle):
    challenge = puzzle['challenge']
    salt = puzzle['salt']
    difficulty = puzzle['difficulty']
    params = puzzle['argon2_params']
    
    for nonce in range(1000000):
        hash_val = compute_argon2_hash(challenge, salt, nonce, params)
        
        if hash_val[:difficulty] == '0' * difficulty:
            return {
                'challenge': challenge,
                'salt': salt,
                'nonce': nonce,
                'argon_hash': hash_val,
                'hmac': puzzle['hmac'],
                'expires_at': puzzle['expires_at']
            }
        
        if nonce % 10000 == 0 and nonce > 0:
            print(f"Solving... {nonce//1000}k attempts", file=sys.stderr)
    
    raise Exception("Failed to solve puzzle")

try:
    # Get puzzle
    response = requests.get(f"{BASE_URL}/auth/puzzle", timeout=10)
    puzzle = response.json()
    
    # Solve puzzle
    solution = solve_puzzle(puzzle)
    
    # Submit solution
    response = requests.post(f"{BASE_URL}/auth/solve", json=solution, timeout=30)
    result = response.json()
    
    # Output token
    print(result['token'])
    
except Exception as e:
    print(f"Error: {e}", file=sys.stderr)
    sys.exit(1)
EOF

# Run solver
python3 "$TEMP_DIR/solver.py" "$BASE_URL"