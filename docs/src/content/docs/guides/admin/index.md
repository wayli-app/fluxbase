---
title: "Admin Dashboard"
---

The Fluxbase Admin Dashboard provides powerful tools for managing your Fluxbase instance, debugging issues, and providing customer support.

## Features

### 🗄️ Database Explorer

Browse, query, and edit your database tables directly from the web interface.

**Key features:**

- View all tables and schemas
- Filter, sort, and search data
- Inline editing with validation
- Batch operations
- Real-time updates

### 👥 User Management

Manage users, roles, and permissions.

**Capabilities:**

- List all users with enriched metadata
- Invite new users
- Update user roles
- Reset passwords
- Delete users

### 🎭 User Impersonation

View the database as different users to debug issues and test RLS policies.

**Learn more:** [User Impersonation Guide](./user-impersonation)

### 📊 Analytics & Monitoring

Monitor your Fluxbase instance health and usage.

**Metrics available:**

- Active sessions
- Database connection pool status
- Query performance
- Storage usage

## Getting Started

### Access the Admin Dashboard

1. Navigate to `http://localhost:8080/admin` (or your configured admin URL)
2. Log in with your admin credentials
3. If this is first setup, create your admin account

### Initial Setup

On first launch, you'll be prompted to create an admin account:

1. Enter your email address
2. Choose a strong password
3. Provide your name
4. Click "Create Admin Account"

Your admin credentials will be stored securely and you'll be logged in automatically.

## Navigation

The admin dashboard is organized into main sections:

- **📋 Tables** - Database explorer with user impersonation
- **👥 Users** - User management interface
- **⚙️ Settings** - Instance configuration
- **📊 Analytics** - Usage metrics and monitoring

## Security Considerations

### Admin Access Control

- Admin accounts are separate from regular user accounts
- Admin credentials are stored in the `dashboard_users` table
- Supports 2FA for enhanced security
- Session management with configurable timeouts

### Audit Logging

All administrative actions are logged for compliance:

- User impersonation sessions
- User management operations
- Configuration changes
- Login attempts

Query the audit logs:

```sql
-- View recent impersonation sessions
SELECT * FROM auth.impersonation_sessions
ORDER BY started_at DESC
LIMIT 50;

-- View admin login history
SELECT * FROM dashboard_auth.sessions
ORDER BY created_at DESC;
```

### Best Practices

1. **Use strong passwords** - Require complex passwords for admin accounts
2. **Enable 2FA** - Add an extra layer of security
3. **Limit admin access** - Only create admin accounts for trusted personnel
4. **Review audit logs** - Regularly check for suspicious activity
5. **Keep sessions short** - Configure appropriate session timeouts

## Common Tasks

### Managing Admin Users

List all admin users:

```bash
fluxbase admin users list
```

Invite a new admin user:

```bash
fluxbase admin users invite --email admin@example.com
fluxbase admin users invite --email admin@example.com --role dashboard_admin
```

View admin user details:

```bash
fluxbase admin users get <user-id>
```

Delete an admin user:

```bash
fluxbase admin users delete <user-id>
```

### Managing Admin Invitations

List pending invitations:

```bash
fluxbase admin invitations list
fluxbase admin invitations list --include-accepted --include-expired
```

Revoke a pending invitation:

```bash
fluxbase admin invitations revoke <token>
```

### Managing Admin Sessions

List active admin sessions:

```bash
fluxbase admin sessions list
```

Revoke a specific session:

```bash
fluxbase admin sessions revoke <session-id>
```

Revoke all sessions for a user:

```bash
fluxbase admin sessions revoke-all <user-id>
```

### Resetting Admin Password

```bash
fluxbase admin password-reset --email admin@example.com
```

### Managing Application Users

The `fluxbase users` command manages application end users (not admin users):

```bash
# List all app users
fluxbase users list

# Search users by email
fluxbase users list --search john

# View user details
fluxbase users get <user-id>

# Invite a new app user
fluxbase users invite --email user@example.com

# Delete an app user
fluxbase users delete <user-id>
```

## Guides

Explore detailed guides for specific admin features:

- [User Impersonation](./user-impersonation) - Debug issues by viewing data as different users
- _More guides coming soon..._

## Support

Need help with the admin dashboard?

- 📖 Check the [documentation](https://docs.fluxbase.io)
- 💬 Join our [Discord community](https://discord.gg/BXPRHkQzkA)
- 🐛 Report issues on [GitHub](https://github.com/fluxbase-eu/fluxbase/issues)
