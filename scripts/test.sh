#!/bin/bash

# TradSys Test Runner Script
# Comprehensive testing framework for all test types

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COVERAGE_THRESHOLD=80
BENCHMARK_DURATION="30s"
LOAD_TEST_DURATION="5m"
STRESS_TEST_DURATION="10m"

# Directories
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COVERAGE_DIR="$PROJECT_ROOT/coverage"
BENCHMARK_DIR="$PROJECT_ROOT/benchmarks"
REPORTS_DIR="$PROJECT_ROOT/test-reports"

# Create directories
mkdir -p "$COVERAGE_DIR" "$BENCHMARK_DIR" "$REPORTS_DIR"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "\n${BLUE}================================${NC}"
    echo -e "${BLUE} $1${NC}"
    echo -e "${BLUE}================================${NC}\n"
}

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    if ! command -v golangci-lint &> /dev/null; then
        log_warning "golangci-lint not found, installing..."
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    fi
    
    if ! command -v gosec &> /dev/null; then
        log_warning "gosec not found, installing..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    fi
    
    log_success "Dependencies checked"
}

# Run unit tests
run_unit_tests() {
    print_header "RUNNING UNIT TESTS"
    
    log_info "Running unit tests with coverage..."
    go test -v -race -coverprofile="$COVERAGE_DIR/unit.out" \
        -covermode=atomic \
        -timeout=10m \
        ./tests/unit/... \
        ./internal/... \
        ./pkg/... \
        ./services/... 2>&1 | tee "$REPORTS_DIR/unit-tests.log"
    
    if [ $? -eq 0 ]; then
        log_success "Unit tests passed"
    else
        log_error "Unit tests failed"
        return 1
    fi
    
    # Generate coverage report
    if [ -f "$COVERAGE_DIR/unit.out" ]; then
        go tool cover -html="$COVERAGE_DIR/unit.out" -o "$COVERAGE_DIR/unit.html"
        
        # Check coverage threshold
        COVERAGE=$(go tool cover -func="$COVERAGE_DIR/unit.out" | tail -1 | awk '{print $3}' | sed 's/%//')
        log_info "Unit test coverage: ${COVERAGE}%"
        
        if (( $(echo "$COVERAGE >= $COVERAGE_THRESHOLD" | bc -l) )); then
            log_success "Coverage threshold met (${COVERAGE}% >= ${COVERAGE_THRESHOLD}%)"
        else
            log_warning "Coverage below threshold (${COVERAGE}% < ${COVERAGE_THRESHOLD}%)"
        fi
    fi
}

# Run integration tests
run_integration_tests() {
    print_header "RUNNING INTEGRATION TESTS"
    
    log_info "Running integration tests..."
    go test -v -race -coverprofile="$COVERAGE_DIR/integration.out" \
        -covermode=atomic \
        -timeout=30m \
        ./tests/integration/... 2>&1 | tee "$REPORTS_DIR/integration-tests.log"
    
    if [ $? -eq 0 ]; then
        log_success "Integration tests passed"
    else
        log_error "Integration tests failed"
        return 1
    fi
    
    # Generate coverage report
    if [ -f "$COVERAGE_DIR/integration.out" ]; then
        go tool cover -html="$COVERAGE_DIR/integration.out" -o "$COVERAGE_DIR/integration.html"
        log_info "Integration test coverage report generated"
    fi
}

# Run performance tests
run_performance_tests() {
    print_header "RUNNING PERFORMANCE TESTS"
    
    log_info "Running performance benchmarks..."
    go test -v -bench=. -benchmem -benchtime="$BENCHMARK_DURATION" \
        -cpuprofile="$BENCHMARK_DIR/cpu.prof" \
        -memprofile="$BENCHMARK_DIR/mem.prof" \
        -timeout=1h \
        ./tests/performance/... 2>&1 | tee "$REPORTS_DIR/performance-tests.log"
    
    if [ $? -eq 0 ]; then
        log_success "Performance tests completed"
    else
        log_error "Performance tests failed"
        return 1
    fi
    
    # Generate benchmark report
    log_info "Generating performance profiles..."
    if [ -f "$BENCHMARK_DIR/cpu.prof" ]; then
        go tool pprof -http=:8080 -no_browser "$BENCHMARK_DIR/cpu.prof" &
        PPROF_PID=$!
        log_info "CPU profile server started at http://localhost:8080 (PID: $PPROF_PID)"
        echo "$PPROF_PID" > "$BENCHMARK_DIR/pprof.pid"
    fi
}

