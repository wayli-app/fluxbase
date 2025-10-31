# Migrating from Supabase to Fluxbase

**Complete guide for migrating your Supabase project to Fluxbase**

This guide helps you migrate your existing Supabase application to Fluxbase with minimal code changes. Fluxbase provides API compatibility with Supabase, making migration straightforward.

## 📊 Feature Comparison

| Feature | Supabase | Fluxbase | Notes |
|---------|----------|----------|-------|
| **REST API** | ✅ PostgREST | ✅ Compatible | Full query compatibility |
| **Authentication** | ✅ GoTrue | ✅ Compatible | JWT-based auth |
| **Realtime** | ✅ WebSocket | ✅ Compatible | PostgreSQL LISTEN/NOTIFY |
| **Storage** | ✅ S3-compatible | ✅ Compatible | File operations |
| **Edge Functions** | ✅ Deno | ✅ Deno | Same runtime |
| **Database** | ✅ PostgreSQL 15+ | ✅ PostgreSQL 15+ | Same database |
| **Row-Level Security** | ✅ Yes | ✅ Yes | PostgreSQL RLS |
| **Client SDK** | ✅ TypeScript/JS | ✅ TypeScript/JS | API-compatible |
| **Self-Hosted** | ✅ Yes | ✅ Yes | Full control |
| **Pricing** | 💰 Hosted/Self-hosted | 🆓 Open Source | No vendor lock-in |

## 🎯 Why Migrate?

### Cost Savings
- **Supabase**: $25/month (Pro) + usage fees
- **Fluxbase**: $0 (self-hosted) or custom pricing

### Full Control
- Deploy anywhere (AWS, GCP, Azure, on-premise)
- No vendor lock-in
- Customize as needed
- Own your data

### Feature Parity
- Same API interface
- Same client SDK patterns
- Same PostgreSQL features
- Same authentication flows

## 🚀 Migration Steps

### Step 1: Export Your Database

```bash
# Export your Supabase database schema
pg_dump -h db.your-project.supabase.co \
  -U postgres \
  -d postgres \
  --schema-only \
  --no-owner \
  --no-privileges \
  > schema.sql

# Export your data
pg_dump -h db.your-project.supabase.co \
  -U postgres \
  -d postgres \
  --data-only \
  --no-owner \
  --no-privileges \
  > data.sql
```

### Step 2: Set Up Fluxbase

```bash
# Clone Fluxbase
git clone https://github.com/yourusername/fluxbase.git
cd fluxbase

# Install dependencies
go mod download

# Configure database
cp fluxbase.yaml.example fluxbase.yaml

# Edit fluxbase.yaml
vim fluxbase.yaml
```

**fluxbase.yaml**:
```yaml
database:
  host: localhost
  port: 5432
  name: your_db_name
  user: postgres
  password: your_password
  max_connections: 50

server:
  host: 0.0.0.0
  port: 8080

auth:
  jwt_secret: your-jwt-secret-min-32-chars-long
  token_expiry: 3600  # 1 hour
```

### Step 3: Import Database

```bash
# Create database
createdb -h localhost -U postgres your_db_name

# Import schema
psql -h localhost -U postgres -d your_db_name < schema.sql

# Import data
psql -h localhost -U postgres -d your_db_name < data.sql

# Run Fluxbase migrations (adds auth tables)
./fluxbase migrate up
```

### Step 4: Update Client Code

#### Install Fluxbase Client

```bash
npm uninstall @supabase/supabase-js
npm install @fluxbase/client
```

#### Update Imports

```typescript
// Before (Supabase)
import { createClient } from '@supabase/supabase-js'

const supabase = createClient(
  'https://your-project.supabase.co',
  'your-anon-key'
)

// After (Fluxbase)
import { createClient } from '@fluxbase/client'

const fluxbase = createClient({
  url: 'https://your-fluxbase-instance.com',  // Your deployment
  anonKey: 'your-anon-key'  // Generate with Fluxbase
})
```

#### Update Method Calls

Most API calls are identical:

```typescript
// Both work the same
const { data, error } = await client
  .from('posts')
  .select('*')
  .eq('published', true)
```

