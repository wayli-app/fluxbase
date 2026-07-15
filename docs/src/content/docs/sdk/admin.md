---
title: "Admin SDK"
description: Programmatic access to Fluxbase admin features. User management, authentication, and system configuration for building admin dashboards and automation.
---

The Admin SDK provides programmatic access to Fluxbase instance management, including user management, authentication, and system configuration.

## Overview

The Admin SDK is designed for:

- Building admin dashboards
- Automating user management
- Managing Fluxbase instances programmatically
- Server-side administration tasks

**Key Features**:

- Admin authentication (setup, login, logout)
- User management (CRUD operations)
- Role management
- Password resets
- User invitations

## Installation

The Admin SDK is included in the main Fluxbase SDK:

```bash
npm install @nimbleflux/fluxbase-sdk
```

## Quick Start

### Initialize Admin Client

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

const client = createClient(
  "http://localhost:8080",
  "your-service-role-key", // Use service role key for admin operations
);

// Access admin module
const admin = client.admin;
```

### Initial Setup

Perform the initial admin setup (first-time only):

```typescript
// Check if setup is needed
const status = await client.admin.getSetupStatus();

if (status.needs_setup) {
  // Perform initial setup
  const response = await client.admin.setup({
    email: "admin@example.com",
    password: "SecurePassword123!",
    name: "Admin User",
  });

  console.log("Setup complete:", response.user.email);
  console.log("Access token:", response.access_token);

  // Token is automatically set in the client
}
```

### Admin Login

```typescript
const response = await client.admin.login({
  email: "admin@example.com",
  password: "password123",
});

console.log("Logged in as:", response.user.email);
console.log("Token expires in:", response.expires_in, "seconds");

// Token is automatically set for subsequent requests
```

---

## Admin Authentication

### Check Setup Status

Check if initial admin setup is required:

```typescript
const status = await client.admin.getSetupStatus();

if (status.needs_setup) {
  console.log("Initial setup required");
} else {
  console.log("Admin user already exists");
}
```

**Response**:

```typescript
{
  needs_setup: boolean;
  has_admin: boolean;
}
```

### Initial Setup

Create the first admin user (can only be called once):

```typescript
const response = await client.admin.setup({
  email: "admin@example.com",
  password: "SecurePassword123!",
  name: "Admin User",
});

console.log("Admin created:", response.user);
```

**Requirements**:

- Password must be at least 12 characters
- Valid email address
- Can only be called when `needs_setup` is `true`

**Response**:

```typescript
{
  user: {
    id: string;
    email: string;
    name: string;
    role: "instance_admin" | "tenant_admin";
    email_verified: boolean;
    created_at: string;
    updated_at: string;
  }
  access_token: string;
  refresh_token: string;
  expires_in: number;
}
```

### Login

Authenticate as an admin user:

```typescript
const response = await client.admin.login({
  email: "admin@example.com",
  password: "password123",
});

// Access token is automatically set
console.log("Access token:", response.access_token);

// Store refresh token for later
localStorage.setItem("admin_refresh_token", response.refresh_token);
```

### Refresh Token

Refresh an expired access token:

```typescript
const refreshToken = localStorage.getItem("admin_refresh_token");

const response = await client.admin.refreshToken({
  refresh_token: refreshToken,
});

// New tokens
console.log("New access token:", response.access_token);
localStorage.setItem("admin_refresh_token", response.refresh_token);
```

### Logout

Invalidate the current admin session:

```typescript
await client.admin.logout();

// Clear stored tokens
localStorage.removeItem("admin_access_token");
localStorage.removeItem("admin_refresh_token");

console.log("Logged out successfully");
```

### Get Current Admin

Get the currently authenticated admin user:

```typescript
const { user } = await client.admin.me();

console.log("Current admin:", user.email);
console.log("Role:", user.role);
```

**Response**:

```typescript
{
  user: {
    id: string;
    email: string;
    role: string;
  }
}
```

---

## User Management

### List Users

List all users with optional filters:

```typescript
// List all users
const { users, total } = await client.admin.listUsers();
console.log(`Total users: ${total}`);