# Run compliance tests
run_compliance_tests() {
    print_header "RUNNING COMPLIANCE TESTS"
    
    log_info "Running compliance validation tests..."
    go test -v -race -timeout=20m \
        ./tests/compliance/... 2>&1 | tee "$REPORTS_DIR/compliance-tests.log"
    
    if [ $? -eq 0 ]; then
        log_success "Compliance tests passed"
    else
        log_error "Compliance tests failed"
        return 1
    fi
}

# Run end-to-end tests
run_e2e_tests() {
    print_header "RUNNING END-TO-END TESTS"
    
    log_info "Running end-to-end tests..."
    go test -v -timeout=45m \
        ./tests/e2e/... 2>&1 | tee "$REPORTS_DIR/e2e-tests.log"
    
    if [ $? -eq 0 ]; then
        log_success "End-to-end tests passed"
    else
        log_error "End-to-end tests failed"
        return 1
    fi
}

# Run load tests
run_load_tests() {
    print_header "RUNNING LOAD TESTS"
    
    log_info "Running load tests (duration: $LOAD_TEST_DURATION)..."
    go test -v -timeout="$LOAD_TEST_DURATION" \
        -run="TestLoad" \
        ./tests/performance/load/... 2>&1 | tee "$REPORTS_DIR/load-tests.log"
    
    if [ $? -eq 0 ]; then
        log_success "Load tests completed"
    else
        log_error "Load tests failed"
        return 1
    fi
}

# Run stress tests
run_stress_tests() {
    print_header "RUNNING STRESS TESTS"
    
    log_info "Running stress tests (duration: $STRESS_TEST_DURATION)..."
    go test -v -timeout="$STRESS_TEST_DURATION" \
        -run="TestStress" \
        ./tests/performance/stress/... 2>&1 | tee "$REPORTS_DIR/stress-tests.log"
    
    if [ $? -eq 0 ]; then
        log_success "Stress tests completed"
    else
        log_error "Stress tests failed"
        return 1
    fi
}

# Run security tests
run_security_tests() {
    print_header "RUNNING SECURITY TESTS"
    
    log_info "Running security scan with gosec..."
    gosec -fmt json -out "$REPORTS_DIR/security-report.json" ./... 2>&1 | tee "$REPORTS_DIR/security-scan.log"
    
    # Check for high severity issues
    if [ -f "$REPORTS_DIR/security-report.json" ]; then
        HIGH_ISSUES=$(jq '.Issues | map(select(.severity == "HIGH")) | length' "$REPORTS_DIR/security-report.json" 2>/dev/null || echo "0")
        MEDIUM_ISSUES=$(jq '.Issues | map(select(.severity == "MEDIUM")) | length' "$REPORTS_DIR/security-report.json" 2>/dev/null || echo "0")
        
        log_info "Security scan results: $HIGH_ISSUES high, $MEDIUM_ISSUES medium severity issues"
        
        if [ "$HIGH_ISSUES" -gt 0 ]; then
            log_error "High severity security issues found"
            return 1
        else
            log_success "No high severity security issues found"
        fi
    fi
}

