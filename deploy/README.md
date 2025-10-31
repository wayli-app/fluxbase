# Fluxbase Deployment Guide

This directory contains everything you need to deploy Fluxbase in various environments.

## Quick Start

### Local Development (Docker Compose)

```bash
# Start all services (PostgreSQL + MinIO + Fluxbase)
cd deploy
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f fluxbase

# Access Fluxbase
open http://localhost:8080
open http://localhost:8080/admin  # Admin UI
```

**Default Credentials:**
- PostgreSQL: `fluxbase/fluxbase`
- MinIO: `minioadmin/minioadmin`
- JWT Secret: `your-super-secret-jwt-key-change-me-in-production`

### Production (Kubernetes with Helm)

```bash
# Add Helm chart repository
cd deploy/helm

# Install with default values (includes PostgreSQL)
helm install my-fluxbase ./fluxbase

# Install with external database
helm install my-fluxbase ./fluxbase \
  --set postgresql.enabled=false \
  --set externalDatabase.host=postgres.example.com \
  --set externalDatabase.password=secure-password

# Install with custom values file
helm install my-fluxbase ./fluxbase -f production-values.yaml
```

## Deployment Options

### 1. Docker Compose (Recommended for Development)

**Use Case:** Local development, testing, demos

**Features:**
- One-command setup
- Includes PostgreSQL and MinIO
- Hot-reload support
- Easy debugging

**Setup:**
```bash
cd deploy
docker-compose up -d
```

**Configuration:**
Edit `docker-compose.yml` to customize:
- Database credentials
- Storage provider (local or S3/MinIO)
- JWT secret
- Port mappings

**Stopping:**
```bash
docker-compose down          # Stop services
docker-compose down -v       # Stop and remove volumes
```

### 2. Kubernetes with Helm (Recommended for Production)

**Use Case:** Production deployments, staging, multi-tenant

**Features:**
- High availability (3+ replicas)
- Auto-scaling (HPA)
- Rolling updates
- Health checks
- Prometheus metrics
- TLS/Ingress support

**Prerequisites:**
- Kubernetes 1.19+
- Helm 3.2.0+
- kubectl configured

**Basic Installation:**
```bash
cd deploy/helm

# Install with default values
helm install fluxbase ./fluxbase

# Check status
kubectl get pods -l app.kubernetes.io/name=fluxbase
kubectl get svc -l app.kubernetes.io/name=fluxbase

# Port-forward for testing
kubectl port-forward svc/fluxbase 8080:8080
```

**Production Installation:**
```bash
# Create namespace
kubectl create namespace production

# Create secrets
kubectl create secret generic fluxbase-secrets \
  --from-literal=database-password='<db-password>' \
  --from-literal=jwt-secret='<jwt-secret>' \
  -n production

# Install with production values
helm install fluxbase ./fluxbase \
  --namespace production \
  --set postgresql.enabled=false \
  --set externalDatabase.host=postgres.production.svc \
  --set existingSecret=fluxbase-secrets \
  --set replicaCount=5 \
  --set ingress.enabled=true \
  --set ingress.hostname=api.example.com \
  --set ingress.tls=true \
  --set autoscaling.enabled=true \
  --set metrics.serviceMonitor.enabled=true
```

**Upgrading:**
```bash
# Update chart
helm upgrade fluxbase ./fluxbase \
  --namespace production \
  -f production-values.yaml

# Rollback if needed
helm rollback fluxbase --namespace production
```

**Uninstalling:**
```bash
helm uninstall fluxbase --namespace production
```

### 3. Binary Deployment

**Use Case:** Single-server deployments, VPS, bare metal

**Setup:**
```bash
# Build binary
make build

# Run with environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=fluxbase
export DB_USER=fluxbase
export DB_PASSWORD=secure-password
export JWT_SECRET=your-secret-key

./fluxbase
```

**Systemd Service:**
```bash
# Create systemd service file
sudo nano /etc/systemd/system/fluxbase.service
```

