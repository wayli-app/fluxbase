#!/bin/bash
#
# Fluxbase Backup Script
#
# Creates backups of both the PostgreSQL database and storage files.
# Supports compression, parallel jobs, retention policies, and verification.
#
# Usage:
#   ./backup.sh --output /backups
#   ./backup.sh --database-only --output /backups/db
#   ./backup.sh --output /backups --retention 30 --verify
#
# Environment Variables:
#   PGHOST          PostgreSQL host (default: localhost)
#   PGPORT          PostgreSQL port (default: 5432)
#   PGUSER          PostgreSQL user (default: postgres)
#   PGPASSWORD      PostgreSQL password
#   PGDATABASE      Database name (default: fluxbase)
#   STORAGE_PATH    Path to storage directory (default: /var/fluxbase/storage)
#   STORAGE_TYPE    Storage type: local, s3 (default: local)
#   S3_BUCKET       S3 bucket name (for s3 storage type)
#   S3_ENDPOINT     S3 endpoint URL (for MinIO/Wasabi)
#
# Exit Codes:
#   0 - Success
#   1 - General error
#   2 - Missing dependencies
#   3 - Backup failed
#   4 - Verification failed

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
OUTPUT_DIR=""
DATABASE_ONLY=false
STORAGE_ONLY=false
COMPRESS=true
PARALLEL_JOBS=4
RETENTION_DAYS=0
VERIFY=false
METRICS_FILE=""
QUIET=false

# Database connection defaults
PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5432}"
PGUSER="${PGUSER:-postgres}"
PGDATABASE="${PGDATABASE:-fluxbase}"

# Storage defaults
STORAGE_PATH="${STORAGE_PATH:-/var/fluxbase/storage}"
STORAGE_TYPE="${STORAGE_TYPE:-local}"
S3_BUCKET="${S3_BUCKET:-}"
S3_ENDPOINT="${S3_ENDPOINT:-}"

# Timestamp for backup naming
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DATE_ONLY=$(date +%Y%m%d)

log() {
    if [ "$QUIET" = false ]; then
        echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
    fi
}

warn() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1" >&2
}

error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1" >&2
}

usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS]

Creates backups of Fluxbase database and storage.

Options:
  --output DIR        Backup destination directory (required)
  --database-only     Only backup database, skip storage
  --storage-only      Only backup storage, skip database
  --no-compress       Disable compression
  --parallel N        Number of parallel jobs for pg_dump (default: 4)
  --retention DAYS    Delete backups older than N days (0 = disabled)
  --verify            Verify backup integrity after creation
  --metrics FILE      Write Prometheus metrics to file
  --quiet             Suppress non-error output
  -h, --help          Show this help message

Environment Variables:
  PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE
  STORAGE_PATH, STORAGE_TYPE, S3_BUCKET, S3_ENDPOINT

Examples:
  # Full backup
  $(basename "$0") --output /backups

  # Database-only with verification
  $(basename "$0") --database-only --output /backups --verify

  # With 30-day retention policy
  $(basename "$0") --output /backups --retention 30

EOF
    exit 0
}

check_dependencies() {
    local missing=()

    if [ "$DATABASE_ONLY" = false ] || [ "$STORAGE_ONLY" = false ]; then
        if ! command -v pg_dump &> /dev/null; then
            missing+=("pg_dump")
        fi
    fi

    if [ "$STORAGE_ONLY" = true ] || [ "$DATABASE_ONLY" = false ]; then
        if [ "$STORAGE_TYPE" = "local" ]; then
            if ! command -v tar &> /dev/null; then
                missing+=("tar")
            fi
        elif [ "$STORAGE_TYPE" = "s3" ]; then
            if ! command -v aws &> /dev/null; then
                missing+=("aws-cli")
            fi
        fi
    fi

    if [ "$VERIFY" = true ] && ! command -v pg_restore &> /dev/null; then
        missing+=("pg_restore")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        error "Missing dependencies: ${missing[*]}"
        exit 2
    fi
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --database-only)
                DATABASE_ONLY=true
                shift
                ;;
            --storage-only)
                STORAGE_ONLY=true
                shift
                ;;
            --no-compress)
                COMPRESS=false
                shift
                ;;
            --parallel)
                PARALLEL_JOBS="$2"
                shift 2
                ;;
            --retention)
                RETENTION_DAYS="$2"
                shift 2
                ;;
            --verify)
                VERIFY=true
                shift
                ;;
            --metrics)
                METRICS_FILE="$2"
                shift 2
                ;;
            --quiet)
                QUIET=true
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                error "Unknown option: $1"
                usage
                ;;
        esac
    done

    if [ -z "$OUTPUT_DIR" ]; then
        error "Output directory is required. Use --output DIR"
        exit 1
    fi

    if [ "$DATABASE_ONLY" = true ] && [ "$STORAGE_ONLY" = true ]; then
        error "Cannot specify both --database-only and --storage-only"
        exit 1
    fi
}

