#!/bin/bash

# TradSys Production Backup Script
# Comprehensive backup solution for database, configurations, and critical data

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/backups/tradsys}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
S3_BUCKET="${S3_BUCKET:-tradsys-backups}"
ENCRYPTION_KEY="${ENCRYPTION_KEY:-/etc/tradsys/backup.key}"
NOTIFICATION_WEBHOOK="${NOTIFICATION_WEBHOOK:-}"

# Database configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-tradsys}"
DB_USER="${DB_USER:-tradsys}"
DB_PASSWORD="${DB_PASSWORD:-}"

# Redis configuration
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"

# Kubernetes configuration
KUBE_NAMESPACE="${KUBE_NAMESPACE:-tradsys-prod}"

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

# Send notification
send_notification() {
    local status="$1"
    local message="$2"
    
    if [ -n "$NOTIFICATION_WEBHOOK" ]; then
        curl -X POST "$NOTIFICATION_WEBHOOK" \
            -H "Content-Type: application/json" \
            -d "{\"text\":\"🔄 TradSys Backup $status: $message\"}" \
            2>/dev/null || log_warning "Failed to send notification"
    fi
}

# Create backup directory structure
create_backup_structure() {
    local timestamp="$1"
    local backup_path="$BACKUP_DIR/$timestamp"
    
    mkdir -p "$backup_path"/{database,redis,configs,kubernetes,logs}
    echo "$backup_path"
}

# Backup PostgreSQL database
backup_database() {
    local backup_path="$1"
    local timestamp="$2"
    
    log_info "Starting database backup..."
    
    # Set password for pg_dump
    export PGPASSWORD="$DB_PASSWORD"
    
    # Create database dump
    pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
        --verbose --no-password --format=custom \
        --file="$backup_path/database/tradsys_${timestamp}.dump"
    
    if [ $? -eq 0 ]; then
        log_success "Database backup completed"
        
        # Create schema-only backup for quick recovery testing
        pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
            --schema-only --no-password \
            --file="$backup_path/database/schema_${timestamp}.sql"
        
        # Backup database statistics
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
            -c "SELECT schemaname, tablename, n_tup_ins, n_tup_upd, n_tup_del FROM pg_stat_user_tables;" \
            --csv > "$backup_path/database/stats_${timestamp}.csv"
        
        log_success "Database schema and statistics backed up"
    else
        log_error "Database backup failed"
        return 1
    fi
    
    unset PGPASSWORD
}

# Backup Redis data
backup_redis() {
    local backup_path="$1"
    local timestamp="$2"
    
    log_info "Starting Redis backup..."
    
    # Create Redis backup using BGSAVE
    if [ -n "$REDIS_PASSWORD" ]; then
        redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" BGSAVE
    else
        redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" BGSAVE
    fi
    
    # Wait for background save to complete
    while true; do
        if [ -n "$REDIS_PASSWORD" ]; then
            result=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" LASTSAVE)
        else
            result=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" LASTSAVE)
        fi
        
        if [ "$result" != "$last_save" ]; then
            break
        fi
        sleep 1
    done
    
    # Copy Redis dump file
    if [ -n "$REDIS_PASSWORD" ]; then
        redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" -a "$REDIS_PASSWORD" \
            --rdb "$backup_path/redis/dump_${timestamp}.rdb"
    else
        redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" \
            --rdb "$backup_path/redis/dump_${timestamp}.rdb"
    fi
    
    if [ $? -eq 0 ]; then
        log_success "Redis backup completed"
    else
        log_error "Redis backup failed"
        return 1
    fi
}