# Generate combined coverage report
generate_coverage_report() {
    print_header "GENERATING COVERAGE REPORT"
    
    log_info "Combining coverage reports..."
    
    # Combine coverage files if they exist
    COVERAGE_FILES=""
    if [ -f "$COVERAGE_DIR/unit.out" ]; then
        COVERAGE_FILES="$COVERAGE_FILES $COVERAGE_DIR/unit.out"
    fi
    if [ -f "$COVERAGE_DIR/integration.out" ]; then
        COVERAGE_FILES="$COVERAGE_FILES $COVERAGE_DIR/integration.out"
    fi
    
    if [ -n "$COVERAGE_FILES" ]; then
        # Create combined coverage file
        echo "mode: atomic" > "$COVERAGE_DIR/combined.out"
        for file in $COVERAGE_FILES; do
            tail -n +2 "$file" >> "$COVERAGE_DIR/combined.out"
        done
        
        # Generate HTML report
        go tool cover -html="$COVERAGE_DIR/combined.out" -o "$COVERAGE_DIR/combined.html"
        
        # Calculate total coverage
        TOTAL_COVERAGE=$(go tool cover -func="$COVERAGE_DIR/combined.out" | tail -1 | awk '{print $3}' | sed 's/%//')
        log_info "Total coverage: ${TOTAL_COVERAGE}%"
        
        if (( $(echo "$TOTAL_COVERAGE >= $COVERAGE_THRESHOLD" | bc -l) )); then
            log_success "Total coverage threshold met (${TOTAL_COVERAGE}% >= ${COVERAGE_THRESHOLD}%)"
        else
            log_warning "Total coverage below threshold (${TOTAL_COVERAGE}% < ${COVERAGE_THRESHOLD}%)"
        fi
    fi
}

# Generate test report
generate_test_report() {
    print_header "GENERATING TEST REPORT"
    
    REPORT_FILE="$REPORTS_DIR/test-summary.md"
    
    cat > "$REPORT_FILE" << EOF
# TradSys Test Report

**Generated:** $(date)
**Coverage Threshold:** ${COVERAGE_THRESHOLD}%

## Test Results Summary

EOF
    
    # Add test results
    for test_type in unit integration performance compliance e2e load stress security; do
        log_file="$REPORTS_DIR/${test_type}-tests.log"
        if [ -f "$log_file" ]; then
            if grep -q "PASS" "$log_file" && ! grep -q "FAIL" "$log_file"; then
                echo "- ✅ ${test_type^} Tests: **PASSED**" >> "$REPORT_FILE"
            else
                echo "- ❌ ${test_type^} Tests: **FAILED**" >> "$REPORT_FILE"
            fi
        else
            echo "- ⏭️ ${test_type^} Tests: **SKIPPED**" >> "$REPORT_FILE"
        fi
    done
    
    # Add coverage information
    if [ -f "$COVERAGE_DIR/combined.out" ]; then
        TOTAL_COVERAGE=$(go tool cover -func="$COVERAGE_DIR/combined.out" | tail -1 | awk '{print $3}' | sed 's/%//')
        echo "" >> "$REPORT_FILE"
        echo "## Coverage Report" >> "$REPORT_FILE"
        echo "" >> "$REPORT_FILE"
        echo "**Total Coverage:** ${TOTAL_COVERAGE}%" >> "$REPORT_FILE"
        echo "**Threshold:** ${COVERAGE_THRESHOLD}%" >> "$REPORT_FILE"
        
        if (( $(echo "$TOTAL_COVERAGE >= $COVERAGE_THRESHOLD" | bc -l) )); then
            echo "**Status:** ✅ **PASSED**" >> "$REPORT_FILE"
        else
            echo "**Status:** ❌ **BELOW THRESHOLD**" >> "$REPORT_FILE"
        fi
    fi
    
    log_success "Test report generated: $REPORT_FILE"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up..."
    
    # Stop pprof server if running
    if [ -f "$BENCHMARK_DIR/pprof.pid" ]; then
        PPROF_PID=$(cat "$BENCHMARK_DIR/pprof.pid")
        if kill -0 "$PPROF_PID" 2>/dev/null; then
            kill "$PPROF_PID"
            log_info "Stopped pprof server (PID: $PPROF_PID)"
        fi
        rm -f "$BENCHMARK_DIR/pprof.pid"
    fi
}