backup_database() {
    local backup_file="$OUTPUT_DIR/database_${TIMESTAMP}.dump"

    log "Starting database backup..."
    log "  Host: $PGHOST:$PGPORT"
    log "  Database: $PGDATABASE"
    log "  Output: $backup_file"

    local pg_args=(-h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE")

    # Use custom format for parallel restore support
    pg_args+=(-Fc)

    # Add parallel jobs if supported
    if [ "$PARALLEL_JOBS" -gt 1 ]; then
        pg_args+=(-j "$PARALLEL_JOBS")
    fi

    # Exclude temporary/cache tables if they exist
    pg_args+=(--exclude-table-data='*.pg_stat_*')

    if pg_dump "${pg_args[@]}" -f "$backup_file" 2>&1; then
        local size
        size=$(du -h "$backup_file" | cut -f1)
        log "Database backup completed: $size"

        # Update metrics
        if [ -n "$METRICS_FILE" ]; then
            local bytes
            bytes=$(stat -c%s "$backup_file" 2>/dev/null || stat -f%z "$backup_file" 2>/dev/null)
            echo "fluxbase_backup_size_bytes{type=\"database\"} $bytes" >> "$METRICS_FILE"
            echo "fluxbase_backup_last_success_timestamp{type=\"database\"} $(date +%s)" >> "$METRICS_FILE"
        fi

        echo "$backup_file"
    else
        error "Database backup failed"
        return 3
    fi
}

backup_storage_local() {
    local backup_file="$OUTPUT_DIR/storage_${TIMESTAMP}.tar"

    if [ "$COMPRESS" = true ]; then
        backup_file="${backup_file}.gz"
    fi

    log "Starting local storage backup..."
    log "  Source: $STORAGE_PATH"
    log "  Output: $backup_file"

    if [ ! -d "$STORAGE_PATH" ]; then
        warn "Storage path does not exist: $STORAGE_PATH"
        return 0
    fi

    local tar_args=(--create)

    if [ "$COMPRESS" = true ]; then
        tar_args+=(--gzip)
    fi

    tar_args+=(--file "$backup_file" -C "$(dirname "$STORAGE_PATH")" "$(basename "$STORAGE_PATH")")

    if tar "${tar_args[@]}" 2>&1; then
        local size
        size=$(du -h "$backup_file" | cut -f1)
        log "Storage backup completed: $size"

        # Update metrics
        if [ -n "$METRICS_FILE" ]; then
            local bytes
            bytes=$(stat -c%s "$backup_file" 2>/dev/null || stat -f%z "$backup_file" 2>/dev/null)
            echo "fluxbase_backup_size_bytes{type=\"storage\"} $bytes" >> "$METRICS_FILE"
            echo "fluxbase_backup_last_success_timestamp{type=\"storage\"} $(date +%s)" >> "$METRICS_FILE"
        fi

        echo "$backup_file"
    else
        error "Storage backup failed"
        return 3
    fi
}

backup_storage_s3() {
    local backup_bucket="${S3_BUCKET}-backup-${DATE_ONLY}"

    log "Starting S3 storage backup..."
    log "  Source: s3://$S3_BUCKET"
    log "  Destination: s3://$backup_bucket"

    local aws_args=(s3 sync "s3://$S3_BUCKET" "s3://$backup_bucket")

    if [ -n "$S3_ENDPOINT" ]; then
        aws_args+=(--endpoint-url "$S3_ENDPOINT")
    fi

    if aws "${aws_args[@]}" 2>&1; then
        log "S3 storage backup completed"

        # Update metrics
        if [ -n "$METRICS_FILE" ]; then
            echo "fluxbase_backup_last_success_timestamp{type=\"storage\"} $(date +%s)" >> "$METRICS_FILE"
        fi

        echo "s3://$backup_bucket"
    else
        error "S3 storage backup failed"
        return 3
    fi
}

backup_storage() {
    case "$STORAGE_TYPE" in
        local)
            backup_storage_local
            ;;
        s3)
            backup_storage_s3
            ;;
        *)
            error "Unknown storage type: $STORAGE_TYPE"
            return 1
            ;;
    esac
}

