---
sidebar_position: 1
---

# User Impersonation

User impersonation allows admins to view the database explorer as different user types to debug issues, test Row Level Security (RLS) policies, and provide customer support.

:::info Admin Dashboard Feature
This is an **admin dashboard feature** designed for debugging and support. It is **not available in the SDK** - you can only use impersonation through the Fluxbase admin UI or by directly calling the REST API with an admin token.
:::

## Overview

The impersonation feature enables admins to see exactly what data users can query based on their RLS policies. This is invaluable for:

- 🐛 **Debugging user-reported issues** - See data exactly as the user sees it
- 🔒 **Testing RLS policies** - Verify security rules work correctly
- 🎯 **Customer support** - Understand what users are experiencing
- ⚙️ **Testing different roles** - Validate permissions for anon and service roles

## Impersonation Modes

### 1. Specific User

Impersonate a real user by their ID to see data exactly as they would see it.

**Use cases:**

- Debugging user-reported data issues
- Verifying RLS policies for specific users
- Customer support investigations

**How it works:**

- Respects all RLS policies for that user
- Uses the user's actual permissions
- All queries execute in their security context

### 2. Anonymous (anon key)

See data as an unauthenticated visitor would see it.

**Use cases:**

- Testing public data access
- Verifying anon-level RLS policies
- Ensuring sensitive data is protected from public access

**How it works:**

- Generates a temporary JWT with `role: "anon"`
- No user account required
- Only public data should be accessible

### 3. Service Role

View data with service-level permissions.

**Use cases:**

- Administrative queries
- Testing privileged operations
- Bypassing RLS for data management

**How it works:**

- Generates a JWT with `role: "service"`
- May bypass RLS (depending on configuration)
- Elevated permissions for admin tasks

## Usage

### Starting Impersonation

1. Navigate to the **Tables** page in the admin dashboard
2. Click the **"Impersonate User"** button in the header
3. Select your impersonation type:
   - **Specific User** - Search and select a user by email
   - **Anonymous** - Impersonate as unauthenticated user
   - **Service Role** - Use service-level permissions
4. Enter a **reason** (required for audit trail)
   - Example: "Support ticket #1234"
   - Example: "Testing RLS policy for premium users"
5. Click **"Start Impersonation"**
6. The page reloads with impersonation active
7. An **orange warning banner** appears showing who you're impersonating

### While Impersonating

When impersonation is active:

- ⚠️ Orange banner displays at the top of the screen
- All table queries use the impersonated user's permissions
- Data grid shows only rows the impersonated user can access
- Edit/delete operations respect RLS policies
- You cannot start another impersonation (must stop first)

### Stopping Impersonation

1. Click **"Stop Impersonation"** in the warning banner
2. Session ends in the database
3. Impersonation tokens are cleared
4. Page reloads with admin context restored

## How It Works

### Two-Token Architecture

The system maintains two separate JWT tokens:

```typescript
// In localStorage
fluxbase_admin_access_token; // Your admin token
fluxbase_admin_user; // Your admin user info
fluxbase_impersonation_token; // Target user's token (when active)
fluxbase_impersonated_user; // Target user info
fluxbase_impersonation_session; // Session metadata
```

### Token Selection

All API requests automatically use the appropriate token:

```typescript
const getActiveToken = () => {
  const impToken = localStorage.getItem("fluxbase_impersonation_token");
  const adminToken = localStorage.getItem("fluxbase_admin_access_token");
  return impToken || adminToken; // Impersonation takes precedence
};
```

### RLS Integration

When querying data while impersonating:

1. **Frontend** sends request with impersonation token
2. **API Client** adds token to Authorization header
3. **Auth Middleware** extracts user ID and role from JWT
4. **RLS Middleware** sets PostgreSQL session variables:
   ```sql
   SET LOCAL app.user_id = '<impersonated_user_id>'
   SET LOCAL app.role = '<impersonated_role>'
   ```
5. **Database** enforces RLS policies based on these variables
6. **Results** show only data the impersonated user can access

## Security Features

### Audit Trail

Every impersonation session is logged in the `auth.impersonation_sessions` table:

| Field                | Description                                             |
| -------------------- | ------------------------------------------------------- |
| `admin_user_id`      | Who performed the impersonation                         |
| `target_user_id`     | Which user was impersonated (nullable for anon/service) |
| `impersonation_type` | Type: 'user', 'anon', or 'service'                      |
| `target_role`        | Role being impersonated                                 |
| `reason`             | Why the impersonation occurred                          |
| `started_at`         | When it started                                         |
| `ended_at`           | When it ended                                           |
| `ip_address`         | IP address of the admin                                 |
| `user_agent`         | Browser/client information                              |
| `is_active`          | Whether session is currently active                     |

