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
  postgres:16-alpine
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

### Method 3: Docker Compose (Full Stack)

Create `docker-compose.yml`:

```yaml
version: "3.8"

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: fluxbase
      POSTGRES_USER: fluxbase
      POSTGRES_PASSWORD: postgres
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U fluxbase"]
      interval: 5s
      timeout: 5s
      retries: 5

  fluxbase:
    image: ghcr.io/wayli-app/fluxbase:latest
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://fluxbase:postgres@postgres:5432/fluxbase?sslmode=disable
      JWT_SECRET: change-this-to-a-secure-random-string
      PORT: 8080
    ports:
      - "8080:8080"
    volumes:
      - ./storage:/app/storage

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data

volumes:
  postgres_data:
  minio_data:
```

Start the stack:

```bash
docker-compose up -d
```

### Method 4: Build from Source

Requirements:

- Go 1.22 or later
- Make
- Git

```bash
# Clone the repository
git clone https://github.com/wayli-app/fluxbase.git
cd fluxbase

# Build the binary
make build

# Install to /usr/local/bin (optional)
sudo make install
```

The binary will be created at `./fluxbase`.

## Configuration

Create a configuration file `fluxbase.yaml`:

```yaml
# Server Configuration
server:
  port: 8080
  host: 0.0.0.0

# Database Configuration
database:
  url: postgres://fluxbase:password@localhost:5432/fluxbase?sslmode=disable
  max_connections: 100
  idle_connections: 10

# JWT Authentication
jwt:
  secret: your-secret-key-change-this-in-production
  access_token_expiry: 15m
  refresh_token_expiry: 7d

# Storage Configuration
storage:
  provider: local # or "s3"
  local_path: ./storage
  max_upload_size: 10485760 # 10MB

# Realtime Configuration
realtime:
  enabled: true
  heartbeat_interval: 30s

# Admin UI
admin:
  enabled: true
  path: /admin
```

Or use environment variables:

```bash
export DATABASE_URL=postgres://fluxbase:password@localhost:5432/fluxbase
export JWT_SECRET=your-secret-key
export PORT=8080
export STORAGE_PROVIDER=local
export STORAGE_LOCAL_PATH=./storage
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

### Database Connection Failed

```
Error: failed to connect to database
```

**Solution:**

- Check PostgreSQL is running: `sudo systemctl status postgresql`
- Verify connection string in `DATABASE_URL`
- Ensure database exists: `psql -U postgres -l`

### Port Already in Use

```
Error: listen tcp :8080: bind: address already in use
```

**Solution:**

- Change port: `PORT=8081 fluxbase`
- Or kill existing process: `lsof -ti:8080 | xargs kill`

### Migrations Failed

```
Error: migration 001_init.up.sql failed
```

**Solution:**

- Drop and recreate database (development only!)
- Check PostgreSQL logs: `sudo journalctl -u postgresql`
- Ensure fluxbase user has proper permissions

### Permission Denied

```
Error: permission denied for schema public
```

**Solution:**

```sql
GRANT ALL PRIVILEGES ON DATABASE fluxbase TO fluxbase;
GRANT ALL ON SCHEMA public TO fluxbase;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO fluxbase;
```

## Upgrading

### Binary Installation

```bash
# Backup your database first!
pg_dump fluxbase > backup.sql

# Download new version
curl -L https://github.com/wayli-app/fluxbase/releases/latest/download/fluxbase-linux-amd64 -o fluxbase
chmod +x fluxbase

# Stop old version
sudo systemctl stop fluxbase  # or kill the process

# Run migrations
./fluxbase migrate

# Start new version
./fluxbase
```

### Docker

```bash
# Pull latest image
docker pull ghcr.io/wayli-app/fluxbase:latest

# Recreate container
docker-compose down
docker-compose up -d
```

## Running as a Service

### systemd (Linux)

Create `/etc/systemd/system/fluxbase.service`:

```ini
[Unit]
Description=Fluxbase Backend as a Service
After=postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=fluxbase
WorkingDirectory=/opt/fluxbase
Environment="DATABASE_URL=postgres://fluxbase:password@localhost:5432/fluxbase"
Environment="JWT_SECRET=your-secret-key"
Environment="PORT=8080"
ExecStart=/usr/local/bin/fluxbase
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable fluxbase
sudo systemctl start fluxbase
sudo systemctl status fluxbase
```

## Next Steps

- [Quick Start Tutorial](./quick-start.md) - Build your first app
- [Configuration Reference](../reference/configuration.md) - Customize Fluxbase
- [SDK Documentation](../guides/typescript-sdk/getting-started.md) - Use TypeScript or Go SDKs
- [Authentication Guide](../guides/authentication.md) - Set up auth in your app

## Need Help?

- **GitHub Issues**: [github.com/wayli-app/fluxbase/issues](https://github.com/wayli-app/fluxbase/issues)
- **GitHub Discussions**: [github.com/wayli-app/fluxbase/discussions](https://github.com/wayli-app/fluxbase/discussions)
- **Discord**: [discord.gg/fluxbase](https://discord.gg/fluxbase)
