# Production Deployment Checklist

Complete this checklist before deploying Fluxbase to production to ensure security, reliability, and performance.

## Security

### Authentication & Authorization

- [ ] **JWT Secret**: Changed from default, at least 32 characters

  ```bash
  # Generate strong secret
  openssl rand -base64 32
  ```

- [ ] **Database Password**: Strong password, not default

  ```bash
  # Generate strong password
  openssl rand -base64 24
  ```

- [ ] **Admin Credentials**: Changed default admin email/password

  ```bash
  FLUXBASE_ADMIN_EMAIL=admin@yourdomain.com
  FLUXBASE_ADMIN_PASSWORD=secure-password
  ```

- [ ] **API Keys**: Rotated and stored in secrets manager
- [ ] **Row Level Security (RLS)**: Enabled and policies configured
  ```sql
  ALTER TABLE users ENABLE ROW LEVEL SECURITY;
  CREATE POLICY user_isolation ON users
    USING (current_setting('app.user_id', true)::uuid = id);
  ```

### Network Security

- [ ] **HTTPS/TLS**: Enabled for all connections

  - [ ] Valid SSL certificate (not self-signed)
  - [ ] TLS 1.2+ only
  - [ ] HTTP → HTTPS redirect enabled

- [ ] **Database SSL**: Enabled

  ```bash
  FLUXBASE_DATABASE_SSL_MODE=require  # or verify-full
  ```

- [ ] **CORS**: Properly configured (not `*` in production)

  ```bash
  FLUXBASE_CORS_ALLOWED_ORIGINS=https://app.yourdomain.com,https://www.yourdomain.com
  ```

- [ ] **Firewall Rules**: Only expose 80/443

  - [ ] Database port (5432) not publicly accessible
  - [ ] Redis port (6379) not publicly accessible
  - [ ] Metrics port (9090) internal only

- [ ] **Rate Limiting**: Enabled
  ```bash
  FLUXBASE_RATE_LIMIT_ENABLED=true
  FLUXBASE_RATE_LIMIT_REQUESTS_PER_SECOND=100
  ```

### Secrets Management

- [ ] **Environment Variables**: Not hardcoded in code
- [ ] **Secrets**: Stored in secure vault

  - [ ] AWS Secrets Manager, Azure Key Vault, or HashiCorp Vault
  - [ ] Not committed to version control
  - [ ] `.env` files in `.gitignore`

- [ ] **IAM Roles**: Used instead of access keys (AWS/GCP/Azure)
- [ ] **Secret Rotation**: Automated or scheduled

### Container Security

- [ ] **Non-root User**: Container runs as non-root

  ```dockerfile
  USER fluxbase  # Already configured in official image
  ```

- [ ] **Read-only Filesystem**: Where possible
- [ ] **Security Scanning**: Images scanned for vulnerabilities

  ```bash
  docker scan ghcr.io/wayli-app/fluxbase:latest
  ```

- [ ] **Minimal Base Image**: Alpine or distroless
- [ ] **No Secrets in Image**: Secrets passed at runtime only

## Configuration

### Application Settings

- [ ] **Environment**: Set to production

  ```bash
  FLUXBASE_ENVIRONMENT=production
  ```

- [ ] **Debug Mode**: Disabled

  ```bash
  FLUXBASE_DEBUG=false
  ```

- [ ] **Log Level**: Set to `info` or `warn`

  ```bash
  FLUXBASE_LOG_LEVEL=info
  ```

- [ ] **Base URL**: Correct production URL
  ```bash
  FLUXBASE_BASE_URL=https://api.yourdomain.com
  ```

### Database Configuration

- [ ] **Connection Pool**: Properly sized

  ```bash
  FLUXBASE_DATABASE_MAX_CONNECTIONS=25
  FLUXBASE_DATABASE_MIN_CONNECTIONS=5
  ```

- [ ] **Connection Timeouts**: Configured

  ```bash
  FLUXBASE_DATABASE_MAX_CONN_LIFETIME=1h
  FLUXBASE_DATABASE_MAX_CONN_IDLE_TIME=30m
  ```

- [ ] **Migrations**: Auto-run on startup or manual deployment
- [ ] **Indexes**: Created for frequently queried columns
- [ ] **Vacuum**: Scheduled (PostgreSQL maintenance)

### Storage Configuration

- [ ] **Storage Provider**: Configured (S3, not local in production)

  ```bash
  FLUXBASE_STORAGE_PROVIDER=s3
  ```

- [ ] **Upload Limits**: Set appropriately

  ```bash
  FLUXBASE_STORAGE_MAX_UPLOAD_SIZE=2147483648  # 2GB
  FLUXBASE_SERVER_BODY_LIMIT=2147483648
  ```

- [ ] **Bucket Permissions**: Properly scoped (not public)
- [ ] **Bucket Versioning**: Enabled for recovery
- [ ] **Lifecycle Policies**: Configured for cost optimization

