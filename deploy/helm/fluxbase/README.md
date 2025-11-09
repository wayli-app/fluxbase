# Fluxbase Helm Chart

Production-ready Helm chart for deploying Fluxbase on Kubernetes, following Bitnami best practices.

## TL;DR

```bash
helm install my-fluxbase ./fluxbase
```

## Introduction

This chart bootstraps a Fluxbase deployment on a Kubernetes cluster using the Helm package manager.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+
- PV provisioner support in the underlying infrastructure (if persistence is enabled)

## Installing the Chart

To install the chart with the release name `my-fluxbase`:

```bash
helm install my-fluxbase ./fluxbase
```

## Uninstalling the Chart

To uninstall/delete the `my-fluxbase` deployment:

```bash
helm delete my-fluxbase
```

## Parameters

### Global parameters

| Name                      | Description                                     | Value |
| ------------------------- | ----------------------------------------------- | ----- |
| `global.imageRegistry`    | Global Docker image registry                    | `""`  |
| `global.imagePullSecrets` | Global Docker registry secret names as an array | `[]`  |
| `global.storageClass`     | Global StorageClass for Persistent Volume(s)    | `""`  |

### Common parameters

| Name                | Description                                    | Value |
| ------------------- | ---------------------------------------------- | ----- |
| `nameOverride`      | String to partially override common.names.name | `""`  |
| `fullnameOverride`  | String to fully override common.names.fullname | `""`  |
| `commonLabels`      | Labels to add to all deployed objects          | `{}`  |
| `commonAnnotations` | Annotations to add to all deployed objects     | `{}`  |

### Fluxbase Image parameters

| Name               | Description                | Value               |
| ------------------ | -------------------------- | ------------------- |
| `image.registry`   | Fluxbase image registry    | `docker.io`         |
| `image.repository` | Fluxbase image repository  | `fluxbase/fluxbase` |
| `image.tag`        | Fluxbase image tag         | `0.1.0`             |
| `image.pullPolicy` | Fluxbase image pull policy | `IfNotPresent`      |

### Deployment Parameters

| Name                | Description                           | Value   |
| ------------------- | ------------------------------------- | ------- |
| `replicaCount`      | Number of Fluxbase replicas           | `3`     |
| `podLabels`         | Extra labels for Fluxbase pods        | `{}`    |
| `podAnnotations`    | Annotations for Fluxbase pods         | `{}`    |
| `resourcesPreset`   | Set container resources preset        | `small` |
| `initContainers`    | Add additional init containers        | `[]`    |
| `sidecars`          | Add additional sidecar containers     | `[]`    |
| `extraVolumes`      | Extra volumes for the pod             | `[]`    |
| `extraVolumeMounts` | Extra volume mounts for the container | `[]`    |

### Service parameters

| Name                 | Description                | Value       |
| -------------------- | -------------------------- | ----------- |
| `service.type`       | Fluxbase service type      | `ClusterIP` |
| `service.ports.http` | Fluxbase service HTTP port | `8080`      |

### Ingress parameters

| Name                       | Description                         | Value            |
| -------------------------- | ----------------------------------- | ---------------- |
| `ingress.enabled`          | Enable ingress record generation    | `false`          |
| `ingress.ingressClassName` | IngressClass to use                 | `""`             |
| `ingress.hostname`         | Default host for the ingress record | `fluxbase.local` |
| `ingress.tls`              | Enable TLS configuration            | `false`          |

### Persistence Parameters

| Name                       | Description                  | Value  |
| -------------------------- | ---------------------------- | ------ |
| `persistence.enabled`      | Enable persistence using PVC | `true` |
| `persistence.storageClass` | Storage class of backing PVC | `""`   |
| `persistence.size`         | Size of data volume          | `8Gi`  |

### PostgreSQL Parameters

Fluxbase supports three PostgreSQL deployment modes:

1. **`standalone`** - Simple StatefulSet with official PostgreSQL 18 image (default)
2. **`cnpg`** - CloudNativePG operator for high availability and automated backups
3. **`none`** - Use external PostgreSQL database (AWS RDS, GCP Cloud SQL, etc.)

#### Common PostgreSQL Parameters

