#!/bin/bash
# Script to apply linting fixes to the codebase

# Set up environment
export PATH=$PATH:$(pwd)/go/bin:$(pwd)/bin

# Ensure we have golangci-lint installed
if [ ! -f "./bin/golangci-lint" ]; then
  echo "Installing golangci-lint..."
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(pwd)/bin v1.64.8
fi

# Download dependencies
echo "Downloading Go dependencies..."
go mod download

# Create a custom .golangci.yml for specific issues
cat > .golangci-custom.yml << EOF
linters:
  disable-all: true
  enable:
    - gofmt
    - goimports
    - misspell
    - ineffassign
    - staticcheck

linters-settings:
  gofmt:
    simplify: true
  goimports:
    local-prefixes: github.com/abdoElHodaky/tradSys

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gosec
  max-issues-per-linter: 0
  max-same-issues: 0
EOF

# Run linting with fixes on specific directories
echo "Running linting with fixes on internal/db..."
./bin/golangci-lint run --config=.golangci-custom.yml --fix ./internal/db/

echo "Running linting with fixes on internal/strategy..."
./bin/golangci-lint run --config=.golangci-custom.yml --fix ./internal/strategy/

echo "Running linting with fixes on internal/trading..."
./bin/golangci-lint run --config=.golangci-custom.yml --fix ./internal/trading/

# Check for any remaining issues
echo "Checking for remaining issues..."
./bin/golangci-lint run --config=.golangci-custom.yml ./internal/db/ ./internal/strategy/ ./internal/trading/

echo "Linting and fixes completed!"

