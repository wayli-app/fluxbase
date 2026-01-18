#!/bin/bash
#
# Fluxbase Restore Script
#
# Restores Fluxbase database and storage from backups.
# Supports dry-run mode, target database selection, and verification.
#
# Usage:
#   ./restore.sh --backup /backups/20260118
#   ./restore.sh --backup /backups/20260118 --database-only
#   ./restore.sh --backup /backups/20260118 --dry-run
#
# Environment Variables:
#   PGHOST          PostgreSQL host (default: localhost)
#   PGPORT          PostgreSQL port (default: 5432)
#   PGUSER          PostgreSQL user (default: postgres)
#   PGPASSWORD      PostgreSQL password
#   PGDATABASE      Target database name (default: fluxbase)
#   STORAGE_PATH    Path to storage directory (default: /var/fluxbase/storage)
#   STORAGE_TYPE    Storage type: local, s3 (default: local)
#   S3_BUCKET       S3 bucket name (for s3 storage type)
#   S3_ENDPOINT     S3 endpoint URL (for MinIO/Wasabi)
#
# Exit Codes:
#   0 - Success
#   1 - General error
#   2 - Missing dependencies
#   3 - Restore failed
#   4 - Verification failed
#   5 - Backup not found

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
BACKUP_PATH=""
DATABASE_ONLY=false
STORAGE_ONLY=false
TARGET_DB=""
DRY_RUN=false
NO_STOP=false
FORCE=false
PARALLEL_JOBS=4

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

log() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1" >&2
}

error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1" >&2
}

info() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] INFO:${NC} $1"
}

usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS]

Restores Fluxbase database and storage from backups.

Options:
  --backup DIR        Backup directory or file to restore from (required)
  --database-only     Only restore database, skip storage
  --storage-only      Only restore storage, skip database
  --target-db NAME    Restore database to different name
  --dry-run           Verify backup without restoring
  --no-stop           Don't stop Fluxbase during restore
  --force             Skip confirmation prompts
  --parallel N        Number of parallel jobs for pg_restore (default: 4)
  -h, --help          Show this help message

Environment Variables:
  PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE
  STORAGE_PATH, STORAGE_TYPE, S3_BUCKET, S3_ENDPOINT

Examples:
  # Full restore
  $(basename "$0") --backup /backups/20260118

  # Database-only to different database
  $(basename "$0") --backup /backups/20260118 --database-only --target-db fluxbase_restored

  # Dry run to verify backup
  $(basename "$0") --backup /backups/20260118 --dry-run

EOF
    exit 0
}

check_dependencies() {
    local missing=()

    if ! command -v pg_restore &> /dev/null; then
        missing+=("pg_restore")
    fi

    if ! command -v psql &> /dev/null; then
        missing+=("psql")
    fi

    if [ "$STORAGE_TYPE" = "local" ]; then
        if ! command -v tar &> /dev/null; then
            missing+=("tar")
        fi
    elif [ "$STORAGE_TYPE" = "s3" ]; then
        if ! command -v aws &> /dev/null; then
            missing+=("aws-cli")
        fi
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        error "Missing dependencies: ${missing[*]}"
        exit 2
    fi
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --backup)
                BACKUP_PATH="$2"
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
            --target-db)
                TARGET_DB="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --no-stop)
                NO_STOP=true
                shift
                ;;
            --force)
                FORCE=true
                shift
                ;;
            --parallel)
                PARALLEL_JOBS="$2"
                shift 2
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

    if [ -z "$BACKUP_PATH" ]; then
        error "Backup path is required. Use --backup DIR"
        exit 1
    fi

    if [ "$DATABASE_ONLY" = true ] && [ "$STORAGE_ONLY" = true ]; then
        error "Cannot specify both --database-only and --storage-only"
        exit 1
    fi

    # Set target database
    if [ -z "$TARGET_DB" ]; then
        TARGET_DB="$PGDATABASE"
    fi
}