// List with filters
const result = await client.admin.listUsers({
  exclude_admins: true, // Exclude admin users
  search: "john", // Search by email
  limit: 50, // Limit results
  type: "app", // User type: 'app' or 'dashboard'
});

result.users.forEach((user) => {
  console.log(
    `${user.email} - ${user.role} - Last login: ${user.last_login_at}`,
  );
});
```

**Options**:

```typescript
interface ListUsersOptions {
  exclude_admins?: boolean; // Exclude admin users
  search?: string; // Search by email
  limit?: number; // Maximum results
  type?: "app" | "dashboard"; // User type
}
```

**Response**:

```typescript
{
  users: Array<{
    id: string;
    email: string;
    role?: string;
    created_at: string;
    updated_at?: string;
    email_verified?: boolean;
    last_login_at?: string;
    session_count?: number;
    is_anonymous?: boolean;
    metadata?: Record<string, any>;
  }>;
  total: number;
}
```

### Invite User

Create a new user and send an invitation email:

```typescript
const response = await client.admin.inviteUser({
  email: "newuser@example.com",
  role: "user",
  send_email: true,
});

console.log("User invited:", response.user.email);
console.log("Invitation link:", response.invitation_link);
```

**Request**:

```typescript
interface InviteUserRequest {
  email: string;
  role?: string;
  send_email?: boolean;
}
```

**Response**:

```typescript
{
  user: EnrichedUser
  invitation_link?: string
  message: string
}
```

### Delete User

Permanently delete a user:

```typescript
const response = await client.admin.deleteUser("user-uuid");
console.log(response.message); // "User deleted successfully"
```

**Warning**: This permanently deletes the user and all associated data.

### Update User Role

Change a user's role:

```typescript
const user = await client.admin.updateUserRole("user-uuid", "admin");

console.log("User role updated:", user.role);
```

**Common Roles**:

- `user` - Regular user
- `admin` - Admin user
- `instance_admin` - Instance-level administrator (full platform access)
- `tenant_admin` - Tenant-level administrator (limited to assigned tenants)
- Custom roles as defined in your application

### Reset User Password

Generate a new password for a user:

```typescript
const response = await client.admin.resetUserPassword("user-uuid");
console.log(response.message); // "Password reset email sent"
```

This sends a password reset email to the user or returns the new password.

---

## Tenant Management

Tenant management requires `instance_admin` role.

### List Tenants

```typescript
const { tenants, total } = await client.tenant.list();

tenants.forEach((tenant) => {
  console.log(
    `${tenant.name} (${tenant.slug}) - Default: ${tenant.is_default}`,
  );
});
```

**Response**:

```typescript
{
  tenants: Array<{
    id: string;
    slug: string;
    name: string;
    is_default: boolean;
    metadata: Record<string, any> | null;
    created_at: string;
    updated_at: string;
    deleted_at: string | null;
  }>;
  total: number;
}
```

### Create Tenant

```typescript
const tenant = await client.tenant.create({
  slug: "acme-corp",
  name: "Acme Corporation",
  metadata: {
    plan: "enterprise",
    billing_email: "billing@acme.com",
  },
});

console.log("Created tenant:", tenant.id);
```

**Request**:

```typescript
interface CreateTenantRequest {
  slug: string;
  name: string;
  is_default?: boolean;
  metadata?: Record<string, any>;
}
```

### Update Tenant

```typescript
const tenant = await client.tenant.update("tenant-uuid", {
  name: "Acme Corp Inc.",
  metadata: { plan: "pro" },
});
```

### Delete Tenant

```typescript
// Soft delete (sets deleted_at)
await client.tenant.delete("tenant-uuid");
```

---

## Service Key Management

Service key management requires `instance_admin` or `tenant_admin` role.

### List Service Keys

```typescript
// List all keys (instance_admin only)
const { data: keys, error } = await client.admin.serviceKeys.list();
```

**Options**:

```typescript
interface ListServiceKeysOptions {
  tenant_id?: string;
  key_type?: "anon" | "service";
  is_active?: boolean;
  user_id?: string;
  limit?: number;
  offset?: number;
}
```

### Create Service Key

```typescript
const { data: key, error } = await client.admin.serviceKeys.create({
  name: "Production API Key",
  key_type: "service",
  scopes: ["*"],
});

