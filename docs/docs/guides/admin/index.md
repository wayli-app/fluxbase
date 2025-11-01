---
sidebar_position: 0
---

# Admin Dashboard

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

1. Navigate to `http://localhost:3001` (or your configured admin URL)
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

## Configuration

Configure the admin dashboard via environment variables:

```bash
# Admin dashboard port
FLUXBASE_ADMIN_PORT=3001

# Session configuration
FLUXBASE_ADMIN_SESSION_SECRET=your-secret-key
FLUXBASE_ADMIN_SESSION_TIMEOUT=24h

# Database connection
DATABASE_URL=postgresql://user:pass@localhost:5432/fluxbase
```

## Common Tasks

### Creating Admin Users

Use the initial setup flow or invite via command line:

```bash
# Create additional admin user
fluxbase admin create-user \
  --email admin@example.com \
  --name "Admin User"
```

### Resetting Admin Password

```bash
# Reset admin password
fluxbase admin reset-password --email admin@example.com
```

### Backing Up Admin Data

Admin data is stored in your PostgreSQL database:

```bash
# Backup admin tables
pg_dump -t 'dashboard_auth.*' \
  $DATABASE_URL > admin_backup.sql
```

## Guides

Explore detailed guides for specific admin features:

- [User Impersonation](./user-impersonation) - Debug issues by viewing data as different users
- *More guides coming soon...*

## Support

Need help with the admin dashboard?

- 📖 Check the [documentation](https://docs.fluxbase.io)
- 💬 Join our [Discord community](https://discord.gg/fluxbase)
- 🐛 Report issues on [GitHub](https://github.com/wayli-app/fluxbase/issues)