find_backup_files() {
    local backup_dir="$1"
    local db_file=""
    local storage_file=""

    # If a specific file was provided, use it
    if [ -f "$backup_dir" ]; then
        if [[ "$backup_dir" == *.dump* ]]; then
            db_file="$backup_dir"
        elif [[ "$backup_dir" == *.tar* ]]; then
            storage_file="$backup_dir"
        fi
    elif [ -d "$backup_dir" ]; then
        # Find most recent database backup
        db_file=$(find "$backup_dir" -name "database_*.dump*" -type f 2>/dev/null | sort -r | head -1)
        # Find most recent storage backup
        storage_file=$(find "$backup_dir" -name "storage_*.tar*" -type f 2>/dev/null | sort -r | head -1)
    fi

    echo "$db_file|$storage_file"
}

verify_backup() {
    local backup_file="$1"
    local backup_type="$2"

    if [ ! -f "$backup_file" ]; then
        error "Backup file not found: $backup_file"
        return 5
    fi

    log "Verifying $backup_type backup: $backup_file"

    case "$backup_type" in
        database)
            # Check if it's a valid PostgreSQL dump
            if ! pg_restore --list "$backup_file" >/dev/null 2>&1; then
                error "Invalid or corrupt database backup"
                return 4
            fi

            # Show backup contents
            local table_count
            table_count=$(pg_restore --list "$backup_file" 2>/dev/null | grep -c "TABLE DATA" || echo "0")
            info "  Tables with data: $table_count"

            local size
            size=$(du -h "$backup_file" | cut -f1)
            info "  File size: $size"
            ;;

        storage)
            if [[ "$backup_file" == *.gz ]]; then
                if ! gzip -t "$backup_file" 2>/dev/null; then
                    error "Corrupt gzip archive"
                    return 4
                fi
                local file_count
                file_count=$(tar -tzf "$backup_file" 2>/dev/null | wc -l)
                info "  Files in archive: $file_count"
            else
                if ! tar -tf "$backup_file" >/dev/null 2>&1; then
                    error "Corrupt tar archive"
                    return 4
                fi
                local file_count
                file_count=$(tar -tf "$backup_file" 2>/dev/null | wc -l)
                info "  Files in archive: $file_count"
            fi

            local size
            size=$(du -h "$backup_file" | cut -f1)
            info "  File size: $size"
            ;;
    esac

    log "Backup verification passed"
    return 0
}

confirm_restore() {
    if [ "$FORCE" = true ]; then
        return 0
    fi

    echo ""
    warn "This will restore data to:"
    warn "  Database: $TARGET_DB on $PGHOST:$PGPORT"
    if [ "$DATABASE_ONLY" = false ]; then
        warn "  Storage: $STORAGE_PATH"
    fi
    echo ""

    if [ "$TARGET_DB" = "$PGDATABASE" ]; then
        warn "WARNING: This will OVERWRITE the production database!"
    fi

    read -p "Are you sure you want to continue? (yes/no): " -r
    if [[ ! "$REPLY" =~ ^[Yy][Ee][Ss]$ ]]; then
        log "Restore cancelled"
        exit 0
    fi
}

stop_fluxbase() {
    if [ "$NO_STOP" = true ]; then
        warn "Continuing without stopping Fluxbase (--no-stop specified)"
        return 0
    fi

    log "Stopping Fluxbase service..."

    # Try systemd first
    if command -v systemctl &> /dev/null && systemctl is-active fluxbase &>/dev/null; then
        sudo systemctl stop fluxbase
        log "Stopped Fluxbase (systemd)"
        return 0
    fi

    # Try docker
    if command -v docker &> /dev/null && docker ps -q -f name=fluxbase &>/dev/null; then
        docker stop fluxbase
        log "Stopped Fluxbase (docker)"
        return 0
    fi

    # Try kubectl
    if command -v kubectl &> /dev/null; then
        if kubectl get deployment fluxbase &>/dev/null 2>&1; then
            kubectl scale deployment fluxbase --replicas=0
            log "Stopped Fluxbase (kubernetes)"
            return 0
        fi
    fi

    warn "Could not detect Fluxbase service. Proceeding without stopping."
}

