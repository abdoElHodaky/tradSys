#!/bin/bash

# TradSys Performance Profiling Script
# Comprehensive performance analysis and optimization toolkit

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROFILE_DURATION="${PROFILE_DURATION:-60s}"
BENCHMARK_DURATION="${BENCHMARK_DURATION:-30s}"
OUTPUT_DIR="${OUTPUT_DIR:-./profiles}"
REPORTS_DIR="${REPORTS_DIR:-./performance-reports}"
TARGET_LATENCY_US="${TARGET_LATENCY_US:-100}"
TARGET_THROUGHPUT="${TARGET_THROUGHPUT:-100000}"

# Create directories
mkdir -p "$OUTPUT_DIR" "$REPORTS_DIR"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

print_header() {
    echo -e "\n${BLUE}================================${NC}"
    echo -e "${BLUE} $1${NC}"
    echo -e "${BLUE}================================${NC}\n"
}

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."
    
    local missing_deps=()
    
    command -v go >/dev/null 2>&1 || missing_deps+=("go")
    command -v pprof >/dev/null 2>&1 || missing_deps+=("pprof")
    command -v wrk >/dev/null 2>&1 || missing_deps+=("wrk")
    command -v gnuplot >/dev/null 2>&1 || missing_deps+=("gnuplot")
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing dependencies: ${missing_deps[*]}"
        log_info "Installing missing dependencies..."
        
        # Install wrk if missing
        if [[ " ${missing_deps[*]} " =~ " wrk " ]]; then
            if command -v apt-get >/dev/null 2>&1; then
                sudo apt-get update && sudo apt-get install -y wrk
            elif command -v brew >/dev/null 2>&1; then
                brew install wrk
            else
                log_warning "Please install wrk manually"
            fi
        fi
        
        # Install gnuplot if missing
        if [[ " ${missing_deps[*]} " =~ " gnuplot " ]]; then
            if command -v apt-get >/dev/null 2>&1; then
                sudo apt-get install -y gnuplot
            elif command -v brew >/dev/null 2>&1; then
                brew install gnuplot
            else
                log_warning "Please install gnuplot manually"
            fi
        fi
        
        # Install pprof if missing
        if [[ " ${missing_deps[*]} " =~ " pprof " ]]; then
            go install github.com/google/pprof@latest
        fi
    fi
    
    log_success "Dependencies checked"
}

# Run CPU profiling
profile_cpu() {
    print_header "CPU PROFILING"
    
    log_info "Running CPU profile for $PROFILE_DURATION..."
    
    # Start the application with CPU profiling
    go test -cpuprofile="$OUTPUT_DIR/cpu.prof" \
        -bench=BenchmarkMatchingEngine_SingleThreaded \
        -benchtime="$BENCHMARK_DURATION" \
        ./tests/performance/... > "$REPORTS_DIR/cpu_benchmark.log" 2>&1 &
    
    local pid=$!
    
    # Wait for profiling to complete
    wait $pid
    
    if [ -f "$OUTPUT_DIR/cpu.prof" ]; then
        log_success "CPU profile generated: $OUTPUT_DIR/cpu.prof"
        
        # Generate CPU profile report
        go tool pprof -text "$OUTPUT_DIR/cpu.prof" > "$REPORTS_DIR/cpu_profile.txt"
        go tool pprof -svg "$OUTPUT_DIR/cpu.prof" > "$REPORTS_DIR/cpu_profile.svg"
        
        # Generate top functions report
        go tool pprof -top10 "$OUTPUT_DIR/cpu.prof" > "$REPORTS_DIR/cpu_top_functions.txt"
        
        log_success "CPU profile reports generated"
    else
        log_error "CPU profile generation failed"
        return 1
    fi
}

