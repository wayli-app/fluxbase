---
title: "Kubernetes Deployment"
---

Deploy Fluxbase on Kubernetes using Helm for production-grade, highly available infrastructure.

## Overview

Fluxbase provides an official Helm chart that includes:

- High availability with multiple replicas
- Horizontal Pod Autoscaling (HPA)
- PostgreSQL database (or external database support)
- Ingress with TLS/SSL support
- Prometheus metrics and ServiceMonitor
- Configurable resource limits
- Security contexts and RBAC

**Helm Chart**: `deploy/helm/fluxbase`

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+
- kubectl configured
- Persistent Volume provisioner
- (Optional) Ingress controller (nginx, Traefik)
- (Optional) cert-manager for TLS certificates

## Quick Start

### 1. Install Helm Chart

```bash
# Add Fluxbase Helm repository (if published)
helm repo add fluxbase https://charts.fluxbase.io
helm repo update

# Or use local chart
cd deploy/helm

# Install with default values
helm install my-fluxbase ./fluxbase

# Install in custom namespace
helm install my-fluxbase ./fluxbase \
  --namespace fluxbase \
  --create-namespace
```

### 2. Verify Installation

```bash
# Check pod status
kubectl get pods -n fluxbase

# Expected output:
# NAME                          READY   STATUS    RESTARTS   AGE
# my-fluxbase-xxxxx-xxxxx       1/1     Running   0          2m
# my-fluxbase-postgresql-0      1/1     Running   0          2m

# Check service
kubectl get svc -n fluxbase

# Port-forward to test locally
kubectl port-forward -n fluxbase svc/my-fluxbase 8080:8080

# Test health endpoint
curl http://localhost:8080/health
```

### 3. Get Connection Info

```bash
# Get connection details
helm status my-fluxbase -n fluxbase

# Get JWT secret (if auto-generated)
kubectl get secret my-fluxbase -n fluxbase -o jsonpath='{.data.jwt-secret}' | base64 -d
```

---
## Production Installation

### Create Values File

Create `production-values.yaml`:

```yaml
# Replica count (minimum 3 for HA)
replicaCount: 3

# Image configuration
image:
  registry: ghcr.io
  repository: wayli-app/fluxbase
  tag: "0.1.0"
  pullPolicy: IfNotPresent

# Fluxbase configuration
config:
  database:
    host: my-postgres.example.com
    port: 5432
    name: fluxbase
    user: fluxbase
    # Don't set password here - use existingSecret
    sslMode: require

  server:
    port: 8080
    host: "0.0.0.0"

  jwt:
    # Don't set secret here - use existingSecret
    expirationMinutes: 60

  storage:
    provider: s3
    s3:
      bucket: fluxbase-production
      region: us-east-1
      endpoint: "" # Leave empty for AWS S3

# Use external database (disable embedded PostgreSQL)
postgresql:
  enabled: false

externalDatabase:
  host: my-postgres.example.com
  port: 5432
  user: fluxbase
  password: "" # Use existingSecret
  database: fluxbase
  existingSecret: fluxbase-db-secret
  existingSecretPasswordKey: password

# Secrets (create separately)
existingSecret: fluxbase-secrets

# Service configuration
service:
  type: ClusterIP
  ports:
    http: 8080

# Ingress configuration
ingress:
  enabled: true
  ingressClassName: nginx
  hostname: api.yourdomain.com
  path: /
  pathType: Prefix
  tls: true
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "2048m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"

# Autoscaling
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPU: 70
  targetMemory: 80

# Resource limits
resources:
  requests:
    cpu: 500m
    memory: 1Gi
  limits:
    cpu: 2000m
    memory: 4Gi

# Pod security
podSecurityContext:
  enabled: true
  fsGroup: 1001
  fsGroupChangePolicy: Always

containerSecurityContext:
  enabled: true
  runAsUser: 1001
  runAsGroup: 1001
  runAsNonRoot: true
  privileged: false
  readOnlyRootFilesystem: false
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  seccompProfile:
    type: RuntimeDefault

# Pod anti-affinity for HA
podAntiAffinityPreset: soft

# Metrics
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    namespace: monitoring
    interval: 30s
    additionalLabels:
      prometheus: kube-prometheus

# Persistence (for local storage)
persistence:
  enabled: false # Using S3, so disabled

# Probes
livenessProbe:
  enabled: true
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  enabled: true
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

startupProbe:
  enabled: true
  initialDelaySeconds: 0
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 30
```

