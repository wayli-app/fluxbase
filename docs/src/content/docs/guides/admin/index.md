---
title: "Admin Dashboard"
description: Manage your Fluxbase instance with the built-in admin dashboard. Database explorer, user management, storage browser, and real-time monitoring.
---

The Fluxbase Admin Dashboard provides tools for managing your instance, debugging issues, and monitoring system health.

## Getting Started

1. Navigate to `http://localhost:8080/admin`
2. Log in with admin credentials (created on first launch)

## Navigation

### Data Management

| Section        | Description                                                                      |
| -------------- | -------------------------------------------------------------------------------- |
| **Tables**     | Browse, edit, and query database tables with inline editing and batch operations |
| **Schema**     | Visual ERD showing table relationships, columns, and constraints                 |
| **SQL Editor** | Execute SQL and GraphQL queries with syntax highlighting and history             |

### Authentication & Access

| Section            | Description                                                       |
| ------------------ | ----------------------------------------------------------------- |
| **Users**          | Manage application users - invite, update roles, reset passwords  |
| **Client Keys**    | Generate API keys for client applications with scoped permissions |
| **Service Keys**   | Manage server-to-server keys for CLI tools and migrations         |
| **Authentication** | Configure OAuth providers, SAML SSO, and auth settings            |

### Multi-Tenancy

| Section               | Description                                                          |
| --------------------- | -------------------------------------------------------------------- |
| **Tenants**           | Create and manage tenants, assign members, configure per-tenant auth |
| **Instance Settings** | Platform-wide configuration and tenant override permissions          |

See: [Tenant Management](./tenants) | [Multi-Tenancy Guide](../multi-tenancy)

### Compute

| Section             | Description                                              | Guide                               |
| ------------------- | -------------------------------------------------------- | ----------------------------------- |
| **Edge Functions**  | Deploy serverless TypeScript/JavaScript functions        | [Edge Functions](../edge-functions) |
| **Background Jobs** | Manage async tasks with scheduling and progress tracking | [Jobs](../jobs)                     |
| **RPC**             | Execute database procedures via API                      | [RPC](../rpc)                       |

### Storage

| Section            | Description                                                                | Guide                 |
| ------------------ | -------------------------------------------------------------------------- | --------------------- |
| **Storage**        | File browser for buckets and objects with upload, preview, and signed URLs | [Storage](../storage) |
| **Storage Config** | Configure storage providers (local/S3) and buckets                         |                       |

### AI Features

| Section             | Description                                       | Guide                                   |
| ------------------- | ------------------------------------------------- | --------------------------------------- |
| **Knowledge Bases** | RAG-powered document stores for AI applications   | [Knowledge Bases](../knowledge-bases)   |
| **Chatbots**        | AI assistants that query your database            | [AI Chatbots](../ai-chatbots)           |
| **MCP Tools**       | Custom tools for AI assistant integration         | [Custom MCP Tools](../mcp/custom-mcp-tools) |
| **AI Providers**    | Configure LLM providers (OpenAI, Anthropic, etc.) |                                         |

### Integrations

| Section            | Description                                     | Guide                               |
| ------------------ | ----------------------------------------------- | ----------------------------------- |
| **Webhooks**       | Event-driven notifications for database changes | [Webhooks](../webhooks)             |
| **Realtime**       | Monitor WebSocket connections and subscriptions | [Realtime](../realtime)             |
| **Email Settings** | Configure SMTP, SendGrid, Mailgun, or SES       | [Email Services](../email-services) |

### Security

| Section               | Description                                             | Guide                                       |
| --------------------- | ------------------------------------------------------- | ------------------------------------------- |
| **Policies**          | Row Level Security management and vulnerability scanner | [Row Level Security](../row-level-security) |
| **Security Settings** | CAPTCHA configuration for auth endpoints                | [CAPTCHA](../captcha)                       |
| **Secrets**           | Manage secrets for functions and jobs                   | [Secrets Management](../secrets-management) |

### Monitoring

| Section        | Description                                      | Guide                                     |
| -------------- | ------------------------------------------------ | ----------------------------------------- |
| **Logs**       | Real-time application log viewer                 | [Logging](../logging)                     |
| **Monitoring** | System health metrics and connection pool status | [Monitoring](../monitoring-observability) |
| **Errors**     | View and manage application errors               |                                           |

### Advanced

| Section        | Description                          |
| -------------- | ------------------------------------ |
| **Extensions** | PostgreSQL extension management      |
| **Features**   | Enable/disable platform features     |
| **Quotas**     | User resource quotas for AI features |

## User Impersonation

View the database as different users to debug RLS policies and support users.

**Enable impersonation:** Click the user icon in the Tables header and select a user. All queries will execute with that user's permissions.

See: [User Impersonation Guide](./user-impersonation)

## Security

### Admin Access

- Admin accounts are separate from application users
- Stored in `dashboard_users` table
- Supports 2FA and configurable session timeouts

### Audit Logging

All admin actions are logged:

```sql
SELECT * FROM auth.impersonation_sessions ORDER BY started_at DESC LIMIT 50;
SELECT * FROM dashboard_auth.sessions ORDER BY created_at DESC;
```

### Best Practices

1. Use strong passwords and enable 2FA
2. Limit admin access to trusted personnel
3. Review audit logs regularly
4. Configure appropriate session timeouts

## CLI Reference

### Admin Users

```bash
fluxbase admin users list
fluxbase admin users invite --email admin@example.com --role instance_admin
fluxbase admin users delete <user-id>
```

### Admin Sessions

```bash
fluxbase admin sessions list
fluxbase admin sessions revoke <session-id>
fluxbase admin sessions revoke-all <user-id>
```

### Application Users

```bash
fluxbase users list
fluxbase users list --search john
fluxbase users get <user-id>
fluxbase users invite --email user@example.com
fluxbase users delete <user-id>
```
