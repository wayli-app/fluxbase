# Docker Deployment

Deploy Fluxbase using Docker and Docker Compose for simple production environments and development.

## Overview

Fluxbase provides official Docker images (~80MB) with:
- Multi-stage build for minimal image size
- Non-root user for security
- Health checks built-in
- Admin UI embedded
- Automatic database migrations

**Image Registry**: `ghcr.io/wayli-app/fluxbase`

## Quick Start

### 1. Pull the Docker Image

```bash
docker pull ghcr.io/wayli-app/fluxbase:latest
```

### 2. Run with Docker Compose

Create `docker-compose.yml`:

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:16-alpine
    container_name: fluxbase-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: your-secure-password
      POSTGRES_DB: fluxbase
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - fluxbase-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  fluxbase:
    image: ghcr.io/wayli-app/fluxbase:latest
    container_name: fluxbase
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      FLUXBASE_DATABASE_HOST: postgres
      FLUXBASE_DATABASE_PORT: 5432
      FLUXBASE_DATABASE_USER: postgres
      FLUXBASE_DATABASE_PASSWORD: your-secure-password
      FLUXBASE_DATABASE_DATABASE: fluxbase
      FLUXBASE_AUTH_JWT_SECRET: your-jwt-secret-change-in-production
      FLUXBASE_BASE_URL: http://localhost:8080
      FLUXBASE_ENVIRONMENT: production
      FLUXBASE_DEBUG: "false"
    ports:
      - "8080:8080"
    volumes:
      - ./storage:/app/storage
      - ./config:/app/config
    networks:
      - fluxbase-network
    restart: unless-stopped

volumes:
  postgres_data:

networks:
  fluxbase-network:
    driver: bridge
```

### 3. Start the Services

```bash
docker-compose up -d
```

### 4. Verify Deployment

```bash
# Check container status
docker-compose ps

# View logs
docker-compose logs -f fluxbase

# Check health endpoint
curl http://localhost:8080/health
```

You should see:
```json
{
  "status": "healthy",
  "database": "connected",
  "version": "0.1.0"
}
```

---

## Production Docker Compose

For production, use this enhanced configuration:

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:16-alpine
    container_name: fluxbase-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: fluxbase
      POSTGRES_INITDB_ARGS: "-E UTF8 --locale=en_US.UTF-8"
    ports:
      - "127.0.0.1:5432:5432"  # Only bind to localhost
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backups:/backups
    networks:
      - fluxbase-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped
    # Resource limits
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G

  redis:
    image: redis:7-alpine
    container_name: fluxbase-redis
    command: redis-server --requirepass ${REDIS_PASSWORD}
    ports:
      - "127.0.0.1:6379:6379"
    volumes:
      - redis_data:/data
    networks:
      - fluxbase-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 5
    restart: unless-stopped

  fluxbase:
    image: ghcr.io/wayli-app/fluxbase:latest
    container_name: fluxbase
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      # Database
      FLUXBASE_DATABASE_HOST: postgres
      FLUXBASE_DATABASE_PORT: 5432
      FLUXBASE_DATABASE_USER: postgres
      FLUXBASE_DATABASE_PASSWORD: ${POSTGRES_PASSWORD}
      FLUXBASE_DATABASE_DATABASE: fluxbase
      FLUXBASE_DATABASE_SSL_MODE: disable
      FLUXBASE_DATABASE_MAX_CONNECTIONS: 25
      FLUXBASE_DATABASE_MIN_CONNECTIONS: 5

      # Authentication
      FLUXBASE_AUTH_JWT_SECRET: ${JWT_SECRET}
      FLUXBASE_AUTH_JWT_EXPIRY: 15m
      FLUXBASE_AUTH_REFRESH_EXPIRY: 168h

      # Redis
      FLUXBASE_REDIS_ENABLED: "true"
      FLUXBASE_REDIS_HOST: redis
      FLUXBASE_REDIS_PORT: 6379
      FLUXBASE_REDIS_PASSWORD: ${REDIS_PASSWORD}

      # Server
      FLUXBASE_BASE_URL: https://api.yourdomain.com
      FLUXBASE_ENVIRONMENT: production
      FLUXBASE_DEBUG: "false"

      # Email (SendGrid example)
      FLUXBASE_EMAIL_ENABLED: "true"
      FLUXBASE_EMAIL_PROVIDER: sendgrid
      FLUXBASE_EMAIL_FROM_ADDRESS: noreply@yourdomain.com
      FLUXBASE_EMAIL_FROM_NAME: "Your App"
      FLUXBASE_EMAIL_SENDGRID_API_KEY: ${SENDGRID_API_KEY}

      # Storage (S3)
      FLUXBASE_STORAGE_PROVIDER: s3
      FLUXBASE_STORAGE_S3_BUCKET: your-bucket
      FLUXBASE_STORAGE_S3_REGION: us-east-1
      FLUXBASE_STORAGE_S3_ACCESS_KEY: ${AWS_ACCESS_KEY}
      FLUXBASE_STORAGE_S3_SECRET_KEY: ${AWS_SECRET_KEY}

      # Rate Limiting
      FLUXBASE_RATE_LIMIT_ENABLED: "true"
      FLUXBASE_RATE_LIMIT_REQUESTS_PER_SECOND: 100

      # Metrics
      FLUXBASE_METRICS_ENABLED: "true"
      FLUXBASE_METRICS_PORT: 9090
    ports:
      - "8080:8080"
      - "127.0.0.1:9090:9090"  # Metrics (internal only)
    volumes:
      - ./storage:/app/storage
      - ./config:/app/config
      - ./logs:/app/logs
    networks:
      - fluxbase-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8G
        reservations:
          cpus: '2'
          memory: 4G

  # Nginx reverse proxy with SSL
  nginx:
    image: nginx:alpine
    container_name: fluxbase-nginx
    depends_on:
      - fluxbase
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
      - ./nginx-logs:/var/log/nginx
    networks:
      - fluxbase-network
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:

networks:
  fluxbase-network:
    driver: bridge
```