### Create Secrets

```bash
# Create database secret
kubectl create secret generic fluxbase-db-secret \
  --namespace fluxbase \
  --from-literal=password='your-postgres-password'

# Create Fluxbase secrets
kubectl create secret generic fluxbase-secrets \
  --namespace fluxbase \
  --from-literal=jwt-secret='your-jwt-secret-at-least-32-chars' \
  --from-literal=database-password='your-postgres-password' \
  --from-literal=s3-access-key-id='AKIAXXXXXXXXXXXXXXXX' \
  --from-literal=s3-secret-access-key='your-aws-secret-key'
```

### Install with Custom Values

```bash
helm install fluxbase ./fluxbase \
  --namespace fluxbase \
  --create-namespace \
  --values production-values.yaml
```
---

## Configuration Options

### Full values.yaml Reference

See [values.yaml](https://github.com/your-org/fluxbase/blob/main/deploy/helm/fluxbase/values.yaml) for all available options.

**Key sections**:

| Section               | Description                   | Default                           |
| --------------------- | ----------------------------- | --------------------------------- |
| `replicaCount`        | Number of Fluxbase pods       | 3                                 |
| `image`               | Container image configuration | ghcr.io/wayli-app/fluxbase:latest |
| `config.database`     | Database connection settings  | PostgreSQL defaults               |
| `config.server`       | HTTP server settings          | Port 8080                         |
| `config.jwt`          | JWT authentication settings   | 60 min expiry                     |
| `config.storage`      | Storage backend (local/s3)    | local                             |
| `postgresql.enabled`  | Deploy PostgreSQL in cluster  | true                              |
| `externalDatabase`    | External database settings    | -                                 |
| `ingress.enabled`     | Enable Ingress                | false                             |
| `autoscaling.enabled` | Enable HPA                    | false                             |
| `metrics.enabled`     | Expose Prometheus metrics     | true                              |
| `persistence.enabled` | Enable persistent storage     | true                              |

---

## Database Configuration

### Option 1: Embedded PostgreSQL (Development/Testing)

```yaml
postgresql:
  enabled: true
  auth:
    username: fluxbase
    password: fluxbase-dev
    database: fluxbase
  primary:
    persistence:
      enabled: true
      size: 8Gi
```

### Option 2: External Managed Database (Production)

```yaml
postgresql:
  enabled: false

externalDatabase:
  host: my-postgres.rds.amazonaws.com
  port: 5432
  user: fluxbase
  database: fluxbase
  existingSecret: fluxbase-db-secret
  existingSecretPasswordKey: password
```

**Create the secret**:

```bash
kubectl create secret generic fluxbase-db-secret \
  --namespace fluxbase \
  --from-literal=password='production-db-password'
```

### Option 3: PostgreSQL Operator (CloudNativePG)

```yaml
# Install CloudNativePG operator first
helm repo add cnpg https://cloudnative-pg.github.io/charts
helm install cnpg cnpg/cloudnative-pg --namespace cnpg-system --create-namespace

# Create PostgreSQL cluster
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: fluxbase-postgres
  namespace: fluxbase
spec:
  instances: 3
  primaryUpdateStrategy: unsupervised

  postgresql:
    parameters:
      shared_buffers: 256MB
      max_connections: "300"
      shared_preload_libraries: "pg_stat_statements"

  bootstrap:
    initdb:
      database: fluxbase
      owner: fluxbase
      secret:
        name: fluxbase-db-secret

  storage:
    size: 100Gi
    storageClass: fast-ssd

  backup:
    barmanObjectStore:
      destinationPath: s3://backups/postgresql
      s3Credentials:
        accessKeyId:
          name: aws-creds
          key: ACCESS_KEY_ID
        secretAccessKey:
          name: aws-creds
          key: SECRET_ACCESS_KEY
    retentionPolicy: "30d"
```

---

## Ingress Configuration

### Nginx Ingress with cert-manager

```yaml
ingress:
  enabled: true
  ingressClassName: nginx
  hostname: api.example.com
  path: /
  tls: true
  annotations:
    # SSL/TLS
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"

    # CORS (if needed)
    nginx.ingress.kubernetes.io/enable-cors: "true"
    nginx.ingress.kubernetes.io/cors-allow-origin: "https://app.example.com"

    # Upload limits
    nginx.ingress.kubernetes.io/proxy-body-size: "2048m"

    # Timeouts
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"

    # Rate limiting
    nginx.ingress.kubernetes.io/limit-rps: "100"

    # WebSocket support
    nginx.ingress.kubernetes.io/proxy-http-version: "1.1"
    nginx.ingress.kubernetes.io/websocket-services: "my-fluxbase"
```

### Install cert-manager

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create ClusterIssuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

### Traefik Ingress

```yaml
ingress:
  enabled: true
  ingressClassName: traefik
  hostname: api.example.com
  annotations:
    traefik.ingress.kubernetes.io/router.tls: "true"
    traefik.ingress.kubernetes.io/router.tls.certresolver: letsencrypt
    traefik.ingress.kubernetes.io/router.middlewares: default-compress@kubernetescrd
```

---

## Autoscaling

### Horizontal Pod Autoscaler (HPA)

```yaml
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 20
  targetCPU: 70
  targetMemory: 80
```

**Verify HPA**:

```bash
kubectl get hpa -n fluxbase
# NAME          REFERENCE                TARGETS         MINPODS   MAXPODS   REPLICAS
# my-fluxbase   Deployment/my-fluxbase   45%/70%,50%/80% 3         20        3
```

### Custom Metrics (Advanced)

```yaml
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
    - type: Pods
      pods:
        metric:
          name: http_requests_per_second
        target:
          type: AverageValue
          averageValue: "1000"
```

---

## Monitoring and Observability

### Prometheus Metrics

```yaml
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    namespace: monitoring
    interval: 30s
    scrapeTimeout: 10s
    additionalLabels:
      prometheus: kube-prometheus
```

### Install Prometheus Stack

```bash
# Add Prometheus Helm repo
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install kube-prometheus-stack
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false
```

### Grafana Dashboards

Import pre-built Fluxbase dashboard (ID: TBD) or create custom:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluxbase-dashboard
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  fluxbase.json: |
    {
      "dashboard": {
        "title": "Fluxbase Metrics",
        "panels": [
          {
            "title": "Request Rate",
            "targets": [
              {
                "expr": "rate(http_requests_total{job=\"fluxbase\"}[5m])"
              }
            ]
          }
        ]
      }
    }
```

### Logging with Loki

```bash
# Install Loki stack
helm install loki grafana/loki-stack \
  --namespace monitoring \
  --set grafana.enabled=true \
  --set promtail.enabled=true
```

Add log annotation to pod:

```yaml
podAnnotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
```

---

## Storage

### Local Storage (Development)

```yaml
persistence:
  enabled: true
  storageClass: "standard"
  accessModes:
    - ReadWriteOnce
  size: 8Gi
```

### S3-Compatible Storage (Production)

```yaml
config:
  storage:
    provider: s3
    s3:
      bucket: fluxbase-production
      region: us-east-1
      endpoint: "" # Empty for AWS S3

# Add S3 credentials to existingSecret
existingSecret: fluxbase-secrets
# Secret should contain: s3-access-key-id, s3-secret-access-key
```

### MinIO (Self-Hosted S3)

```bash
# Install MinIO
helm install minio bitnami/minio \
  --namespace fluxbase \
  --set auth.rootUser=admin \
  --set auth.rootPassword=minio-secret \
  --set defaultBuckets=fluxbase
```

Configure Fluxbase:

```yaml
config:
  storage:
    provider: s3
    s3:
      bucket: fluxbase
      region: us-east-1
      endpoint: http://minio:9000
```

---

## High Availability Configuration

### Multi-Zone Deployment

```yaml
# Pod anti-affinity to spread across zones
podAntiAffinityPreset: hard

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
                - fluxbase
        topologyKey: topology.kubernetes.io/zone

# Topology spread constraints
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: DoNotSchedule
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: fluxbase
```

### Pod Disruption Budget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: fluxbase-pdb
  namespace: fluxbase
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app.kubernetes.io/name: fluxbase
```

### Database High Availability

Use managed database with multi-AZ:

- AWS RDS Multi-AZ
- Google Cloud SQL HA
- Azure Database for PostgreSQL HA

Or PostgreSQL operator with replication:

```yaml
# CloudNativePG cluster with 3 replicas
spec:
  instances: 3
  minSyncReplicas: 1
  maxSyncReplicas: 2
```

---

## Security

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: fluxbase-network-policy
  namespace: fluxbase
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: fluxbase
  policyTypes:
    - Ingress
    - Egress
  ingress:
    # Allow from ingress controller
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
        ports:
          - protocol: TCP
            port: 8080
  egress:
    # Allow to database
    - to:
        - podSelector:
            matchLabels:
              app.kubernetes.io/name: postgresql
        ports:
          - protocol: TCP
            port: 5432
    # Allow DNS
    - to:
        - namespaceSelector: {}
        ports:
          - protocol: UDP
            port: 53
    # Allow to internet (for external APIs)
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
            except:
              - 169.254.169.254/32  # Block metadata service
```

### Pod Security Standards

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: fluxbase
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

### Secrets Management with External Secrets Operator

```bash
# Install External Secrets Operator
helm install external-secrets external-secrets/external-secrets \
  --namespace external-secrets-system \
  --create-namespace
```

Configure AWS Secrets Manager integration:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secrets-manager
  namespace: fluxbase
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        jwt:
          serviceAccountRef:
            name: fluxbase

---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: fluxbase-secrets
  namespace: fluxbase
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: fluxbase-secrets
  data:
    - secretKey: jwt-secret
      remoteRef:
        key: fluxbase/jwt-secret
    - secretKey: database-password
      remoteRef:
        key: fluxbase/database-password
```

---

## Backup and Disaster Recovery

### Automated Backups with Velero

```bash
# Install Velero
helm install velero vmware-tanzu/velero \
  --namespace velero \
  --create-namespace \
  --set configuration.provider=aws \
  --set configuration.backupStorageLocation.bucket=backups \
  --set configuration.backupStorageLocation.config.region=us-east-1

# Create backup schedule
velero schedule create fluxbase-daily \
  --schedule="0 2 * * *" \
  --include-namespaces fluxbase \
  --ttl 720h0m0s
```

### Database Backups

```bash
# Create CronJob for database backups
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
  namespace: fluxbase
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: backup
              image: postgres:18-alpine
              env:
                - name: PGHOST
                  value: postgresql
                - name: PGUSER
                  value: fluxbase
                - name: PGPASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: fluxbase-db-secret
                      key: password
              command:
                - sh
                - -c
                - |
                  pg_dump fluxbase | gzip > /backups/backup-$(date +%Y%m%d-%H%M%S).sql.gz
              volumeMounts:
                - name: backups
                  mountPath: /backups
          volumes:
            - name: backups
              persistentVolumeClaim:
                claimName: postgres-backups
          restartPolicy: OnFailure
```

---

## Upgrading

### Rolling Update

```bash
# Update values
helm upgrade fluxbase ./fluxbase \
  --namespace fluxbase \
  --values production-values.yaml

# Monitor rollout
kubectl rollout status deployment/fluxbase -n fluxbase

# Rollback if needed
helm rollback fluxbase -n fluxbase
```

### Blue-Green Deployment

```bash
# Install new version alongside old
helm install fluxbase-blue ./fluxbase \
  --namespace fluxbase-blue \
  --values production-values.yaml

# Test new version
kubectl port-forward -n fluxbase-blue svc/fluxbase-blue 8081:8080

# Switch traffic (update Ingress or Service)
kubectl patch ingress fluxbase -n fluxbase \
  --type json \
  -p '[{"op": "replace", "path": "/spec/rules/0/http/paths/0/backend/service/name", "value": "fluxbase-blue"}]'

# Delete old version after verification
helm uninstall fluxbase -n fluxbase
```

---

## Troubleshooting

### Pod Not Starting

```bash
# Check pod status
kubectl describe pod <pod-name> -n fluxbase

# Check logs
kubectl logs <pod-name> -n fluxbase

# Check events
kubectl get events -n fluxbase --sort-by='.lastTimestamp'
```

### Database Connection Issues

```bash
# Test database connectivity
kubectl run -it --rm debug --image=postgres:18-alpine --restart=Never -n fluxbase -- \
  psql -h postgresql -U fluxbase -d fluxbase

# Check database service
kubectl get svc -n fluxbase
kubectl describe svc postgresql -n fluxbase
```

### Ingress Not Working

```bash
# Check ingress status
kubectl describe ingress fluxbase -n fluxbase

# Check ingress controller logs
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller

# Verify TLS certificate
kubectl get certificate -n fluxbase
kubectl describe certificate fluxbase-tls -n fluxbase
```

---

## Next Steps

- [Production Checklist](production-checklist) - Pre-deployment verification
- [Scaling Guide](scaling) - Performance optimization
- [Docker Deployment](docker) - Alternative deployment method
