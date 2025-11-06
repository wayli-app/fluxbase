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

| Name                       | Description                    | Value      |
| -------------------------- | ------------------------------ | ---------- |
| `postgresql.enabled`       | Deploy PostgreSQL container(s) | `true`     |
| `postgresql.auth.username` | PostgreSQL username            | `fluxbase` |
| `postgresql.auth.password` | PostgreSQL password            | `fluxbase` |
| `postgresql.auth.database` | PostgreSQL database            | `fluxbase` |

## Configuration Examples

### Production with External Database

```yaml
# production-values.yaml
replicaCount: 5

postgresql:
  enabled: false

externalDatabase:
  host: my-postgres.example.com
  port: 5432
  user: fluxbase
  database: fluxbase
  existingSecret: fluxbase-db-secret

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

# Optional: Enable service key for backend services
serviceKey:
  enabled: true
  existingSecret: fluxbase-service-key
  secretKey: service-key
```

Deploy:

```bash
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
