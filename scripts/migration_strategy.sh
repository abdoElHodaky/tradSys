#!/bin/bash

# TradSys Migration Strategy Script
# Comprehensive migration from legacy system to TradSys v3

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
MIGRATION_DIR="${MIGRATION_DIR:-./migration}"
BACKUP_DIR="${BACKUP_DIR:-./migration-backups}"
LEGACY_DB="${LEGACY_DB:-legacy_trading_db}"
TARGET_DB="${TARGET_DB:-tradsys_v3_db}"
MIGRATION_MODE="${MIGRATION_MODE:-phased}"

# Create directories
mkdir -p "$MIGRATION_DIR" "$BACKUP_DIR"

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

# Pre-migration assessment
run_pre_migration_assessment() {
    print_header "PRE-MIGRATION ASSESSMENT"
    
    log_info "Analyzing legacy system..."
    
    # Database assessment
    cat > "$MIGRATION_DIR/db_assessment.sql" << 'EOF'
-- Legacy Database Assessment
SELECT 
    'users' as table_name,
    COUNT(*) as record_count,
    MIN(created_at) as oldest_record,
    MAX(created_at) as newest_record
FROM users
UNION ALL
SELECT 
    'orders' as table_name,
    COUNT(*) as record_count,
    MIN(created_at) as oldest_record,
    MAX(created_at) as newest_record
FROM orders
UNION ALL
SELECT 
    'trades' as table_name,
    COUNT(*) as record_count,
    MIN(executed_at) as oldest_record,
    MAX(executed_at) as newest_record
FROM trades;
EOF
    
    # API assessment
    cat > "$MIGRATION_DIR/api_assessment.json" << 'EOF'
{
  "legacy_endpoints": [
    {"path": "/api/v1/orders", "usage": "high", "migration_priority": "critical"},
    {"path": "/api/v1/trades", "usage": "high", "migration_priority": "critical"},
    {"path": "/api/v1/users", "usage": "medium", "migration_priority": "high"},
    {"path": "/api/v1/portfolios", "usage": "medium", "migration_priority": "high"},
    {"path": "/api/v1/reports", "usage": "low", "migration_priority": "medium"}
  ],
  "integration_points": [
    {"system": "market_data_feed", "type": "websocket", "criticality": "high"},
    {"system": "clearing_house", "type": "rest_api", "criticality": "high"},
    {"system": "risk_system", "type": "message_queue", "criticality": "critical"},
    {"system": "reporting_system", "type": "batch_export", "criticality": "medium"}
  ]
}
EOF
    
    log_success "Pre-migration assessment completed"
}

# Create migration plan
create_migration_plan() {
    print_header "CREATING MIGRATION PLAN"
    
    cat > "$MIGRATION_DIR/migration_plan.md" << 'EOF'
# TradSys Migration Plan

## Migration Strategy: Phased Approach

### Phase 1: Infrastructure Setup (Week 1)
- Deploy TradSys v3 infrastructure in parallel
- Set up data replication from legacy to new system
- Configure monitoring and alerting
- Establish rollback procedures

### Phase 2: Data Migration (Week 2-3)
- Migrate historical data (users, orders, trades)
- Validate data integrity and completeness
- Set up real-time data synchronization
- Test data consistency between systems

### Phase 3: API Migration (Week 4-5)
- Deploy API compatibility layer
- Migrate non-critical endpoints first
- Gradually migrate high-traffic endpoints
- Monitor performance and error rates

### Phase 4: User Migration (Week 6-7)
- Migrate users in batches (10% per day)
- Provide dual-system access during transition
- Monitor user experience and feedback
- Address migration issues promptly

### Phase 5: Full Cutover (Week 8)
- Complete migration of all users
- Decommission legacy system
- Remove compatibility layers
- Celebrate successful migration! 🎉

## Risk Mitigation
- Comprehensive backup strategy
- Real-time monitoring and alerting
- Automated rollback procedures
- 24/7 support during migration
- Gradual user migration to minimize impact

## Success Criteria
- Zero data loss during migration
- <1% increase in error rates
- <10% increase in response times
- >95% user satisfaction score
- Complete migration within 8 weeks
EOF
    
    log_success "Migration plan created"
}