Create `.env` file:

```bash
# PostgreSQL
POSTGRES_PASSWORD=your-very-secure-postgres-password

# Redis
REDIS_PASSWORD=your-very-secure-redis-password

# Fluxbase Auth
JWT_SECRET=your-jwt-secret-at-least-32-characters-long

# Email (SendGrid)
SENDGRID_API_KEY=SG.xxxxxxxxxxxxx

# AWS S3 Storage
AWS_ACCESS_KEY=AKIAXXXXXXXXXXXXXXXX
AWS_SECRET_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

**Important**: Add `.env` to `.gitignore`!

---

## Building Custom Images

### Build from Source

```bash
# Clone the repository
git clone https://github.com/your-org/fluxbase.git
cd fluxbase

# Build the image
docker build -t fluxbase:custom .

# With build args for versioning
docker build \
  --build-arg VERSION=1.0.0 \
  --build-arg COMMIT=$(git rev-parse HEAD) \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  -t fluxbase:1.0.0 \
  .
```

### Multi-Architecture Build

```bash
# Enable buildx
docker buildx create --use

# Build for multiple platforms
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/your-org/fluxbase:latest \
  --push \
  .
```

---

## Nginx Configuration

Create `nginx.conf`:

```nginx
events {
    worker_connections 1024;
}

http {
    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;

    # Upstream Fluxbase
    upstream fluxbase {
        server fluxbase:8080;
        keepalive 32;
    }

    # Redirect HTTP to HTTPS
    server {
        listen 80;
        server_name api.yourdomain.com;
        return 301 https://$host$request_uri;
    }

    # HTTPS Server
    server {
        listen 443 ssl http2;
        server_name api.yourdomain.com;

        # SSL Configuration
        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers HIGH:!aNULL:!MD5;
        ssl_prefer_server_ciphers on;

        # Security Headers
        add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
        add_header X-Frame-Options "SAMEORIGIN" always;
        add_header X-Content-Type-Options "nosniff" always;
        add_header X-XSS-Protection "1; mode=block" always;

        # Logging
        access_log /var/log/nginx/access.log;
        error_log /var/log/nginx/error.log;

        # Client body size (for file uploads)
        client_max_body_size 2G;

        # Proxy settings
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # API endpoints
        location / {
            limit_req zone=api_limit burst=20 nodelay;
            proxy_pass http://fluxbase;
        }

        # WebSocket endpoint (realtime)
        location /realtime {
            proxy_pass http://fluxbase;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_read_timeout 86400;
        }

        # Health check (no rate limit)
        location /health {
            proxy_pass http://fluxbase;
            access_log off;
        }
    }
}
```

---

## Database Backups

### Automated Backup Script

Create `backup.sh`:

```bash
#!/bin/bash

BACKUP_DIR="/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
FILENAME="fluxbase_backup_${TIMESTAMP}.sql.gz"

# Create backup
docker exec fluxbase-postgres pg_dump \
  -U postgres \
  -d fluxbase \
  -F c \
  | gzip > "${BACKUP_DIR}/${FILENAME}"

# Keep only last 7 days of backups
find ${BACKUP_DIR} -name "fluxbase_backup_*.sql.gz" -mtime +7 -delete

