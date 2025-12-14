---
title: "Installation"
---

This guide walks you through installing Fluxbase on your system.

## Prerequisites

Before installing Fluxbase, ensure you have:

- **PostgreSQL 15+** - Fluxbase requires PostgreSQL as its database
- **64-bit Operating System** - Linux
- **1GB RAM minimum** (2GB+ recommended for production)
- **100MB disk space** (plus space for your data)

## Installing PostgreSQL

### Ubuntu/Debian

```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

### macOS (Homebrew)

```bash
brew install postgresql@16
brew services start postgresql@16
```

### Docker

```bash
docker run -d \
  --name fluxbase-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=fluxbase \
  -p 5432:5432 \
  postgis/postgis:18-3.6
```

### Create Database

```bash
# Connect to PostgreSQL
psql -U postgres

# Create database and user
CREATE DATABASE fluxbase;
CREATE USER fluxbase WITH PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE fluxbase TO fluxbase;
\q
```

## Installing Fluxbase

Choose one of the following installation methods:

### Method 1: Download Pre-built Binary (Recommended)

Download the latest release for your platform (~40MB binary):

**Linux (x86_64)**

```bash
curl -L https://github.com/fluxbase-eu/fluxbase/releases/latest/download/fluxbase-linux-amd64 -o fluxbase
chmod +x fluxbase
sudo mv fluxbase /usr/local/bin/
```

**macOS (Intel)**

```bash
curl -L https://github.com/fluxbase-eu/fluxbase/releases/latest/download/fluxbase-darwin-amd64 -o fluxbase
chmod +x fluxbase
sudo mv fluxbase /usr/local/bin/
```

### Method 2: Docker

Pull and run the official Docker image (~80MB container):

```bash
docker pull ghcr.io/fluxbase-eu/fluxbase:latest

docker run -d \
  --name fluxbase \
  -p 8080:8080 \
  -e DATABASE_URL=postgres://fluxbase:password@host.docker.internal:5432/fluxbase \
  -e JWT_SECRET=your-secret-key-change-this \
  ghcr.io/fluxbase-eu/fluxbase:latest
```

### Method 3: Docker Compose

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

  fluxbase:
    image: ghcr.io/fluxbase-eu/fluxbase:latest
    depends_on:
      - postgres
    environment:
      DATABASE_URL: postgres://fluxbase:postgres@postgres:5432/fluxbase?sslmode=disable
      JWT_SECRET: change-this-to-a-secure-random-string
    ports:
      - "8080:8080"
```

Start: `docker-compose up -d`

### Method 4: Build from Source

```bash
git clone https://github.com/fluxbase-eu/fluxbase.git
cd fluxbase
make build
```

## Configuration

### Required Environment Variables

Fluxbase requires these environment variables to be set:

| Variable                        | Description                                           | How to Generate           |
| ------------------------------- | ----------------------------------------------------- | ------------------------- |
| `FLUXBASE_AUTH_JWT_SECRET`      | Secret key for signing JWT tokens (min 32 characters) | `openssl rand -base64 32` |
| `FLUXBASE_SECURITY_SETUP_TOKEN` | Token for initial admin setup (min 32 characters)     | `openssl rand -base64 32` |
| `FLUXBASE_DATABASE_*`           | Database connection settings                          | See below                 |

:::caution[Security Warning]
Never use default or weak secrets in production. Both `JWT_SECRET` and `SETUP_TOKEN` should be strong, random strings.
:::

**Generate secure secrets:**

```bash
# Generate JWT secret
openssl rand -base64 32
# Example output: K7gNU3sdo+OL0wNhqoVWhr3g6s1xYv72ol/pe/Unols=

# Generate setup token
openssl rand -base64 32
# Example output: 8mHBJQTVx2XUd7s4ZKrqMWJB5sGhYm9kP3nXcLfRabc=
```

**Minimal environment variables:**

```bash
# Required: Database connection
export FLUXBASE_DATABASE_HOST=localhost
export FLUXBASE_DATABASE_PORT=5432
export FLUXBASE_DATABASE_USER=fluxbase
export FLUXBASE_DATABASE_PASSWORD=your-db-password
export FLUXBASE_DATABASE_DATABASE=fluxbase

# Required: JWT secret for authentication (min 32 chars)
export FLUXBASE_AUTH_JWT_SECRET=your-secure-jwt-secret-min-32-chars

# Required: Setup token for admin dashboard access
export FLUXBASE_SECURITY_SETUP_TOKEN=your-secure-setup-token-min-32-chars

# Optional: Server port (default: 8080)
export FLUXBASE_SERVER_ADDRESS=:8080

# Optional: Base URL for callbacks (magic links, OAuth)
export FLUXBASE_BASE_URL=http://localhost:8080
```