### Step 5: Migrate Environment Variables

```bash
# .env.local (Before)
NEXT_PUBLIC_SUPABASE_URL=https://your-project.supabase.co
NEXT_PUBLIC_SUPABASE_ANON_KEY=eyJ...

# .env.local (After)
NEXT_PUBLIC_FLUXBASE_URL=https://your-fluxbase-instance.com
NEXT_PUBLIC_FLUXBASE_ANON_KEY=eyJ...  # Generate new key
```

### Step 6: Generate API Keys

```bash
# Generate anon key (public, for client-side)
./fluxbase generate-key --role anon

# Generate service role key (private, for server-side)
./fluxbase generate-key --role service_role

# Store these securely
```

### Step 7: Update Authentication

Authentication code is mostly identical:

```typescript
// Sign up (identical)
const { data, error } = await fluxbase.auth.signUp({
  email: 'user@example.com',
  password: 'password123'
})

// Sign in (identical)
const { data, error } = await fluxbase.auth.signIn({
  email: 'user@example.com',
  password: 'password123'
})

// Get user (identical)
const { data } = await fluxbase.auth.getUser()
```

### Step 8: Migrate Edge Functions

Edge Functions use the same Deno runtime:

```typescript
// Supabase Edge Function
import { serve } from 'https://deno.land/std@0.168.0/http/server.ts'

serve(async (req) => {
  const { name } = await req.json()
  return new Response(
    JSON.stringify({ message: `Hello ${name}!` }),
    { headers: { 'Content-Type': 'application/json' } }
  )
})

// Fluxbase Edge Function (same code!)
function handler(request) {
  const { name } = JSON.parse(request.body)
  return {
    status: 200,
    body: JSON.stringify({ message: `Hello ${name}!` })
  }
}
```

Deploy function:

```bash
# Create function via API
curl -X POST https://your-fluxbase-instance.com/api/v1/functions \
  -H "Authorization: Bearer YOUR_SERVICE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hello",
    "code": "function handler(request) { ... }",
    "permissions": ["--allow-net"]
  }'
```

### Step 9: Migrate Storage

Storage API is identical:

```typescript
// Upload (identical)
const { data, error } = await fluxbase.storage
  .from('avatars')
  .upload('user-123/avatar.jpg', file)

// Download (identical)
const { data, error } = await fluxbase.storage
  .from('avatars')
  .download('user-123/avatar.jpg')

// Get public URL (identical)
const { publicURL } = fluxbase.storage
  .from('avatars')
  .getPublicUrl('user-123/avatar.jpg')
```

### Step 10: Test & Deploy

```bash
# Run Fluxbase locally
./fluxbase serve

# Test all endpoints
npm run test

# Deploy to production
docker build -t fluxbase:latest .
docker push your-registry/fluxbase:latest
```

## 🔄 API Migration Guide

### Database Queries

**No changes needed** - API is identical:

```typescript
// Supabase
const { data, error } = await supabase
  .from('posts')
  .select('id, title, author:users(name)')
  .eq('published', true)
  .order('created_at', { ascending: false })
  .limit(10)

// Fluxbase (identical!)
const { data, error } = await fluxbase
  .from('posts')
  .select('id, title, author:users(name)')
  .eq('published', true)
  .order('created_at', { ascending: false })
  .limit(10)
```

### Authentication

**Minimal changes**:

```typescript
// Supabase
supabase.auth.onAuthStateChange((event, session) => {
  console.log(event, session)
})

// Fluxbase
fluxbase.auth.onAuthStateChange((event, session) => {
  console.log(event, session)
})
```

### Realtime

**Identical API**:

```typescript
// Supabase
supabase
  .from('posts')
  .on('INSERT', (payload) => console.log(payload))
  .subscribe()

// Fluxbase (same!)
fluxbase
  .from('posts')
  .on('INSERT', (payload) => console.log(payload))
  .subscribe()
```

### Storage

**Identical API**:

```typescript
// Both use the same API
await client.storage.from('bucket').upload('path', file)
await client.storage.from('bucket').download('path')
await client.storage.from('bucket').remove(['path'])
```

## 🔧 Configuration Mapping

