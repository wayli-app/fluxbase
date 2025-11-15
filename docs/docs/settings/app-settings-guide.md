# Application Settings Guide

This guide explains how to use Fluxbase's unified application settings system with the `app.settings` table.

## Table of Contents

1. [Overview](#overview)
2. [Database Schema](#database-schema)
3. [SDK Usage](#sdk-usage)
4. [RLS Policies](#rls-policies)
5. [Backend Usage](#backend-usage)
6. [Migration Guide](#migration-guide)

## Overview

Fluxbase provides a unified `app.settings` table in the `app` schema for storing all application-level configuration. This replaces the previous separate tables (`dashboard.auth_settings`, `dashboard.system_settings`, `dashboard.custom_settings`).

### Key Features

- **Unified Storage**: One table for all settings with flexible JSONB values
- **Categorization**: Settings organized by category (auth, system, storage, functions, realtime, custom)
- **Access Control**: Built-in RLS policies with `is_public` and `is_secret` flags
- **Flexible Values**: Store any JSON-serializable data
- **Role-Based Editing**: Control who can edit settings with `editable_by` array

## Database Schema

### Table Structure

```sql
CREATE TABLE app.settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT UNIQUE NOT NULL,
    value JSONB NOT NULL,
    value_type TEXT NOT NULL DEFAULT 'string'
        CHECK (value_type IN ('string', 'number', 'boolean', 'json', 'array')),
    category TEXT NOT NULL DEFAULT 'custom'
        CHECK (category IN ('auth', 'system', 'storage', 'functions', 'realtime', 'custom')),
    description TEXT,
    is_public BOOLEAN DEFAULT false,
    is_secret BOOLEAN DEFAULT false,
    editable_by TEXT[] NOT NULL DEFAULT ARRAY['dashboard_admin']::TEXT[],
    metadata JSONB DEFAULT '{}'::JSONB,
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Default RLS Policies

```sql
-- Service role has full access (bypasses RLS anyway)
CREATE POLICY "Service role has full access to app settings"
    ON app.settings FOR ALL TO service_role
    USING (true) WITH CHECK (true);

-- Anon/authenticated can read public, non-secret settings
CREATE POLICY "Public settings are readable by anyone"
    ON app.settings FOR SELECT TO anon, authenticated
    USING (is_public = true AND is_secret = false);

-- Authenticated users can read all non-secret settings
CREATE POLICY "Authenticated users can read non-secret settings"
    ON app.settings FOR SELECT TO authenticated
    USING (is_secret = false);
```

## SDK Usage

### TypeScript/JavaScript SDK

#### Framework Settings (Structured)

```typescript
import { FluxbaseClient } from "@fluxbase/sdk";

const client = new FluxbaseClient("https://api.myapp.com", "admin_key");

// Get all app settings (structured)
const settings = await client.admin.settings.app.get();
console.log(settings.authentication.enable_signup);
console.log(settings.features.enable_realtime);

// Update framework settings
await client.admin.settings.app.update({
  authentication: {
    enable_signup: true,
    password_min_length: 12,
    require_email_verification: true,
  },
  features: {
    enable_realtime: true,
    enable_storage: true,
  },
});

// Use convenience methods
await client.admin.settings.app.enableSignup();
await client.admin.settings.app.setPasswordMinLength(12);
await client.admin.settings.app.setFeature("realtime", true);

// Configure email providers
await client.admin.settings.app.configureSMTP({
  host: "smtp.gmail.com",
  port: 587,
  username: "noreply@myapp.com",
  password: "app-password",
  use_tls: true,
});
```

#### Custom Settings (Key-Value)

```typescript
// Set custom settings
await client.admin.settings.app.setSetting(
  "billing.tiers",
  {
    free: 1000,
    pro: 10000,
    enterprise: 100000,
  },
  {
    description: "API quotas per billing tier",
    is_public: false,
    is_secret: false,
  },
);

// Get single custom setting value
const tiers = await client.admin.settings.app.getSetting("billing.tiers");
console.log(tiers); // { free: 1000, pro: 10000, enterprise: 100000 }

// Get multiple custom settings
const values = await client.admin.settings.app.getSettings([
  "billing.tiers",
  "features.beta_enabled",
  "features.dark_mode",
]);
console.log(values);
// {
//   'billing.tiers': { free: 1000, pro: 10000, enterprise: 100000 },
//   'features.beta_enabled': { enabled: true },
//   'features.dark_mode': { enabled: false }
// }

// List all custom settings
const allSettings = await client.admin.settings.app.listSettings();
allSettings.forEach((s) => console.log(s.key, s.value));

// Delete custom setting
await client.admin.settings.app.deleteSetting("billing.tiers");
```

#### Public Access (Non-Admin Users)

```typescript
// Users with regular tokens can access public settings
const userClient = new FluxbaseClient("https://api.myapp.com", "user_token");

// Get single public setting (respects RLS)
const betaEnabled = await userClient.settings.get("features.beta_enabled");
console.log(betaEnabled); // { enabled: true }

// Get multiple public settings
const publicValues = await userClient.settings.getMany([
  "features.beta_enabled",
  "features.dark_mode",
  "public.app_version",
  "internal.secret_key", // Will be filtered out by RLS
]);
console.log(publicValues);
// {
//   'features.beta_enabled': { enabled: true },
//   'features.dark_mode': { enabled: false },
//   'public.app_version': '1.0.0'
//   // 'internal.secret_key' is omitted by RLS
// }
```

## RLS Policies

### Creating Custom Policies

You can create custom RLS policies in user migrations to implement fine-grained access control.

#### Example: Users Can Only Read Specific Settings

```sql
-- In /migrations/user/001_custom_settings_policies.sql

-- Allow authenticated users to ONLY read feature flags
CREATE POLICY "Users can read feature flags only"
    ON app.settings
    FOR SELECT
    TO authenticated
    USING (
        key ~ '^features\.'  -- Key starts with 'features.'
        AND is_public = true
        AND is_secret = false
    );
```

#### Example: Category-Based Access

```sql
-- Only allow reading from 'custom' category
CREATE POLICY "Users can read custom settings only"
    ON app.settings
    FOR SELECT
    TO authenticated
    USING (
        category = 'custom'
        AND is_public = true
    );
```

#### Example: Restrictive Policy (AND Logic)

```sql
-- Use RESTRICTIVE policy to enforce multiple conditions
CREATE POLICY "Restrict anon to specific keys"
    ON app.settings
    AS RESTRICTIVE
    FOR SELECT
    TO anon
    USING (
        key IN ('features.beta_enabled', 'public.app_version')
        AND is_public = true
        AND is_secret = false
    );
```

#### Example: Metadata-Based Access

```sql
-- Use metadata field for advanced access control
CREATE POLICY "Users can read settings with public_api flag"
    ON app.settings
    FOR SELECT
    TO authenticated
    USING (
        (metadata->>'public_api')::boolean = true
        OR is_public = true
    );
```

## Backend Usage

### Go Backend

```go
import (
    "context"
    "time"
    "github.com/fluxbase/fluxbase/internal/auth"
)

// Initialize settings cache
settingsCache := auth.NewSettingsCache(settingsService, 5*time.Minute)

// Get boolean setting
enableSignup := settingsCache.GetBool(ctx, "app.auth.enable_signup", false)
if enableSignup {
    // Allow user registration
}

// Get integer setting
passwordMinLength := settingsCache.GetInt(ctx, "app.auth.password_min_length", 8)

// Get string setting
appVersion := settingsCache.GetString(ctx, "public.app_version", "1.0.0")

// Get JSON setting
type FeatureConfig struct {
    Enabled     bool   `json:"enabled"`
    Description string `json:"description"`
}

var config FeatureConfig
err := settingsCache.GetJSON(ctx, "features.config", &config)
if err != nil {
    log.Printf("Failed to get feature config: %v", err)
}

// Get multiple settings at once
settings, err := settingsCache.GetMany(ctx, []string{
    "features.beta_enabled",
    "features.dark_mode",
    "public.app_version",
})
if err != nil {
    log.Printf("Failed to get settings: %v", err)
}

// Environment Variable Override
// Set FLUXBASE_AUTH_ENABLE_SIGNUP=true to override database value
// Priority: Env Var > Cache > Database > Default
```

### Cache Management

```go
// Invalidate specific setting
settingsCache.Invalidate("app.auth.enable_signup")

// Invalidate all cached settings
settingsCache.InvalidateAll()

// Check if setting is overridden by env var
if settingsCache.IsOverriddenByEnv("app.auth.enable_signup") {
    log.Println("Setting is overridden by environment variable")
}

// Get environment variable name for a setting
envVar := settingsCache.GetEnvVarName("app.auth.enable_signup")
// Returns: "FLUXBASE_AUTH_ENABLE_SIGNUP"
```

## Migration Guide

### Migrating from Old Settings Tables

If you have existing data in the old settings tables, here's how to migrate:

```sql
-- Migration script to move data from old tables to app.settings

-- Migrate dashboard.auth_settings
INSERT INTO app.settings (key, value, category, description, created_at, updated_at)
SELECT
    key,
    value,
    'auth' as category,
    description,
    created_at,
    updated_at
FROM dashboard.auth_settings
ON CONFLICT (key) DO NOTHING;

-- Migrate dashboard.system_settings
INSERT INTO app.settings (key, value, category, description, created_at, updated_at)
SELECT
    key,
    value,
    'system' as category,
    description,
    created_at,
    updated_at
FROM dashboard.system_settings
ON CONFLICT (key) DO NOTHING;

-- Migrate dashboard.custom_settings
INSERT INTO app.settings (
    key, value, value_type, category, description,
    editable_by, metadata, created_by, updated_by,
    created_at, updated_at
)
SELECT
    key,
    value,
    value_type,
    'custom' as category,
    description,
    editable_by,
    metadata,
    created_by,
    updated_by,
    created_at,
    updated_at
FROM dashboard.custom_settings
ON CONFLICT (key) DO NOTHING;

-- After verifying migration, drop old tables
-- DROP TABLE dashboard.auth_settings;
-- DROP TABLE dashboard.system_settings;
-- DROP TABLE dashboard.custom_settings;
```

## Best Practices

### 1. Use Appropriate Categories

```typescript
// Framework settings (Fluxbase manages these)
category: "auth" | "system" | "storage" | "functions" | "realtime";

// Your custom settings
category: "custom";
```

### 2. Set Access Flags Correctly

```typescript
// Public feature flags (anyone can read)
is_public: true, is_secret: false

// Internal config (only authenticated users)
is_public: false, is_secret: false

// Sensitive data (only service_role)
is_public: false, is_secret: true
```

### 3. Use Meaningful Keys

```typescript
// Good - hierarchical, descriptive
"billing.tiers.api_quotas";
"features.beta.enabled";
"integrations.stripe.webhook_secret";

// Bad - flat, unclear
"config1";
"setting_123";
"data";
```

### 4. Add Descriptions

```typescript
await client.admin.settings.app.setSetting('billing.tiers', {...}, {
  description: 'API request quotas per billing tier (free/pro/enterprise)'
})
```

### 5. Use Metadata for Advanced Use Cases

```typescript
await client.admin.settings.app.setSetting('features.beta', {...}, {
  metadata: {
    public_api: true,
    version: '2.0',
    deprecated: false,
    rollout_percentage: 50
  }
})
```

## Examples

### Example 1: Feature Flag System

```typescript
// Create feature flags
await client.admin.settings.app.setSetting(
  "features.new_dashboard",
  { enabled: true, rollout: 100 },
  { is_public: true, description: "New dashboard UI" },
);

await client.admin.settings.app.setSetting(
  "features.ai_assistant",
  { enabled: false, rollout: 0 },
  { is_public: true, description: "AI-powered assistant" },
);

// Users check features in their app
const features = await userClient.settings.getMany([
  "features.new_dashboard",
  "features.ai_assistant",
]);

if (features["features.new_dashboard"]?.enabled) {
  // Show new dashboard
}
```

### Example 2: API Quotas

```typescript
// Set quotas
await client.admin.settings.app.setSetting('billing.quotas', {
  free: { requests: 1000, storage: 100 },
  pro: { requests: 100000, storage: 10000 },
  enterprise: { requests: -1, storage: -1 }
}, {
  description: 'API and storage quotas per plan tier'
})

// Backend checks quota
const quotas = await settingsCache.GetJSON(ctx, "billing.quotas", &quotaConfig)
userQuota := quotas[user.PlanTier]
```

### Example 3: A/B Testing Configuration

```typescript
// Configure A/B tests
await client.admin.settings.app.setSetting(
  "experiments.new_checkout",
  {
    enabled: true,
    variant_a: { traffic: 50, name: "Original" },
    variant_b: { traffic: 50, name: "Simplified" },
  },
  {
    is_public: true,
    metadata: { experiment_id: "exp_001", started_at: "2024-01-01" },
  },
);
```

## Troubleshooting

### Setting Not Accessible

**Problem**: User gets 403 when accessing a setting

**Solution**: Check RLS policies and access flags

```sql
-- Check setting access flags
SELECT key, is_public, is_secret, category
FROM app.settings
WHERE key = 'your.setting.key';

-- Check active policies
SELECT * FROM pg_policies
WHERE schemaname = 'app' AND tablename = 'settings';
```

### Setting Not Found

**Problem**: `getSetting()` returns 404

**Solution**: Verify setting exists and key is correct

```sql
-- List all settings
SELECT key, category FROM app.settings ORDER BY key;

-- Search for similar keys
SELECT key FROM app.settings WHERE key LIKE '%search_term%';
```

### Cache Not Updating

**Problem**: Backend shows old value after update

**Solution**: Invalidate cache

```go
settingsCache.Invalidate("app.auth.enable_signup")
// or
settingsCache.InvalidateAll()
```

## Related Documentation

- [RLS Guide](../security/rls-guide.md)
- [User Migrations](../migrations/user-migrations.md)
- [SDK Reference](../sdk/settings.md)