### Access Control

- ✅ Only users with `role = 'admin'` can impersonate
- ✅ Self-impersonation is prevented (for user mode)
- ✅ Reason field is required (cannot be empty)
- ✅ Only one active session per admin at a time
- ✅ Previous session auto-ends when starting new one

### Visual Indicators

- 🟠 Bright orange warning banner (cannot be dismissed)
- 📝 Shows impersonation type and target user
- 🔒 Impersonate button disabled while already impersonating

## API Reference

### Endpoints

```http
# Start impersonating a specific user
POST /api/v1/auth/impersonate
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "target_user_id": "uuid",
  "reason": "Support ticket #1234"
}
```

```http
# Start impersonating anonymous user
POST /api/v1/auth/impersonate/anon
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "reason": "Testing public data access"
}
```

```http
# Start impersonating with service role
POST /api/v1/auth/impersonate/service
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "reason": "Administrative query"
}
```

```http
# Stop impersonation
DELETE /api/v1/auth/impersonate
Authorization: Bearer <admin_token>
```

```http
# Get active impersonation session
GET /api/v1/auth/impersonate
Authorization: Bearer <admin_token>
```

```http
# List impersonation sessions (audit trail)
GET /api/v1/auth/impersonate/sessions?limit=50&offset=0
Authorization: Bearer <admin_token>
```

### Response Format

```typescript
// Start impersonation response
{
  "session": {
    "id": "uuid",
    "admin_user_id": "uuid",
    "target_user_id": "uuid",
    "impersonation_type": "user",
    "target_role": "user",
    "reason": "Support ticket #1234",
    "started_at": "2024-01-15T10:30:00Z",
    "is_active": true,
    "ip_address": "192.168.1.1",
    "user_agent": "Mozilla/5.0..."
  },
  "target_user": {
    "id": "uuid",
    "email": "user@example.com",
    "role": "user"
  },
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_in": 900
}
```

## Troubleshooting

### Impersonation token not working

**Symptoms:** Queries return admin data instead of impersonated user's data

**Solutions:**

- Check localStorage for `fluxbase_impersonation_token`
- Verify token is being sent in Authorization header
- Check backend logs for JWT validation errors
- Ensure you reloaded the page after starting impersonation

### Data not filtered correctly

**Symptoms:** Seeing more/less data than expected

**Solutions:**

- Verify RLS policies exist on the tables
- Check PostgreSQL session variables: `SHOW app.user_id`
- Ensure RLS middleware is enabled in backend config
- Verify the RLS policies use `app.user_id` and `app.role` correctly

### Cannot stop impersonation

**Symptoms:** Stop button doesn't work or session persists

**Solutions:**

- Clear localStorage manually via browser DevTools
- Check for active session in `auth.impersonation_sessions` table
- Verify DELETE endpoint is accessible (check CORS/network)
- Logout and login again to reset session

### Users not appearing in search

**Symptoms:** User search returns no results

**Solutions:**

- Verify `exclude_admins` filter isn't hiding all users
- Check user has non-admin role in database
- Ensure user exists and is not deleted
- Try searching with full email address

## Best Practices

### 1. Always Provide a Clear Reason

```typescript
// ✅ Good
reason: "Support ticket #12345 - user reports missing data";

// ❌ Bad
reason: "testing";
```

### 2. Stop Impersonation When Done

Don't leave impersonation sessions running. Always click "Stop Impersonation" when finished to:

- Clear audit trail properly
- Avoid confusion
- Prevent accidental data modifications

### 3. Review Audit Logs Regularly

Query the impersonation sessions table to monitor usage:

```sql
SELECT
  admin_user_id,
  target_user_id,
  impersonation_type,
  reason,
  started_at,
  ended_at,
  EXTRACT(EPOCH FROM (ended_at - started_at)) as duration_seconds
FROM auth.impersonation_sessions
WHERE started_at > NOW() - INTERVAL '7 days'
ORDER BY started_at DESC;
```

### 4. Test RLS Policies with Multiple Scenarios

Use all three impersonation modes to thoroughly test:

- **User mode**: Test with regular users
- **Anon mode**: Verify public data access
- **Service mode**: Test admin operations

## Related Documentation

- [Authentication Guide](/docs/guides/authentication) - Learn about authentication and user roles
- [API Cookbook](/docs/api-cookbook) - Examples of common API patterns
- [Advanced Guides](/docs/advanced-guides) - Advanced Fluxbase features
