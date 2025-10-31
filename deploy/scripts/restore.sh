#!/bin/bash
#
# Fluxbase Restore Script
# Restores PostgreSQL database and storage from backup
#
# Usage: ./restore.sh <backup_date> [s3]
# Example: ./restore.sh 20251031_140530
#

set -euo pipefail

# Check arguments
if [ $# -lt 1 ]; then
    echo "Usage: $0 <backup_date> [s3]"
    echo "Example: $0 20251031_140530"
    exit 1
fi

BACKUP_DATE=$1
RESTORE_FROM_S3=${2:-false}

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/fluxbase}"
PGHOST="${FLUXBASE_DATABASE_HOST:-postgres}"
PGPORT="${FLUXBASE_DATABASE_PORT:-5432}"
PGUSER="${FLUXBASE_DATABASE_USER:-fluxbase}"
PGDATABASE="${FLUXBASE_DATABASE_DATABASE:-fluxbase}"

# S3 configuration
S3_BUCKET="${S3_BACKUP_BUCKET:-}"
S3_PREFIX="${S3_BACKUP_PREFIX:-fluxbase}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

log_info "Starting Fluxbase restore - $BACKUP_DATE"

# Download from S3 if requested
if [ "$RESTORE_FROM_S3" = "s3" ] && [ -n "$S3_BUCKET" ]; then
    log_info "Downloading backups from S3..."
    mkdir -p "$BACKUP_DIR"
    aws s3 cp "s3://$S3_BUCKET/$S3_PREFIX/" "$BACKUP_DIR/" --recursive --exclude "*" --include "*_${BACKUP_DATE}.*"
fi

# Check if backup files exist
DB_BACKUP_FILE="$BACKUP_DIR/db_${BACKUP_DATE}.dump"
if [ ! -f "$DB_BACKUP_FILE" ]; then
    log_error "Database backup not found: $DB_BACKUP_FILE"
    exit 1
fi

# Confirmation prompt
log_warn "⚠️  WARNING: This will OVERWRITE the current database!"
log_warn "Database: $PGDATABASE on $PGHOST"
log_warn "Backup: $DB_BACKUP_FILE"
read -p "Are you sure you want to continue? (yes/no): " -r
if [[ ! $REPLY =~ ^yes$ ]]; then
    log_info "Restore cancelled"
    exit 0
fi

# 1. Restore Database
log_info "Restoring PostgreSQL database..."
log_info "Terminating active connections..."

# Terminate existing connections
psql -h "$PGHOST" -p "$PGPORT" -U postgres -c "
SELECT pg_terminate_backend(pid) 
FROM pg_stat_activity 
WHERE datname = '$PGDATABASE' AND pid <> pg_backend_pid();" 2>/dev/null || true

# Drop and recreate database (optional - use with caution)
if [ "${DROP_DATABASE:-false}" = "true" ]; then
    log_warn "Dropping and recreating database..."
    dropdb -h "$PGHOST" -p "$PGPORT" -U postgres "$PGDATABASE" || true
    createdb -h "$PGHOST" -p "$PGPORT" -U postgres -O "$PGUSER" "$PGDATABASE"
fi

# Restore from backup
pg_restore -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" -c -v "$DB_BACKUP_FILE"
log_info "Database restored successfully"

# 2. Restore Configuration (if exists)
CONFIG_BACKUP_FILE="$BACKUP_DIR/config_${BACKUP_DATE}.tar.gz"
if [ -f "$CONFIG_BACKUP_FILE" ]; then
    log_info "Restoring configuration..."
    tar -xzf "$CONFIG_BACKUP_FILE" -C / 2>/dev/null || true
    log_info "Configuration restored"
fi

# 3. Restore Storage (if exists)
STORAGE_BACKUP_FILE="$BACKUP_DIR/storage_${BACKUP_DATE}.tar.gz"
if [ -f "$STORAGE_BACKUP_FILE" ]; then
    log_info "Restoring storage..."
    STORAGE_DIR="${FLUXBASE_STORAGE_LOCAL_PATH:-/app/storage}"
    mkdir -p "$(dirname $STORAGE_DIR)"
    tar -xzf "$STORAGE_BACKUP_FILE" -C "$(dirname $STORAGE_DIR)"
    log_info "Storage restored"
fi

log_info "✅ Restore completed successfully!"
log_info "Please restart Fluxbase to apply changes"

# Optional: Send notification
if [ -n "${RESTORE_WEBHOOK_URL:-}" ]; then
    curl -X POST "$RESTORE_WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d "{\"status\":\"restored\",\"backup_date\":\"$BACKUP_DATE\"}" \
        2>/dev/null || true
fi

exit 0
