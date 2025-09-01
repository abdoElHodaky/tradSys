#!/bin/bash

# Replace timeframe aggregator
mv internal/trading/market_data/timeframe/aggregator_fixed.go internal/trading/market_data/timeframe/aggregator.go

# Replace latency tracker
mv internal/performance/latency/tracker_fixed.go internal/performance/latency/tracker.go

# Replace strategy framework
mv internal/strategy/optimized_framework_fixed.go internal/strategy/optimized_framework.go

# Replace risk interface
mv internal/risk/interface_fixed.go internal/risk/interface.go

# Replace orders interface
mv internal/orders/interface_fixed.go internal/orders/interface.go

# Replace marketdata/external files
mv internal/marketdata/external/manager_fixed.go internal/marketdata/external/manager.go
mv internal/marketdata/external/module_fixed.go internal/marketdata/external/module.go
mv internal/marketdata/external/provider_fixed.go internal/marketdata/external/provider.go
mv internal/**_fixed.go internal/**.go
echo "All files have been replaced with fixed versions."

