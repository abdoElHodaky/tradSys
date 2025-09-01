#!/bin/bash
# Script to fix duplicate file issues in the codebase

# Set up environment
export PATH=$PATH:$(pwd)/go/bin:$(pwd)/bin

echo "Fixing duplicate files in the codebase..."

# Fix duplicate files in internal/db
echo "Fixing duplicate files in internal/db..."

# Rename connection_pool_fixed.go to connection_pool.go.new
if [ -f "internal/db/connection_pool_fixed.go" ] && [ -f "internal/db/connection_pool.go" ]; then
  echo "Merging connection_pool_fixed.go into connection_pool.go..."
  # Create a backup of the original file
  cp internal/db/connection_pool.go internal/db/connection_pool.go.bak
  # Replace the original with the fixed version
  cp internal/db/connection_pool_fixed.go internal/db/connection_pool.go
  # Remove the fixed version
  rm internal/db/connection_pool_fixed.go
fi

# Rename query_cache_fixed.go to query_cache.go.new
if [ -f "internal/db/query_cache_fixed.go" ] && [ -f "internal/db/query_cache.go" ]; then
  echo "Merging query_cache_fixed.go into query_cache.go..."
  # Create a backup of the original file
  cp internal/db/query_cache.go internal/db/query_cache.go.bak
  # Replace the original with the fixed version
  cp internal/db/query_cache_fixed.go internal/db/query_cache.go
  # Remove the fixed version
  rm internal/db/query_cache_fixed.go
fi

# Fix duplicate files in proto/orders
if [ -f "proto/orders/orders_fixed.go" ] && [ -f "proto/orders/orders.go" ]; then
  echo "Merging orders_fixed.go into orders.go..."
  # Create a backup of the original file
  cp proto/orders/orders.go proto/orders/orders.go.bak
  # Replace the original with the fixed version
  cp proto/orders/orders_fixed.go proto/orders/orders.go
  # Remove the fixed version
  rm proto/orders/orders_fixed.go
fi

echo "Duplicate files fixed!"

# Run linting again to check for remaining issues
echo "Running linting again to check for remaining issues..."
./bin/golangci-lint run --config=.golangci-custom.yml ./internal/db/

echo "Fix complete!"