### Supabase Dashboard Settings → Fluxbase Config

| Supabase Dashboard | Fluxbase Config | File |
|-------------------|-----------------|------|
| Auth > Email Templates | `auth.email_templates` | `fluxbase.yaml` |
| Auth > Providers | `auth.providers` | `fluxbase.yaml` |
| Database > Connection String | `database.*` | `fluxbase.yaml` |
| Storage > Policies | RLS policies | SQL |
| Edge Functions > Secrets | Environment variables | `.env` |

### Example fluxbase.yaml

```yaml
database:
  host: your-db-host
  port: 5432
  name: your_db
  user: postgres
  password: ${DB_PASSWORD}  # Use env vars for secrets
  max_connections: 50
  ssl_mode: require  # If using cloud database

server:
  host: 0.0.0.0
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

auth:
  jwt_secret: ${JWT_SECRET}  # Min 32 characters
  token_expiry: 3600  # 1 hour
  refresh_token_expiry: 2592000  # 30 days
  smtp_host: smtp.sendgrid.net
  smtp_port: 587
  smtp_user: apikey
  smtp_password: ${SENDGRID_API_KEY}
  from_email: noreply@yourdomain.com

storage:
  provider: s3  # or 'local' for filesystem
  s3_bucket: your-bucket
  s3_region: us-east-1
  s3_access_key: ${AWS_ACCESS_KEY}
  s3_secret_key: ${AWS_SECRET_KEY}

security:
  rate_limit:
    enabled: true
    requests_per_minute: 100
  csrf:
    enabled: true  # Enable for browser apps
    secret: ${CSRF_SECRET}

observability:
  metrics:
    enabled: true
    port: 9090  # Prometheus metrics
  tracing:
    enabled: true
    endpoint: http://jaeger:14268/api/traces
```

## 📝 SQL Schema Adjustments

### Auth Schema

Fluxbase uses a similar auth schema. Map your existing users:

```sql
-- If you have custom user fields, migrate them
INSERT INTO auth.users (id, email, encrypted_password, email_confirmed_at, created_at, updated_at)
SELECT
  id,
  email,
  encrypted_password,
  email_confirmed_at,
  created_at,
  updated_at
FROM supabase_auth_schema.users;

-- Copy user metadata
UPDATE auth.users
SET user_metadata = (
  SELECT raw_user_meta_data
  FROM supabase_auth_schema.users su
  WHERE su.id = auth.users.id
);
```

### RLS Policies

RLS policies work identically:

```sql
-- Existing Supabase RLS policy
CREATE POLICY "Users can view own posts"
  ON posts
  FOR SELECT
  USING (auth.uid() = user_id);

-- Fluxbase equivalent (update function reference)
CREATE POLICY "Users can view own posts"
  ON posts
  FOR SELECT
  USING (user_id::text = current_setting('app.user_id', true));
```

The only change: Replace `auth.uid()` with `current_setting('app.user_id', true)::uuid`.

**Migration script**:

```bash
#!/bin/bash
# migrate_rls.sh

# Replace auth.uid() with Fluxbase equivalent
sed -i 's/auth\.uid()/current_setting('\''app.user_id'\'', true)::uuid/g' schema.sql

# Re-import updated schema
psql -h localhost -U postgres -d your_db < schema.sql
```

## 🎛️ Framework-Specific Migrations

### Next.js

**Before** (`pages/_app.tsx`):
```typescript
import { createClient } from '@supabase/supabase-js'

const supabase = createClient(
  process.env.NEXT_PUBLIC_SUPABASE_URL!,
  process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!
)
```

**After**:
```typescript
import { createClient } from '@fluxbase/client'

const fluxbase = createClient({
  url: process.env.NEXT_PUBLIC_FLUXBASE_URL!,
  anonKey: process.env.NEXT_PUBLIC_FLUXBASE_ANON_KEY!
})
```

### React (Create React App)

**Before** (`src/supabaseClient.ts`):
```typescript
import { createClient } from '@supabase/supabase-js'

export const supabase = createClient(
  import.meta.env.VITE_SUPABASE_URL,
  import.meta.env.VITE_SUPABASE_ANON_KEY
)
```

