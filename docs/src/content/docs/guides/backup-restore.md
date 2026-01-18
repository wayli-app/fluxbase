---
title: "Backup & Restore"
---

This guide covers backup and restore procedures for Fluxbase deployments. Regular backups are critical for disaster recovery and should be part of your operational practices.

## Overview

Fluxbase stores data in two primary locations:
- **PostgreSQL Database**: User data, authentication, metadata, jobs, webhooks
- **Storage Backend**: Uploaded files (local filesystem or S3-compatible storage)

Both must be backed up together to ensure consistent recovery.

---

## Quick Start

Fluxbase provides backup and restore scripts for common scenarios:

```bash
# Full backup (database + storage)
./scripts/backup.sh --output /backups/$(date +%Y%m%d)

# Database-only backup
./scripts/backup.sh --database-only --output /backups/db-$(date +%Y%m%d).sql

# Restore from backup
./scripts/restore.sh --backup /backups/20260118
```

---

## Database Backup

### Using pg_dump (Recommended for Small-Medium Databases)

```bash
# Full backup with custom format (supports parallel restore)
pg_dump -Fc -h localhost -U postgres -d fluxbase \
  -f fluxbase_$(date +%Y%m%d_%H%M%S).dump

# Plain SQL backup (readable, portable)
pg_dump -h localhost -U postgres -d fluxbase \
  -f fluxbase_$(date +%Y%m%d_%H%M%S).sql

# Schema-only backup (useful for migrations)
pg_dump -h localhost -U postgres -d fluxbase \
  --schema-only -f fluxbase_schema.sql
```

**Recommended Options:**
| Option | Description |
|--------|-------------|
| `-Fc` | Custom format - compressed, supports parallel restore |
| `-j N` | Number of parallel jobs (for pg_dump 9.3+) |
| `--no-owner` | Don't include ownership commands |
| `--no-privileges` | Don't include privilege grants |
| `-T pattern` | Exclude tables matching pattern |

### Using pg_basebackup (For Large Databases)

For databases larger than 10GB, use physical backups:

```bash
# Base backup with WAL files
pg_basebackup -h localhost -U postgres -D /backups/base_$(date +%Y%m%d) \
  -Ft -z -P -Xs

# Options:
# -Ft: tar format
# -z: gzip compression
# -P: show progress
# -Xs: stream WAL files during backup
```

### Continuous Archiving (WAL Archiving)

For point-in-time recovery, configure WAL archiving in `postgresql.conf`:

```ini
# Enable WAL archiving
wal_level = replica
archive_mode = on
archive_command = 'cp %p /wal_archive/%f'
archive_timeout = 300  # Force archive every 5 minutes

# Retention (optional, using pg_archivecleanup)
# archive_cleanup_command = 'pg_archivecleanup /wal_archive %r'
```

---

## Storage Backup

### Local Filesystem Storage

```bash
# Using rsync (incremental, efficient)
rsync -avz --delete /var/fluxbase/storage/ /backups/storage/

# Using tar (full archive)
tar -czf storage_$(date +%Y%m%d).tar.gz /var/fluxbase/storage/
```

### S3-Compatible Storage

```bash
# Using AWS CLI (works with MinIO, Wasabi, etc.)
aws s3 sync s3://fluxbase-storage s3://fluxbase-backup-$(date +%Y%m%d) \
  --source-region us-east-1

# Using rclone (supports many providers)
rclone sync fluxbase:storage backup:storage-$(date +%Y%m%d)
```

**Enable S3 Versioning** for automatic file history:
```bash
aws s3api put-bucket-versioning \
  --bucket fluxbase-storage \
  --versioning-configuration Status=Enabled
```

---

## Automated Backup Script

The provided `scripts/backup.sh` handles common backup scenarios:

```bash
# Environment variables
export PGHOST=localhost
export PGUSER=postgres
export PGDATABASE=fluxbase
export STORAGE_PATH=/var/fluxbase/storage
export BACKUP_RETENTION_DAYS=30

# Run backup
./scripts/backup.sh --output /backups
```

### Script Options

| Option | Description |
|--------|-------------|
| `--output DIR` | Backup destination directory |
| `--database-only` | Skip storage backup |
| `--storage-only` | Skip database backup |
| `--compress` | Compress backup files (default: enabled) |
| `--parallel N` | Parallel jobs for pg_dump (default: 4) |
| `--retention DAYS` | Delete backups older than N days |
| `--verify` | Verify backup integrity after creation |

