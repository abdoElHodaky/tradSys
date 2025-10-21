#!/bin/bash

echo "🔄 Updating HFT import paths to trading..."

# Update all Go files to use trading instead of hft
find . -name "*.go" -type f -exec sed -i \
    -e 's|github.com/abdoElHodaky/tradSys/internal/hft/config|github.com/abdoElHodaky/tradSys/internal/trading/config|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/hft/memory|github.com/abdoElHodaky/tradSys/internal/trading/memory|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/hft/metrics|github.com/abdoElHodaky/tradSys/internal/trading/metrics|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/hft/middleware|github.com/abdoElHodaky/tradSys/internal/trading/middleware|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/hft/pools|github.com/abdoElHodaky/tradSys/internal/trading/pools|g' \
    {} \;

echo "✅ HFT import paths updated successfully!"

# Verify the changes
echo "🔍 Verifying updated imports..."
echo "Remaining HFT imports:"
grep -r "github.com/abdoElHodaky/tradSys/internal/hft" . --include="*.go" || echo "None found!"

echo "🎉 HFT to trading migration completed!"