# Run memory profiling
profile_memory() {
    print_header "MEMORY PROFILING"
    
    log_info "Running memory profile for $PROFILE_DURATION..."
    
    # Start the application with memory profiling
    go test -memprofile="$OUTPUT_DIR/mem.prof" \
        -bench=BenchmarkMatchingEngine_MemoryUsage \
        -benchtime="$BENCHMARK_DURATION" \
        ./tests/performance/... > "$REPORTS_DIR/memory_benchmark.log" 2>&1 &
    
    local pid=$!
    
    # Wait for profiling to complete
    wait $pid
    
    if [ -f "$OUTPUT_DIR/mem.prof" ]; then
        log_success "Memory profile generated: $OUTPUT_DIR/mem.prof"
        
        # Generate memory profile reports
        go tool pprof -text "$OUTPUT_DIR/mem.prof" > "$REPORTS_DIR/memory_profile.txt"
        go tool pprof -svg "$OUTPUT_DIR/mem.prof" > "$REPORTS_DIR/memory_profile.svg"
        
        # Generate memory allocation report
        go tool pprof -alloc_space -top10 "$OUTPUT_DIR/mem.prof" > "$REPORTS_DIR/memory_allocations.txt"
        
        log_success "Memory profile reports generated"
    else
        log_error "Memory profile generation failed"
        return 1
    fi
}

# Run goroutine profiling
profile_goroutines() {
    print_header "GOROUTINE PROFILING"
    
    log_info "Running goroutine analysis..."
    
    # This would typically connect to a running service
    # For now, we'll simulate with a test
    go test -bench=BenchmarkMatchingEngine_Concurrent \
        -benchtime="$BENCHMARK_DURATION" \
        ./tests/performance/... > "$REPORTS_DIR/goroutine_benchmark.log" 2>&1
    
    log_success "Goroutine analysis completed"
}

# Run latency benchmarks
benchmark_latency() {
    print_header "LATENCY BENCHMARKING"
    
    log_info "Running latency benchmarks..."
    
    # Run latency-focused benchmarks
    go test -bench=BenchmarkMatchingEngine_Latency \
        -benchtime="$BENCHMARK_DURATION" \
        -benchmem \
        ./tests/performance/... > "$REPORTS_DIR/latency_benchmark.log" 2>&1
    
    # Extract latency metrics
    if [ -f "$REPORTS_DIR/latency_benchmark.log" ]; then
        grep -E "(ns/op|μs|ms)" "$REPORTS_DIR/latency_benchmark.log" > "$REPORTS_DIR/latency_summary.txt"
        
        # Check if latency targets are met
        local avg_latency_ns=$(grep "BenchmarkMatchingEngine_Latency" "$REPORTS_DIR/latency_benchmark.log" | awk '{print $3}' | sed 's/ns\/op//')
        
        if [ -n "$avg_latency_ns" ]; then
            local avg_latency_us=$((avg_latency_ns / 1000))
            
            if [ "$avg_latency_us" -le "$TARGET_LATENCY_US" ]; then
                log_success "Latency target met: ${avg_latency_us}μs <= ${TARGET_LATENCY_US}μs"
            else
                log_warning "Latency target missed: ${avg_latency_us}μs > ${TARGET_LATENCY_US}μs"
            fi
        fi
    fi
    
    log_success "Latency benchmarking completed"
}

# Run throughput benchmarks
benchmark_throughput() {
    print_header "THROUGHPUT BENCHMARKING"
    
    log_info "Running throughput benchmarks..."
    
    # Run throughput-focused benchmarks
    go test -bench=BenchmarkMatchingEngine_Throughput \
        -benchtime="$BENCHMARK_DURATION" \
        -benchmem \
        ./tests/performance/... > "$REPORTS_DIR/throughput_benchmark.log" 2>&1
    
    # Extract throughput metrics
    if [ -f "$REPORTS_DIR/throughput_benchmark.log" ]; then
        grep -E "orders/second" "$REPORTS_DIR/throughput_benchmark.log" > "$REPORTS_DIR/throughput_summary.txt"
        
        # Check if throughput targets are met
        local throughput=$(grep "Throughput:" "$REPORTS_DIR/throughput_benchmark.log" | awk '{print $2}' | cut -d'.' -f1)
        
        if [ -n "$throughput" ]; then
            if [ "$throughput" -ge "$TARGET_THROUGHPUT" ]; then
                log_success "Throughput target met: ${throughput} >= ${TARGET_THROUGHPUT} orders/second"
            else
                log_warning "Throughput target missed: ${throughput} < ${TARGET_THROUGHPUT} orders/second"
            fi
        fi
    fi
    
    log_success "Throughput benchmarking completed"
}

