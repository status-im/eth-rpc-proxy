#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export LUA_PATH="$SCRIPT_DIR/../lua/?.lua;$SCRIPT_DIR/../lua/?/init.lua;$SCRIPT_DIR/../lua/cache/?.lua;$SCRIPT_DIR/../lua/auth/?.lua;$SCRIPT_DIR/../lua/providers/?.lua;$SCRIPT_DIR/../lua/utils/?.lua;$SCRIPT_DIR/?.lua;$SCRIPT_DIR/?/init.lua;;"

echo "Running Lua tests..."

total_tests=0
passed_tests=0

for test_file in $(find "$SCRIPT_DIR" -name "*_test.lua" -type f); do
    total_tests=$((total_tests + 1))
    test_dir=$(dirname "$test_file")
    cd "$test_dir"
    
    if lua "$(basename "$test_file")"; then
        echo "✓ $(basename "$test_file") - PASSED"
        passed_tests=$((passed_tests + 1))
    else
        echo "✗ $(basename "$test_file") - FAILED"
        exit 1
    fi
done

echo ""
echo "========================================"
echo "Tests completed: $passed_tests/$total_tests passed"
echo "========================================" 