```ini
[Unit]
Description=Fluxbase Backend-as-a-Service
After=network.target postgresql.service

[Service]
Type=simple
User=fluxbase
WorkingDirectory=/opt/fluxbase
ExecStart=/opt/fluxbase/fluxbase
Restart=always
RestartSec=10

# Environment variables
Environment="DB_HOST=localhost"
Environment="DB_PORT=5432"
Environment="DB_NAME=fluxbase"
Environment="DB_USER=fluxbase"
Environment="DB_PASSWORD=secure-password"
Environment="JWT_SECRET=your-secret-key"
Environment="SERVER_PORT=8080"

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start
sudo systemctl enable fluxbase
sudo systemctl start fluxbase
sudo systemctl status fluxbase
```

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DB_HOST` | PostgreSQL host | `localhost` | Yes |
| `DB_PORT` | PostgreSQL port | `5432` | Yes |
| `DB_NAME` | Database name | `fluxbase` | Yes |
| `DB_USER` | Database user | `fluxbase` | Yes |
| `DB_PASSWORD` | Database password | - | Yes |
| `DB_SSL_MODE` | SSL mode | `disable` | No |
| `SERVER_PORT` | HTTP server port | `8080` | No |
| `SERVER_HOST` | HTTP server host | `0.0.0.0` | No |
| `JWT_SECRET` | JWT signing key | - | Yes |
| `JWT_EXPIRATION_MINUTES` | Token expiration | `60` | No |
| `STORAGE_PROVIDER` | Storage type (`local` or `s3`) | `local` | No |
| `STORAGE_LOCAL_BASE_PATH` | Local storage path | `/data/storage` | No |
| `STORAGE_S3_BUCKET` | S3 bucket name | - | If using S3 |
| `STORAGE_S3_REGION` | S3 region | - | If using S3 |
| `STORAGE_S3_ENDPOINT` | S3 endpoint (for MinIO) | - | If using S3 |
| `STORAGE_S3_ACCESS_KEY_ID` | S3 access key | - | If using S3 |
| `STORAGE_S3_SECRET_ACCESS_KEY` | S3 secret key | - | If using S3 |
| `LOG_LEVEL` | Log level | `info` | No |
| `LOG_FORMAT` | Log format (`json` or `text`) | `json` | No |
| `METRICS_ENABLED` | Enable Prometheus metrics | `true` | No |

### Helm Chart Values

See [helm/fluxbase/README.md](helm/fluxbase/README.md) for complete Helm configuration options.

**Key Values:**
- `replicaCount` - Number of replicas (default: 3)
- `resourcesPreset` - Resource allocation (nano/micro/small/medium/large/xlarge/2xlarge)
- `postgresql.enabled` - Deploy PostgreSQL (default: true)
- `ingress.enabled` - Enable ingress (default: false)
- `autoscaling.enabled` - Enable HPA (default: false)
- `metrics.enabled` - Enable Prometheus metrics (default: true)

## Production Checklist

### Security

- [ ] Change default JWT secret
- [ ] Use strong database passwords
- [ ] Enable TLS/HTTPS (via ingress or load balancer)
- [ ] Configure CORS for your domain
- [ ] Enable rate limiting
- [ ] Use Kubernetes secrets (not plaintext values)
- [ ] Enable network policies
- [ ] Configure pod security policies
- [ ] Regular security updates

### High Availability

- [ ] Run 3+ replicas
- [ ] Enable pod anti-affinity
- [ ] Configure health checks (liveness/readiness)
- [ ] Use rolling updates
- [ ] Set resource limits
- [ ] Enable autoscaling (HPA)
- [ ] Configure PodDisruptionBudget

### Observability

- [ ] Enable Prometheus metrics
- [ ] Configure ServiceMonitor (if using Prometheus Operator)
- [ ] Set up log aggregation (ELK, Loki, etc.)
- [ ] Configure alerts
- [ ] Monitor database connections
- [ ] Track API latency
- [ ] Set up dashboards (Grafana)

### Database

- [ ] Use external managed PostgreSQL (AWS RDS, Google Cloud SQL, etc.)
- [ ] Enable automated backups
- [ ] Configure connection pooling
- [ ] Monitor slow queries
- [ ] Set up replication (read replicas)
- [ ] Plan for disaster recovery

### Storage

- [ ] Use S3-compatible storage (AWS S3, MinIO, etc.)
- [ ] Enable versioning
- [ ] Configure lifecycle policies
- [ ] Set up CDN (CloudFront, CloudFlare)
- [ ] Monitor storage costs

### Performance

- [ ] Configure appropriate resource limits
- [ ] Enable caching headers
- [ ] Use CDN for static assets
- [ ] Optimize database indexes
- [ ] Monitor and tune connection pools
- [ ] Profile slow endpoints

## Troubleshooting

### Common Issues

**1. Database Connection Failed**
```bash
# Check PostgreSQL is running
kubectl get pods -l app.kubernetes.io/name=postgresql