| Name                             | Description                                            | Value        |
| -------------------------------- | ------------------------------------------------------ | ------------ |
| `postgresql.mode`                | Deployment mode: `standalone`, `cnpg`, or `none`       | `standalone` |
| `postgresql.auth.username`       | PostgreSQL username                                    | `fluxbase`   |
| `postgresql.auth.password`       | PostgreSQL password (use existingSecret in production) | `fluxbase`   |
| `postgresql.auth.database`       | PostgreSQL database name                               | `fluxbase`   |
| `postgresql.auth.existingSecret` | Name of existing secret with database credentials      | `""`         |

#### Standalone Mode Parameters

| Name                                              | Description                             | Value       |
| ------------------------------------------------- | --------------------------------------- | ----------- |
| `postgresql.standalone.enabled`                   | Enable standalone PostgreSQL deployment | `true`      |
| `postgresql.standalone.image.tag`                 | PostgreSQL image tag                    | `18-alpine` |
| `postgresql.standalone.persistence.enabled`       | Enable persistence for PostgreSQL       | `true`      |
| `postgresql.standalone.persistence.size`          | Size of PostgreSQL data volume          | `8Gi`       |
| `postgresql.standalone.resources.requests.cpu`    | CPU request for PostgreSQL              | `250m`      |
| `postgresql.standalone.resources.requests.memory` | Memory request for PostgreSQL           | `256Mi`     |

#### CloudNativePG Mode Parameters

Requires CloudNativePG operator: `kubectl apply -f https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.25/releases/cnpg-1.25.0.yaml`

| Name                                     | Description                                   | Value                                  |
| ---------------------------------------- | --------------------------------------------- | -------------------------------------- |
| `postgresql.cnpg.enabled`                | Enable CloudNativePG deployment               | `false`                                |
| `postgresql.cnpg.instances`              | Number of PostgreSQL instances in the cluster | `3`                                    |
| `postgresql.cnpg.imageName`              | PostgreSQL image for CNPG                     | `ghcr.io/cloudnative-pg/postgresql:18` |
| `postgresql.cnpg.storage.size`           | Size of PostgreSQL data volume per instance   | `8Gi`                                  |
| `postgresql.cnpg.monitoring.enabled`     | Enable Prometheus monitoring                  | `true`                                 |
| `postgresql.cnpg.backup.enabled`         | Enable automated backups                      | `false`                                |
| `postgresql.cnpg.backup.retentionPolicy` | Backup retention policy                       | `30d`                                  |
| `postgresql.cnpg.pooler.enabled`         | Enable PgBouncer connection pooler            | `false`                                |
| `postgresql.cnpg.pooler.instances`       | Number of pooler instances                    | `2`                                    |

#### External Database Parameters

| Name                                         | Description                              | Value      |
| -------------------------------------------- | ---------------------------------------- | ---------- |
| `externalDatabase.host`                      | External database host                   | `""`       |
| `externalDatabase.port`                      | External database port                   | `5432`     |
| `externalDatabase.user`                      | External database user                   | `fluxbase` |
| `externalDatabase.password`                  | External database password               | `""`       |
| `externalDatabase.database`                  | External database name                   | `fluxbase` |
| `externalDatabase.existingSecret`            | Name of existing secret with credentials | `""`       |
| `externalDatabase.existingSecretPasswordKey` | Key in secret containing password        | `password` |

## Configuration Examples

Fluxbase includes pre-configured example values files in the `examples/` directory:

- **`values-standalone.yaml`** - Standalone PostgreSQL deployment (simple, single-instance)
- **`values-cnpg.yaml`** - CloudNativePG deployment (high availability, automated backups)
- **`values-external.yaml`** - External database configuration (AWS RDS, GCP Cloud SQL, etc.)

### Quick Start - Standalone Mode (Default)

```bash
# Install with default standalone PostgreSQL
helm install fluxbase ./fluxbase \
  --set config.jwt.secret=$(openssl rand -base64 32)
```

### CloudNativePG Mode (High Availability)

```bash
# 1. Install CloudNativePG operator
kubectl apply -f https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.25/releases/cnpg-1.25.0.yaml

# 2. Install Fluxbase with CNPG
helm install fluxbase ./fluxbase \
  -f examples/values-cnpg.yaml \
  --set config.jwt.secret=$(openssl rand -base64 32) \
  --set postgresql.mode=cnpg
```