# Main execution
main() {
    cd "$PROJECT_ROOT"
    
    # Set trap for cleanup
    trap cleanup EXIT
    
    print_header "TRADSYS COMPREHENSIVE TEST SUITE"
    log_info "Starting comprehensive test execution..."
    log_info "Project root: $PROJECT_ROOT"
    
    # Parse command line arguments
    RUN_ALL=true
    RUN_UNIT=false
    RUN_INTEGRATION=false
    RUN_PERFORMANCE=false
    RUN_COMPLIANCE=false
    RUN_E2E=false
    RUN_LOAD=false
    RUN_STRESS=false
    RUN_SECURITY=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --unit)
                RUN_ALL=false
                RUN_UNIT=true
                shift
                ;;
            --integration)
                RUN_ALL=false
                RUN_INTEGRATION=true
                shift
                ;;
            --performance)
                RUN_ALL=false
                RUN_PERFORMANCE=true
                shift
                ;;
            --compliance)
                RUN_ALL=false
                RUN_COMPLIANCE=true
                shift
                ;;
            --e2e)
                RUN_ALL=false
                RUN_E2E=true
                shift
                ;;
            --load)
                RUN_ALL=false
                RUN_LOAD=true
                shift
                ;;
            --stress)
                RUN_ALL=false
                RUN_STRESS=true
                shift
                ;;
            --security)
                RUN_ALL=false
                RUN_SECURITY=true
                shift
                ;;
            --coverage-threshold)
                COVERAGE_THRESHOLD="$2"
                shift 2
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --unit              Run unit tests only"
                echo "  --integration       Run integration tests only"
                echo "  --performance       Run performance tests only"
                echo "  --compliance        Run compliance tests only"
                echo "  --e2e              Run end-to-end tests only"
                echo "  --load             Run load tests only"
                echo "  --stress           Run stress tests only"
                echo "  --security         Run security tests only"
                echo "  --coverage-threshold N  Set coverage threshold (default: 80)"
                echo "  --help             Show this help message"
                echo ""
                echo "If no specific test type is specified, all tests will be run."
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Check dependencies
    check_dependencies
    
    # Track overall success
    OVERALL_SUCCESS=true
    
    # Run tests based on arguments
    if [ "$RUN_ALL" = true ] || [ "$RUN_UNIT" = true ]; then
        if ! run_unit_tests; then
            OVERALL_SUCCESS=false
        fi
    fi
    
    if [ "$RUN_ALL" = true ] || [ "$RUN_INTEGRATION" = true ]; then
        if ! run_integration_tests; then
            OVERALL_SUCCESS=false
        fi
    fi
    
    if [ "$RUN_ALL" = true ] || [ "$RUN_PERFORMANCE" = true ]; then
        if ! run_performance_tests; then
            OVERALL_SUCCESS=false
        fi
    fi
    
    if [ "$RUN_ALL" = true ] || [ "$RUN_COMPLIANCE" = true ]; then
        if ! run_compliance_tests; then
            OVERALL_SUCCESS=false
        fi
    fi
    
    if [ "$RUN_ALL" = true ] || [ "$RUN_E2E" = true ]; then
        if ! run_e2e_tests; then
            OVERALL_SUCCESS=false
        fi
    fi
    
    if [ "$RUN_ALL" = true ] || [ "$RUN_LOAD" = true ]; then
        if ! run_load_tests; then
            OVERALL_SUCCESS=false
        fi
    fi
    
    if [ "$RUN_ALL" = true ] || [ "$RUN_STRESS" = true ]; then
        if ! run_stress_tests; then
            OVERALL_SUCCESS=false
        fi
    fi
    
    if [ "$RUN_ALL" = true ] || [ "$RUN_SECURITY" = true ]; then
        if ! run_security_tests; then
            OVERALL_SUCCESS=false
        fi
    fi
    
    # Generate reports
    generate_coverage_report
    generate_test_report
    
    # Final result
    print_header "TEST EXECUTION COMPLETE"
    
    if [ "$OVERALL_SUCCESS" = true ]; then
        log_success "All tests passed successfully! 🎉"
        log_info "Reports available in: $REPORTS_DIR"
        log_info "Coverage reports available in: $COVERAGE_DIR"
        exit 0
    else
        log_error "Some tests failed. Check the reports for details."
        log_info "Reports available in: $REPORTS_DIR"
        exit 1
    fi
}

# Run main function
main "$@"
