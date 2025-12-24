---
title: "Quick Start"
---

Get Fluxbase running in under 5 minutes using Docker.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose installed

## 1. Create docker-compose.yml

Create a `docker-compose.yml` file with the following content:

```yaml
services:
  postgres:
    image: postgis/postgis:18-3.6
    environment:
      POSTGRES_DB: fluxbase
      POSTGRES_USER: fluxbase
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql

  fluxbase:
    image: ghcr.io/fluxbase-eu/fluxbase:latest
    depends_on:
      - postgres
    environment:
      FLUXBASE_DATABASE_HOST: postgres
      FLUXBASE_DATABASE_PORT: 5432
      FLUXBASE_DATABASE_USER: fluxbase
      FLUXBASE_DATABASE_PASSWORD: postgres
      FLUXBASE_DATABASE_DATABASE: fluxbase
      FLUXBASE_DATABASE_SSL_MODE: disable
      FLUXBASE_AUTH_JWT_SECRET: change-this-to-a-secure-random-string-min-32-chars
      FLUXBASE_SECURITY_SETUP_TOKEN: change-this-to-another-secure-random-string
    ports:
      - "8080:8080"

volumes:
  postgres_data:
```

## 2. Set Secure Secrets

Generate secure values for the secrets:

```bash
# Generate JWT secret (copy output to FLUXBASE_AUTH_JWT_SECRET)
openssl rand -base64 32

# Generate setup token (copy output to FLUXBASE_SECURITY_SETUP_TOKEN)
openssl rand -base64 32
```

Update `docker-compose.yml` with your generated secrets.

:::caution[Security Warning]
Never use the example secrets in production. Both secrets should be strong, random strings of at least 32 characters.
:::

## 3. Start Fluxbase

```bash
docker compose up -d
```

Wait a few seconds for the services to start. Check logs with:

```bash
docker compose logs fluxbase
```

## 4. Complete Setup

1. Open [http://localhost:8080/admin/setup](http://localhost:8080/admin/setup) in your browser
2. Enter your **Setup Token** (the `FLUXBASE_SECURITY_SETUP_TOKEN` value)
3. Create your admin account (email and password)
4. You'll be redirected to the admin dashboard

## 5. Explore the Admin Dashboard

After logging in at [http://localhost:8080/admin](http://localhost:8080/admin), you can:

- **Tables Browser** - Create tables and manage data
- **Authentication** - View and manage users
- **Storage** - Upload and manage files
- **Functions** - Deploy edge functions
- **Realtime** - Monitor WebSocket connections

## Next Steps

- [TypeScript SDK Guide](/docs/sdk/getting-started) - Build applications with the SDK
- [Authentication Guide](/docs/guides/authentication) - Set up user authentication
- [Row-Level Security](/docs/guides/row-level-security) - Secure your data
- [AI Chatbots](/docs/guides/ai-chatbots) - Build natural language interfaces
- [Configuration Reference](/docs/reference/configuration) - All configuration options
