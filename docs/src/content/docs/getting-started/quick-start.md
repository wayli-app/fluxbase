---
title: "Quick Start"
---

Get Fluxbase running in under 5 minutes using Docker.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose installed
- 500MB disk space (plus space for your data)

## 1. Clone the Repository

```bash
git clone https://github.com/fluxbase-eu/fluxbase.git
cd fluxbase/deploy
```

## 2. Generate Secrets

Run the secrets generator:

```bash
./generate-keys.sh
```

Select option **1** (Docker Compose) when prompted. This creates a `.env` file with all required secrets.

:::tip[Save Your Setup Token]
The script displays your **Setup Token** once. Save it - you'll need it to access the admin dashboard.
:::

## 3. Start Fluxbase

```bash
docker compose -f docker-compose.minimal.yaml up -d
```

Wait for the services to start (first run takes ~30 seconds for migrations). Check the logs:

```bash
docker logs -f fluxbase
```

You should see:

```
Fluxbase starting...
Database connected
Migrations applied
Fluxbase is ready!
```

## 4. Complete Setup

1. Open [http://localhost:8080/admin/setup](http://localhost:8080/admin/setup)
2. Enter your **Setup Token** (from step 2)
3. Create your admin account
4. You're in!

## Test the API

```bash
curl http://localhost:8080/health
```

```json
{"status": "healthy", "database": "connected"}
```

## Explore the Admin Dashboard

At [http://localhost:8080/admin](http://localhost:8080/admin):

- **Tables Browser** - Create tables and manage data
- **Authentication** - View and manage users
- **Storage** - Upload and manage files
- **Functions** - Deploy edge functions
- **Realtime** - Monitor WebSocket connections

## Troubleshooting

**Database connection errors after changing secrets:**

```bash
docker compose -f docker-compose.minimal.yaml down -v
docker compose -f docker-compose.minimal.yaml up -d
```

The `-v` flag resets volumes so PostgreSQL reinitializes with the new password.

## Next Steps

- [TypeScript SDK Guide](/docs/sdk/getting-started) - Build applications with the SDK
- [Authentication Guide](/docs/guides/authentication) - Set up user authentication
- [Row-Level Security](/docs/guides/row-level-security) - Secure your data
- [Edge Functions](/docs/guides/edge-functions) - Deploy serverless functions
- [Configuration Reference](/docs/reference/configuration) - All configuration options
- [Docker Deployment](/docs/deployment/docker) - Production deployment guide