console.log("Key prefix:", key?.key_prefix);
console.log("Full key (store securely):", key?.key);
```

**Request**:

```typescript
interface CreateServiceKeyRequest {
  name: string;
  description?: string;
  key_type: "anon" | "service";
  scopes?: string[];
  allowed_namespaces?: string[];
  rate_limit_per_minute?: number;
  rate_limit_per_hour?: number;
  expires_at?: string;
}
```

**Key Types**:

- `anon` - Anonymous access, no tenant context
- `service` - Backend service key (bypasses RLS)

### Rotate Service Key

```typescript
// Rotate key - creates a replacement and deprecates the old one
const { data: newKey, error } = await client.admin.serviceKeys.rotate("old-key-id");

console.log("New key:", newKey?.key);
console.log("Old key deprecated at:", newKey?.deprecated_at);
```

### Revoke Service Key

```typescript
await client.admin.serviceKeys.revoke("key-id", { reason: "Security incident" });
```

### Deprecate Service Key

```typescript
// Mark for deprecation with grace period
await client.admin.serviceKeys.deprecate("key-id", {
  grace_period_hours: 48,
});
```

---

## Complete Examples

### Admin Dashboard

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

// Initialize client
const client = createClient("http://localhost:8080", "your-service-role-key");

// Admin login
async function adminLogin(email: string, password: string) {
  try {
    const response = await client.admin.login({ email, password });

    // Store tokens
    localStorage.setItem("admin_access_token", response.access_token);
    localStorage.setItem("admin_refresh_token", response.refresh_token);

    return response.user;
  } catch (error) {
    console.error("Login failed:", error);
    throw error;
  }
}

// Load users with pagination
async function loadUsers(page: number = 1, pageSize: number = 50) {
  const { users, total } = await client.admin.listUsers({
    exclude_admins: false,
    limit: pageSize,
    type: "app",
  });

  return {
    users,
    total,
    pages: Math.ceil(total / pageSize),
    currentPage: page,
  };
}

// Search users
async function searchUsers(query: string) {
  const { users } = await client.admin.listUsers({
    search: query,
    limit: 20,
  });

  return users;
}

// Create new user
async function createUser(email: string, role: string = "user") {
  const response = await client.admin.inviteUser({
    email,
    role,
    send_email: true,
  });

  console.log("Invitation sent to:", email);
  return response.user;
}

// Make user admin
async function promoteToAdmin(userId: string) {
  const user = await client.admin.updateUserRole(userId, "admin");
  console.log(`${user.email} is now an admin`);
  return user;
}

// Remove user
async function removeUser(userId: string) {
  if (!confirm("Are you sure you want to delete this user?")) {
    return;
  }

  await client.admin.deleteUser(userId);
  console.log("User deleted");
}

// Usage
async function main() {
  // Login
  const admin = await adminLogin("admin@example.com", "password123");
  console.log("Logged in as:", admin.email);

  // Load users
  const { users, total, pages } = await loadUsers(1, 50);
  console.log(`Showing ${users.length} of ${total} users (${pages} pages)`);

  // Search
  const results = await searchUsers("john");
  console.log(`Found ${results.length} users matching "john"`);

  // Create user
  const newUser = await createUser("newuser@example.com", "user");
  console.log("Created user:", newUser.id);

  // Promote to admin
  await promoteToAdmin(newUser.id);

  // Cleanup
  // await removeUser(newUser.id)
}

main().catch(console.error);
```

### React Admin Hook

