---
title: "Operational Runbook"
---

This runbook provides guidance for operating Fluxbase in production, including troubleshooting common issues, handling incidents, and maintaining system health.

## Overview

This document covers:
- Common failure scenarios and remediation
- Database troubleshooting
- Performance debugging
- Security incident response
- Monitoring alert responses

---

## Quick Reference

### Health Check Endpoints

| Endpoint | Purpose | Expected Response |
|----------|---------|-------------------|
| `GET /health` | Basic health check | `{"status":"healthy"}` |
| `GET /api/v1/monitoring/health` | Detailed component health | JSON with component status |
| `GET /metrics` | Prometheus metrics | Prometheus text format |

### Common Issues Quick Fix

| Symptom | First Check | Quick Fix |
|---------|-------------|-----------|
| 502 Bad Gateway | Database connection | Restart PostgreSQL connection pool |
| High latency | Connection pool exhaustion | Increase `database.max_connections` |
| Memory growth | RLS cache size | Reduce `realtime.rls_cache_size` |
| WebSocket drops | Slow clients | Check `realtime.slow_client_threshold` |

---

## Database Troubleshooting

### Connection Pool Exhaustion

**Symptoms:**
- Requests timing out
- "too many connections" errors in logs
- High `fluxbase_db_connections_waiting` metric

**Diagnosis:**
```sql
-- Check active connections
SELECT count(*), state
FROM pg_stat_activity
WHERE datname = 'fluxbase'
GROUP BY state;

-- Find long-running queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query
FROM pg_stat_activity
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes'
ORDER BY duration DESC;
```

**Remediation:**
1. Identify and terminate long-running queries:
```sql
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE duration > interval '5 minutes'
AND state != 'idle';
```

2. Increase pool size if consistently at limit:
```yaml
database:
  max_connections: 100  # Up from default 50
```

3. Review application for connection leaks (queries not being closed).

---

### Slow Queries

**Symptoms:**
- High p99 latency
- Elevated `fluxbase_db_query_duration_seconds` histogram

**Diagnosis:**
```sql
-- Enable slow query logging (temporarily)
ALTER SYSTEM SET log_min_duration_statement = '100ms';
SELECT pg_reload_conf();

-- Check for missing indexes
SELECT schemaname, tablename, attname, null_frac, n_distinct, correlation
FROM pg_stats
WHERE schemaname = 'public' AND tablename = 'your_table';

-- Analyze query plan
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT * FROM your_query;
```

**Remediation:**
1. Add missing indexes:
```sql
CREATE INDEX CONCURRENTLY idx_users_email ON auth.users(email);
```

2. Update table statistics:
```sql
ANALYZE your_table;
```

3. Consider query optimization or pagination.

---

### Replication Lag (If Using Replicas)

**Symptoms:**
- Stale data in read queries
- `pg_stat_replication.replay_lag` increasing

**Diagnosis:**
```sql
-- Check replication status
SELECT client_addr, state, sent_lsn, replay_lsn,
       sent_lsn - replay_lsn AS lag_bytes
FROM pg_stat_replication;
```

**Remediation:**
1. Check replica disk I/O and network
2. Increase `max_wal_senders` if needed
3. Consider reducing write-heavy workloads temporarily

---

## Performance Debugging

### High CPU Usage

**Diagnosis:**
1. Check process-level CPU:
```bash
top -c -p $(pgrep -f fluxbase)
```

2. Enable CPU profiling (if compiled with pprof):
```bash
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof
```

3. Check for hot paths in metrics:
```bash
curl -s http://localhost:8080/metrics | grep -E 'fluxbase_(http|db)_requests_total'
```

**Common Causes:**
- JSON serialization of large responses
- Complex RLS policies
- Inefficient filter queries

**Remediation:**
1. Add pagination to large result sets
2. Simplify RLS policies
3. Add database indexes

---

### High Memory Usage

**Diagnosis:**
```bash
# Check process memory
ps aux | grep fluxbase

# If pprof enabled
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**Common Causes:**
- Large RLS cache
- Many concurrent WebSocket connections
- Large response bodies in memory

**Remediation:**
1. Reduce cache sizes:
```yaml
realtime:
  rls_cache_size: 50000  # Down from 100000
  rls_cache_ttl: 15s     # Down from 30s
```

2. Limit concurrent connections:
```yaml
realtime:
  max_connections_per_user: 5
  max_connections_per_ip: 10