### Email Configuration

- [ ] **Email Provider**: Configured and tested

  ```bash
  FLUXBASE_EMAIL_ENABLED=true
  FLUXBASE_EMAIL_PROVIDER=sendgrid  # or smtp, mailgun, ses
  ```

- [ ] **From Address**: Verified domain
- [ ] **SPF/DKIM/DMARC**: Configured for deliverability
- [ ] **Templates**: Customized with branding

## High Availability

### Redundancy

- [ ] **Multiple Replicas**: At least 3 instances

  ```yaml
  replicaCount: 3 # Kubernetes
  ```

- [ ] **Database Replication**: Primary + replicas

  - [ ] Synchronous or asynchronous replication
  - [ ] Automatic failover configured

- [ ] **Load Balancer**: Configured

  - [ ] Health checks enabled
  - [ ] Session affinity if needed

- [ ] **Multi-Zone Deployment**: Across availability zones
  ```yaml
  podAntiAffinityPreset: soft
  topologySpreadConstraints:
    - maxSkew: 1
      topologyKey: topology.kubernetes.io/zone
  ```

### Auto-scaling

- [ ] **Horizontal Pod Autoscaler (HPA)**: Configured

  ```yaml
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
    targetCPU: 70
  ```

- [ ] **Database Auto-scaling**: For managed databases
- [ ] **Storage Auto-scaling**: For managed storage

### Health Checks

- [ ] **Liveness Probe**: Configured

  ```yaml
  livenessProbe:
    httpGet:
      path: /health
      port: 8080
    initialDelaySeconds: 30
    periodSeconds: 10
  ```

- [ ] **Readiness Probe**: Configured

  ```yaml
  readinessProbe:
    httpGet:
      path: /health
      port: 8080
    initialDelaySeconds: 5
    periodSeconds: 10
  ```

- [ ] **Startup Probe**: Configured for slow starts

## Monitoring & Logging

### Metrics

- [ ] **Prometheus Metrics**: Enabled

  ```bash
  FLUXBASE_METRICS_ENABLED=true
  FLUXBASE_METRICS_PORT=9090
  ```

- [ ] **ServiceMonitor**: Created (Kubernetes)
- [ ] **Dashboards**: Imported into Grafana
  - [ ] Request rate
  - [ ] Response time (p50, p95, p99)
  - [ ] Error rate
  - [ ] Database connections
  - [ ] Memory/CPU usage

### Logging

- [ ] **Structured Logging**: JSON format
- [ ] **Log Aggregation**: Centralized (ELK, Loki, CloudWatch)
- [ ] **Log Retention**: Configured (30-90 days)
- [ ] **Log Levels**: Appropriate for production
- [ ] **Sensitive Data**: Not logged (passwords, tokens)

### Alerting

- [ ] **Alert Rules**: Configured

  - [ ] High error rate (&gt;1%)
  - [ ] Slow response time (p95 &gt; 1s)
  - [ ] High CPU/memory usage (&gt;80%)
  - [ ] Database connection pool exhaustion
  - [ ] Disk space low (&lt;20%)

- [ ] **Alert Channels**: Configured

  - [ ] Email
  - [ ] Slack/Discord
  - [ ] PagerDuty/OpsGenie

- [ ] **On-call Schedule**: Defined

### Tracing

- [ ] **Distributed Tracing**: Enabled (optional but recommended)

  ```bash
  FLUXBASE_TRACING_ENABLED=true
  FLUXBASE_TRACING_EXPORTER=jaeger
  ```

- [ ] **Sampling Rate**: Configured (10-100%)

## Backup & Disaster Recovery

### Database Backups

- [ ] **Automated Backups**: Daily or more frequent

  ```bash
  # Example cron job
  0 2 * * * /scripts/backup-db.sh
  ```

- [ ] **Backup Retention**: Defined policy

  - [ ] Daily: 7 days
  - [ ] Weekly: 4 weeks
  - [ ] Monthly: 12 months

- [ ] **Backup Testing**: Restore tested monthly
- [ ] **Point-in-Time Recovery (PITR)**: Enabled
- [ ] **Off-site Backups**: Stored in different region

### File Storage Backups

- [ ] **S3 Versioning**: Enabled
- [ ] **Cross-region Replication**: Enabled
- [ ] **Lifecycle Policies**: Configured

### Disaster Recovery Plan

- [ ] **RTO (Recovery Time Objective)**: Defined (e.g., < 1 hour)
- [ ] **RPO (Recovery Point Objective)**: Defined (e.g., < 5 minutes)
- [ ] **DR Region**: Configured (multi-region setup)
- [ ] **Runbook**: Documented disaster recovery procedures
- [ ] **DR Drills**: Practiced quarterly

## Performance