# Backup Kubernetes configurations
backup_kubernetes() {
    local backup_path="$1"
    local timestamp="$2"
    
    log_info "Starting Kubernetes configuration backup..."
    
    # Backup all resources in the namespace
    kubectl get all -n "$KUBE_NAMESPACE" -o yaml > "$backup_path/kubernetes/all_resources_${timestamp}.yaml"
    
    # Backup ConfigMaps
    kubectl get configmaps -n "$KUBE_NAMESPACE" -o yaml > "$backup_path/kubernetes/configmaps_${timestamp}.yaml"
    
    # Backup Secrets (metadata only for security)
    kubectl get secrets -n "$KUBE_NAMESPACE" -o yaml | \
        sed 's/data:.*/data: <REDACTED>/g' > "$backup_path/kubernetes/secrets_metadata_${timestamp}.yaml"
    
    # Backup PersistentVolumeClaims
    kubectl get pvc -n "$KUBE_NAMESPACE" -o yaml > "$backup_path/kubernetes/pvc_${timestamp}.yaml"
    
    # Backup Ingress
    kubectl get ingress -n "$KUBE_NAMESPACE" -o yaml > "$backup_path/kubernetes/ingress_${timestamp}.yaml"
    
    # Backup NetworkPolicies
    kubectl get networkpolicies -n "$KUBE_NAMESPACE" -o yaml > "$backup_path/kubernetes/networkpolicies_${timestamp}.yaml"
    
    # Backup RBAC
    kubectl get rolebindings,roles -n "$KUBE_NAMESPACE" -o yaml > "$backup_path/kubernetes/rbac_${timestamp}.yaml"
    
    log_success "Kubernetes configuration backup completed"
}

# Backup application configurations
backup_configs() {
    local backup_path="$1"
    local timestamp="$2"
    
    log_info "Starting configuration backup..."
    
    # Backup local configuration files
    if [ -d "/etc/tradsys" ]; then
        cp -r /etc/tradsys "$backup_path/configs/etc_tradsys_${timestamp}"
    fi
    
    # Backup application configs from ConfigMaps
    kubectl get configmap tradsys-config -n "$KUBE_NAMESPACE" -o yaml > \
        "$backup_path/configs/app_config_${timestamp}.yaml"
    
    # Backup environment-specific configurations
    if [ -f "/opt/tradsys/config.yaml" ]; then
        cp "/opt/tradsys/config.yaml" "$backup_path/configs/app_config_${timestamp}.yaml"
    fi
    
    log_success "Configuration backup completed"
}

# Backup application logs
backup_logs() {
    local backup_path="$1"
    local timestamp="$2"
    
    log_info "Starting log backup..."
    
    # Backup recent application logs from Kubernetes
    kubectl logs -n "$KUBE_NAMESPACE" -l app=tradsys-api --tail=10000 > \
        "$backup_path/logs/api_logs_${timestamp}.log" 2>/dev/null || true
    
    kubectl logs -n "$KUBE_NAMESPACE" -l app=tradsys-matching --tail=10000 > \
        "$backup_path/logs/matching_logs_${timestamp}.log" 2>/dev/null || true
    
    kubectl logs -n "$KUBE_NAMESPACE" -l app=tradsys-risk --tail=10000 > \
        "$backup_path/logs/risk_logs_${timestamp}.log" 2>/dev/null || true
    
    # Backup system logs if available
    if [ -d "/var/log/tradsys" ]; then
        cp -r /var/log/tradsys "$backup_path/logs/system_logs_${timestamp}"
    fi
    
    log_success "Log backup completed"
}

# Encrypt backup
encrypt_backup() {
    local backup_path="$1"
    local timestamp="$2"
    
    if [ ! -f "$ENCRYPTION_KEY" ]; then
        log_warning "Encryption key not found, skipping encryption"
        return 0
    fi
    
    log_info "Encrypting backup..."
    
    # Create encrypted archive
    tar -czf - -C "$BACKUP_DIR" "$timestamp" | \
        openssl enc -aes-256-cbc -salt -kfile "$ENCRYPTION_KEY" > \
        "$BACKUP_DIR/${timestamp}.tar.gz.enc"
    
    if [ $? -eq 0 ]; then
        # Remove unencrypted backup
        rm -rf "$backup_path"
        log_success "Backup encrypted successfully"
    else
        log_error "Backup encryption failed"
        return 1
    fi
}