```

3. Add response size limits

---

### High Latency

**Diagnosis:**
1. Check component latencies in metrics:
```bash
curl -s http://localhost:8080/metrics | grep -E 'duration.*bucket'
```

2. Check database query times:
```sql
SELECT query, calls, mean_exec_time, total_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 20;
```

3. Check network latency to database:
```bash
ping -c 10 your-database-host
```

**Remediation:**
1. Add caching (Redis/Dragonfly) for frequent queries
2. Optimize slow queries (indexes, query rewrite)
3. Scale horizontally if single-instance bottleneck

---

## Realtime/WebSocket Issues

### Connections Dropping

**Symptoms:**
- Clients frequently reconnecting
- "connection closed" errors in client logs

**Diagnosis:**
```bash
# Check connection metrics
curl -s http://localhost:8080/metrics | grep -E 'fluxbase_realtime'

# Check for slow clients
curl -s http://localhost:8080/api/v1/monitoring/metrics | jq '.realtime'
```

**Common Causes:**
- Slow clients not consuming messages fast enough
- Network issues between client and server
- Server resource exhaustion

**Remediation:**
1. Increase slow client timeout:
```yaml
realtime:
  slow_client_timeout: 60s  # Up from 30s
  slow_client_threshold: 200  # Up from 100
```

2. Reduce message frequency if broadcasting too much
3. Check load balancer/proxy timeouts

---

### Messages Not Delivered

**Symptoms:**
- Clients subscribed but not receiving updates
- Database changes not triggering notifications

**Diagnosis:**
```sql
-- Check if pg_notify is working
NOTIFY test_channel, 'test message';

-- Check if triggers exist
SELECT * FROM pg_trigger
WHERE tgname LIKE 'fluxbase%';
```

**Remediation:**
1. Verify realtime is enabled on the table:
```sql
-- Check for realtime trigger
SELECT * FROM pg_trigger
WHERE tgrelid = 'your_table'::regclass;
```

2. Restart the listener if stuck:
```bash
# API endpoint to reset realtime (if available)
curl -X POST http://localhost:8080/api/v1/admin/realtime/restart
```

---

## Security Incident Response

### Suspected Account Compromise

**Immediate Actions:**
1. Revoke all sessions for affected user:
```sql
DELETE FROM auth.sessions WHERE user_id = 'affected-user-uuid';
```

2. Reset user password:
```sql
UPDATE auth.users
SET encrypted_password = crypt('temporary-password', gen_salt('bf'))
WHERE id = 'affected-user-uuid';
```

3. Revoke OAuth tokens:
```sql
DELETE FROM auth.identities WHERE user_id = 'affected-user-uuid';
```

4. Check audit logs for suspicious activity:
```sql
SELECT * FROM auth.audit_log_entries
WHERE actor_id = 'affected-user-uuid'
ORDER BY created_at DESC
LIMIT 100;
```

---

### Service Key Compromise

**Immediate Actions:**
1. Revoke the compromised key:
```bash
curl -X POST http://localhost:8080/api/v1/admin/service-keys/{key-id}/revoke \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"reason": "Security incident - suspected compromise"}'
```

2. Review access logs:
```bash
grep "X-Service-Key" /var/log/fluxbase/access.log | tail -1000
```

3. Rotate all service keys as precaution:
```bash
# List all active keys
curl http://localhost:8080/api/v1/admin/service-keys

# Rotate each key
curl -X POST http://localhost:8080/api/v1/admin/service-keys/{key-id}/rotate
```

---

### DDoS or Rate Limit Abuse

**Symptoms:**
- Spike in request volume
- Many 429 Too Many Requests responses
- Single IP making excessive requests

**Immediate Actions:**
1. Check top requesters:
```bash
# If using nginx
awk '{print $1}' /var/log/nginx/access.log | sort | uniq -c | sort -rn | head -20
```

2. Block abusive IPs at firewall level:
```bash
# Using iptables
iptables -A INPUT -s 1.2.3.4 -j DROP

# Using UFW
ufw deny from 1.2.3.4
```

3. Enable stricter rate limiting temporarily:
```yaml
server:
  rate_limit:
    requests_per_second: 10  # Down from default
    burst: 20
```

---

## Storage Issues

### Disk Space Running Low

**Diagnosis:**
```bash
# Check disk usage
df -h