### Scheduling with Cron

```bash
# Daily backup at 2 AM
0 2 * * * /opt/fluxbase/scripts/backup.sh --output /backups --retention 30 >> /var/log/fluxbase-backup.log 2>&1

# Weekly full backup on Sunday
0 3 * * 0 /opt/fluxbase/scripts/backup.sh --output /backups/weekly >> /var/log/fluxbase-backup.log 2>&1
```

---

## Database Restore

### From Custom Format Dump

```bash
# Create target database
createdb -h localhost -U postgres fluxbase_restored

# Restore with parallel jobs
pg_restore -h localhost -U postgres -d fluxbase_restored \
  -j 4 --no-owner --no-privileges \
  fluxbase_20260118.dump

# Verify restoration
psql -h localhost -U postgres -d fluxbase_restored \
  -c "SELECT count(*) FROM auth.users;"
```

### From Plain SQL Dump

```bash
# Create target database
createdb -h localhost -U postgres fluxbase_restored

# Restore from SQL file
psql -h localhost -U postgres -d fluxbase_restored \
  -f fluxbase_20260118.sql
```

### Point-in-Time Recovery (PITR)

For WAL-archived databases:

```bash
# 1. Stop PostgreSQL
sudo systemctl stop postgresql

# 2. Clear data directory
rm -rf /var/lib/postgresql/14/main/*

# 3. Restore base backup
tar -xzf base_20260118.tar.gz -C /var/lib/postgresql/14/main/

# 4. Create recovery.signal and configure recovery target
cat > /var/lib/postgresql/14/main/postgresql.auto.conf << EOF
restore_command = 'cp /wal_archive/%f %p'
recovery_target_time = '2026-01-18 10:30:00'
recovery_target_action = 'promote'
EOF

# 5. Start PostgreSQL (will recover to target time)
sudo systemctl start postgresql
```

---

## Storage Restore

### Local Filesystem

```bash
# Stop Fluxbase to prevent writes
sudo systemctl stop fluxbase

# Restore from rsync backup
rsync -avz /backups/storage/ /var/fluxbase/storage/

# Restore from tar archive
tar -xzf storage_20260118.tar.gz -C /

# Fix permissions
chown -R fluxbase:fluxbase /var/fluxbase/storage

# Restart Fluxbase
sudo systemctl start fluxbase
```

### S3-Compatible Storage

```bash
# Sync from backup bucket
aws s3 sync s3://fluxbase-backup-20260118 s3://fluxbase-storage \
  --delete

# Or restore specific version (if versioning enabled)
aws s3api list-object-versions --bucket fluxbase-storage --prefix uploads/
aws s3api get-object --bucket fluxbase-storage --key uploads/file.jpg \
  --version-id "abc123" restored-file.jpg
```

---

## Automated Restore Script

The provided `scripts/restore.sh` handles common restore scenarios:

```bash
# Full restore
./scripts/restore.sh --backup /backups/20260118

# Database-only restore to different database
./scripts/restore.sh --backup /backups/20260118 \
  --database-only --target-db fluxbase_test

# Dry run (verify without restoring)
./scripts/restore.sh --backup /backups/20260118 --dry-run
```

### Script Options

| Option | Description |
|--------|-------------|
| `--backup DIR` | Backup directory to restore from |
| `--database-only` | Restore only database |
| `--storage-only` | Restore only storage |
| `--target-db NAME` | Restore to different database name |
| `--dry-run` | Verify backup without restoring |
| `--no-stop` | Don't stop Fluxbase during restore |

---

## Backup Verification

Always verify backups can be restored:

```bash
# 1. Restore to test database
createdb fluxbase_verify
pg_restore -d fluxbase_verify --no-owner fluxbase_backup.dump

# 2. Run verification queries
psql -d fluxbase_verify << 'EOF'
-- Check table counts
SELECT 'auth.users' as table_name, count(*) FROM auth.users
UNION ALL
SELECT 'storage.objects', count(*) FROM storage.objects
UNION ALL
SELECT 'jobs.jobs', count(*) FROM jobs.jobs;

-- Check for data integrity
SELECT 'orphaned_identities' as check_name, count(*)
FROM auth.identities i
LEFT JOIN auth.users u ON i.user_id = u.id
WHERE u.id IS NULL;
EOF

# 3. Clean up
dropdb fluxbase_verify
```