**Or create `fluxbase.yaml`:**

```yaml
database:
  host: localhost
  port: 5432
  user: fluxbase
  password: your-db-password
  database: fluxbase
  ssl_mode: disable

auth:
  jwt_secret: your-secure-jwt-secret-min-32-chars

security:
  setup_token: your-secure-setup-token-min-32-chars

storage:
  provider: local
  local_path: ./storage

base_url: http://localhost:8080
```

## Initialize Database

Run database migrations:

```bash
fluxbase migrate
```

This will:

- Create the `auth` schema with user tables
- Create the `storage` schema for file metadata
- Set up realtime triggers
- Initialize system functions

## Start Fluxbase

```bash
fluxbase
```

You should see:

```
🚀 Fluxbase starting...
✅ Database connected: PostgreSQL 16.0
✅ Migrations applied: 4 migrations
✅ Admin UI available at: http://localhost:8080/admin
✅ REST API available at: http://localhost:8080/api/v1
✅ Realtime WebSocket at: ws://localhost:8080/realtime
🎉 Fluxbase is ready! Listening on http://localhost:8080
```

## Verify Installation

### 1. Check Health

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{
  "status": "healthy",
  "database": "connected",
  "version": "0.1.0"
}
```

### 2. Create First Admin Account

:::note[Important First Step]
Before using Fluxbase, you must create the first admin account through the setup wizard.
:::

**Navigate to the Admin Setup page:**

Open `http://localhost:8080/admin/setup` in your browser.

You will be prompted to:

1. **Enter the Setup Token** - This is the `FLUXBASE_SECURITY_SETUP_TOKEN` you configured
2. **Create Admin Account** - Enter email and password for the first admin user

```text
┌─────────────────────────────────────────┐
│         Fluxbase Admin Setup            │
├─────────────────────────────────────────┤
│  Setup Token: [________________]        │
│                                         │
│  Admin Email: [________________]        │
│  Admin Password: [________________]     │
│                                         │
│  [Create Admin Account]                 │
└─────────────────────────────────────────┘
```

After successful setup, you'll be redirected to the admin login page.

### 3. Access Admin Dashboard

Once the admin account is created:

1. Navigate to `http://localhost:8080/admin`
2. Log in with the admin credentials you just created
3. Explore the dashboard: Tables, Users, Storage, Functions, Jobs, etc.

### 4. Create Application Users

Regular application users can sign up via the API:

```bash
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123"
  }'
```

:::tip
Admin accounts (created via `/admin/setup`) are separate from application users. Admin accounts can access the dashboard, while application users interact with your app's data via the API.
:::

### 5. Test REST API

Create a table:

```sql
-- Connect to your database
psql postgres://fluxbase:password@localhost:5432/fluxbase

-- Create a test table
CREATE TABLE tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  completed BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

Query via REST API:

```bash
curl http://localhost:8080/api/v1/tables/tasks
```

## Troubleshooting

| Issue                          | Solution                                                                                               |
| ------------------------------ | ------------------------------------------------------------------------------------------------------ |
| **Database connection failed** | Check PostgreSQL running: `systemctl status postgresql`, verify `DATABASE_URL`, ensure database exists |
| **Port already in use**        | Change port: `PORT=8081 fluxbase` or kill process: `lsof -ti:8080 \| xargs kill`                       |
| **Migrations failed**          | Check PostgreSQL logs, ensure user has permissions, drop/recreate DB (dev only)                        |
| **Permission denied**          | Grant permissions: `GRANT ALL ON SCHEMA public TO fluxbase;`                                           |

## Upgrading

**Binary:**

```bash
pg_dump fluxbase > backup.sql  # Backup first!
curl -L https://github.com/fluxbase-eu/fluxbase/releases/latest/download/fluxbase-linux-amd64 -o fluxbase
chmod +x fluxbase && ./fluxbase migrate && ./fluxbase
```

**Docker:**

```bash
docker pull ghcr.io/fluxbase-eu/fluxbase:latest && docker-compose up -d
```

## Running as systemd Service

Create `/etc/systemd/system/fluxbase.service`:

```ini
[Unit]
Description=Fluxbase
After=postgresql.service

[Service]
Type=simple
Environment="DATABASE_URL=postgres://fluxbase:password@localhost:5432/fluxbase"
Environment="JWT_SECRET=your-secret-key"
ExecStart=/usr/local/bin/fluxbase
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Enable: `systemctl enable fluxbase && systemctl start fluxbase`

## Next Steps

- [Quick Start Tutorial](./quick-start.md)
- [Configuration Reference](../reference/configuration.md)
- [TypeScript SDK](../guides/typescript-sdk/getting-started.md)
