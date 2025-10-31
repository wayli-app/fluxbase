#!/bin/bash
#
# Fluxbase Backup Script
# Backs up PostgreSQL database and storage to local or S3
#
# Usage: ./backup.sh [local|s3]
#

set -euo pipefail

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/fluxbase}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_MODE="${1:-local}"

# Database configuration
PGHOST="${FLUXBASE_DATABASE_HOST:-postgres}"
PGPORT="${FLUXBASE_DATABASE_PORT:-5432}"
PGUSER="${FLUXBASE_DATABASE_USER:-fluxbase}"
PGDATABASE="${FLUXBASE_DATABASE_DATABASE:-fluxbase}"

# S3 configuration (optional)
S3_BUCKET="${S3_BACKUP_BUCKET:-}"
S3_PREFIX="${S3_BACKUP_PREFIX:-fluxbase}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create backup directory
mkdir -p "$BACKUP_DIR"

log_info "Starting Fluxbase backup - $DATE"

# 1. Backup PostgreSQL Database
log_info "Backing up PostgreSQL database..."
DB_BACKUP_FILE="$BACKUP_DIR/db_$DATE.dump"

if command -v pg_dump &> /dev/null; then
    pg_dump -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -Fc "$PGDATABASE" > "$DB_BACKUP_FILE"
    log_info "Database backup completed: $DB_BACKUP_FILE"
else
    log_error "pg_dump not found. Install postgresql-client"
    exit 1
fi

# 2. Backup Configuration
log_info "Backing up configuration..."
CONFIG_BACKUP_FILE="$BACKUP_DIR/config_$DATE.tar.gz"

if [ -d "/app/config" ]; then
    tar -czf "$CONFIG_BACKUP_FILE" -C /app config/ 2>/dev/null || true
elif [ -d "/etc/fluxbase" ]; then
    tar -czf "$CONFIG_BACKUP_FILE" /etc/fluxbase/ 2>/dev/null || true
fi

if [ -f "$CONFIG_BACKUP_FILE" ]; then
    log_info "Config backup completed: $CONFIG_BACKUP_FILE"
fi

# 3. Backup Storage (if using local storage)
STORAGE_DIR="${FLUXBASE_STORAGE_LOCAL_PATH:-/app/storage}"
if [ -d "$STORAGE_DIR" ] && [ "$(ls -A $STORAGE_DIR 2>/dev/null)" ]; then
    log_info "Backing up local storage..."
    STORAGE_BACKUP_FILE="$BACKUP_DIR/storage_$DATE.tar.gz"
    tar -czf "$STORAGE_BACKUP_FILE" -C "$(dirname $STORAGE_DIR)" "$(basename $STORAGE_DIR)"
    log_info "Storage backup completed: $STORAGE_BACKUP_FILE"
fi

# 4. Generate backup manifest
MANIFEST_FILE="$BACKUP_DIR/manifest_$DATE.txt"
cat > "$MANIFEST_FILE" << EOF
Fluxbase Backup Manifest
========================
Date: $DATE
Database: $PGDATABASE
Host: $PGHOST

Files:
$(ls -lh $BACKUP_DIR/*_$DATE.*)
EOF

log_info "Backup manifest: $MANIFEST_FILE"

# 5. Upload to S3 (if configured)
if [ "$BACKUP_MODE" = "s3" ] && [ -n "$S3_BUCKET" ]; then
    log_info "Uploading backups to S3..."

    if command -v aws &> /dev/null; then
        for file in $BACKUP_DIR/*_$DATE.*; do
            aws s3 cp "$file" "s3://$S3_BUCKET/$S3_PREFIX/$(basename $file)"
        done
        log_info "S3 upload completed"
    else
        log_warn "AWS CLI not found. Skipping S3 upload"
    fi
fi

# 6. Clean old backups (retention policy)
log_info "Cleaning old backups (retention: $RETENTION_DAYS days)..."
find "$BACKUP_DIR" -type f -mtime +$RETENTION_DAYS -delete
log_info "Old backups cleaned"

# 7. Calculate backup size
TOTAL_SIZE=$(du -sh "$BACKUP_DIR" | cut -f1)
log_info "Total backup size: $TOTAL_SIZE"

log_info "Backup completed successfully!"

# Optional: Send notification (webhook, email, Slack, etc.)
if [ -n "${BACKUP_WEBHOOK_URL:-}" ]; then
    curl -X POST "$BACKUP_WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d "{\"status\":\"success\",\"date\":\"$DATE\",\"size\":\"$TOTAL_SIZE\"}" \
        2>/dev/null || true
fi

exit 0