# Find large files
du -sh /var/fluxbase/storage/* | sort -rh | head -20

# Check database size
psql -c "SELECT pg_size_pretty(pg_database_size('fluxbase'));"
```

**Remediation:**
1. Clean up old storage objects:
```sql
-- Find orphaned objects
SELECT * FROM storage.objects o
LEFT JOIN storage.buckets b ON o.bucket_id = b.id
WHERE b.id IS NULL;

-- Delete old objects (be careful!)
DELETE FROM storage.objects
WHERE created_at < NOW() - INTERVAL '90 days'
AND bucket_id = 'temp-uploads';
```

2. Vacuum database:
```sql
VACUUM FULL VERBOSE;
```

3. Archive old data

---

### Storage Upload Failures

**Symptoms:**
- 500 errors on file uploads
- "no space left on device" in logs

**Diagnosis:**
```bash
# Check storage backend
ls -la /var/fluxbase/storage/

# Check S3 connectivity (if using S3)
aws s3 ls s3://your-bucket --region your-region
```

**Remediation:**
1. For local storage: expand disk or clean up
2. For S3: check IAM permissions and bucket policy
3. Verify storage configuration in fluxbase.yaml

---

## Background Jobs Issues

### Jobs Stuck in Pending

**Symptoms:**
- Jobs not executing
- `fluxbase_jobs_pending` metric growing

**Diagnosis:**
```sql
-- Check pending jobs
SELECT id, name, status, scheduled_at, attempts
FROM jobs.jobs
WHERE status = 'pending'
ORDER BY scheduled_at
LIMIT 20;

-- Check worker status
SELECT * FROM jobs.workers WHERE status = 'active';
```

**Remediation:**
1. Check if workers are running:
```bash
curl http://localhost:8080/api/v1/monitoring/metrics | jq '.jobs.workers'
```

2. Restart workers if stuck
3. Increase worker concurrency:
```yaml
jobs:
  max_concurrent_per_worker: 10  # Up from 5
```

---

### Jobs Failing Repeatedly

**Diagnosis:**
```sql
-- Check failed jobs with errors
SELECT id, name, attempts, error, updated_at
FROM jobs.jobs
WHERE status = 'failed'
ORDER BY updated_at DESC
LIMIT 20;
```

**Remediation:**
1. Fix underlying error (check job code)
2. Retry failed jobs:
```sql
UPDATE jobs.jobs
SET status = 'pending', attempts = 0, error = NULL
WHERE id = 'job-uuid';
```

3. Adjust retry policy if needed

---

## Alerting Response Guide

### Alert: High Error Rate

**Threshold:** Error rate > 1% of requests

**Response:**
1. Check recent deployments for regressions
2. Review error logs for common patterns
3. Check downstream dependencies (database, storage)
4. Consider rollback if recently deployed

---

### Alert: Database Connection Saturation

**Threshold:** Available connections < 10%

**Response:**
1. Terminate idle long-held connections
2. Review application for connection leaks
3. Scale database or increase connection limits

---

### Alert: Disk Space Critical

**Threshold:** < 10% disk space remaining

**Response:**
1. Identify largest space consumers
2. Clean up logs, temp files
3. Expand storage if needed
4. Set up automated cleanup

---

### Alert: Memory Usage High

**Threshold:** > 90% memory usage

**Response:**
1. Check for memory leaks (heap profile)
2. Reduce cache sizes
3. Restart service if immediate relief needed
4. Scale vertically if persistent

---

## Maintenance Procedures

### Rolling Restart

For zero-downtime restarts across multiple instances:

```bash
# If using Kubernetes
kubectl rollout restart deployment/fluxbase

# If using systemd (per-instance)
for host in host1 host2 host3; do
  ssh $host "sudo systemctl restart fluxbase"
  sleep 30  # Wait for health check
done
```

---

### Database Maintenance

**Weekly:**
```sql
-- Update statistics
ANALYZE;

-- Check for bloat
SELECT schemaname, tablename,
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
LIMIT 20;
```

**Monthly:**
```sql
-- Reindex if needed
REINDEX DATABASE fluxbase;

-- Check for unused indexes
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0
ORDER BY pg_relation_size(indexrelid) DESC;
```

---

## Escalation Matrix

| Severity | Response Time | Escalation |
|----------|---------------|------------|
| P1 - Service Down | 15 min | On-call → Team Lead → Engineering Manager |
| P2 - Major Degradation | 1 hour | On-call → Team Lead |
| P3 - Minor Issue | 4 hours | On-call |
| P4 - Non-urgent | Next business day | Ticket |

---

## Learn More

- [Backup & Restore](/docs/guides/backup-restore) - Disaster recovery procedures
- [Monitoring & Observability](/docs/guides/monitoring-observability) - Setting up monitoring
- [Production Checklist](/docs/deployment/production-checklist) - Pre-production checklist
- [Scaling](/docs/deployment/scaling) - Scaling strategies

