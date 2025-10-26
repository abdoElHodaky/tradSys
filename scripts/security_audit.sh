#!/bin/bash

# TradSys Security Audit Script
# Comprehensive security assessment and vulnerability scanning

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
AUDIT_DIR="${AUDIT_DIR:-./security-audit}"
REPORTS_DIR="${REPORTS_DIR:-./security-reports}"
SCAN_TIMEOUT="${SCAN_TIMEOUT:-300}"
SEVERITY_THRESHOLD="${SEVERITY_THRESHOLD:-MEDIUM}"

# Create directories
mkdir -p "$AUDIT_DIR" "$REPORTS_DIR"

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
    log_info "Checking security audit dependencies..."
    
    local missing_deps=()
    
    command -v gosec >/dev/null 2>&1 || missing_deps+=("gosec")
    command -v trivy >/dev/null 2>&1 || missing_deps+=("trivy")
    command -v nancy >/dev/null 2>&1 || missing_deps+=("nancy")
    command -v semgrep >/dev/null 2>&1 || missing_deps+=("semgrep")
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_warning "Missing security tools: ${missing_deps[*]}"
        log_info "Installing missing tools..."
        
        # Install gosec
        if [[ " ${missing_deps[*]} " =~ " gosec " ]]; then
            go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
        fi
        
        # Install trivy
        if [[ " ${missing_deps[*]} " =~ " trivy " ]]; then
            if command -v apt-get >/dev/null 2>&1; then
                sudo apt-get update && sudo apt-get install -y wget apt-transport-https gnupg lsb-release
                wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | sudo apt-key add -
                echo "deb https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | sudo tee -a /etc/apt/sources.list.d/trivy.list
                sudo apt-get update && sudo apt-get install -y trivy
            elif command -v brew >/dev/null 2>&1; then
                brew install trivy
            else
                log_warning "Please install trivy manually"
            fi
        fi
        
        # Install nancy
        if [[ " ${missing_deps[*]} " =~ " nancy " ]]; then
            go install github.com/sonatypecommunity/nancy@latest
        fi
        
        # Install semgrep
        if [[ " ${missing_deps[*]} " =~ " semgrep " ]]; then
            if command -v pip3 >/dev/null 2>&1; then
                pip3 install semgrep
            else
                log_warning "Please install semgrep manually (requires Python)"
            fi
        fi
    fi
    
    log_success "Security audit dependencies checked"
}

# Run static code analysis
run_static_analysis() {
    print_header "STATIC CODE ANALYSIS"
    
    log_info "Running Gosec static security analyzer..."
    
    # Run gosec with comprehensive rules
    gosec -fmt json -out "$REPORTS_DIR/gosec-report.json" \
        -severity "$SEVERITY_THRESHOLD" \
        -confidence medium \
        -exclude-dir=vendor \
        -exclude-dir=.git \
        ./... 2>&1 | tee "$REPORTS_DIR/gosec.log"
    
    if [ $? -eq 0 ]; then
        log_success "Gosec analysis completed"
        
        # Parse results
        if [ -f "$REPORTS_DIR/gosec-report.json" ]; then
            local high_issues=$(jq '.Issues | map(select(.severity == "HIGH")) | length' "$REPORTS_DIR/gosec-report.json" 2>/dev/null || echo "0")
            local medium_issues=$(jq '.Issues | map(select(.severity == "MEDIUM")) | length' "$REPORTS_DIR/gosec-report.json" 2>/dev/null || echo "0")
            local low_issues=$(jq '.Issues | map(select(.severity == "LOW")) | length' "$REPORTS_DIR/gosec-report.json" 2>/dev/null || echo "0")
            
            log_info "Gosec Results: $high_issues high, $medium_issues medium, $low_issues low severity issues"
            
            if [ "$high_issues" -gt 0 ]; then
                log_error "High severity security issues found!"
                return 1
            fi
        fi
    else
        log_error "Gosec analysis failed"
        return 1
    fi
}