```typescript
import { useState, useEffect } from 'react'
import { createClient } from '@nimbleflux/fluxbase-sdk'

const client = createClient(
  'http://localhost:8080',
  'your-service-role-key'
)

export function useAdmin() {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [admin, setAdmin] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    checkAuth()
  }, [])

  async function checkAuth() {
    const token = localStorage.getItem('admin_access_token')

    if (!token) {
      setLoading(false)
      return
    }

    try {
      client.admin.setToken(token)
      const { user } = await client.admin.me()
      setAdmin(user)
      setIsAuthenticated(true)
    } catch (error) {
      // Token invalid or expired
      localStorage.removeItem('admin_access_token')
      localStorage.removeItem('admin_refresh_token')
    } finally {
      setLoading(false)
    }
  }

  async function login(email: string, password: string) {
    const response = await client.admin.login({ email, password })

    localStorage.setItem('admin_access_token', response.access_token)
    localStorage.setItem('admin_refresh_token', response.refresh_token)

    setAdmin(response.user)
    setIsAuthenticated(true)

    return response
  }

  async function logout() {
    await client.admin.logout()

    localStorage.removeItem('admin_access_token')
    localStorage.removeItem('admin_refresh_token')

    setAdmin(null)
    setIsAuthenticated(false)
  }

  return {
    isAuthenticated,
    admin,
    loading,
    login,
    logout
  }
}

// Usage in component
function AdminDashboard() {
  const { isAuthenticated, admin, loading, login, logout } = useAdmin()

  if (loading) return <div>Loading...</div>

  if (!isAuthenticated) {
    return <LoginForm onLogin={login} />
  }

  return (
    <div>
      <h1>Welcome, {admin.email}</h1>
      <button onClick={logout}>Logout</button>
      <UserManagement />
    </div>
  )
}
```

### Bulk Operations

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

const client = createClient("http://localhost:8080", "your-service-role-key");

// Bulk invite users
async function bulkInviteUsers(emails: string[], role: string = "user") {
  const results = {
    success: [],
    failed: [],
  };

  for (const email of emails) {
    try {
      const response = await client.admin.inviteUser({
        email,
        role,
        send_email: true,
      });

      results.success.push({
        email,
        userId: response.user.id,
      });
    } catch (error) {
      results.failed.push({
        email,
        error: error.message,
      });
    }
  }

  return results;
}

// Bulk delete inactive users
async function deleteInactiveUsers(daysSinceLastLogin: number = 90) {
  const { users } = await client.admin.listUsers({ type: "app" });

  const cutoffDate = new Date();
  cutoffDate.setDate(cutoffDate.getDate() - daysSinceLastLogin);

  const inactiveUsers = users.filter((user) => {
    if (!user.last_login_at) return true;
    const lastLogin = new Date(user.last_login_at);
    return lastLogin < cutoffDate;
  });

  console.log(`Found ${inactiveUsers.length} inactive users`);

  for (const user of inactiveUsers) {
    try {
      await client.admin.deleteUser(user.id);
      console.log(`Deleted: ${user.email}`);
    } catch (error) {
      console.error(`Failed to delete ${user.email}:`, error);
    }
  }

  return inactiveUsers.length;
}

// Usage
const emails = ["user1@example.com", "user2@example.com", "user3@example.com"];

const results = await bulkInviteUsers(emails, "user");
console.log(`Invited: ${results.success.length}`);
console.log(`Failed: ${results.failed.length}`);

// Delete inactive users (with confirmation)
if (confirm("Delete all users inactive for 90+ days?")) {
  const deleted = await deleteInactiveUsers(90);
  console.log(`Deleted ${deleted} inactive users`);
}
```

---

## Error Handling

```typescript
import { createClient } from "@nimbleflux/fluxbase-sdk";

const client = createClient("http://localhost:8080", "your-service-role-key");

