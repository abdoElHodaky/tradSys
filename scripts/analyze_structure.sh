#!/bin/bash

echo "🔍 TradSys Deep Structure Analysis"
echo "=================================="

echo ""
echo "📊 DIRECTORY STRUCTURE OVERVIEW"
echo "--------------------------------"

# Count files and lines in each directory
echo "internal/:"
find internal -name "*.go" -not -name "*_test.go" | wc -l | xargs echo "  Files:"
find internal -name "*.go" -not -name "*_test.go" | xargs wc -l | tail -1 | awk '{print "  Lines: " $1}'

echo "pkg/:"
find pkg -name "*.go" -not -name "*_test.go" | wc -l | xargs echo "  Files:"
find pkg -name "*.go" -not -name "*_test.go" | xargs wc -l | tail -1 | awk '{print "  Lines: " $1}'

echo "services/:"
find services -name "*.go" -not -name "*_test.go" 2>/dev/null | wc -l | xargs echo "  Files:"
find services -name "*.go" -not -name "*_test.go" 2>/dev/null | xargs wc -l 2>/dev/null | tail -1 | awk '{print "  Lines: " $1}' || echo "  Lines: 0"

echo ""
echo "🚨 FILE SIZE VIOLATIONS (>500 lines)"
echo "------------------------------------"

echo "CRITICAL (>1000 lines):"
find . -name "*.go" -not -name "*_test.go" -not -path "./vendor/*" | xargs wc -l | awk '$1 > 1000 {print "  🔴 " $2 ": " $1 " lines"}' | head -10

echo ""
echo "HIGH (750-1000 lines):"
find . -name "*.go" -not -name "*_test.go" -not -path "./vendor/*" | xargs wc -l | awk '$1 > 750 && $1 <= 1000 {print "  🟠 " $2 ": " $1 " lines"}' | head -10

echo ""
echo "MEDIUM (500-750 lines):"
find . -name "*.go" -not -name "*_test.go" -not -path "./vendor/*" | xargs wc -l | awk '$1 > 500 && $1 <= 750 {print "  🟡 " $2 ": " $1 " lines"}' | head -15

echo ""
echo "🔗 IMPORT DEPENDENCY ANALYSIS"
echo "-----------------------------"

echo "pkg/ importing internal/ (VIOLATION):"
grep -r "github.com/abdoElHodaky/tradSys/internal" pkg/ 2>/dev/null | wc -l | xargs echo "  Count:"
grep -r "github.com/abdoElHodaky/tradSys/internal" pkg/ 2>/dev/null | head -5

echo ""
echo "internal/ importing services/ (SHOULD ELIMINATE):"
grep -r "github.com/abdoElHodaky/tradSys/services" internal/ 2>/dev/null | wc -l | xargs echo "  Count:"

echo ""
echo "pkg/ importing services/ (SHOULD ELIMINATE):"
grep -r "github.com/abdoElHodaky/tradSys/services" pkg/ 2>/dev/null | wc -l | xargs echo "  Count:"

echo ""
echo "📦 SERVICES DIRECTORY ANALYSIS (NON-STANDARD)"
echo "---------------------------------------------"

if [ -d "services" ]; then
    echo "Services packages requiring relocation:"
    find services -type d -mindepth 1 -maxdepth 1 | while read dir; do
        files=$(find "$dir" -name "*.go" -not -name "*_test.go" | wc -l)
        lines=$(find "$dir" -name "*.go" -not -name "*_test.go" | xargs wc -l 2>/dev/null | tail -1 | awk '{print $1}' || echo "0")
        echo "  $dir: $files files, $lines lines"
    done
else
    echo "  No services directory found"
fi

echo ""
echo "🎯 REORGANIZATION PRIORITIES"
echo "----------------------------"

echo "1. CRITICAL - Services Directory Elimination:"
if [ -d "services" ]; then
    find services -name "*.go" -not -name "*_test.go" | wc -l | xargs echo "   Files to relocate:"
    find services -name "*.go" -not -name "*_test.go" | xargs wc -l 2>/dev/null | tail -1 | awk '{print "   Lines to relocate: " $1}' || echo "   Lines to relocate: 0"
fi

echo ""
echo "2. HIGH - File Size Violations:"
find . -name "*.go" -not -name "*_test.go" -not -path "./vendor/*" | xargs wc -l | awk '$1 > 500' | wc -l | xargs echo "   Files exceeding 500 lines:"

echo ""
echo "3. MEDIUM - Import Path Cleanup:"
grep -r "github.com/abdoElHodaky/tradSys" . --include="*.go" | grep -v "_test.go" | wc -l | xargs echo "   Total internal imports to review:"

echo ""
echo "📈 ESTIMATED EFFORT BREAKDOWN"
echo "-----------------------------"

# Calculate rough estimates
services_files=$(find services -name "*.go" -not -name "*_test.go" 2>/dev/null | wc -l || echo "0")
large_files=$(find . -name "*.go" -not -name "*_test.go" -not -path "./vendor/*" | xargs wc -l | awk '$1 > 500' | wc -l)
import_refs=$(grep -r "github.com/abdoElHodaky/tradSys" . --include="*.go" | grep -v "_test.go" | wc -l)

echo "Services elimination: $((services_files * 2)) hours"
echo "File splitting: $((large_files * 2)) hours"  
echo "Import cleanup: $((import_refs / 10)) hours"
echo "Testing & validation: 12 hours"
echo "TOTAL ESTIMATED: $(((services_files * 2) + (large_files * 2) + (import_refs / 10) + 12)) hours"

echo ""
echo "✅ ANALYSIS COMPLETE"

