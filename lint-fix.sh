#!/bin/bash
set -e

# Update dependencies
go mod tidy

# Run linter
echo "Running linter..."
./bin/golangci-lint run ./...

# If linting fails, try to fix automatically
if [ $? -ne 0 ]; then
  echo "Linting failed, attempting to fix issues..."
  ./bin/golangci-lint run --fix ./...
  
  # Run tests to verify fixes didn't break anything
  go test ./...
  
  # If tests pass, commit the changes
  if [ $? -eq 0 ]; then
    git add .
    git commit -m "Fix linting issues with golangci-lint"
    echo "Linting issues fixed and committed."
  else
    echo "Tests failed after fixing linting issues. Manual intervention required."
    exit 1
  fi
fi