try {
  await client.admin.login({
    email: "admin@example.com",
    password: "wrong-password",
  });
} catch (error) {
  if (error.status === 401) {
    console.error("Invalid credentials");
  } else if (error.status === 429) {
    console.error("Too many login attempts. Please try again later.");
  } else {
    console.error("Login failed:", error.message);
  }
}

// Retry logic with exponential backoff
async function loginWithRetry(
  email: string,
  password: string,
  maxRetries: number = 3,
) {
  let lastError;

  for (let i = 0; i < maxRetries; i++) {
    try {
      return await client.admin.login({ email, password });
    } catch (error) {
      lastError = error;

      if (error.status === 401) {
        // Don't retry on invalid credentials
        throw error;
      }

      if (i < maxRetries - 1) {
        // Exponential backoff: 1s, 2s, 4s
        const delay = Math.pow(2, i) * 1000;
        await new Promise((resolve) => setTimeout(resolve, delay));
      }
    }
  }

  throw lastError;
}
```

---

## Security Best Practices

### 1. Secure Token Storage

```typescript
// DO NOT store in localStorage for production (XSS vulnerable)
// Use secure, httpOnly cookies instead

// Bad (development only)
localStorage.setItem("admin_token", token);

// Good (production)
// Let your backend set httpOnly cookies
// The SDK will send them automatically
```

### 2. Token Refresh

```typescript
// Implement automatic token refresh
async function refreshTokenIfNeeded() {
  const expiresAt = localStorage.getItem("admin_token_expires_at");

  if (!expiresAt || Date.now() >= parseInt(expiresAt)) {
    const refreshToken = localStorage.getItem("admin_refresh_token");

    const response = await client.admin.refreshToken({
      refresh_token: refreshToken,
    });

    localStorage.setItem("admin_access_token", response.access_token);
    localStorage.setItem("admin_refresh_token", response.refresh_token);
    localStorage.setItem(
      "admin_token_expires_at",
      String(Date.now() + response.expires_in * 1000),
    );

    client.admin.setToken(response.access_token);
  }
}

// Call before admin operations
await refreshTokenIfNeeded();
await client.admin.listUsers();
```

### 3. Role Verification

```typescript
async function requireAdminRole() {
  const { user } = await client.admin.me();

  if (
    user.role !== "admin" &&
    user.role !== "instance_admin" &&
    user.role !== "tenant_admin"
  ) {
    throw new Error("Admin role required");
  }

  return user;
}

// Use in operations
await requireAdminRole();
await client.admin.deleteUser("user-id");
```

### 4. Audit Logging

```typescript
async function deleteUserWithAudit(userId: string, reason: string) {
  const admin = await client.admin.me();

  // Log the action
  console.log(
    `[AUDIT] ${admin.user.email} deleted user ${userId}. Reason: ${reason}`,
  );

  // Or send to audit service
  await fetch("/api/audit", {
    method: "POST",
    body: JSON.stringify({
      action: "DELETE_USER",
      admin_id: admin.user.id,
      target_user_id: userId,
      reason,
      timestamp: new Date().toISOString(),
    }),
  });

  // Perform deletion
  await client.admin.deleteUser(userId);
}
```

---

## TypeScript Types

```typescript
import type {
  // Admin Auth
  AdminSetupStatusResponse,
  AdminSetupRequest,
  AdminUser,
  AdminAuthResponse,
  AdminLoginRequest,
  AdminRefreshRequest,
  AdminRefreshResponse,
  AdminMeResponse,

  // User Management
  EnrichedUser,
  ListUsersResponse,
  ListUsersOptions,
  InviteUserRequest,
  InviteUserResponse,
  UpdateUserRoleRequest,
  ResetUserPasswordResponse,
  DeleteUserResponse,
} from "@nimbleflux/fluxbase-sdk";
```

---

## Next Steps

- [Authentication Guide](/guides/authentication) - User authentication methods
- [SDK Reference](/api/sdk/readme/) - Query and manipulate data
- [Storage Guide](/guides/storage) - File upload and management
- [Realtime Guide](/guides/realtime) - WebSocket subscriptions