# Data migration
run_data_migration() {
    print_header "DATA MIGRATION"
    
    log_info "Starting data migration process..."
    
    # Create migration scripts
    cat > "$MIGRATION_DIR/migrate_users.sql" << 'EOF'
-- Migrate Users
INSERT INTO tradsys_v3.users (
    id, email, username, first_name, last_name, 
    phone, country, kyc_status, created_at, updated_at
)
SELECT 
    id, email, username, first_name, last_name,
    phone, country, 
    CASE 
        WHEN verification_status = 'verified' THEN 'approved'
        WHEN verification_status = 'pending' THEN 'pending'
        ELSE 'rejected'
    END as kyc_status,
    created_at, updated_at
FROM legacy_trading_db.users
WHERE active = 1;
EOF
    
    cat > "$MIGRATION_DIR/migrate_orders.sql" << 'EOF'
-- Migrate Orders
INSERT INTO tradsys_v3.orders (
    id, user_id, symbol, side, type, quantity, price,
    status, time_in_force, created_at, updated_at
)
SELECT 
    id, user_id, symbol, 
    CASE side WHEN 'B' THEN 'buy' WHEN 'S' THEN 'sell' END as side,
    CASE order_type 
        WHEN 'M' THEN 'market'
        WHEN 'L' THEN 'limit'
        WHEN 'S' THEN 'stop'
        ELSE 'limit'
    END as type,
    quantity, price,
    CASE status
        WHEN 'N' THEN 'pending'
        WHEN 'F' THEN 'filled'
        WHEN 'C' THEN 'cancelled'
        WHEN 'R' THEN 'rejected'
        ELSE 'pending'
    END as status,
    CASE time_in_force
        WHEN 'D' THEN 'day'
        WHEN 'G' THEN 'gtc'
        WHEN 'I' THEN 'ioc'
        ELSE 'gtc'
    END as time_in_force,
    created_at, updated_at
FROM legacy_trading_db.orders
WHERE created_at >= DATE_SUB(NOW(), INTERVAL 1 YEAR);
EOF
    
    log_success "Data migration scripts created"
}

# API compatibility layer
setup_api_compatibility() {
    print_header "API COMPATIBILITY LAYER"
    
    log_info "Setting up API compatibility layer..."
    
    cat > "$MIGRATION_DIR/api_compatibility.go" << 'EOF'
package migration

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

// LegacyAPIHandler provides compatibility with legacy API endpoints
type LegacyAPIHandler struct {
    newAPIClient *http.Client
    legacyAPIURL string
    newAPIURL    string
}

// HandleLegacyOrder converts legacy order format to new format
func (h *LegacyAPIHandler) HandleLegacyOrder(c *gin.Context) {
    var legacyOrder LegacyOrderRequest
    if err := c.ShouldBindJSON(&legacyOrder); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request format"})
        return
    }
    
    // Convert legacy format to new format
    newOrder := ConvertLegacyOrder(legacyOrder)
    
    // Forward to new API
    response, err := h.forwardToNewAPI("/api/v1/orders", newOrder)
    if err != nil {
        c.JSON(500, gin.H{"error": "Migration error"})
        return
    }
    
    // Convert response back to legacy format
    legacyResponse := ConvertToLegacyResponse(response)
    c.JSON(200, legacyResponse)
}

type LegacyOrderRequest struct {
    Symbol   string  `json:"symbol"`
    Side     string  `json:"side"`     // "B" or "S"
    Type     string  `json:"type"`     // "M", "L", "S"
    Quantity float64 `json:"quantity"`
    Price    float64 `json:"price"`
}

func ConvertLegacyOrder(legacy LegacyOrderRequest) map[string]interface{} {
    side := "buy"
    if legacy.Side == "S" {
        side = "sell"
    }
    
    orderType := "limit"
    switch legacy.Type {
    case "M":
        orderType = "market"
    case "S":
        orderType = "stop"
    }
    
    return map[string]interface{}{
        "symbol":   legacy.Symbol,
        "side":     side,
        "type":     orderType,
        "quantity": legacy.Quantity,
        "price":    legacy.Price,
    }
}
EOF
    
    log_success "API compatibility layer configured"
}

