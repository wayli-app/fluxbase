---
sidebar_position: 1
---

# Installation

This guide walks you through installing Fluxbase on your system.

## Prerequisites

Before installing Fluxbase, ensure you have:

- **PostgreSQL 14+** - Fluxbase requires PostgreSQL as its database
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
  postgres:18-alpine
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
curl -L https://github.com/wayli-app/fluxbase/releases/latest/download/fluxbase-linux-amd64 -o fluxbase
chmod +x fluxbase
sudo mv fluxbase /usr/local/bin/
```

**macOS (Intel)**

```bash
curl -L https://github.com/wayli-app/fluxbase/releases/latest/download/fluxbase-darwin-amd64 -o fluxbase
chmod +x fluxbase
sudo mv fluxbase /usr/local/bin/
```

### Method 2: Docker

Pull and run the official Docker image (~80MB container):

```bash
docker pull ghcr.io/wayli-app/fluxbase:latest

docker run -d \
  --name fluxbase \
  -p 8080:8080 \
  -e DATABASE_URL=postgres://fluxbase:password@host.docker.internal:5432/fluxbase \
  -e JWT_SECRET=your-secret-key-change-this \
  ghcr.io/wayli-app/fluxbase:latest
```

### Method 3: Docker Compose

```yaml
services:
  postgres:
    image: postgres:18-alpine
    environment:
      POSTGRES_DB: fluxbase
      POSTGRES_USER: fluxbase
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"

  fluxbase:
    image: ghcr.io/wayli-app/fluxbase:latest
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
git clone https://github.com/wayli-app/fluxbase.git
cd fluxbase
make build
```

## Configuration

**Environment variables:**

```bash
export DATABASE_URL=postgres://fluxbase:password@localhost:5432/fluxbase
export JWT_SECRET=your-secret-key
export PORT=8080
```

**Or create `fluxbase.yaml`:**

```yaml
database:
  url: postgres://fluxbase:password@localhost:5432/fluxbase
jwt:
  secret: your-secret-key
storage:
  provider: local
  local_path: ./storage
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

### 2. Access Admin UI

Open http://localhost:8080/admin in your browser.

### 3. Create First User

```bash
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "SecurePassword123"
  }'
```

### 4. Test REST API

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
curl -L https://github.com/wayli-app/fluxbase/releases/latest/download/fluxbase-linux-amd64 -o fluxbase
chmod +x fluxbase && ./fluxbase migrate && ./fluxbase
```

**Docker:**

```bash
docker pull ghcr.io/wayli-app/fluxbase:latest && docker-compose up -d
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