# Upload to S3
upload_to_s3() {
    local timestamp="$1"
    local file_path="$BACKUP_DIR/${timestamp}.tar.gz.enc"
    
    if [ -z "$S3_BUCKET" ]; then
        log_warning "S3 bucket not configured, skipping upload"
        return 0
    fi
    
    log_info "Uploading backup to S3..."
    
    aws s3 cp "$file_path" "s3://$S3_BUCKET/tradsys/$(date +%Y/%m/%d)/" \
        --storage-class STANDARD_IA \
        --metadata "backup-type=full,timestamp=$timestamp,retention-days=$RETENTION_DAYS"
    
    if [ $? -eq 0 ]; then
        log_success "Backup uploaded to S3 successfully"
    else
        log_error "S3 upload failed"
        return 1
    fi
}

# Cleanup old backups
cleanup_old_backups() {
    log_info "Cleaning up old backups (retention: $RETENTION_DAYS days)..."
    
    # Cleanup local backups
    find "$BACKUP_DIR" -name "*.tar.gz.enc" -mtime +$RETENTION_DAYS -delete
    find "$BACKUP_DIR" -type d -mtime +$RETENTION_DAYS -exec rm -rf {} + 2>/dev/null || true
    
    # Cleanup S3 backups if configured
    if [ -n "$S3_BUCKET" ]; then
        aws s3api list-objects-v2 --bucket "$S3_BUCKET" --prefix "tradsys/" \
            --query "Contents[?LastModified<='$(date -d "$RETENTION_DAYS days ago" --iso-8601)'].Key" \
            --output text | xargs -I {} aws s3 rm "s3://$S3_BUCKET/{}" 2>/dev/null || true
    fi
    
    log_success "Old backup cleanup completed"
}

# Verify backup integrity
verify_backup() {
    local timestamp="$1"
    local file_path="$BACKUP_DIR/${timestamp}.tar.gz.enc"
    
    log_info "Verifying backup integrity..."
    
    if [ ! -f "$file_path" ]; then
        log_error "Backup file not found: $file_path"
        return 1
    fi
    
    # Test decryption
    if [ -f "$ENCRYPTION_KEY" ]; then
        openssl enc -aes-256-cbc -d -kfile "$ENCRYPTION_KEY" -in "$file_path" | \
            tar -tzf - > /dev/null 2>&1
        
        if [ $? -eq 0 ]; then
            log_success "Backup integrity verified"
        else
            log_error "Backup integrity check failed"
            return 1
        fi
    else
        log_warning "Cannot verify encrypted backup without encryption key"
    fi
    
    # Check file size
    file_size=$(stat -f%z "$file_path" 2>/dev/null || stat -c%s "$file_path" 2>/dev/null)
    if [ "$file_size" -lt 1024 ]; then
        log_error "Backup file is suspiciously small: $file_size bytes"
        return 1
    fi
    
    log_success "Backup verification completed"
}

# Generate backup report
generate_report() {
    local timestamp="$1"
    local backup_path="$BACKUP_DIR/${timestamp}.tar.gz.enc"
    local report_file="$BACKUP_DIR/backup_report_${timestamp}.txt"
    
    cat > "$report_file" << EOF
TradSys Backup Report
====================

Backup Timestamp: $timestamp
Backup Date: $(date)
Backup File: $backup_path
File Size: $(du -h "$backup_path" 2>/dev/null | cut -f1 || echo "Unknown")

Components Backed Up:
- PostgreSQL Database: ✓
- Redis Cache: ✓
- Kubernetes Configurations: ✓
- Application Configurations: ✓
- Application Logs: ✓

Backup Status: SUCCESS
Encryption: $([ -f "$ENCRYPTION_KEY" ] && echo "Enabled" || echo "Disabled")
S3 Upload: $([ -n "$S3_BUCKET" ] && echo "Enabled" || echo "Disabled")
Retention: $RETENTION_DAYS days

Next Backup: $(date -d "+1 day")
EOF
    
    log_success "Backup report generated: $report_file"
}