### Production with External Database

```yaml
# production-values.yaml
postgresql:
  mode: none # Use external database

externalDatabase:
  host: my-postgres.example.com
  port: 5432
  user: fluxbase
  database: fluxbase
  existingSecret: fluxbase-db-secret

config:
  database:
    sslMode: require
  jwt:
    secret: "" # Set via existingSecret

replicaCount: 5

ingress:
  enabled: true
  ingressClassName: nginx
  hostname: api.example.com
  tls: true
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod

autoscaling:
  enabled: true
  minReplicas: 5
  maxReplicas: 20
  targetCPU: 70

metrics:
  enabled: true
  serviceMonitor:
    enabled: true

existingSecret: fluxbase-secrets # Contains DB password, JWT secret, etc.
```

Deploy:

```bash
# Create secrets
kubectl create secret generic fluxbase-secrets \
  --from-literal=database-password=your-db-password \
  --from-literal=jwt-secret=$(openssl rand -base64 32)

# Install
helm install fluxbase ./fluxbase -f production-values.yaml
```

### With Custom Labels and Annotations

```yaml
commonLabels:
  team: backend
  env: production

commonAnnotations:
  owner: platform-team

podLabels:
  app.kubernetes.io/component: api

podAnnotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
```

### With Extra Volumes and Sidecar Container

```yaml
extraVolumes:
  - name: custom-config
    configMap:
      name: my-custom-config

extraVolumeMounts:
  - name: custom-config
    mountPath: /etc/custom

sidecars:
  - name: log-shipper
    image: fluentbit/fluent-bit:2.0
    volumeMounts:
      - name: logs
        mountPath: /var/log/fluxbase
```

## Security Configuration

### Service Role Keys

Service keys provide elevated privileges for backend services, bypassing Row-Level Security (RLS) policies.

**⚠️ WARNING**: Service keys have full database access. Never expose them to clients or commit to version control.

#### Creating a Service Key

1. **Generate a random key**:

   ```bash
   openssl rand -base64 32
   ```

2. **Format the key**:

   ```
   sk_live_<your_random_string>  # Production
   sk_test_<your_random_string>  # Development
   ```

3. **Create Kubernetes secret**:

   ```bash
   kubectl create secret generic fluxbase-service-key \
     --from-literal=service-key="sk_live_abc123xyz..."
   ```

4. **Hash and store in database**:

   ```sql
   INSERT INTO auth.service_keys (name, description, key_hash, key_prefix, enabled)
   VALUES (
     'Backend Service',
     'Service key for backend API calls',
     crypt('sk_live_abc123xyz...', gen_salt('bf', 12)),  -- bcrypt hash
     'sk_live_',  -- First 8 characters
     true
   );
   ```

5. **Enable in Helm values**:
   ```yaml
   serviceKey:
     enabled: true
     existingSecret: fluxbase-service-key
     secretKey: service-key
   ```

#### Using Service Keys

Service keys are passed via HTTP headers to Fluxbase:

```bash
# Via X-Service-Key header (recommended)
curl -H "X-Service-Key: sk_live_abc123..." https://api.example.com/api/v1/tables/users

# Via Authorization header
curl -H "Authorization: ServiceKey sk_live_abc123..." https://api.example.com/api/v1/tables/users
```

#### Best Practices

- ✅ Store service keys in Kubernetes Secrets or external secret managers (Vault, AWS Secrets Manager)
- ✅ Use separate keys for different environments (dev/staging/prod)
- ✅ Rotate service keys regularly
- ✅ Monitor usage via `last_used_at` timestamp
- ✅ Set expiration dates where possible
- ❌ Never commit service keys to version control
- ❌ Never expose service keys in client-side code
- ❌ Never log service keys in plaintext

For more details, see the [Authentication Guide](../../../docs/docs/guides/authentication.md).

## Upgrading

### To 1.0.0

This version uses Bitnami common chart patterns and includes:

- Enhanced security contexts
- Resource presets (nano, micro, small, medium, large, xlarge, 2xlarge)
- Full Prometheus metrics support
- Comprehensive extensibility (init containers, sidecars, extra volumes)
- Service key support for backend services

## License

MIT
