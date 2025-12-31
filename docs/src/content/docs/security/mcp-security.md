---
title: "MCP Security"
description: "Security best practices for the MCP server"
---

This guide covers security considerations and best practices for the MCP server.

## Authentication

### Client Keys (Recommended)

Create client keys with minimal required scopes:

```bash
# Read-only access for AI assistants
fluxbase clientkeys create --name "AI Reader" \
  --scopes "read:tables,read:storage"

# Limited write access
fluxbase clientkeys create --name "AI Writer" \
  --scopes "read:tables,write:tables"
```

### Service Keys (Admin Only)

Service keys bypass Row Level Security and have full access. Use only for:
- Administrative operations
- Trusted backend services
- Development/debugging

**Never expose service keys to client applications or AI assistants in untrusted environments.**

## Authorization Layers

### 1. Scope-Based Access

Each MCP tool requires specific scopes:

```yaml
# Example: Restrict to read-only operations
mcp:
  allowed_tools:
    - query_table
    - list_objects
    - download_object
```

### 2. Row Level Security (RLS)

All database operations respect PostgreSQL RLS policies:

```sql
-- Users can only see their own data
CREATE POLICY user_isolation ON public.orders
  FOR SELECT
  USING (user_id = current_setting('request.jwt.claims')::json->>'sub');
```

MCP queries execute within the user's security context.

### 3. Tool Whitelisting

Restrict available tools in production:

```yaml
mcp:
  allowed_tools:
    - query_table      # Allow reads
    - search_vectors   # Allow vector search
    # - insert_record  # Block writes
    # - delete_record  # Block deletes
```

### 4. Resource Whitelisting

Restrict available resources:

```yaml
mcp:
  allowed_resources:
    - "fluxbase://schema/tables"
    - "fluxbase://functions"
    # Exclude sensitive resources
```

## SQL Injection Prevention

The MCP server prevents SQL injection through:

1. **Identifier Validation** - Table and column names validated against `^[a-zA-Z_][a-zA-Z0-9_]*$`
2. **Parameterized Queries** - All values use prepared statements
3. **Schema Cache** - Table/column existence verified before query execution
4. **Quoting** - Identifiers properly quoted and escaped

## Safe Defaults

### Mandatory Filters

`update_record` and `delete_record` require a filter parameter:

```json
// This will fail - no filter provided
{
  "name": "delete_record",
  "arguments": {"table": "users"}
}

// This works - filter specified
{
  "name": "delete_record",
  "arguments": {
    "table": "users",
    "filter": {"id": "eq.123"}
  }
}
```

### Query Limits

- Maximum 1000 rows per query
- Maximum 10MB file download
- Maximum 100 vector search results

### Rate Limiting

Configure per-client rate limits:

```yaml
mcp:
  rate_limit_per_min: 100  # 100 requests per minute per client
```

## Audit Logging

MCP operations are logged with:

- Client key ID/name
- User ID and role
- Tool/resource accessed
- Timestamp

Enable debug logging for detailed traces:

```yaml
logging:
  level: debug
```

## Best Practices

### 1. Principle of Least Privilege

Create dedicated client keys with minimal scopes:

```bash
# For a chatbot that only needs to search knowledge base
fluxbase clientkeys create --name "Support Bot" \
  --scopes "read:vectors"
```

### 2. Separate Keys per Application

Don't share client keys between applications:

```bash
fluxbase clientkeys create --name "Mobile App - Production"
fluxbase clientkeys create --name "Web App - Production"
fluxbase clientkeys create --name "AI Assistant - Production"
```

### 3. Rotate Keys Regularly

Rotate client keys periodically:

```bash
# Create new key
fluxbase clientkeys create --name "AI Assistant - 2024-Q2"

# Update application configuration
# Then revoke old key
fluxbase clientkeys delete "old-key-id"
```

### 4. Monitor Usage

Review MCP access patterns:

```sql
SELECT
  client_key_name,
  COUNT(*) as requests,
  DATE_TRUNC('hour', created_at) as hour
FROM auth.audit_log
WHERE path LIKE '/mcp%'
GROUP BY 1, 3
ORDER BY 3 DESC;
```

### 5. Restrict in Production

Disable unnecessary features in production:

```yaml
mcp:
  enabled: true
  allowed_tools:
    - query_table
    - search_vectors
  allowed_resources:
    - "fluxbase://schema/tables"
```

## Common Attack Vectors

### Prevented Attacks

| Attack | Prevention |
|--------|------------|
| SQL Injection | Parameterized queries, identifier validation |
| Unauthorized Access | Scope-based access control |
| Data Leakage | Row Level Security |
| Bulk Deletion | Mandatory filters |
| Resource Exhaustion | Query limits, rate limiting |

### Configuration Mistakes to Avoid

1. **Don't use service keys for AI assistants** - Use scoped client keys instead
2. **Don't disable RLS** - Always use RLS in production
3. **Don't allow all tools** - Whitelist only required tools
4. **Don't expose internal schemas** - Non-admin users can't see system tables
5. **Don't skip rate limiting** - Configure appropriate limits

## Monitoring and Alerts

Set up alerts for:

- Unusual query patterns
- Failed authentication attempts
- Rate limit violations
- Access to sensitive tables

```sql
-- Example: Alert on failed auth
SELECT COUNT(*)
FROM auth.audit_log
WHERE path LIKE '/mcp%'
  AND status_code = 401
  AND created_at > NOW() - INTERVAL '1 hour';
```