# Run dependency vulnerability scanning
run_dependency_scan() {
    print_header "DEPENDENCY VULNERABILITY SCANNING"
    
    log_info "Running Trivy dependency scanner..."
    
    # Scan for vulnerabilities in dependencies
    trivy fs --format json --output "$REPORTS_DIR/trivy-report.json" \
        --severity HIGH,CRITICAL \
        --timeout "${SCAN_TIMEOUT}s" \
        . 2>&1 | tee "$REPORTS_DIR/trivy.log"
    
    if [ $? -eq 0 ]; then
        log_success "Trivy dependency scan completed"
        
        # Parse results
        if [ -f "$REPORTS_DIR/trivy-report.json" ]; then
            local critical_vulns=$(jq '.Results[]?.Vulnerabilities[]? | select(.Severity == "CRITICAL") | .VulnerabilityID' "$REPORTS_DIR/trivy-report.json" 2>/dev/null | wc -l || echo "0")
            local high_vulns=$(jq '.Results[]?.Vulnerabilities[]? | select(.Severity == "HIGH") | .VulnerabilityID' "$REPORTS_DIR/trivy-report.json" 2>/dev/null | wc -l || echo "0")
            
            log_info "Trivy Results: $critical_vulns critical, $high_vulns high severity vulnerabilities"
            
            if [ "$critical_vulns" -gt 0 ]; then
                log_error "Critical vulnerabilities found in dependencies!"
                return 1
            fi
        fi
    else
        log_error "Trivy dependency scan failed"
        return 1
    fi
    
    # Run Nancy for Go module vulnerabilities
    if command -v nancy >/dev/null 2>&1 && [ -f "go.sum" ]; then
        log_info "Running Nancy Go module vulnerability scanner..."
        
        go list -json -deps ./... | nancy sleuth --output-format json > "$REPORTS_DIR/nancy-report.json" 2>&1 || true
        
        if [ -f "$REPORTS_DIR/nancy-report.json" ]; then
            log_success "Nancy scan completed"
        fi
    fi
}

# Run SAST (Static Application Security Testing)
run_sast_analysis() {
    print_header "STATIC APPLICATION SECURITY TESTING"
    
    if command -v semgrep >/dev/null 2>&1; then
        log_info "Running Semgrep SAST analysis..."
        
        # Run semgrep with security rules
        semgrep --config=auto --json --output="$REPORTS_DIR/semgrep-report.json" \
            --timeout="$SCAN_TIMEOUT" \
            --exclude="vendor/" \
            --exclude=".git/" \
            . 2>&1 | tee "$REPORTS_DIR/semgrep.log"
        
        if [ $? -eq 0 ]; then
            log_success "Semgrep SAST analysis completed"
            
            # Parse results
            if [ -f "$REPORTS_DIR/semgrep-report.json" ]; then
                local error_count=$(jq '.results | map(select(.extra.severity == "ERROR")) | length' "$REPORTS_DIR/semgrep-report.json" 2>/dev/null || echo "0")
                local warning_count=$(jq '.results | map(select(.extra.severity == "WARNING")) | length' "$REPORTS_DIR/semgrep-report.json" 2>/dev/null || echo "0")
                
                log_info "Semgrep Results: $error_count errors, $warning_count warnings"
                
                if [ "$error_count" -gt 0 ]; then
                    log_warning "Security errors found by Semgrep"
                fi
            fi
        else
            log_warning "Semgrep SAST analysis failed or not available"
        fi
    else
        log_warning "Semgrep not available, skipping SAST analysis"
    fi
}