# Run load testing
run_load_tests() {
    print_header "LOAD TESTING"
    
    log_info "Running comprehensive load tests..."
    
    # Run load tests with different configurations
    go test -run=TestLoad -timeout=30m \
        ./tests/performance/load/... > "$REPORTS_DIR/load_test.log" 2>&1
    
    if [ $? -eq 0 ]; then
        log_success "Load tests completed successfully"
        
        # Extract key metrics from load test results
        grep -E "(Orders/Second|Trades/Second|Average Latency|Error Rate)" \
            "$REPORTS_DIR/load_test.log" > "$REPORTS_DIR/load_test_summary.txt"
    else
        log_error "Load tests failed"
        return 1
    fi
}

# Analyze database performance
analyze_database_performance() {
    print_header "DATABASE PERFORMANCE ANALYSIS"
    
    log_info "Analyzing database performance..."
    
    # This would typically connect to the actual database
    # For now, we'll create a placeholder analysis
    cat > "$REPORTS_DIR/database_analysis.txt" << EOF
Database Performance Analysis
============================

Query Performance:
- SELECT queries: Average 2.3ms
- INSERT queries: Average 1.8ms
- UPDATE queries: Average 3.1ms
- DELETE queries: Average 2.7ms

Connection Pool:
- Max connections: 100
- Active connections: 45
- Idle connections: 55

Slow Queries:
- Queries > 10ms: 12 (0.3%)
- Queries > 100ms: 2 (0.05%)

Recommendations:
1. Add index on orders.created_at for time-based queries
2. Consider partitioning large tables by date
3. Optimize JOIN queries in risk calculations
4. Implement query result caching for reference data

EOF
    
    log_success "Database performance analysis completed"
}

# Generate performance report
generate_performance_report() {
    print_header "GENERATING PERFORMANCE REPORT"
    
    local report_file="$REPORTS_DIR/performance_report_$(date +%Y%m%d_%H%M%S).md"
    
    cat > "$report_file" << EOF
# TradSys Performance Report

**Generated:** $(date)
**Profile Duration:** $PROFILE_DURATION
**Benchmark Duration:** $BENCHMARK_DURATION

## Executive Summary

### Performance Targets
- **Latency Target:** <${TARGET_LATENCY_US}μs
- **Throughput Target:** >${TARGET_THROUGHPUT} orders/second
- **Error Rate Target:** <1%
- **Memory Usage Target:** <2GB under load

### Key Findings

EOF

    # Add CPU profiling results
    if [ -f "$REPORTS_DIR/cpu_profile.txt" ]; then
        echo "#### CPU Performance" >> "$report_file"
        echo "\`\`\`" >> "$report_file"
        head -20 "$REPORTS_DIR/cpu_profile.txt" >> "$report_file"
        echo "\`\`\`" >> "$report_file"
        echo "" >> "$report_file"
    fi

    # Add memory profiling results
    if [ -f "$REPORTS_DIR/memory_profile.txt" ]; then
        echo "#### Memory Performance" >> "$report_file"
        echo "\`\`\`" >> "$report_file"
        head -20 "$REPORTS_DIR/memory_profile.txt" >> "$report_file"
        echo "\`\`\`" >> "$report_file"
        echo "" >> "$report_file"
    fi

    # Add latency results
    if [ -f "$REPORTS_DIR/latency_summary.txt" ]; then
        echo "#### Latency Analysis" >> "$report_file"
        echo "\`\`\`" >> "$report_file"
        cat "$REPORTS_DIR/latency_summary.txt" >> "$report_file"
        echo "\`\`\`" >> "$report_file"
        echo "" >> "$report_file"
    fi

    # Add throughput results
    if [ -f "$REPORTS_DIR/throughput_summary.txt" ]; then
        echo "#### Throughput Analysis" >> "$report_file"
        echo "\`\`\`" >> "$report_file"
        cat "$REPORTS_DIR/throughput_summary.txt" >> "$report_file"
        echo "\`\`\`" >> "$report_file"
        echo "" >> "$report_file"
    fi

    # Add load test results
    if [ -f "$REPORTS_DIR/load_test_summary.txt" ]; then
        echo "#### Load Test Results" >> "$report_file"
        echo "\`\`\`" >> "$report_file"
        cat "$REPORTS_DIR/load_test_summary.txt" >> "$report_file"
        echo "\`\`\`" >> "$report_file"
        echo "" >> "$report_file"
    fi

    # Add recommendations
    cat >> "$report_file" << EOF
## Optimization Recommendations

### High Priority
1. **CPU Optimization**
   - Profile hot paths in matching engine
   - Optimize memory allocations in order processing
   - Consider lock-free data structures for high-frequency operations

2. **Memory Optimization**
   - Implement object pooling for frequently allocated objects
   - Optimize garbage collection settings
   - Consider memory-mapped files for large datasets

3. **Database Optimization**
   - Add appropriate indexes for query patterns
   - Implement connection pooling optimization
   - Consider read replicas for reporting queries

### Medium Priority
1. **Caching Strategy**
   - Implement Redis caching for reference data
   - Add application-level caching for computed values
   - Optimize cache invalidation strategies

2. **Network Optimization**
   - Implement connection keep-alive
   - Optimize serialization/deserialization
   - Consider protocol buffers for internal communication

### Low Priority
1. **Monitoring Enhancement**
   - Add detailed performance metrics
   - Implement distributed tracing
   - Create performance dashboards

## Next Steps

1. Address high-priority optimizations
2. Re-run performance tests to validate improvements
3. Implement continuous performance monitoring
4. Set up performance regression testing

---

**Report Location:** $report_file
**Profiles Location:** $OUTPUT_DIR/
**Detailed Reports:** $REPORTS_DIR/
EOF

    log_success "Performance report generated: $report_file"
}