**After**:
```typescript
import { createClient } from '@fluxbase/client'

export const fluxbase = createClient({
  url: import.meta.env.VITE_FLUXBASE_URL,
  anonKey: import.meta.env.VITE_FLUXBASE_ANON_KEY
})
```

### Vue.js

**Before** (`src/supabase.js`):
```javascript
import { createClient } from '@supabase/supabase-js'

export const supabase = createClient(
  process.env.VUE_APP_SUPABASE_URL,
  process.env.VUE_APP_SUPABASE_ANON_KEY
)
```

**After**:
```javascript
import { createClient } from '@fluxbase/client'

export const fluxbase = createClient({
  url: process.env.VUE_APP_FLUXBASE_URL,
  anonKey: process.env.VUE_APP_FLUXBASE_ANON_KEY
})
```

### React Native / Expo

**Before**:
```typescript
import 'react-native-url-polyfill/auto'
import { createClient } from '@supabase/supabase-js'

const supabase = createClient(
  'YOUR_SUPABASE_URL',
  'YOUR_SUPABASE_ANON_KEY'
)
```

**After**:
```typescript
import 'react-native-url-polyfill/auto'
import { createClient } from '@fluxbase/client'

const fluxbase = createClient({
  url: 'YOUR_FLUXBASE_URL',
  anonKey: 'YOUR_FLUXBASE_ANON_KEY'
})
```

## 🐳 Deployment Options

### Docker

```dockerfile
# Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o fluxbase cmd/fluxbase/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/fluxbase .
COPY --from=builder /app/fluxbase.yaml .

EXPOSE 8080
CMD ["./fluxbase", "serve"]
```

```bash
# Build and run
docker build -t fluxbase:latest .
docker run -p 8080:8080 \
  -e DB_PASSWORD=yourpassword \
  -e JWT_SECRET=your-jwt-secret \
  fluxbase:latest
```

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: fluxbase
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  fluxbase:
    build: .
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_NAME: fluxbase
      DB_USER: postgres
      DB_PASSWORD: ${DB_PASSWORD}
      JWT_SECRET: ${JWT_SECRET}
    depends_on:
      - postgres

volumes:
  postgres_data:
```

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f fluxbase

# Stop services
docker-compose down
```

### Kubernetes

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fluxbase
spec:
  replicas: 3
  selector:
    matchLabels:
      app: fluxbase
  template:
    metadata:
      labels:
        app: fluxbase
    spec:
      containers:
      - name: fluxbase
        image: your-registry/fluxbase:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: postgres-service
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: fluxbase-secrets
              key: db-password
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: fluxbase-secrets
              key: jwt-secret
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: fluxbase-service
spec:
  selector:
    app: fluxbase
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

```bash
# Deploy
kubectl apply -f k8s/deployment.yaml

# Create secrets
kubectl create secret generic fluxbase-secrets \
  --from-literal=db-password=yourpassword \
  --from-literal=jwt-secret=your-jwt-secret
```

### AWS (ECS)

```bash
# Create ECR repository
aws ecr create-repository --repository-name fluxbase

# Build and push
docker build -t fluxbase:latest .
docker tag fluxbase:latest 123456789.dkr.ecr.us-east-1.amazonaws.com/fluxbase:latest
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/fluxbase:latest

# Create ECS task definition
aws ecs register-task-definition --cli-input-json file://task-definition.json

# Create service
aws ecs create-service \
  --cluster your-cluster \
  --service-name fluxbase \
  --task-definition fluxbase:1 \
  --desired-count 3 \
  --launch-type FARGATE
```

## ✅ Migration Checklist

### Pre-Migration

- [ ] Export Supabase database schema
- [ ] Export Supabase database data
- [ ] Document custom SQL functions
- [ ] List all Edge Functions
- [ ] Inventory storage buckets and files
- [ ] Note all environment variables
- [ ] Review RLS policies
- [ ] Check API usage patterns

### Migration