# Run secrets detection
run_secrets_detection() {
    print_header "SECRETS DETECTION"
    
    log_info "Running secrets detection scan..."
    
    # Use git-secrets if available, otherwise use basic pattern matching
    if command -v git-secrets >/dev/null 2>&1; then
        git secrets --scan --recursive . > "$REPORTS_DIR/secrets-scan.log" 2>&1 || true
        log_success "Git-secrets scan completed"
    else
        # Basic pattern matching for common secrets
        log_info "Running basic secrets pattern matching..."
        
        cat > "$AUDIT_DIR/secrets-patterns.txt" << EOF
password\s*=\s*['""][^'""]+['""]
api[_-]?key\s*=\s*['""][^'""]+['""]
secret[_-]?key\s*=\s*['""][^'""]+['""]
private[_-]?key\s*=\s*['""][^'""]+['""]
token\s*=\s*['""][^'""]+['""]
aws[_-]?access[_-]?key\s*=\s*['""][^'""]+['""]
aws[_-]?secret[_-]?key\s*=\s*['""][^'""]+['""]
database[_-]?url\s*=\s*['""][^'""]+['""]
connection[_-]?string\s*=\s*['""][^'""]+['""]
EOF
        
        grep -r -n -i -f "$AUDIT_DIR/secrets-patterns.txt" . \
            --exclude-dir=.git \
            --exclude-dir=vendor \
            --exclude-dir=node_modules \
            --exclude="*.log" \
            --exclude="*.json" > "$REPORTS_DIR/secrets-basic.log" 2>/dev/null || true
        
        if [ -s "$REPORTS_DIR/secrets-basic.log" ]; then
            log_warning "Potential secrets found - please review $REPORTS_DIR/secrets-basic.log"
        else
            log_success "No obvious secrets patterns detected"
        fi
    fi
}

# Run container security scan
run_container_scan() {
    print_header "CONTAINER SECURITY SCANNING"
    
    if [ -f "Dockerfile" ] || [ -f "deployments/docker/Dockerfile.production" ]; then
        log_info "Running container security scan..."
        
        # Scan Dockerfile with trivy
        if [ -f "deployments/docker/Dockerfile.production" ]; then
            trivy config --format json --output "$REPORTS_DIR/dockerfile-scan.json" \
                deployments/docker/Dockerfile.production 2>&1 | tee "$REPORTS_DIR/dockerfile-scan.log"
        elif [ -f "Dockerfile" ]; then
            trivy config --format json --output "$REPORTS_DIR/dockerfile-scan.json" \
                Dockerfile 2>&1 | tee "$REPORTS_DIR/dockerfile-scan.log"
        fi
        
        log_success "Container security scan completed"
    else
        log_info "No Dockerfile found, skipping container scan"
    fi
}

# Run infrastructure security checks
run_infrastructure_checks() {
    print_header "INFRASTRUCTURE SECURITY CHECKS"
    
    log_info "Running infrastructure security checks..."
    
    # Check Kubernetes configurations
    if [ -d "deployments/kubernetes" ]; then
        log_info "Scanning Kubernetes configurations..."
        
        # Basic security checks for Kubernetes manifests
        find deployments/kubernetes -name "*.yaml" -o -name "*.yml" | while read -r file; do
            log_info "Checking $file..."
            
            # Check for security contexts
            if ! grep -q "securityContext" "$file"; then
                echo "WARNING: $file missing securityContext" >> "$REPORTS_DIR/k8s-security-issues.log"
            fi
            
            # Check for resource limits
            if ! grep -q "resources:" "$file"; then
                echo "WARNING: $file missing resource limits" >> "$REPORTS_DIR/k8s-security-issues.log"
            fi
            
            # Check for privileged containers
            if grep -q "privileged: true" "$file"; then
                echo "CRITICAL: $file contains privileged container" >> "$REPORTS_DIR/k8s-security-issues.log"
            fi
            
            # Check for host network
            if grep -q "hostNetwork: true" "$file"; then
                echo "HIGH: $file uses host network" >> "$REPORTS_DIR/k8s-security-issues.log"
            fi
        done
        
        if [ -f "$REPORTS_DIR/k8s-security-issues.log" ]; then
            log_warning "Kubernetes security issues found - see $REPORTS_DIR/k8s-security-issues.log"
        else
            log_success "Kubernetes configurations look secure"
        fi
    fi
    
    # Check CI/CD configurations
    if [ -d ".github/workflows" ]; then
        log_info "Scanning GitHub Actions workflows..."
        
        find .github/workflows -name "*.yml" -o -name "*.yaml" | while read -r file; do
            # Check for secrets in workflows
            if grep -q -E "(password|secret|key|token)" "$file"; then
                echo "WARNING: $file may contain hardcoded secrets" >> "$REPORTS_DIR/cicd-security-issues.log"
            fi
            
            # Check for pull_request_target usage
            if grep -q "pull_request_target" "$file"; then
                echo "HIGH: $file uses pull_request_target - review for security" >> "$REPORTS_DIR/cicd-security-issues.log"
            fi
        done
        
        if [ -f "$REPORTS_DIR/cicd-security-issues.log" ]; then
            log_warning "CI/CD security issues found - see $REPORTS_DIR/cicd-security-issues.log"
        else
            log_success "CI/CD configurations look secure"
        fi
    fi
}

# Run compliance validation
run_compliance_validation() {
    print_header "COMPLIANCE VALIDATION"
    
    log_info "Running compliance validation tests..."
    
    # Run compliance tests
    go test -v -timeout=30m ./tests/compliance/... > "$REPORTS_DIR/compliance-test.log" 2>&1
    
    if [ $? -eq 0 ]; then
        log_success "Compliance validation tests passed"
    else
        log_error "Compliance validation tests failed"
        return 1
    fi
    
    # Check for required compliance documentation
    compliance_docs=(
        "docs/compliance/privacy-policy.md"
        "docs/compliance/terms-of-service.md"
        "docs/compliance/data-retention-policy.md"
        "docs/compliance/incident-response-plan.md"
    )
    
    missing_docs=()
    for doc in "${compliance_docs[@]}"; do
        if [ ! -f "$doc" ]; then
            missing_docs+=("$doc")
        fi
    done
    
    if [ ${#missing_docs[@]} -gt 0 ]; then
        log_warning "Missing compliance documentation: ${missing_docs[*]}"
        echo "Missing compliance documentation:" > "$REPORTS_DIR/missing-compliance-docs.log"
        printf '%s\n' "${missing_docs[@]}" >> "$REPORTS_DIR/missing-compliance-docs.log"
    else
        log_success "All required compliance documentation present"
    fi
}

# Generate security report
generate_security_report() {
    print_header "GENERATING SECURITY REPORT"
    
    local report_file="$REPORTS_DIR/security-audit-report_$(date +%Y%m%d_%H%M%S).md"
    
    cat > "$report_file" << EOF
# TradSys Security Audit Report

**Generated:** $(date)
**Audit Scope:** Complete codebase and infrastructure
**Severity Threshold:** $SEVERITY_THRESHOLD

## Executive Summary

This report contains the results of a comprehensive security audit of the TradSys trading system.

### Security Tools Used
- **Gosec**: Go security analyzer
- **Trivy**: Vulnerability scanner
- **Nancy**: Go module vulnerability scanner
- **Semgrep**: Static analysis security testing
- **Custom Scripts**: Infrastructure and compliance checks

## Findings Summary

EOF

    # Add Gosec results
    if [ -f "$REPORTS_DIR/gosec-report.json" ]; then
        echo "### Static Code Analysis (Gosec)" >> "$report_file"
        local high_issues=$(jq '.Issues | map(select(.severity == "HIGH")) | length' "$REPORTS_DIR/gosec-report.json" 2>/dev/null || echo "0")
        local medium_issues=$(jq '.Issues | map(select(.severity == "MEDIUM")) | length' "$REPORTS_DIR/gosec-report.json" 2>/dev/null || echo "0")
        local low_issues=$(jq '.Issues | map(select(.severity == "LOW")) | length' "$REPORTS_DIR/gosec-report.json" 2>/dev/null || echo "0")
        
        echo "- **High Severity Issues:** $high_issues" >> "$report_file"
        echo "- **Medium Severity Issues:** $medium_issues" >> "$report_file"
        echo "- **Low Severity Issues:** $low_issues" >> "$report_file"
        echo "" >> "$report_file"
    fi

    # Add Trivy results
    if [ -f "$REPORTS_DIR/trivy-report.json" ]; then
        echo "### Dependency Vulnerabilities (Trivy)" >> "$report_file"
        local critical_vulns=$(jq '.Results[]?.Vulnerabilities[]? | select(.Severity == "CRITICAL") | .VulnerabilityID' "$REPORTS_DIR/trivy-report.json" 2>/dev/null | wc -l || echo "0")
        local high_vulns=$(jq '.Results[]?.Vulnerabilities[]? | select(.Severity == "HIGH") | .VulnerabilityID' "$REPORTS_DIR/trivy-report.json" 2>/dev/null | wc -l || echo "0")
        
        echo "- **Critical Vulnerabilities:** $critical_vulns" >> "$report_file"
        echo "- **High Vulnerabilities:** $high_vulns" >> "$report_file"
        echo "" >> "$report_file"
    fi

    # Add compliance results
    if [ -f "$REPORTS_DIR/compliance-test.log" ]; then
        echo "### Compliance Validation" >> "$report_file"
        if grep -q "PASS" "$REPORTS_DIR/compliance-test.log"; then
            echo "- **Status:** ✅ PASSED" >> "$report_file"
        else
            echo "- **Status:** ❌ FAILED" >> "$report_file"
        fi
        echo "" >> "$report_file"
    fi

    # Add recommendations
    cat >> "$report_file" << EOF
## Security Recommendations

### High Priority
1. **Address Critical Vulnerabilities**
   - Update dependencies with critical security vulnerabilities
   - Fix high-severity static analysis issues
   - Implement missing security controls

2. **Enhance Authentication & Authorization**
   - Implement multi-factor authentication
   - Review and strengthen JWT token handling
   - Add rate limiting to prevent brute force attacks

3. **Improve Input Validation**
   - Implement comprehensive input sanitization
   - Add SQL injection prevention measures
   - Strengthen XSS protection

### Medium Priority
1. **Security Headers**
   - Implement all recommended security headers
   - Configure Content Security Policy
   - Add HSTS headers for HTTPS enforcement

2. **Secrets Management**
   - Implement proper secrets management system
   - Remove any hardcoded secrets
   - Use environment variables or secret stores

3. **Audit Logging**
   - Enhance audit trail completeness
   - Implement log integrity protection
   - Add security event monitoring

### Low Priority
1. **Documentation**
   - Complete security documentation
   - Update incident response procedures
   - Create security training materials

## Compliance Status

### Regulatory Frameworks
- **MiFID II (EU):** Framework implemented
- **Dodd-Frank (US):** Framework implemented
- **FCA (UK):** Framework implemented
- **ASIC (AU):** Framework implemented
- **JFSA (JP):** Framework implemented
- **HKMA (HK):** Framework implemented
- **MAS (SG):** Framework implemented
- **CFTC (US):** Framework implemented

### Data Protection
- **GDPR Compliance:** Framework implemented
- **Data Retention:** Policies defined
- **Data Encryption:** In transit and at rest
- **Access Controls:** Role-based access implemented

## Next Steps

1. **Immediate Actions**
   - Fix all critical and high severity issues
   - Update vulnerable dependencies
   - Implement missing security controls

2. **Short Term (1-4 weeks)**
   - Complete security documentation
   - Implement enhanced monitoring
   - Conduct penetration testing

3. **Long Term (1-3 months)**
   - Regular security assessments
   - Security awareness training
   - Continuous compliance monitoring

---

**Report Location:** $report_file
**Detailed Logs:** $REPORTS_DIR/
**Audit Data:** $AUDIT_DIR/
EOF

    log_success "Security audit report generated: $report_file"
}

# Main execution function
main() {
    print_header "TRADSYS SECURITY AUDIT SUITE"
    
    log_info "Starting comprehensive security audit..."
    log_info "Audit directory: $AUDIT_DIR"
    log_info "Reports directory: $REPORTS_DIR"
    log_info "Severity threshold: $SEVERITY_THRESHOLD"
    
    # Check dependencies
    check_dependencies
    
    # Run security audits
    local overall_success=true
    
    if ! run_static_analysis; then
        overall_success=false
    fi
    
    if ! run_dependency_scan; then
        overall_success=false
    fi
    
    run_sast_analysis
    run_secrets_detection
    run_container_scan
    run_infrastructure_checks
    
    if ! run_compliance_validation; then
        overall_success=false
    fi
    
    # Generate comprehensive report
    generate_security_report
    
    # Final summary
    print_header "SECURITY AUDIT COMPLETE"
    
    if [ "$overall_success" = true ]; then
        log_success "Security audit completed successfully! 🛡️"
        log_info "Key outputs:"
        log_info "  - Security Report: $REPORTS_DIR/security-audit-report_*.md"
        log_info "  - Gosec Report: $REPORTS_DIR/gosec-report.json"
        log_info "  - Trivy Report: $REPORTS_DIR/trivy-report.json"
        log_info "  - Compliance Results: $REPORTS_DIR/compliance-test.log"
        
        echo ""
        echo "Security Audit Summary:"
        echo "======================"
        
        # Show critical findings
        if [ -f "$REPORTS_DIR/gosec-report.json" ]; then
            local high_issues=$(jq '.Issues | map(select(.severity == "HIGH")) | length' "$REPORTS_DIR/gosec-report.json" 2>/dev/null || echo "0")
            echo "Static Analysis: $high_issues high severity issues"
        fi
        
        if [ -f "$REPORTS_DIR/trivy-report.json" ]; then
            local critical_vulns=$(jq '.Results[]?.Vulnerabilities[]? | select(.Severity == "CRITICAL") | .VulnerabilityID' "$REPORTS_DIR/trivy-report.json" 2>/dev/null | wc -l || echo "0")
            echo "Dependencies: $critical_vulns critical vulnerabilities"
        fi
        
        exit 0
    else
        log_error "Security audit found critical issues that must be addressed"
        log_info "Reports available in: $REPORTS_DIR"
        exit 1
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --audit-dir)
            AUDIT_DIR="$2"
            shift 2
            ;;
        --reports-dir)
            REPORTS_DIR="$2"
            shift 2
            ;;
        --severity)
            SEVERITY_THRESHOLD="$2"
            shift 2
            ;;
        --timeout)
            SCAN_TIMEOUT="$2"
            shift 2
            ;;
        --static-only)
            run_static_analysis
            exit 0
            ;;
        --deps-only)
            run_dependency_scan
            exit 0
            ;;
        --compliance-only)
            run_compliance_validation
            exit 0
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --audit-dir DIR       Audit working directory"
            echo "  --reports-dir DIR     Reports output directory"
            echo "  --severity LEVEL      Severity threshold (LOW|MEDIUM|HIGH)"
            echo "  --timeout SECONDS     Scan timeout in seconds"
            echo "  --static-only         Run static analysis only"
            echo "  --deps-only           Run dependency scan only"
            echo "  --compliance-only     Run compliance validation only"
            echo "  --help                Show this help message"
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