start_fluxbase() {
    if [ "$NO_STOP" = true ]; then
        return 0
    fi

    log "Starting Fluxbase service..."

    # Try systemd first
    if command -v systemctl &> /dev/null && systemctl list-unit-files fluxbase.service &>/dev/null 2>&1; then
        sudo systemctl start fluxbase
        log "Started Fluxbase (systemd)"
        return 0
    fi

    # Try docker
    if command -v docker &> /dev/null; then
        docker start fluxbase 2>/dev/null && log "Started Fluxbase (docker)" && return 0
    fi

    # Try kubectl
    if command -v kubectl &> /dev/null; then
        if kubectl get deployment fluxbase &>/dev/null 2>&1; then
            kubectl scale deployment fluxbase --replicas=1
            log "Started Fluxbase (kubernetes)"
            return 0
        fi
    fi

    warn "Could not restart Fluxbase service automatically"
}

restore_database() {
    local backup_file="$1"

    log "Restoring database from: $backup_file"
    log "  Target database: $TARGET_DB"

    # Check if target database exists
    if psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -lqt | cut -d \| -f 1 | grep -qw "$TARGET_DB"; then
        if [ "$TARGET_DB" = "$PGDATABASE" ]; then
            warn "Target database exists. Dropping and recreating..."

            # Terminate existing connections
            psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d postgres -c \
                "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$TARGET_DB' AND pid <> pg_backend_pid();" \
                >/dev/null 2>&1 || true

            # Drop and recreate
            dropdb -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" "$TARGET_DB" || {
                error "Failed to drop existing database"
                return 3
            }
        fi
    fi

    # Create target database
    log "Creating database: $TARGET_DB"
    createdb -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" "$TARGET_DB" || {
        error "Failed to create target database"
        return 3
    }

    # Restore
    log "Restoring data (this may take a while)..."

    local restore_args=(-h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$TARGET_DB")
    restore_args+=(--no-owner --no-privileges)

    if [ "$PARALLEL_JOBS" -gt 1 ]; then
        restore_args+=(-j "$PARALLEL_JOBS")
    fi

    if pg_restore "${restore_args[@]}" "$backup_file" 2>&1; then
        log "Database restore completed successfully"
    else
        # pg_restore returns non-zero even for warnings, so check if data exists
        local user_count
        user_count=$(psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$TARGET_DB" -t \
            -c "SELECT count(*) FROM auth.users" 2>/dev/null | tr -d ' ')

        if [ -n "$user_count" ] && [ "$user_count" -gt 0 ]; then
            warn "Restore completed with warnings (this is often normal)"
            log "  Users restored: $user_count"
        else
            error "Database restore may have failed. Check database contents."
            return 3
        fi
    fi
}

restore_storage_local() {
    local backup_file="$1"

    log "Restoring local storage from: $backup_file"
    log "  Target path: $STORAGE_PATH"

    # Backup current storage if it exists
    if [ -d "$STORAGE_PATH" ]; then
        local backup_timestamp
        backup_timestamp=$(date +%Y%m%d_%H%M%S)
        local old_storage="${STORAGE_PATH}.old.${backup_timestamp}"

        warn "Moving existing storage to: $old_storage"
        mv "$STORAGE_PATH" "$old_storage"
    fi

    # Create parent directory
    mkdir -p "$(dirname "$STORAGE_PATH")"

    # Extract backup
    local tar_args=(--extract)

    if [[ "$backup_file" == *.gz ]]; then
        tar_args+=(--gzip)
    fi

    tar_args+=(--file "$backup_file" -C "$(dirname "$STORAGE_PATH")")

    if tar "${tar_args[@]}" 2>&1; then
        log "Storage restore completed"

        # Fix permissions if running as root
        if [ "$(id -u)" = "0" ] && id fluxbase &>/dev/null; then
            chown -R fluxbase:fluxbase "$STORAGE_PATH"
            log "Fixed storage permissions"
        fi
    else
        error "Storage restore failed"
        return 3
    fi
}

restore_storage_s3() {
    local backup_bucket="$1"

    log "Restoring S3 storage from: $backup_bucket"
    log "  Target bucket: s3://$S3_BUCKET"

    local aws_args=(s3 sync "$backup_bucket" "s3://$S3_BUCKET" --delete)

    if [ -n "$S3_ENDPOINT" ]; then
        aws_args+=(--endpoint-url "$S3_ENDPOINT")
    fi

    if aws "${aws_args[@]}" 2>&1; then
        log "S3 storage restore completed"
    else
        error "S3 storage restore failed"
        return 3
    fi
}

restore_storage() {
    local backup_file="$1"

    case "$STORAGE_TYPE" in
        local)
            restore_storage_local "$backup_file"
            ;;
        s3)
            restore_storage_s3 "$backup_file"
            ;;
        *)
            error "Unknown storage type: $STORAGE_TYPE"
            return 1
            ;;
    esac
}