# User communication
setup_user_communication() {
    print_header "USER COMMUNICATION SETUP"
    
    log_info "Preparing user communication materials..."
    
    # Migration announcement
    cat > "$MIGRATION_DIR/migration_announcement.md" << 'EOF'
# Important: TradSys Platform Migration

Dear Valued Traders,

We are excited to announce the migration to our new and improved TradSys v3 platform! This upgrade will provide you with enhanced performance, better security, and new features.

## What's New in TradSys v3?
- **10x Faster Order Execution**: Sub-100μs matching engine
- **Enhanced Security**: Multi-factor authentication and advanced encryption
- **Improved User Interface**: Modern, intuitive design
- **Advanced Order Types**: Iceberg, TWAP, VWAP orders
- **Real-time Analytics**: Enhanced portfolio and risk management tools

## Migration Timeline
- **Week 1-3**: Infrastructure setup and data migration
- **Week 4-5**: API migration and testing
- **Week 6-7**: Gradual user migration (you'll receive specific instructions)
- **Week 8**: Full platform launch

## What You Need to Do
1. **Update Your API Keys**: New API keys will be provided
2. **Test Your Integrations**: Use our sandbox environment
3. **Review New Features**: Attend our webinar sessions
4. **Backup Your Data**: Export your trading history if needed

## Support During Migration
- **24/7 Support Hotline**: +1-800-TRADSYS
- **Live Chat**: Available on both platforms
- **Migration Webinars**: Weekly sessions with Q&A
- **Documentation**: Comprehensive migration guides

## Important Dates
- **Migration Start**: [DATE]
- **Your Migration Window**: [SPECIFIC_DATE_RANGE]
- **Platform Launch**: [LAUNCH_DATE]

We appreciate your patience during this transition and are committed to making this migration as smooth as possible.

Best regards,
The TradSys Team
EOF
    
    # User migration guide
    cat > "$MIGRATION_DIR/user_migration_guide.md" << 'EOF'
# User Migration Guide

## Before Migration
1. **Export Your Data**
   - Download trading history
   - Save favorite watchlists
   - Export portfolio reports

2. **Update Contact Information**
   - Verify email address
   - Update phone number
   - Confirm mailing address

3. **Review Account Settings**
   - Check risk preferences
   - Verify trading permissions
   - Update notification settings

## During Migration
1. **You Will Receive**
   - Email notification 48 hours before your migration
   - New login credentials
   - API key migration instructions
   - Direct support contact

2. **Migration Process**
   - Your account will be temporarily unavailable (2-4 hours)
   - All data will be transferred automatically
   - You'll receive confirmation when complete

3. **First Login**
   - Use new credentials provided
   - Verify your account information
   - Test basic functionality
   - Report any issues immediately

## After Migration
1. **Verify Your Data**
   - Check account balance
   - Review order history
   - Confirm portfolio positions
   - Validate watchlists

2. **Update Your Systems**
   - Install new API keys
   - Update trading software
   - Test automated strategies
   - Verify integrations

3. **Explore New Features**
   - Try the new interface
   - Test advanced order types
   - Use new analytics tools
   - Provide feedback

## Need Help?
- **Migration Hotline**: +1-800-MIGRATE
- **Email Support**: migration@tradsys.com
- **Live Chat**: Available 24/7
- **Video Tutorials**: Available on our website

## Rollback Plan
If you experience issues, we can temporarily restore access to the legacy system while we resolve problems. Contact support immediately if needed.
EOF
    
    log_success "User communication materials prepared"
}

# Rollback procedures
setup_rollback_procedures() {
    print_header "ROLLBACK PROCEDURES"
    
    log_info "Setting up rollback procedures..."
    
    cat > "$MIGRATION_DIR/rollback_plan.sh" << 'EOF'
#!/bin/bash

# Emergency Rollback Procedures
# Use only in case of critical migration issues

set -e

ROLLBACK_REASON="${1:-unspecified}"
AFFECTED_USERS="${2:-all}"

log_emergency() {
    echo "[EMERGENCY] $(date) - $1" | tee -a /var/log/migration-emergency.log
}

# Immediate rollback steps
immediate_rollback() {
    log_emergency "INITIATING EMERGENCY ROLLBACK - Reason: $ROLLBACK_REASON"
    
    # 1. Stop new system traffic
    kubectl scale deployment tradsys-api --replicas=0 -n tradsys-prod
    kubectl scale deployment tradsys-matching --replicas=0 -n tradsys-prod
    
    # 2. Redirect traffic to legacy system
    kubectl patch service tradsys-api-service -n tradsys-prod -p '{"spec":{"selector":{"app":"legacy-api"}}}'
    
    # 3. Restore database from backup
    if [ "$AFFECTED_USERS" = "all" ]; then
        mysql -u root -p$DB_PASSWORD -e "DROP DATABASE IF EXISTS tradsys_v3_db;"
        mysql -u root -p$DB_PASSWORD -e "CREATE DATABASE tradsys_v3_db;"
        mysql -u root -p$DB_PASSWORD tradsys_v3_db < "$BACKUP_DIR/pre_migration_backup.sql"
    fi
    
    # 4. Notify users
    curl -X POST "$NOTIFICATION_API/emergency" \
        -H "Content-Type: application/json" \
        -d "{\"message\":\"System temporarily restored to previous version. We apologize for any inconvenience.\",\"severity\":\"high\"}"
    
    log_emergency "ROLLBACK COMPLETED - Legacy system restored"
}

# Gradual rollback for specific users
gradual_rollback() {
    log_emergency "INITIATING GRADUAL ROLLBACK for users: $AFFECTED_USERS"
    
    # Move specific users back to legacy system
    echo "$AFFECTED_USERS" | tr ',' '\n' | while read user_id; do
        mysql -u root -p$DB_PASSWORD tradsys_v3_db -e "UPDATE users SET migration_status='rolled_back' WHERE id='$user_id';"
        log_emergency "User $user_id rolled back to legacy system"
    done
    
    log_emergency "GRADUAL ROLLBACK COMPLETED"
}

# Main rollback logic
if [ "$AFFECTED_USERS" = "all" ]; then
    immediate_rollback
else
    gradual_rollback
fi

# Send alerts
curl -X POST "$SLACK_WEBHOOK" \
    -H "Content-Type: application/json" \
    -d "{\"text\":\"🚨 MIGRATION ROLLBACK EXECUTED - Reason: $ROLLBACK_REASON, Users: $AFFECTED_USERS\"}"
EOF
    
    chmod +x "$MIGRATION_DIR/rollback_plan.sh"
    
    log_success "Rollback procedures configured"
}

# Migration monitoring
setup_migration_monitoring() {
    print_header "MIGRATION MONITORING"
    
    log_info "Setting up migration monitoring..."
    
    cat > "$MIGRATION_DIR/migration_monitor.py" << 'EOF'
#!/usr/bin/env python3

import time
import requests
import json
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class MigrationMonitor:
    def __init__(self):
        self.legacy_api = "http://legacy-api:8080"
        self.new_api = "http://tradsys-api:8080"
        self.metrics = {
            'users_migrated': 0,
            'data_consistency_errors': 0,
            'api_errors': 0,
            'performance_degradation': 0
        }
    
    def check_data_consistency(self):
        """Check data consistency between legacy and new systems"""
        try:
            # Compare user counts
            legacy_users = requests.get(f"{self.legacy_api}/api/v1/users/count").json()
            new_users = requests.get(f"{self.new_api}/api/v1/users/count").json()
            
            if abs(legacy_users['count'] - new_users['count']) > 10:
                logger.error(f"User count mismatch: Legacy={legacy_users['count']}, New={new_users['count']}")
                self.metrics['data_consistency_errors'] += 1
                return False
            
            # Compare order counts
            legacy_orders = requests.get(f"{self.legacy_api}/api/v1/orders/count").json()
            new_orders = requests.get(f"{self.new_api}/api/v1/orders/count").json()
            
            if abs(legacy_orders['count'] - new_orders['count']) > 100:
                logger.error(f"Order count mismatch: Legacy={legacy_orders['count']}, New={new_orders['count']}")
                self.metrics['data_consistency_errors'] += 1
                return False
            
            logger.info("Data consistency check passed")
            return True
            
        except Exception as e:
            logger.error(f"Data consistency check failed: {e}")
            self.metrics['data_consistency_errors'] += 1
            return False
    
    def check_api_performance(self):
        """Monitor API performance during migration"""
        try:
            start_time = time.time()
            response = requests.get(f"{self.new_api}/api/v1/health", timeout=5)
            response_time = time.time() - start_time
            
            if response.status_code != 200:
                logger.error(f"API health check failed: {response.status_code}")
                self.metrics['api_errors'] += 1
                return False
            
            if response_time > 1.0:  # 1 second threshold
                logger.warning(f"API response time degraded: {response_time:.2f}s")
                self.metrics['performance_degradation'] += 1
            
            logger.info(f"API performance check passed: {response_time:.2f}s")
            return True
            
        except Exception as e:
            logger.error(f"API performance check failed: {e}")
            self.metrics['api_errors'] += 1
            return False
    
    def send_alert(self, message, severity="warning"):
        """Send alert to monitoring system"""
        alert_data = {
            "timestamp": datetime.now().isoformat(),
            "message": message,
            "severity": severity,
            "metrics": self.metrics
        }
        
        try:
            # Send to Slack
            requests.post(
                "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK",
                json={"text": f"🚨 Migration Alert: {message}"}
            )
            
            # Send to monitoring system
            requests.post(
                "http://monitoring-api:8080/alerts",
                json=alert_data
            )
            
        except Exception as e:
            logger.error(f"Failed to send alert: {e}")
    
    def run_monitoring_cycle(self):
        """Run one complete monitoring cycle"""
        logger.info("Starting migration monitoring cycle")
        
        issues = []
        
        if not self.check_data_consistency():
            issues.append("Data consistency issues detected")
        
        if not self.check_api_performance():
            issues.append("API performance issues detected")
        
        if issues:
            self.send_alert(f"Migration issues: {', '.join(issues)}", "error")
        else:
            logger.info("All migration checks passed")
    
    def run(self):
        """Run continuous monitoring"""
        logger.info("Starting migration monitoring...")
        
        while True:
            try:
                self.run_monitoring_cycle()
                time.sleep(60)  # Check every minute
            except KeyboardInterrupt:
                logger.info("Monitoring stopped by user")
                break
            except Exception as e:
                logger.error(f"Monitoring error: {e}")
                time.sleep(60)

if __name__ == "__main__":
    monitor = MigrationMonitor()
    monitor.run()
EOF
    
    chmod +x "$MIGRATION_DIR/migration_monitor.py"
    
    log_success "Migration monitoring configured"
}

# Main execution
main() {
    print_header "TRADSYS MIGRATION STRATEGY"
    
    log_info "Initializing migration strategy..."
    log_info "Migration mode: $MIGRATION_MODE"
    log_info "Legacy database: $LEGACY_DB"
    log_info "Target database: $TARGET_DB"
    
    # Run migration preparation steps
    run_pre_migration_assessment
    create_migration_plan
    run_data_migration
    setup_api_compatibility
    setup_user_communication
    setup_rollback_procedures
    setup_migration_monitoring
    
    print_header "MIGRATION STRATEGY COMPLETE"
    
    log_success "Migration strategy prepared successfully! 🎉"
    log_info "Key deliverables:"
    log_info "  - Migration Plan: $MIGRATION_DIR/migration_plan.md"
    log_info "  - Data Migration Scripts: $MIGRATION_DIR/migrate_*.sql"
    log_info "  - API Compatibility Layer: $MIGRATION_DIR/api_compatibility.go"
    log_info "  - User Communication: $MIGRATION_DIR/migration_announcement.md"
    log_info "  - Rollback Procedures: $MIGRATION_DIR/rollback_plan.sh"
    log_info "  - Migration Monitoring: $MIGRATION_DIR/migration_monitor.py"
    
    echo ""
    echo "Next Steps:"
    echo "==========="
    echo "1. Review migration plan with stakeholders"
    echo "2. Set up staging environment for testing"
    echo "3. Schedule user communication timeline"
    echo "4. Prepare support team for migration period"
    echo "5. Execute migration plan in phases"
    
    log_success "Ready to begin migration process! 🚀"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --mode)
            MIGRATION_MODE="$2"
            shift 2
            ;;
        --legacy-db)
            LEGACY_DB="$2"
            shift 2
            ;;
        --target-db)
            TARGET_DB="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --mode MODE           Migration mode (phased|big-bang)"
            echo "  --legacy-db DB        Legacy database name"
            echo "  --target-db DB        Target database name"
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