# Check connection string
kubectl exec -it <fluxbase-pod> -- env | grep DB_

# Test connection manually
kubectl exec -it <fluxbase-pod> -- psql -h $DB_HOST -U $DB_USER -d $DB_NAME
```

**2. Pods Not Starting**
```bash
# Check pod status
kubectl describe pod <pod-name>

# Check logs
kubectl logs <pod-name>

# Check resource limits
kubectl top pods
```

**3. Ingress Not Working**
```bash
# Check ingress status
kubectl get ingress

# Check ingress controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx

# Verify DNS
nslookup api.example.com
```

**4. High Memory Usage**
- Reduce `DB_MAX_CONNECTIONS`
- Lower `replicaCount` or adjust `resources.limits.memory`
- Check for memory leaks in logs

**5. High CPU Usage**
- Enable autoscaling with HPA
- Check slow queries in database
- Review rate limiting configuration
- Profile application with `/debug/pprof` (if enabled)

## Monitoring

### Health Checks

```bash
# Health endpoint
curl http://localhost:8080/health

# Metrics endpoint (Prometheus format)
curl http://localhost:8080/metrics

# Realtime stats
curl http://localhost:8080/api/v1/realtime/stats
```

### Key Metrics

- `fluxbase_http_requests_total` - Total HTTP requests
- `fluxbase_http_request_duration_seconds` - Request latency
- `fluxbase_db_queries_total` - Database query count
- `fluxbase_db_query_duration_seconds` - Query latency
- `fluxbase_realtime_connections` - Active WebSocket connections
- `fluxbase_storage_operations_total` - Storage operations
- `fluxbase_auth_attempts_total` - Authentication attempts

### Grafana Dashboard

Import the provided Grafana dashboard:
```bash
kubectl apply -f deploy/grafana-dashboard.json
```

## Backup and Recovery

### Database Backup

```bash
# Manual backup
kubectl exec <postgres-pod> -- pg_dump -U fluxbase fluxbase > backup.sql

# Restore
kubectl exec -i <postgres-pod> -- psql -U fluxbase fluxbase < backup.sql
```

### Automated Backups

Use managed database services or configure automated backups:
- AWS RDS: Automated snapshots
- Google Cloud SQL: Automated backups
- Self-hosted: Use `pg_dump` with cron

### Disaster Recovery

1. **Database:** Restore from latest backup
2. **Storage:** S3 versioning enabled (rollback if needed)
3. **Configuration:** Helm values stored in Git
4. **Secrets:** Backed up securely (Vault, AWS Secrets Manager)

## Support

- Documentation: https://docs.fluxbase.io
- GitHub Issues: https://github.com/yourusername/fluxbase/issues
- Production Runbook: See [PRODUCTION_RUNBOOK.md](../PRODUCTION_RUNBOOK.md)

## License

MIT
