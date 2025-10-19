#!/bin/bash

echo "🔄 Updating import paths for TradSys unified structure..."

# Update all Go files to use new import paths
find . -name "*.go" -type f -exec sed -i \
    -e 's|github.com/abdoElHodaky/tradSys/internal/hft/matching|github.com/abdoElHodaky/tradSys/internal/core/matching|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/hft/risk|github.com/abdoElHodaky/tradSys/internal/core/risk|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/hft/settlement|github.com/abdoElHodaky/tradSys/internal/core/settlement|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/trading/connectivity|github.com/abdoElHodaky/tradSys/internal/connectivity|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/trading/compliance|github.com/abdoElHodaky/tradSys/internal/compliance|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/trading/strategies|github.com/abdoElHodaky/tradSys/internal/strategies|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/trading/order_matching|github.com/abdoElHodaky/tradSys/internal/core/matching|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/trading/risk_management|github.com/abdoElHodaky/tradSys/internal/core/risk|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/trading/settlement|github.com/abdoElHodaky/tradSys/internal/core/settlement|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/strategy|github.com/abdoElHodaky/tradSys/internal/strategies|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/hft/monitoring|github.com/abdoElHodaky/tradSys/internal/monitoring|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/config|github.com/abdoElHodaky/tradSys/internal/unified-config|g' \
    {} \;

echo "✅ Import paths updated successfully!"

# Verify the changes
echo "🔍 Verifying updated imports..."
grep -r "github.com/abdoElHodaky/tradSys/internal/core" . --include="*.go" | wc -l | xargs echo "Core imports found:"
grep -r "github.com/abdoElHodaky/tradSys/internal/connectivity" . --include="*.go" | wc -l | xargs echo "Connectivity imports found:"
grep -r "github.com/abdoElHodaky/tradSys/internal/compliance" . --include="*.go" | wc -l | xargs echo "Compliance imports found:"
grep -r "github.com/abdoElHodaky/tradSys/internal/strategies" . --include="*.go" | wc -l | xargs echo "Strategies imports found:"
grep -r "github.com/abdoElHodaky/tradSys/internal/unified-config" . --include="*.go" | wc -l | xargs echo "Unified-config imports found:"

echo "🎉 Import path migration completed!"