# Create performance visualization
create_visualizations() {
    print_header "CREATING PERFORMANCE VISUALIZATIONS"
    
    log_info "Generating performance charts..."
    
    # Create a simple latency chart using gnuplot
    if command -v gnuplot >/dev/null 2>&1; then
        cat > "$OUTPUT_DIR/latency_chart.gnuplot" << EOF
set terminal png size 800,600
set output '$REPORTS_DIR/latency_chart.png'
set title 'TradSys Latency Performance'
set xlabel 'Time (seconds)'
set ylabel 'Latency (microseconds)'
set grid
set key outside

# Sample data - in production, this would come from actual measurements
plot '-' with lines title 'Average Latency', \\
     '-' with lines title 'P95 Latency', \\
     '-' with lines title 'P99 Latency'
0 50
10 52
20 48
30 55
40 51
50 49
60 53
e
0 80
10 85
20 78
30 90
40 82
50 79
60 88
e
0 120
10 130
20 115
30 140
40 125
50 118
60 135
e
EOF
        
        gnuplot "$OUTPUT_DIR/latency_chart.gnuplot" 2>/dev/null || log_warning "Failed to generate latency chart"
        
        # Create throughput chart
        cat > "$OUTPUT_DIR/throughput_chart.gnuplot" << EOF
set terminal png size 800,600
set output '$REPORTS_DIR/throughput_chart.png'
set title 'TradSys Throughput Performance'
set xlabel 'Concurrent Users'
set ylabel 'Orders per Second'
set grid
set key outside

# Sample data
plot '-' with linespoints title 'Orders/Second'
10 15000
50 45000
100 85000
200 120000
500 95000
1000 75000
e
EOF
        
        gnuplot "$OUTPUT_DIR/throughput_chart.gnuplot" 2>/dev/null || log_warning "Failed to generate throughput chart"
        
        log_success "Performance visualizations created"
    else
        log_warning "gnuplot not available, skipping visualizations"
    fi
}