verify_database_backup() {
    local backup_file="$1"
    local test_db="fluxbase_verify_$$"

    log "Verifying database backup..."

    # Create test database
    if ! createdb -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" "$test_db" 2>/dev/null; then
        error "Failed to create verification database"
        return 4
    fi

    # Restore to test database
    if ! pg_restore -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$test_db" \
        --no-owner --no-privileges "$backup_file" 2>/dev/null; then
        dropdb -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" "$test_db" 2>/dev/null || true
        error "Failed to restore backup for verification"
        return 4
    fi

    # Check critical tables
    local user_count
    user_count=$(psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$test_db" -t \
        -c "SELECT count(*) FROM auth.users" 2>/dev/null | tr -d ' ')

    # Clean up
    dropdb -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" "$test_db" 2>/dev/null || true

    if [ -z "$user_count" ]; then
        warn "Could not verify auth.users table"
    else
        log "Verification passed: $user_count users in backup"
    fi

    return 0
}

verify_storage_backup() {
    local backup_file="$1"

    log "Verifying storage backup..."

    if [[ "$backup_file" == s3://* ]]; then
        # S3 verification - just check bucket exists
        local bucket
        bucket=$(echo "$backup_file" | sed 's|s3://||')
        if aws s3 ls "s3://$bucket" --max-items 1 >/dev/null 2>&1; then
            log "S3 backup verified"
        else
            error "S3 backup verification failed"
            return 4
        fi
    else
        # Local verification - check archive integrity
        if [[ "$backup_file" == *.gz ]]; then
            if gzip -t "$backup_file" 2>/dev/null; then
                log "Storage backup verified (gzip integrity OK)"
            else
                error "Storage backup verification failed (corrupt gzip)"
                return 4
            fi
        else
            if tar -tf "$backup_file" >/dev/null 2>&1; then
                log "Storage backup verified (tar integrity OK)"
            else
                error "Storage backup verification failed (corrupt tar)"
                return 4
            fi
        fi
    fi

    return 0
}

cleanup_old_backups() {
    if [ "$RETENTION_DAYS" -le 0 ]; then
        return 0
    fi

    log "Cleaning up backups older than $RETENTION_DAYS days..."

    local deleted=0

    # Find and delete old database backups
    while IFS= read -r -d '' file; do
        rm -f "$file"
        ((deleted++))
    done < <(find "$OUTPUT_DIR" -name "database_*.dump*" -type f -mtime +"$RETENTION_DAYS" -print0 2>/dev/null)

    # Find and delete old storage backups
    while IFS= read -r -d '' file; do
        rm -f "$file"
        ((deleted++))
    done < <(find "$OUTPUT_DIR" -name "storage_*.tar*" -type f -mtime +"$RETENTION_DAYS" -print0 2>/dev/null)

    if [ "$deleted" -gt 0 ]; then
        log "Deleted $deleted old backup files"
    fi
}

write_manifest() {
    local manifest_file="$OUTPUT_DIR/backup_${TIMESTAMP}.manifest"

    cat > "$manifest_file" << EOF
# Fluxbase Backup Manifest
# Created: $(date -Iseconds)
# Host: $(hostname)

[backup]
timestamp=$TIMESTAMP
database_host=$PGHOST
database_name=$PGDATABASE
storage_type=$STORAGE_TYPE
compressed=$COMPRESS

[files]
EOF

    for file in "$@"; do
        if [ -n "$file" ]; then
            echo "$file" >> "$manifest_file"
        fi
    done

    log "Manifest written: $manifest_file"
}

main() {
    parse_args "$@"
    check_dependencies

    # Create output directory
    mkdir -p "$OUTPUT_DIR"

    # Initialize metrics file
    if [ -n "$METRICS_FILE" ]; then
        cat > "$METRICS_FILE" << EOF
# HELP fluxbase_backup_size_bytes Backup size in bytes
# TYPE fluxbase_backup_size_bytes gauge
# HELP fluxbase_backup_last_success_timestamp Last successful backup timestamp
# TYPE fluxbase_backup_last_success_timestamp gauge
EOF
    fi

    local db_backup=""
    local storage_backup=""
    local exit_code=0

    # Database backup
    if [ "$STORAGE_ONLY" = false ]; then
        if db_backup=$(backup_database); then
            if [ "$VERIFY" = true ]; then
                verify_database_backup "$db_backup" || exit_code=$?
            fi
        else
            exit_code=3
        fi
    fi

    # Storage backup
    if [ "$DATABASE_ONLY" = false ]; then
        if storage_backup=$(backup_storage); then
            if [ "$VERIFY" = true ]; then
                verify_storage_backup "$storage_backup" || exit_code=$?
            fi
        else
            exit_code=3
        fi
    fi

    # Write manifest
    if [ "$exit_code" -eq 0 ]; then
        write_manifest "$db_backup" "$storage_backup"
    fi

    # Cleanup old backups
    cleanup_old_backups

    if [ "$exit_code" -eq 0 ]; then
        log "Backup completed successfully"
    else
        error "Backup completed with errors (exit code: $exit_code)"
    fi

    exit "$exit_code"
}

main "$@"
