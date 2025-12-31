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
2. Enter your **Setup Token** (the `FLUXBASE_SECURITY_SETUP_TOKEN` value from step 2)
3. Fill in your admin account details:
   - **Full Name** - Your display name
   - **Email** - Used for login
   - **Password** - Minimum 12 characters
4. Click **Complete Setup**

You'll be automatically logged in and redirected to the dashboard.

:::note[Setup Token]
The setup token is a one-time security measure. It ensures only someone with access to your server configuration can create the initial admin account. See the [Initial Setup Guide](/guides/admin/setup-guide/) for more details.
:::

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

## What's Next?

Now that Fluxbase is running, here's what to do next:

### Build Your First API

1. **Create a table** in the Tables Browser
2. **Generate a client key** in Settings > Client Keys
3. **Query your data** using the SDK or REST API

For a complete walkthrough, see the [First API Tutorial](/guides/tutorials/first-api/).

### Connect Your Application

Install the TypeScript SDK:

```bash
npm install @fluxbase/sdk
```

```typescript
import { createClient } from '@fluxbase/sdk'

const fluxbase = createClient('http://localhost:8080', 'your-client-key')

// Query data
const { data, error } = await fluxbase.from('users').select('*')
```

### Set Up Authentication

Enable user signups in Settings > Authentication, then:

```typescript
// Sign up a user
const { user, error } = await fluxbase.auth.signUp({
  email: 'user@example.com',
  password: 'securepassword123'
})
```

See the [Authentication Guide](/guides/authentication/) for OAuth, magic links, and more.

### Secure Your Data

Use Row-Level Security (RLS) to control access:

```sql
-- Users can only read their own data
CREATE POLICY "Users read own data"
ON public.profiles FOR SELECT
USING (auth.uid() = user_id);
```

See the [Row-Level Security Guide](/guides/row-level-security/).

## Learn More

- [First API Tutorial](/guides/tutorials/first-api/) - Complete beginner walkthrough
- [TypeScript SDK Guide](/guides/typescript-sdk/) - SDK reference and examples
- [Authentication Guide](/guides/authentication/) - User authentication options
- [Edge Functions](/guides/edge-functions/) - Deploy serverless functions
- [Configuration Reference](/reference/configuration/) - All configuration options
- [Docker Deployment](/deployment/docker/) - Production deployment guide