post_restore_verification() {
    log "Running post-restore verification..."

    # Check database
    local checks_passed=0
    local checks_total=0

    # Check auth.users
    ((checks_total++))
    local user_count
    user_count=$(psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$TARGET_DB" -t \
        -c "SELECT count(*) FROM auth.users" 2>/dev/null | tr -d ' ')
    if [ -n "$user_count" ]; then
        info "  auth.users: $user_count records"
        ((checks_passed++))
    else
        warn "  auth.users: check failed"
    fi

    # Check storage.objects
    ((checks_total++))
    local object_count
    object_count=$(psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$TARGET_DB" -t \
        -c "SELECT count(*) FROM storage.objects" 2>/dev/null | tr -d ' ')
    if [ -n "$object_count" ]; then
        info "  storage.objects: $object_count records"
        ((checks_passed++))
    else
        warn "  storage.objects: check failed"
    fi

    # Check jobs.jobs
    ((checks_total++))
    local job_count
    job_count=$(psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$TARGET_DB" -t \
        -c "SELECT count(*) FROM jobs.jobs" 2>/dev/null | tr -d ' ')
    if [ -n "$job_count" ]; then
        info "  jobs.jobs: $job_count records"
        ((checks_passed++))
    else
        warn "  jobs.jobs: check failed"
    fi

    log "Verification: $checks_passed/$checks_total checks passed"

    if [ "$checks_passed" -lt "$checks_total" ]; then
        warn "Some verification checks failed. Please review manually."
    fi
}

main() {
    parse_args "$@"
    check_dependencies

    # Find backup files
    IFS='|' read -r db_file storage_file <<< "$(find_backup_files "$BACKUP_PATH")"

    log "Backup discovery:"
    if [ -n "$db_file" ]; then
        info "  Database: $db_file"
    else
        if [ "$STORAGE_ONLY" = false ]; then
            error "No database backup found in: $BACKUP_PATH"
            exit 5
        fi
    fi

    if [ -n "$storage_file" ]; then
        info "  Storage: $storage_file"
    else
        if [ "$DATABASE_ONLY" = false ]; then
            warn "No storage backup found in: $BACKUP_PATH"
        fi
    fi

    # Verify backups
    if [ "$STORAGE_ONLY" = false ] && [ -n "$db_file" ]; then
        verify_backup "$db_file" "database" || exit $?
    fi

    if [ "$DATABASE_ONLY" = false ] && [ -n "$storage_file" ]; then
        verify_backup "$storage_file" "storage" || exit $?
    fi

    # Dry run ends here
    if [ "$DRY_RUN" = true ]; then
        log "Dry run completed. Backups are valid."
        exit 0
    fi

    # Confirm restore
    confirm_restore

    # Stop Fluxbase
    stop_fluxbase

    local exit_code=0

    # Restore database
    if [ "$STORAGE_ONLY" = false ] && [ -n "$db_file" ]; then
        restore_database "$db_file" || exit_code=$?
    fi

    # Restore storage
    if [ "$DATABASE_ONLY" = false ] && [ -n "$storage_file" ]; then
        restore_storage "$storage_file" || exit_code=$?
    fi

    # Post-restore verification
    if [ "$exit_code" -eq 0 ] && [ "$STORAGE_ONLY" = false ]; then
        post_restore_verification
    fi

    # Start Fluxbase
    start_fluxbase

    if [ "$exit_code" -eq 0 ]; then
        log "Restore completed successfully!"
        echo ""
        info "Next steps:"
        info "  1. Verify application is running: curl http://localhost:8080/health"
        info "  2. Test user authentication"
        info "  3. Check file uploads/downloads"
        info "  4. Review application logs for errors"
    else
        error "Restore completed with errors (exit code: $exit_code)"
    fi

    exit "$exit_code"
}

main "$@"