- [ ] Set up Fluxbase instance
- [ ] Import database schema
- [ ] Import database data
- [ ] Run Fluxbase migrations
- [ ] Generate API keys
- [ ] Update client code imports
- [ ] Update environment variables
- [ ] Migrate Edge Functions
- [ ] Test authentication flows
- [ ] Test database queries
- [ ] Test realtime subscriptions
- [ ] Test storage operations
- [ ] Migrate RLS policies
- [ ] Update CI/CD pipelines

### Post-Migration

- [ ] Run full test suite
- [ ] Performance testing
- [ ] Monitor error rates
- [ ] Verify data integrity
- [ ] Test backup/restore
- [ ] Document changes
- [ ] Train team on Fluxbase
- [ ] Decommission Supabase instance

## 🐛 Common Issues & Solutions

### Issue: Authentication Tokens Not Working

**Problem**: Existing JWT tokens from Supabase don't work with Fluxbase.

**Solution**: Tokens are not portable between systems. Users need to re-authenticate after migration.

```typescript
// Force re-authentication
await fluxbase.auth.signOut()
// Redirect to login page
```

### Issue: RLS Policies Not Working

**Problem**: Data not being filtered by RLS.

**Solution**: Update RLS policy functions:

```sql
-- Replace
auth.uid()

-- With
current_setting('app.user_id', true)::uuid
```

### Issue: Storage Files Not Accessible

**Problem**: File URLs return 404.

**Solution**: Configure storage correctly:

```yaml
# fluxbase.yaml
storage:
  provider: s3
  s3_bucket: your-bucket
  s3_region: us-east-1
  # Or use local storage
  # provider: local
  # local_path: /var/lib/fluxbase/storage
```

### Issue: Edge Functions Timing Out

**Problem**: Functions work in Supabase but timeout in Fluxbase.

**Solution**: Increase timeout in function configuration:

```typescript
await fluxbase.functions.update('function-name', {
  timeout: 30000  // 30 seconds
})
```

### Issue: Realtime Not Receiving Events

**Problem**: WebSocket connection established but no events.

**Solution**: Ensure PostgreSQL LISTEN/NOTIFY is configured:

```sql
-- Create trigger for realtime
CREATE OR REPLACE FUNCTION notify_changes()
RETURNS TRIGGER AS $$
BEGIN
  PERFORM pg_notify(
    'fluxbase_realtime',
    json_build_object(
      'table', TG_TABLE_NAME,
      'type', TG_OP,
      'record', row_to_json(NEW),
      'old_record', row_to_json(OLD)
    )::text
  );
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add trigger to table
CREATE TRIGGER posts_notify
AFTER INSERT OR UPDATE OR DELETE ON posts
FOR EACH ROW EXECUTE FUNCTION notify_changes();
```

## 💰 Cost Comparison

### Supabase Hosted

| Tier | Price/month | Includes |
|------|-------------|----------|
| Free | $0 | 500MB DB, 1GB storage, 50K users |
| Pro | $25 | 8GB DB, 100GB storage, 100K users |
| Team | $599 | Custom DB, custom storage |
| Enterprise | Custom | Dedicated instance |

### Fluxbase Self-Hosted

| Deployment | Cost/month | Includes |
|------------|------------|----------|
| Local | $0 | Unlimited |
| AWS EC2 (t3.medium) | ~$30 | 2 vCPU, 4GB RAM |
| AWS EC2 (t3.large) | ~$60 | 2 vCPU, 8GB RAM |
| AWS ECS Fargate | ~$50-200 | Scalable containers |

**Savings**: 50-90% for most workloads

## 📚 Additional Resources

- [Fluxbase Documentation](https://docs.fluxbase.io)
- [API Reference](https://docs.fluxbase.io/api)
- [Example Migrations](https://github.com/fluxbase/migration-examples)
- [Community Discord](https://discord.gg/fluxbase)
- [Migration Support](mailto:support@fluxbase.io)

## 🤝 Migration Support

Need help migrating? We offer:

- **Free Migration Consultation** (30 minutes)
- **Migration Assistance** (paid, for complex projects)
- **Custom Feature Development**
- **Enterprise Support Contracts**

Contact: support@fluxbase.io

---

**Last Updated**: 2025-10-30
**Status**: Complete ✅
**Migration Time**: 2-8 hours (typical project)