echo "Backup created: ${FILENAME}"
```

Make it executable:

```bash
chmod +x backup.sh
```

### Schedule with Cron

```bash
# Edit crontab
crontab -e

# Add daily backup at 2 AM
0 2 * * * /path/to/backup.sh >> /var/log/fluxbase-backup.log 2>&1
```

### Restore from Backup

```bash
# Stop the application
docker-compose stop fluxbase

# Restore database
gunzip -c /backups/fluxbase_backup_20240126_020000.sql.gz | \
  docker exec -i fluxbase-postgres pg_restore \
    -U postgres \
    -d fluxbase \
    --clean

# Start the application
docker-compose start fluxbase
```

---

## Monitoring with Prometheus

Add Prometheus to `docker-compose.yml`:

```yaml
  prometheus:
    image: prom/prometheus:latest
    container_name: fluxbase-prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    ports:
      - "127.0.0.1:9091:9090"
    networks:
      - fluxbase-network
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    container_name: fluxbase-grafana
    environment:
      GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_PASSWORD}
    volumes:
      - grafana_data:/var/lib/grafana
    ports:
      - "3000:3000"
    networks:
      - fluxbase-network
    restart: unless-stopped
```

Create `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'fluxbase'
    static_configs:
      - targets: ['fluxbase:9090']

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']
```

---

## Logging

### Centralized Logging with Loki

```yaml
  loki:
    image: grafana/loki:latest
    container_name: fluxbase-loki
    ports:
      - "127.0.0.1:3100:3100"
    volumes:
      - ./loki-config.yml:/etc/loki/local-config.yaml
      - loki_data:/loki
    networks:
      - fluxbase-network
    restart: unless-stopped

  promtail:
    image: grafana/promtail:latest
    container_name: fluxbase-promtail
    volumes:
      - /var/log:/var/log
      - ./promtail-config.yml:/etc/promtail/config.yml
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
    networks:
      - fluxbase-network
    restart: unless-stopped
```

---

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker-compose logs fluxbase

# Common issues:
# 1. Database not ready - wait for healthcheck
# 2. Missing environment variables
# 3. Port already in use

# Verify environment variables
docker-compose config

# Check port conflicts
sudo netstat -tlnp | grep 8080
```

### Database Connection Failed

```bash
# Test database connectivity
docker exec -it fluxbase-postgres psql -U postgres -d fluxbase

# Check connection from Fluxbase container
docker exec -it fluxbase curl postgres:5432

# Verify network
docker network inspect fluxbase-network
```

### Out of Memory

```bash
# Check container stats
docker stats

# Increase memory limits in docker-compose.yml
deploy:
  resources:
    limits:
      memory: 8G
```

### Slow Performance

1. **Check database indexes**:
   ```sql
   SELECT schemaname, tablename, indexname
   FROM pg_indexes
   WHERE schemaname = 'public';
   ```

2. **Monitor connection pool**:
   ```sql
   SELECT count(*) FROM pg_stat_activity;
   ```

3. **Enable query logging**:
   ```bash
   FLUXBASE_DEBUG=true
   ```

---

## Upgrading

### Rolling Update

```bash
# Pull latest image
docker-compose pull fluxbase

# Recreate container with new image
docker-compose up -d --no-deps fluxbase

# Check health
curl http://localhost:8080/health
```

### With Downtime

```bash
# Stop all services
docker-compose down

# Pull latest images
docker-compose pull

# Start services
docker-compose up -d

# Verify
docker-compose ps
```

### Rollback

```bash
# Use specific version
docker-compose stop fluxbase
docker run -d --name fluxbase ghcr.io/wayli-app/fluxbase:0.9.0
```

---

## Security Best Practices

1. **Use secrets management**:
   ```bash
   # Docker secrets (Swarm mode)
   echo "my-secret" | docker secret create jwt_secret -
   ```

2. **Run as non-root**:
   ```dockerfile
   USER fluxbase  # Already configured in official image
   ```

3. **Limit container capabilities**:
   ```yaml
   cap_drop:
     - ALL
   cap_add:
     - NET_BIND_SERVICE
   ```

4. **Use read-only filesystem**:
   ```yaml
   read_only: true
   tmpfs:
     - /tmp
     - /app/logs
   ```

5. **Scan images for vulnerabilities**:
   ```bash
   docker scan ghcr.io/wayli-app/fluxbase:latest
   ```

---

## Next Steps

- [Kubernetes Deployment](kubernetes) - Scale with Kubernetes
- [Production Checklist](production-checklist) - Pre-deployment checklist
- [Scaling Guide](scaling) - Optimize performance