# Main backup function
main() {
    local timestamp=$(date +%Y%m%d_%H%M%S)
    
    log_info "Starting TradSys backup process..."
    log_info "Backup timestamp: $timestamp"
    
    # Send start notification
    send_notification "STARTED" "Backup process initiated at $(date)"
    
    # Create backup directory structure
    local backup_path=$(create_backup_structure "$timestamp")
    log_info "Backup directory created: $backup_path"
    
    # Perform backups
    local backup_success=true
    
    if ! backup_database "$backup_path" "$timestamp"; then
        backup_success=false
    fi
    
    if ! backup_redis "$backup_path" "$timestamp"; then
        backup_success=false
    fi
    
    if ! backup_kubernetes "$backup_path" "$timestamp"; then
        backup_success=false
    fi
    
    backup_configs "$backup_path" "$timestamp"
    backup_logs "$backup_path" "$timestamp"
    
    if [ "$backup_success" = false ]; then
        log_error "Some backup components failed"
        send_notification "FAILED" "Backup process failed at $(date)"
        exit 1
    fi
    
    # Encrypt backup
    if ! encrypt_backup "$backup_path" "$timestamp"; then
        log_error "Backup encryption failed"
        send_notification "FAILED" "Backup encryption failed at $(date)"
        exit 1
    fi
    
    # Upload to S3
    upload_to_s3 "$timestamp"
    
    # Verify backup
    if ! verify_backup "$timestamp"; then
        log_error "Backup verification failed"
        send_notification "FAILED" "Backup verification failed at $(date)"
        exit 1
    fi
    
    # Cleanup old backups
    cleanup_old_backups
    
    # Generate report
    generate_report "$timestamp"
    
    log_success "TradSys backup completed successfully!"
    send_notification "SUCCESS" "Backup completed successfully at $(date)"
}

# Check dependencies
check_dependencies() {
    local missing_deps=()
    
    command -v pg_dump >/dev/null 2>&1 || missing_deps+=("postgresql-client")
    command -v redis-cli >/dev/null 2>&1 || missing_deps+=("redis-tools")
    command -v kubectl >/dev/null 2>&1 || missing_deps+=("kubectl")
    command -v openssl >/dev/null 2>&1 || missing_deps+=("openssl")
    command -v aws >/dev/null 2>&1 || missing_deps+=("awscli")
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing dependencies: ${missing_deps[*]}"
        exit 1
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --help              Show this help message"
            echo "  --dry-run           Show what would be backed up without doing it"
            echo "  --verify-only       Only verify existing backups"
            echo "  --cleanup-only      Only cleanup old backups"
            echo ""
            echo "Environment Variables:"
            echo "  BACKUP_DIR          Backup directory (default: /backups/tradsys)"
            echo "  RETENTION_DAYS      Backup retention in days (default: 30)"
            echo "  S3_BUCKET           S3 bucket for backup storage"
            echo "  ENCRYPTION_KEY      Path to encryption key file"
            echo "  DB_HOST             Database host (default: localhost)"
            echo "  DB_NAME             Database name (default: tradsys)"
            echo "  DB_USER             Database user (default: tradsys)"
            echo "  DB_PASSWORD         Database password"
            exit 0
            ;;
        --dry-run)
            log_info "DRY RUN MODE - No actual backup will be performed"
            exit 0
            ;;
        --verify-only)
            log_info "VERIFY ONLY MODE"
            # Find latest backup and verify
            latest_backup=$(ls -t "$BACKUP_DIR"/*.tar.gz.enc 2>/dev/null | head -1)
            if [ -n "$latest_backup" ]; then
                timestamp=$(basename "$latest_backup" .tar.gz.enc)
                verify_backup "$timestamp"
            else
                log_error "No backups found to verify"
                exit 1
            fi
            exit 0
            ;;
        --cleanup-only)
            log_info "CLEANUP ONLY MODE"
            cleanup_old_backups
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Check dependencies and run main function
check_dependencies
main
