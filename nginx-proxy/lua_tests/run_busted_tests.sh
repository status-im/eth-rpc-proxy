#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Check if busted is installed
if ! command -v busted &> /dev/null; then
    echo "Busted is not installed. Please install it first:"
    echo "  luarocks install busted"
    echo ""
    echo "Or to install it locally in the project:"
    echo "  luarocks install --local busted"
    exit 1
fi

echo "Running Busted tests..."
echo "========================================"

# Change to project directory where .busted config is located
cd "$PROJECT_DIR"

# Run busted with configuration
echo "Running tests with Busted v$(busted --version)..."
echo ""
if busted "$@"; then
    echo ""
    echo "✓ All tests passed!"
    exit 0
else
    echo ""
    echo "✗ Some tests failed!"
    exit 1
fi 