### Resource Allocation

- [ ] **CPU Limits**: Set appropriately

  ```yaml
  resources:
    requests:
      cpu: 500m
    limits:
      cpu: 2000m
  ```

- [ ] **Memory Limits**: Set appropriately

  ```yaml
  resources:
    requests:
      memory: 1Gi
    limits:
      memory: 4Gi
  ```

- [ ] **Database Resources**: Right-sized for workload

### Optimization

- [ ] **Database Indexes**: Created for all queries

  ```sql
  CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
  ```

- [ ] **Query Optimization**: Slow queries identified and fixed
- [ ] **Caching**: Redis configured

  ```bash
  FLUXBASE_REDIS_ENABLED=true
  ```

- [ ] **CDN**: Configured for static assets
- [ ] **Compression**: Enabled (gzip/brotli)
- [ ] **HTTP/2**: Enabled

## Compliance

### Data Protection

- [ ] **GDPR Compliance**: If applicable

  - [ ] Data retention policies
  - [ ] Right to deletion implemented
  - [ ] Data export functionality

- [ ] **Data Encryption**:

  - [ ] At rest (database, backups)
  - [ ] In transit (TLS)

- [ ] **PII Handling**: Documented and secure

### Audit

- [ ] **Audit Logging**: Enabled for sensitive operations
- [ ] **Access Logs**: Retained (6-12 months)
- [ ] **Change Tracking**: Version control for infrastructure

## Documentation

- [ ] **Architecture Diagram**: Up to date
- [ ] **Deployment Guide**: Complete
- [ ] **Runbooks**: For common operations

  - [ ] Deployment
  - [ ] Rollback
  - [ ] Scaling
  - [ ] Incident response

- [ ] **API Documentation**: Published
- [ ] **Security Policies**: Documented
- [ ] **Contact Information**: Emergency contacts listed

## Testing

### Load Testing

- [ ] **Load Test**: Performed at expected peak load

  ```bash
  # Example with k6
  k6 run --vus 100 --duration 30s load-test.js
  ```

- [ ] **Stress Test**: Performed beyond expected load
- [ ] **Results**: Documented and acceptable

### Security Testing

- [ ] **Penetration Test**: Performed
- [ ] **Vulnerability Scan**: Performed
- [ ] **Dependency Audit**: No critical vulnerabilities
  ```bash
  npm audit
  go list -json -m all | nancy sleuth
  ```

### Integration Testing

- [ ] **End-to-End Tests**: Passing
- [ ] **API Tests**: Passing
- [ ] **Database Migrations**: Tested

## Launch Preparation

### Final Checks

- [ ] **DNS**: Configured and propagated
- [ ] **SSL Certificate**: Valid and not expiring soon
- [ ] **Email Deliverability**: Tested
- [ ] **Monitoring**: All alerts firing correctly (test)
- [ ] **Backups**: Verified working
- [ ] **Documentation**: Reviewed and complete

### Go-Live Checklist

- [ ] **Maintenance Window**: Scheduled and communicated
- [ ] **Rollback Plan**: Prepared
- [ ] **Team Availability**: On-call team ready
- [ ] **Communication Plan**: Status page and announcements ready
- [ ] **Performance Baseline**: Established for comparison

### Post-Launch

- [ ] **Monitor Metrics**: First 24-48 hours closely
- [ ] **Check Logs**: For errors or warnings
- [ ] **Verify Backups**: First backup successful
- [ ] **Performance Review**: Compare to baseline
- [ ] **User Feedback**: Monitored and addressed
- [ ] **Post-Mortem**: Document any issues

## Cost Optimization

- [ ] **Resource Usage**: Monitored and right-sized
- [ ] **Auto-scaling**: Configured to scale down during off-peak
- [ ] **Reserved Instances**: For predictable workloads
- [ ] **Spot Instances**: For non-critical workloads
- [ ] **Storage Lifecycle**: Old backups/logs archived or deleted
- [ ] **Database Connection Pool**: Optimized to reduce DB instance size

## Checklist Summary

**Critical (Must Have)**:

- Security (JWT secret, HTTPS, database password)
- Backups (automated, tested)
- Monitoring (metrics, logs, alerts)
- High Availability (multiple replicas, health checks)

**Important (Should Have)**:

- Auto-scaling
- Disaster recovery plan
- Load testing
- Documentation

**Nice to Have**:

- Distributed tracing
- Cost optimization
- Advanced security (vault, IAM roles)

---

## Next Steps

After completing this checklist:

1. Review with your team
2. Schedule deployment window
3. Execute deployment following [Deployment Guide](overview)
4. Monitor closely for first 24-48 hours
5. Document any issues for future improvements

For ongoing operations, see:

- [Scaling Guide](scaling) - Performance optimization
- [Monitoring Guide](../guides/monitoring-observability) - Observability setup