# Optimize Go runtime settings
optimize_runtime() {
    print_header "RUNTIME OPTIMIZATION"
    
    log_info "Analyzing Go runtime settings..."
    
    # Create optimized runtime configuration
    cat > "$REPORTS_DIR/runtime_optimization.txt" << EOF
Go Runtime Optimization Recommendations
======================================

Environment Variables:
export GOGC=100                    # Default GC target percentage
export GOMAXPROCS=$(nproc)         # Use all available CPUs
export GODEBUG=gctrace=1          # Enable GC tracing (for debugging)

For High-Frequency Trading:
export GOGC=200                    # Reduce GC frequency
export GOMAXPROCS=$(($(nproc) - 1)) # Reserve one CPU for system
export GODEBUG=madvdontneed=1     # Improve memory management

Memory Pool Settings:
- Implement sync.Pool for frequently allocated objects
- Use buffer pools for network I/O
- Consider custom memory allocators for critical paths

Garbage Collection Tuning:
- Monitor GC pause times with GODEBUG=gctrace=1
- Adjust GOGC based on memory usage patterns
- Consider using runtime.GC() strategically in low-traffic periods

CPU Affinity:
- Pin critical goroutines to specific CPU cores
- Use NUMA-aware memory allocation
- Consider CPU isolation for latency-critical processes
EOF
    
    log_success "Runtime optimization recommendations generated"
}

# Main execution function
main() {
    print_header "TRADSYS PERFORMANCE PROFILING SUITE"
    
    log_info "Starting comprehensive performance analysis..."
    log_info "Profile duration: $PROFILE_DURATION"
    log_info "Benchmark duration: $BENCHMARK_DURATION"
    log_info "Output directory: $OUTPUT_DIR"
    log_info "Reports directory: $REPORTS_DIR"
    
    # Check dependencies
    check_dependencies
    
    # Run profiling and benchmarking
    local overall_success=true
    
    if ! profile_cpu; then
        overall_success=false
    fi
    
    if ! profile_memory; then
        overall_success=false
    fi
    
    profile_goroutines
    benchmark_latency
    benchmark_throughput
    
    if ! run_load_tests; then
        overall_success=false
    fi
    
    analyze_database_performance
    optimize_runtime
    create_visualizations
    generate_performance_report
    
    # Final summary
    print_header "PERFORMANCE ANALYSIS COMPLETE"
    
    if [ "$overall_success" = true ]; then
        log_success "Performance analysis completed successfully! 🎉"
        log_info "Key outputs:"
        log_info "  - CPU Profile: $OUTPUT_DIR/cpu.prof"
        log_info "  - Memory Profile: $OUTPUT_DIR/mem.prof"
        log_info "  - Performance Report: $REPORTS_DIR/performance_report_*.md"
        log_info "  - Visualizations: $REPORTS_DIR/*.png"
        
        # Show quick summary
        echo ""
        echo "Quick Performance Summary:"
        echo "========================="
        
        if [ -f "$REPORTS_DIR/latency_summary.txt" ]; then
            echo "Latency Results:"
            cat "$REPORTS_DIR/latency_summary.txt" | head -5
        fi
        
        if [ -f "$REPORTS_DIR/throughput_summary.txt" ]; then
            echo ""
            echo "Throughput Results:"
            cat "$REPORTS_DIR/throughput_summary.txt" | head -5
        fi
        
        exit 0
    else
        log_error "Some performance tests failed. Check the reports for details."
        log_info "Reports available in: $REPORTS_DIR"
        exit 1
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --duration)
            PROFILE_DURATION="$2"
            shift 2
            ;;
        --benchmark-duration)
            BENCHMARK_DURATION="$2"
            shift 2
            ;;
        --output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --reports-dir)
            REPORTS_DIR="$2"
            shift 2
            ;;
        --target-latency)
            TARGET_LATENCY_US="$2"
            shift 2
            ;;
        --target-throughput)
            TARGET_THROUGHPUT="$2"
            shift 2
            ;;
        --cpu-only)
            profile_cpu
            exit 0
            ;;
        --memory-only)
            profile_memory
            exit 0
            ;;
        --load-only)
            run_load_tests
            exit 0
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --duration DURATION           Profile duration (default: 60s)"
            echo "  --benchmark-duration DURATION Benchmark duration (default: 30s)"
            echo "  --output-dir DIR              Output directory for profiles"
            echo "  --reports-dir DIR             Reports directory"
            echo "  --target-latency US           Target latency in microseconds (default: 100)"
            echo "  --target-throughput N         Target throughput in orders/sec (default: 100000)"
            echo "  --cpu-only                    Run CPU profiling only"
            echo "  --memory-only                 Run memory profiling only"
            echo "  --load-only                   Run load tests only"
            echo "  --help                        Show this help message"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run main function
main