### Automated Verification

Add to your backup script:

```bash
#!/bin/bash
# Verify backup after creation
verify_backup() {
    local backup_file=$1
    local test_db="fluxbase_verify_$(date +%s)"

    createdb "$test_db" || return 1
    pg_restore -d "$test_db" --no-owner "$backup_file" || return 1

    # Check critical tables
    local user_count=$(psql -d "$test_db" -t -c "SELECT count(*) FROM auth.users")
    if [ "$user_count" -eq 0 ]; then
        echo "WARNING: No users found in backup"
    fi

    dropdb "$test_db"
    return 0
}
```

---

## Disaster Recovery Checklist

### Before Disaster

- [ ] Automated daily backups configured
- [ ] Backups stored in separate location/region
- [ ] Backup verification running weekly
- [ ] Recovery procedure documented and tested
- [ ] Recovery time objective (RTO) defined
- [ ] Recovery point objective (RPO) defined
- [ ] Team trained on recovery procedures

### During Recovery

1. **Assess the situation**
   - Identify what failed (database, storage, both)
   - Determine data loss window
   - Choose recovery target time

2. **Communicate**
   - Notify stakeholders
   - Set expectations for downtime

3. **Execute recovery**
   ```bash
   # Stop application
   kubectl scale deployment fluxbase --replicas=0

   # Restore database
   ./scripts/restore.sh --backup /backups/latest --database-only

   # Restore storage
   ./scripts/restore.sh --backup /backups/latest --storage-only

   # Verify data
   ./scripts/restore.sh --backup /backups/latest --dry-run

   # Start application
   kubectl scale deployment fluxbase --replicas=3
   ```

4. **Verify recovery**
   - Check application health endpoints
   - Verify user authentication works
   - Test file uploads/downloads
   - Review application logs for errors

5. **Post-mortem**
   - Document what happened
   - Identify improvements
   - Update procedures if needed

---

## Backup Strategy Recommendations

### Development/Testing
- Daily pg_dump backups
- 7-day retention
- Manual verification monthly

### Production (Small-Medium)
- Daily pg_dump with custom format
- Hourly WAL archiving
- 30-day retention
- Weekly automated verification
- Off-site backup replication

### Production (Large/Critical)
- Continuous WAL archiving (5-minute intervals)
- Physical backups (pg_basebackup) weekly
- Real-time replication to standby
- 90-day retention
- Daily automated verification
- Multi-region backup storage
- Tested disaster recovery quarterly

---

## Monitoring Backup Health

### Prometheus Metrics

If you're using the provided backup script, it exports metrics:

```
# HELP fluxbase_backup_last_success_timestamp Last successful backup timestamp
# TYPE fluxbase_backup_last_success_timestamp gauge
fluxbase_backup_last_success_timestamp{type="database"} 1705574400
fluxbase_backup_last_success_timestamp{type="storage"} 1705574400

# HELP fluxbase_backup_size_bytes Backup size in bytes
# TYPE fluxbase_backup_size_bytes gauge
fluxbase_backup_size_bytes{type="database"} 1073741824
fluxbase_backup_size_bytes{type="storage"} 5368709120
```

### Alerting Rules

```yaml
groups:
- name: backup
  rules:
  - alert: BackupMissing
    expr: time() - fluxbase_backup_last_success_timestamp > 86400
    for: 1h
    labels:
      severity: critical
    annotations:
      summary: "Fluxbase backup is missing"
      description: "No successful backup in the last 24 hours"

  - alert: BackupSizeAnomaly
    expr: |
      abs(fluxbase_backup_size_bytes - fluxbase_backup_size_bytes offset 1d)
      / fluxbase_backup_size_bytes offset 1d > 0.5
    for: 1h
    labels:
      severity: warning
    annotations:
      summary: "Backup size changed significantly"
      description: "Backup size changed by more than 50%"
```

---

## Learn More

- [Deployment Overview](/docs/deployment/overview) - Production deployment guide
- [Production Checklist](/docs/deployment/production-checklist) - Pre-production checklist
- [Monitoring & Observability](/docs/guides/monitoring-observability) - Monitoring setup
- [Scaling](/docs/deployment/scaling) - Scaling Fluxbase

